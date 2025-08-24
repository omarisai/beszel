package agent

import (
	"beszel/internal/entities/system"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/shirou/gopsutil/v4/common"
	"github.com/shirou/gopsutil/v4/sensors"
)

type SensorConfig struct {
	context        context.Context
	sensors        map[string]struct{}
	genericSensors map[string]GenericSensorConfig
	primarySensor  string
	isBlacklist    bool
	hasWildcards   bool
	skipCollection bool
}

type GenericSensorConfig struct {
	Name    string
	Unit    string
	Maximum float64
	Minimum float64
}

func (a *Agent) newSensorConfig() *SensorConfig {
	primarySensor, _ := GetEnv("PRIMARY_SENSOR")
	sysSensors, _ := GetEnv("SYS_SENSORS")
	sensorsEnvVal, sensorsSet := GetEnv("SENSORS")
	skipCollection := sensorsSet && sensorsEnvVal == ""

	return a.newSensorConfigWithEnv(primarySensor, sysSensors, sensorsEnvVal, skipCollection)
}

// Matches sensors.TemperaturesWithContext to allow for panic recovery (gopsutil/issues/1832)
type getTempsFn func(ctx context.Context) ([]sensors.TemperatureStat, error)

// newSensorConfigWithEnv creates a SensorConfig with the provided environment variables
// sensorsSet indicates if the SENSORS environment variable was explicitly set (even to empty string)
func (a *Agent) newSensorConfigWithEnv(primarySensor, sysSensors, sensorsEnvVal string, skipCollection bool) *SensorConfig {
	config := &SensorConfig{
		context:        context.Background(),
		primarySensor:  primarySensor,
		skipCollection: skipCollection,
		sensors:        make(map[string]struct{}),
		genericSensors: make(map[string]GenericSensorConfig),
	}

	// Set sensors context (allows overriding sys location for sensors)
	if sysSensors != "" {
		slog.Info("SYS_SENSORS", "path", sysSensors)
		config.context = context.WithValue(config.context,
			common.EnvKey, common.EnvMap{common.HostSysEnvKey: sysSensors},
		)
	}

	// handle blacklist
	if strings.HasPrefix(sensorsEnvVal, "-") {
		config.isBlacklist = true
		sensorsEnvVal = sensorsEnvVal[1:]
	}

	for sensor := range strings.SplitSeq(sensorsEnvVal, ",") {
		sensor = strings.TrimSpace(sensor)
		if sensor != "" {
			// Check if it's new generic sensor format
			if strings.HasPrefix(sensor, "(") && strings.HasSuffix(sensor, ")") {
				if err := config.parseGenericSensor(sensor); err != nil {
					slog.Warn("Invalid generic sensor format", "sensor", sensor, "err", err)
					continue
				}
			} else {
				// Existing temperature sensor logic
				config.sensors[sensor] = struct{}{}
				if strings.Contains(sensor, "*") {
					config.hasWildcards = true
				}
			}
		}
	}

	return config
}

// parseGenericSensor parses a generic sensor configuration in the format "(name,unit,maximum,minimum)"
func (config *SensorConfig) parseGenericSensor(sensor string) error {
	// Remove parentheses
	content := sensor[1 : len(sensor)-1]
	parts := strings.Split(content, ",")
	if len(parts) != 4 {
		return fmt.Errorf("expected 4 parts (name,unit,maximum,minimum), got %d", len(parts))
	}

	name := strings.TrimSpace(parts[0])
	unit := strings.TrimSpace(parts[1])
	maximumStr := strings.TrimSpace(parts[2])
	minimumStr := strings.TrimSpace(parts[3])

	if name == "" {
		return fmt.Errorf("sensor name cannot be empty")
	}
	if unit == "" {
		return fmt.Errorf("sensor unit cannot be empty")
	}

	maximum, err := strconv.ParseFloat(maximumStr, 64)
	if err != nil {
		return fmt.Errorf("invalid maximum value '%s': %w", maximumStr, err)
	}

	minimum, err := strconv.ParseFloat(minimumStr, 64)
	if err != nil {
		return fmt.Errorf("invalid minimum value '%s': %w", minimumStr, err)
	}

	if minimum >= maximum {
		return fmt.Errorf("minimum value (%f) must be less than maximum value (%f)", minimum, maximum)
	}

	config.genericSensors[name] = GenericSensorConfig{
		Name:    name,
		Unit:    unit,
		Maximum: maximum,
		Minimum: minimum,
	}

	slog.Info("Configured generic sensor", "name", name, "unit", unit, "min", minimum, "max", maximum)
	return nil
}

