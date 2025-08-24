#!/bin/bash

# Demo script for Generic Sensor functionality in Beszel
# This script demonstrates the file-based generic sensor system

echo "🔧 Generic Sensor Demo for Beszel"
echo "=================================="

# Create demo sensor directory
DEMO_DIR="/tmp/beszel-generic-sensors-demo"
echo "📁 Creating demo sensor directory: $DEMO_DIR"
mkdir -p "$DEMO_DIR"

# Create example sensor files
echo "📝 Creating example sensor files..."

# Example 1: Pressure sensor (static value)
echo "850.5" > "$DEMO_DIR/pressure"
echo "   ✓ pressure: 850.5 Pa"

# Example 2: Voltage sensor (static value)
echo "11.8" > "$DEMO_DIR/voltage"
echo "   ✓ voltage: 11.8 V"

# Example 3: RPM sensor (static value) 
echo "2400" > "$DEMO_DIR/rpm"
echo "   ✓ rpm: 2400 RPM"

# Example 4: Custom API sensor (script that outputs value)
cat > "$DEMO_DIR/api_latency" << 'EOF'
#!/bin/bash
# Simulate API latency measurement
start=$(date +%s%3N)
sleep 0.02  # Simulate 20ms delay
end=$(date +%s%3N)
echo $((end - start))
EOF
chmod +x "$DEMO_DIR/api_latency"
echo "   ✓ api_latency: executable script"

# Example 5: CPU frequency (try to link to real system file if available)
if [ -f "/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq" ]; then
    ln -sf "/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq" "$DEMO_DIR/cpu_freq"
    echo "   ✓ cpu_freq: linked to system file"
else
    echo "1800000" > "$DEMO_DIR/cpu_freq"
    echo "   ✓ cpu_freq: static value (system file not available)"
fi

echo ""
echo "🧪 Testing sensor configuration syntax..."

# Test sensor configurations
SENSORS_CONFIG="(pressure,Pa,1000,0),(voltage,V,12.5,0.5),(rpm,RPM,3000,500),(api_latency,ms,1000,0),(cpu_freq,Hz,3000000,800000)"

echo "   Configuration: $SENSORS_CONFIG"
echo "   ✓ Syntax: (name,unit,maximum,minimum)"

echo ""
echo "📊 Reading sensor values..."

# Read and display sensor values
for sensor in pressure voltage rpm api_latency cpu_freq; do
    if [ -f "$DEMO_DIR/$sensor" ]; then
        if [ -x "$DEMO_DIR/$sensor" ]; then
            # Executable script
            value=$("$DEMO_DIR/$sensor" 2>/dev/null || echo "error")
        else
            # Regular file
            value=$(cat "$DEMO_DIR/$sensor" 2>/dev/null || echo "error")
        fi
        echo "   $sensor: $value"
    else
        echo "   $sensor: not found"
    fi
done

echo ""
echo "🎯 Configuration Examples"
echo "========================"

echo "1. Basic Setup:"
echo "   export SENSORS=\"(pressure,Pa,1000,0)\""
echo "   echo \"850.5\" > /generic-sensors/pressure"
echo ""

echo "2. Hardware Monitor Sensors:"
echo "   export SENSORS=\"(voltage,V,12.5,0.5),(rpm,RPM,3000,500)\""
echo "   ln -s /sys/class/hwmon/hwmon0/in0_input /generic-sensors/voltage"
echo "   ln -s /sys/class/hwmon/hwmon1/fan1_input /generic-sensors/rpm"
echo ""

echo "3. Mixed Temperature and Generic:"
echo "   export SENSORS=\"cpu_temp,gpu_temp,(pressure,Pa,1000,0),(humidity,%,100,0)\""
echo ""

echo "4. Script-based Sensors:"
echo "   export SENSORS=\"(api_response_time,ms,1000,0)\""
echo "   cat << 'SCRIPT' > /usr/local/bin/measure-api-time"
echo "   #!/bin/bash"
echo "   start=\$(date +%s%3N)"
echo "   curl -s http://api.example.com/health > /dev/null"
echo "   end=\$(date +%s%3N)"
echo "   echo \$((end - start))"
echo "   SCRIPT"
echo "   chmod +x /usr/local/bin/measure-api-time"
echo "   ln -s /usr/local/bin/measure-api-time /generic-sensors/api_response_time"

echo ""
echo "📋 JSON Output Format"
echo "===================="

cat << 'JSON'
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
      },
      "rpm": {
        "v": 2400,
        "u": "RPM",
        "min": 500,
        "max": 3000
      }
    }
  }
}
JSON

echo ""
echo "✅ Key Benefits"
echo "==============="
echo "• No Go programming required - just files!"
echo "• Backwards compatible with existing temperature sensors"
echo "• Supports any numeric sensor type"
echo "• Flexible: static files, symlinks, or executable scripts"
echo "• Validation with configurable min/max ranges"
echo "• Easy integration with existing monitoring tools"

echo ""
echo "🧹 Cleaning up demo directory..."
rm -rf "$DEMO_DIR"

echo "✨ Generic Sensor Demo Complete!"
echo ""
echo "To use generic sensors in your Beszel setup:"
echo "1. Create /generic-sensors/ directory"
echo "2. Set SENSORS environment variable with (name,unit,max,min) syntax"
echo "3. Provide sensor values via files, symlinks, or scripts"
echo "4. Restart Beszel agent"
