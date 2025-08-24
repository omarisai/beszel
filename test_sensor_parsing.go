package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type GenericSensorData struct {
	V   float64 `json:"v"`   // Value
	U   string  `json:"u"`   // Unit
	Min *float64 `json:"min"` // Minimum value (optional)
	Max *float64 `json:"max"` // Maximum value (optional)
}

func readGenericSensors(sensorDir string) map[string]GenericSensorData {
	sensors := make(map[string]GenericSensorData)
	
	// Check if sensor directory exists
	if _, err := os.Stat(sensorDir); os.IsNotExist(err) {
		return sensors
	}
	
	// Read all files in the sensor directory
	files, err := os.ReadDir(sensorDir)
	if err != nil {
		return sensors
	}
	
	metadataPattern := regexp.MustCompile(`^\(([^,]+),([^,]+),([^,]+),([^,]+)\)$`)
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		filePath := filepath.Join(sensorDir, file.Name())
		f, err := os.Open(filePath)
		if err != nil {
			continue
		}
		
		scanner := bufio.NewScanner(f)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, strings.TrimSpace(scanner.Text()))
		}
		f.Close()
		
		if len(lines) == 0 {
			continue
		}
		
		var sensorData GenericSensorData
		var valueStr string
		var sensorName string
		
		// Check if first line contains metadata
		if metadataPattern.MatchString(lines[0]) {
			matches := metadataPattern.FindStringSubmatch(lines[0])
			if len(matches) == 5 {
				sensorName = matches[1]
				sensorData.U = matches[2]
				
				// Parse max value
				if maxVal, err := strconv.ParseFloat(matches[3], 64); err == nil {
					sensorData.Max = &maxVal
				}
				
				// Parse min value
				if minVal, err := strconv.ParseFloat(matches[4], 64); err == nil {
					sensorData.Min = &minVal
				}
			}
			
			// Value should be on the second line
			if len(lines) > 1 {
				valueStr = lines[1]
			}
		} else {
			// No metadata, first line is the value
			valueStr = lines[0]
			sensorName = file.Name()
			sensorData.U = "" // No unit specified
		}
		
		// Parse the sensor value
		if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
			sensorData.V = value
			sensors[sensorName] = sensorData
		}
	}
	
	return sensors
}

func main() {
	sensors := readGenericSensors("/tmp/generic-sensors")
	
	fmt.Println("Generic Sensors Data:")
	for name, data := range sensors {
		fmt.Printf("  %s: %.2f %s", name, data.V, data.U)
		if data.Min != nil && data.Max != nil {
			fmt.Printf(" (range: %.1f-%.1f)", *data.Min, *data.Max)
		}
		fmt.Println()
	}
}