// updateTemperatures updates the agent with the latest sensor temperatures
func (a *Agent) updateTemperatures(systemStats *system.Stats) {
	// skip if sensors whitelist is set to empty string
	if a.sensorConfig.skipCollection {
		slog.Debug("Skipping temperature collection")
		return
	}

	// reset high temp
	a.systemInfo.DashboardTemp = 0

	temps, err := a.getTempsWithPanicRecovery(getSensorTemps)
	if err != nil {
		// retry once on panic (gopsutil/issues/1832)
		temps, err = a.getTempsWithPanicRecovery(getSensorTemps)
		if err != nil {
			slog.Warn("Error updating temperatures", "err", err)
			if len(systemStats.Temperatures) > 0 {
				systemStats.Temperatures = make(map[string]float64)
			}
			return
		}
	}
	slog.Debug("Temperature", "sensors", temps)

	// return if no sensors
	if len(temps) == 0 {
		return
	}

	systemStats.Temperatures = make(map[string]float64, len(temps))
	for i, sensor := range temps {
		// check for malformed strings on darwin (gopsutil/issues/1832)
		if runtime.GOOS == "darwin" && !utf8.ValidString(sensor.SensorKey) {
			continue
		}

		// scale temperature
		if sensor.Temperature != 0 && sensor.Temperature < 1 {
			sensor.Temperature = scaleTemperature(sensor.Temperature)
		}
		// skip if temperature is unreasonable
		if sensor.Temperature <= 0 || sensor.Temperature >= 200 {
			continue
		}
		sensorName := sensor.SensorKey
		if _, ok := systemStats.Temperatures[sensorName]; ok {
			// if key already exists, append int to key
			sensorName = sensorName + "_" + strconv.Itoa(i)
		}
		// skip if not in whitelist or blacklist
		if !isValidSensor(sensorName, a.sensorConfig) {
			continue
		}
		// set dashboard temperature
		switch a.sensorConfig.primarySensor {
		case "":
			a.systemInfo.DashboardTemp = max(a.systemInfo.DashboardTemp, sensor.Temperature)
		case sensorName:
			a.systemInfo.DashboardTemp = sensor.Temperature
		}
		systemStats.Temperatures[sensorName] = twoDecimals(sensor.Temperature)
	}
}

// updateGenericSensors updates the agent with the latest generic sensor data
func (a *Agent) updateGenericSensors(systemStats *system.Stats) {
	// Skip if no generic sensors are configured
	if len(a.sensorConfig.genericSensors) == 0 {
		return
	}

	// Initialize the map if needed
	if systemStats.GenericSensors == nil {
		systemStats.GenericSensors = make(map[string]system.SensorData)
	}

	// Collect data for each configured generic sensor
	for name, config := range a.sensorConfig.genericSensors {
		value, err := a.collectGenericSensorValue(name, config)
		if err != nil {
			slog.Warn("Failed to collect generic sensor data", "sensor", name, "err", err)
			continue
		}

		// Validate the value is within the configured range
		if value < config.Minimum || value > config.Maximum {
			slog.Warn("Generic sensor value out of range", "sensor", name, "value", value, "min", config.Minimum, "max", config.Maximum)
			continue
		}

		systemStats.GenericSensors[name] = system.SensorData{
			Value: twoDecimals(value),
			Unit:  config.Unit,
			Min:   config.Minimum,
			Max:   config.Maximum,
		}
	}
}

