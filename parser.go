// El Parsor
// Parse some tings
package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type BuoyReading struct {
	WindSpeed     *float64
	WindDirection *float64
	WaveHeight    *float64
	WavePeriod    *float64
	WaterTemp     *float64
	WaveDirection *float64
}

func parseRealtime2(data string, rowLimit int) ([]BuoyReading, error) {
	var readings []BuoyReading
	scanner := bufio.NewScanner(strings.NewReader(data))
	rowCounter := 0

	for scanner.Scan() {
		// NOTE: We could use a timestamp filter instead
		if rowLimit != 0 && rowCounter == rowLimit {
			break
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue // Skip becase these are the headers
		}
		fields := strings.Fields(line)

		if len(fields) < columnLength {
			continue // skip messed up line
		}

		reading := BuoyReading{
			WindSpeed:     parseDataPoint(fields[colWindSpeed]),
			WindDirection: parseDataPoint(fields[colWindDir]),
			WaveHeight:    parseDataPoint(fields[colWaveHeight]),
			WavePeriod:    parseDataPoint(fields[colAvgPeriod]),
			WaterTemp:     parseDataPoint(fields[colWaterTemp]),
			WaveDirection: parseDataPoint(fields[colMeanWaveDir]),
		}

		readings = append(readings, reading)
		rowCounter++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning buoy data: %w", err)
	}
	return readings, nil
}

func parseDataPoint(s string) *float64 {
	if s == "MM" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// Break out todays summary and the full fors
func parseForecastSummary(data string) (forcastSummary, error) {
	// Finds pattern of the next section header (.TONIGHT..., .SAT..., etc).
	re := regexp.MustCompile(`(?s)\.\.\.(.*?)\n\.`)
	m := re.FindStringSubmatch(data)
	todaySummary := ""
	if m != nil {
		todaySummary = strings.TrimSpace(m[1])
	}

	return forcastSummary{Today: todaySummary, Full: data}, nil
}

func parseBuoyWebpage(html, baseURL string) (buoyImageData, error) {
	var image360Url *string
	re := regexp.MustCompile(`/images/buoycam/[^"]+`)
	match := re.FindString(html)

	if match != "" {
		url := baseURL + match
		image360Url = &url
	}

	return buoyImageData{Image360Url: image360Url}, nil

}
