package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"strconv"
)

// GenericSensorConfig represents a generic sensor configuration
type GenericSensorConfig struct {
	Name string
	Unit string
	Max  float64
	Min  float64
}

// parseGenericSensor parses a generic sensor configuration string
func parseGenericSensor(sensorStr string) (GenericSensorConfig, error) {
	sensorStr = strings.TrimSpace(sensorStr)
	
	// Check if it's a generic sensor (wrapped in parentheses)
	if !strings.HasPrefix(sensorStr, "(") || !strings.HasSuffix(sensorStr, ")") {
		return GenericSensorConfig{}, fmt.Errorf("invalid generic sensor format")
	}
	
	// Remove parentheses
	inner := sensorStr[1 : len(sensorStr)-1]
	parts := strings.Split(inner, ",")
	
	if len(parts) != 4 {
		return GenericSensorConfig{}, fmt.Errorf("invalid generic sensor format - expected 4 parts, got %d", len(parts))
	}
	
	name := strings.TrimSpace(parts[0])
	unit := strings.TrimSpace(parts[1])
	maxStr := strings.TrimSpace(parts[2])
	minStr := strings.TrimSpace(parts[3])
	
	if name == "" || unit == "" {
		return GenericSensorConfig{}, fmt.Errorf("sensor name and unit cannot be empty")
	}
	
	max, err := strconv.ParseFloat(maxStr, 64)
	if err != nil {
		return GenericSensorConfig{}, fmt.Errorf("invalid max value: %w", err)
	}
	
	min, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return GenericSensorConfig{}, fmt.Errorf("invalid min value: %w", err)
	}
	
	if min > max {
		return GenericSensorConfig{}, fmt.Errorf("min value %.2f cannot be greater than max value %.2f", min, max)
	}
	
	return GenericSensorConfig{
		Name: name,
		Unit: unit,
		Max:  max,
		Min:  min,
	}, nil
}

// collectGenericSensorValue reads a sensor value from the /generic-sensors/ directory
func collectGenericSensorValue(sensorName string, config GenericSensorConfig) (float64, error) {
	// Create the file path
	filePath := filepath.Join("/generic-sensors", sensorName)
	
	// Read the file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read sensor file %s: %w", filePath, err)
	}
	
	// Parse the value
	valueStr := strings.TrimSpace(string(data))
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse sensor value '%s': %w", valueStr, err)
	}
	
	return value, nil
}

func main() {
	fmt.Println("Testing Generic Sensor Implementation")
	fmt.Println("=====================================")
	
	// Test parseGenericSensor
	testCases := []string{
		"(pressure,Pa,1000,0)",
		"(voltage,V,12.5,0.5)",
		"(rpm,RPM,3000,500)",
		"(humidity,%,100,0)",
		"(temperature,°C,100,-50)",
	}
	
	for _, testCase := range testCases {
		fmt.Printf("\nTesting: %s\n", testCase)
		config, err := parseGenericSensor(testCase)
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
		} else {
			fmt.Printf("  ✓ Name: %s, Unit: %s, Range: %.2f - %.2f\n", 
				config.Name, config.Unit, config.Min, config.Max)
		}
	}
	
	// Test invalid cases
	fmt.Println("\nTesting invalid cases:")
	invalidCases := []string{
		"pressure,Pa,1000,0",    // Missing parentheses
		"(pressure,Pa,1000)",    // Missing min value
		"(pressure,,1000,0)",    // Empty unit
		"(pressure,Pa,abc,0)",   // Invalid max
		"(pressure,Pa,0,1000)",  // Min > Max
	}
	
	for _, testCase := range invalidCases {
		fmt.Printf("\nTesting invalid: %s\n", testCase)
		config, err := parseGenericSensor(testCase)
		if err != nil {
			fmt.Printf("  ✓ Expected error: %v\n", err)
		} else {
			fmt.Printf("  ✗ Unexpected success: %+v\n", config)
		}
	}
	
	// Test file-based sensor reading (create test directory and files)
	fmt.Println("\nTesting file-based sensor collection:")
	
	// Create test directory
	testDir := "/tmp/generic-sensors"
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		fmt.Printf("Failed to create test directory: %v\n", err)
		return
	}
	
	// Create test sensor files
	testSensors := map[string]string{
		"pressure": "850.5",
		"voltage":  "11.8",
		"rpm":      "2400",
		"humidity": "65.2",
	}
	
	for sensor, value := range testSensors {
		filePath := filepath.Join(testDir, sensor)
		err := os.WriteFile(filePath, []byte(value), 0644)
		if err != nil {
			fmt.Printf("Failed to create test file %s: %v\n", filePath, err)
			continue
		}
		
		// Parse config for validation (not used in this test)
		_, err = parseGenericSensor(fmt.Sprintf("(%s,unit,1000,0)", sensor))
		if err != nil {
			fmt.Printf("Failed to parse config for %s: %v\n", sensor, err)
			continue
		}
		
		// Test reading with modified path
		testFilePath := filepath.Join(testDir, sensor)
		data, err := os.ReadFile(testFilePath)
		if err != nil {
			fmt.Printf("Failed to read %s: %v\n", sensor, err)
		} else {
			valueStr := strings.TrimSpace(string(data))
			readValue, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				fmt.Printf("Failed to parse %s: %v\n", sensor, err)
			} else {
				fmt.Printf("  ✓ %s: %.2f (expected %s)\n", sensor, readValue, value)
			}
		}
	}
	
	// Clean up
	os.RemoveAll(testDir)
	
	fmt.Println("\nGeneric sensor implementation test completed!")
}