// collectGenericSensorValue collects the current value for a generic sensor
// It reads the value from the corresponding file in /generic-sensors/
func (a *Agent) collectGenericSensorValue(sensorName string, config GenericSensorConfig) (float64, error) {
	// Look for sensor file in /generic-sensors/
	sensorPath := filepath.Join("/generic-sensors", sensorName)
	
	// Check if the sensor file exists
	if _, err := os.Stat(sensorPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("sensor file not found at %s - create a file or symlink with the sensor value", sensorPath)
	}
	
	// Read the sensor value from the file
	value, err := ReadSensorFromFile(sensorPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read sensor '%s' from %s: %w", sensorName, sensorPath, err)
	}
	
	return value, nil
}

// Helper functions for implementing custom sensor collection

// ReadSensorFromFile reads a numeric value from a file path (useful for Linux sysfs sensors)
func ReadSensorFromFile(filePath string) (float64, error) {
	// Read the file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read sensor file %s: %w", filePath, err)
	}

	// Parse the numeric value
	valueStr := strings.TrimSpace(string(data))
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse sensor value '%s' from %s: %w", valueStr, filePath, err)
	}

	return value, nil
}

// GetGenericSensorNames returns the names of all configured generic sensors
func (a *Agent) GetGenericSensorNames() []string {
	names := make([]string, 0, len(a.sensorConfig.genericSensors))
	for name := range a.sensorConfig.genericSensors {
		names = append(names, name)
	}
	return names
}

// NewSensorConfigWithEnv creates a SensorConfig with the provided environment variables (exported for testing)
func (a *Agent) NewSensorConfigWithEnv(primarySensor, sysSensors, sensorsEnvVal string, skipCollection bool) *SensorConfig {
	return a.newSensorConfigWithEnv(primarySensor, sysSensors, sensorsEnvVal, skipCollection)
}

// GetTemperatureSensors returns the configured temperature sensors
func (config *SensorConfig) GetTemperatureSensors() map[string]struct{} {
	return config.sensors
}

// GetGenericSensors returns the configured generic sensors
func (config *SensorConfig) GetGenericSensors() map[string]GenericSensorConfig {
	return config.genericSensors
}

// getTempsWithPanicRecovery wraps sensors.TemperaturesWithContext to recover from panics (gopsutil/issues/1832)
func (a *Agent) getTempsWithPanicRecovery(getTemps getTempsFn) (temps []sensors.TemperatureStat, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	// get sensor data (error ignored intentionally as it may be only with one sensor)
	temps, _ = getTemps(a.sensorConfig.context)
	return
}

// isValidSensor checks if a sensor is valid based on the sensor name and the sensor config
func isValidSensor(sensorName string, config *SensorConfig) bool {
	// Check if it's a configured generic sensor
	if _, exists := config.genericSensors[sensorName]; exists {
		return true
	}

	// if no sensors configured, everything is valid
	if len(config.sensors) == 0 {
		return true
	}

	// Exact match - return true if whitelist, false if blacklist
	if _, exactMatch := config.sensors[sensorName]; exactMatch {
		return !config.isBlacklist
	}

	// If no wildcards, return true if blacklist, false if whitelist
	if !config.hasWildcards {
		return config.isBlacklist
	}

	// Check for wildcard patterns
	for pattern := range config.sensors {
		if !strings.Contains(pattern, "*") {
			continue
		}
		if match, _ := path.Match(pattern, sensorName); match {
			return !config.isBlacklist
		}
	}

	return config.isBlacklist
}

// scaleTemperature scales temperatures in fractional values to reasonable Celsius values
func scaleTemperature(temp float64) float64 {
	if temp > 1 {
		return temp
	}
	scaled100 := temp * 100
	scaled1000 := temp * 1000

	if scaled100 >= 15 && scaled100 <= 95 {
		return scaled100
	} else if scaled1000 >= 15 && scaled1000 <= 95 {
		return scaled1000
	}
	return scaled100
}
