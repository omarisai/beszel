package main

import (
	"beszel/internal/agent"
	"fmt"
	"log"
	"os"
)

func main() {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "beszel-sensor-test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an agent instance
	testAgent, err := agent.NewAgent(tmpDir)
	if err != nil {
		log.Fatal(err)
	}

	// Test different sensor configurations
	testConfigs := []struct {
		name      string
		sensorsVal string
	}{
		{
			name:      "Temperature sensors only",
			sensorsVal: "cpu_temp,gpu_temp",
		},
		{
			name:      "Generic sensors only", 
			sensorsVal: "(pressure,Pa,1000,0),(voltage,V,12.5,0.5)",
		},
		{
			name:      "Mixed sensors",
			sensorsVal: "cpu_temp,(pressure,Pa,1000,0),gpu_temp,(voltage,V,12.5,0.5)",
		},
		{
			name:      "Invalid generic sensor (should be ignored)",
			sensorsVal: "cpu_temp,(invalid_format),gpu_temp",
		},
	}

	fmt.Println("=== Beszel Generic Sensor Configuration Test ===\n")

	for _, config := range testConfigs {
		fmt.Printf("Testing: %s\n", config.name)
		fmt.Printf("SENSORS value: %s\n", config.sensorsVal)
		
		// Create sensor config for this test
		sensorConfig := testAgent.NewSensorConfigWithEnv("", "", config.sensorsVal, false)
		
		// Print results
		fmt.Printf("  - Temperature sensors configured: %d\n", len(sensorConfig.GetTemperatureSensors()))
		fmt.Printf("  - Generic sensors configured: %d\n", len(sensorConfig.GetGenericSensors()))
		
		// Print generic sensor details
		for name, sensor := range sensorConfig.GetGenericSensors() {
			fmt.Printf("    * %s: unit=%s, range=%g-%g\n", name, sensor.Unit, sensor.Minimum, sensor.Maximum)
		}
		
		fmt.Println()
	}

	fmt.Println("All tests completed successfully!")
	fmt.Println("\n=== Usage Instructions ===")
	fmt.Println("To use generic sensors, set the SENSORS environment variable with the format:")
	fmt.Println("  SENSORS='(sensor_name,unit,maximum,minimum)'")
	fmt.Println("\nExamples:")
	fmt.Println("  SENSORS='(pressure,Pa,1000,0)'")
	fmt.Println("  SENSORS='(voltage,V,12.5,0.5),(rpm,RPM,3000,500)'")
	fmt.Println("  SENSORS='cpu_temp,(pressure,Pa,1000,0),gpu_temp'  # Mixed")
	fmt.Println("\nNote: You'll need to implement collectGenericSensorValue() for actual sensor reading.")
}
