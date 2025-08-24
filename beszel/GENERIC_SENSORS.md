# Generic Sensor Support in Beszel

Beszel now supports generic sensors in addition to temperature sensors. This allows you to monitor custom sensors with different units and value ranges using a simple file-based approach.

## Configuration Syntax

### Temperature Sensors (Existing)
```bash
SENSORS="cpu_temp,gpu_temp,*acpi*"
```

### Generic Sensors (New)
```bash
SENSORS="(sensor_name,unit,maximum,minimum)"
```

### Mixed Configuration
```bash
SENSORS="cpu_temp,(pressure,Pa,1000,0),gpu_temp,(voltage,V,12.5,0.5)"
```

## File-Based Sensor System

Generic sensors read their values from files in the `/generic-sensors/` directory. Each sensor corresponds to a file with the same name as the sensor.

### Directory Structure
```
/generic-sensors/
├── pressure          # File containing pressure value
├── voltage           # File containing voltage value
├── rpm               # File containing RPM value
└── humidity          # File containing humidity value
```

## Examples

### Single Generic Sensor
```bash
# Configure pressure sensor
export SENSORS="(pressure,Pa,1000,0)"

# Create sensor file
echo "850.5" > /generic-sensors/pressure
# OR create symlink to actual sensor file
ln -s /sys/class/hwmon/hwmon0/pressure1_input /generic-sensors/pressure
```

### Multiple Generic Sensors
```bash
# Configure multiple sensors
export SENSORS="(pressure,Pa,1000,0),(voltage,V,12.5,0.5)"

# Create sensor files
echo "850.5" > /generic-sensors/pressure
echo "11.8" > /generic-sensors/voltage
```

### Mixed Temperature and Generic Sensors
```bash
# Temperature sensors + custom sensors
export SENSORS="cpu_temp,gpu_temp,(rpm,RPM,3000,500),(pressure,Pa,1000,0)"

# Create generic sensor files
echo "2400" > /generic-sensors/rpm
ln -s /sys/class/hwmon/hwmon0/pressure1_input /generic-sensors/pressure
```

### Using Symlinks to System Sensors
```bash
# Link to hardware monitor files
ln -s /sys/class/hwmon/hwmon0/in0_input /generic-sensors/voltage
ln -s /sys/class/hwmon/hwmon1/fan1_input /generic-sensors/rpm
ln -s /sys/class/hwmon/hwmon2/humidity1_input /generic-sensors/humidity

# Configure sensors
export SENSORS="(voltage,V,12.5,0.5),(rpm,RPM,3000,500),(humidity,%,100,0)"
```

## Common Sensor File Locations

### Linux Hardware Monitoring (hwmon)
```bash
# Voltage sensors
/sys/class/hwmon/hwmon*/in*_input

# Fan RPM sensors  
/sys/class/hwmon/hwmon*/fan*_input

# Power sensors
/sys/class/hwmon/hwmon*/power*_input

# Current sensors
/sys/class/hwmon/hwmon*/curr*_input
```

### CPU Frequency
```bash
/sys/devices/system/cpu/cpu*/cpufreq/scaling_cur_freq
```

### Custom Script Output
```bash
# Create script that outputs sensor value
#!/bin/bash
echo "42.5"  # Your sensor reading logic here

# Make executable and link
chmod +x /usr/local/bin/my-sensor-script
ln -s /usr/local/bin/my-sensor-script /generic-sensors/custom_sensor
```

## Setup Instructions

### 1. Create Generic Sensors Directory
```bash
sudo mkdir -p /generic-sensors
sudo chmod 755 /generic-sensors
```

### 2. Configure Sensors
Set your sensor configuration:
```bash
export SENSORS="(pressure,Pa,1000,0),(voltage,V,12.5,0.5)"
```

### 3. Provide Sensor Values

**Option A: Direct Files**
```bash
# Static value
echo "850.5" > /generic-sensors/pressure

# Updated by external script
echo "11.8" > /generic-sensors/voltage
```

**Option B: Symlinks to System Files**
```bash
# Link to hardware sensors
ln -s /sys/class/hwmon/hwmon0/in0_input /generic-sensors/voltage
ln -s /sys/class/hwmon/hwmon1/fan1_input /generic-sensors/rpm
```

**Option C: Script Output**
```bash
# Create executable script
cat << 'EOF' > /usr/local/bin/read-pressure
#!/bin/bash
# Your sensor reading logic
curl -s http://192.168.1.100/api/pressure | jq -r '.value'
EOF
chmod +x /usr/local/bin/read-pressure
ln -s /usr/local/bin/read-pressure /generic-sensors/pressure
```

## Data Structure

Generic sensors are stored separately from temperature sensors in the system stats:

```json
{
  "stats": {
    "t": {
      "cpu_temp": 45.2,
      "gpu_temp": 38.7
    },
    "gs": {
      "pressure": {
        "v": 850.5,
        "u": "Pa",
        "min": 0,
        "max": 1000
      },
      "voltage": {
        "v": 11.8,
        "u": "V", 
        "min": 0.5,
        "max": 12.5
      }
    }
  }
}
```

## Validation

- Sensor values are validated against the configured min/max range
- Values outside the range are logged as warnings and excluded
- Invalid configuration formats are logged and ignored
- The format must be exactly: `(name,unit,maximum,minimum)`

## Backward Compatibility

- Existing temperature sensor configurations work unchanged
- Temperature sensors continue to use the `t` field in JSON/CBOR
- Generic sensors use the new `gs` field
- No breaking changes to existing functionality

## Error Handling

- Invalid generic sensor formats are logged and skipped
- Sensor collection errors are logged but don't stop other sensors
- Out-of-range values are logged as warnings
- Missing sensor implementations return helpful error messages

## Helper Functions

The following helper functions are available:

```go
// Get list of configured generic sensor names
sensorNames := agent.GetGenericSensorNames()

// Access sensor configuration
config := agent.NewSensorConfigWithEnv("", "", sensorsEnvVal, false)
genericSensors := config.GetGenericSensors()
tempSensors := config.GetTemperatureSensors()
```

## Usage Tips

1. **Sensor Names**: Use descriptive names without spaces or special characters
2. **Units**: Use standard unit abbreviations (Pa, V, RPM, etc.)
3. **Ranges**: Set realistic min/max values for proper validation
4. **Implementation**: Start with file-based sensors for easy testing
5. **Testing**: Use environment variables to test different configurations

## Next Steps

1. Implement `collectGenericSensorValue` for your specific sensors
2. Test with simple file-based sensors first
3. Add error handling for your sensor sources
4. Consider implementing the `ReadSensorFromFile` helper for common use cases
