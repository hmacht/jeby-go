// El Parsor
// Parse some tings
package main

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// These are the Realtime2 columns.
// YY  MM DD hh mm WDIR WSPD GST  WVHT   DPD   APD MWD   PRES  ATMP  WTMP  DEWP  VIS PTDY  TIDE
// yr  mo dy hr mn degT m/s  m/s     m   sec   sec degT   hPa  degC  degC  degC  nmi  hPa    ft
const (
	colYear = iota
	colMonth
	colDay
	colHour
	colMinute
	colWindDir
	colWindSpeed
	colGust
	colWaveHeight
	colDomPeriod
	colAvgPeriod
	colMeanWaveDir
	colPressure
	colAirTemp
	colWaterTemp
	colDewpoint
	colVisibility
	colPressureTendency
	colTide
)

// The minimum number of columns we need (through water temp).
const columnLength = 18

type BuoyReading struct {
	Time time.Time

	WindSpeed     *float64
	WindDirection *float64
	WaveHeight    *float64
	WavePeriod    *float64
	WaterTemp     *float64
	WaveDirection *float64
}

// parseRecentBuoyReading returns a single reading where each field holds its most
// recent non-nil value from within the last hour. Sensors drop out independently,
// so fields may come from different timestamps (all within the last hour). Time
// is the newest contributing row. Rows may arrive in any order — the NDBC feed is
// newest-first — and each field keeps the value from its newest row.
func parseRecentBuoyReading(data string) (BuoyReading, error) {
	cutoff := time.Now().Add(-time.Hour)

	var out BuoyReading
	// fieldTimes records the source timestamp of each field currently in out, so a
	// field is only replaced by a strictly newer reading — independent of order.
	fieldTimes := make(map[**float64]time.Time)
	found := false

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue // header rows
		}
		fields := strings.Fields(line)
		if len(fields) < columnLength {
			continue // malformed / short line
		}
		row, err := parseBuoyRow(fields)
		if err != nil {
			continue // unparseable timestamp
		}
		if row.Time.Before(cutoff) {
			continue // older than an hour → ignore
		}
		if mergeBuoyReading(&out, fieldTimes, row) {
			if row.Time.After(out.Time) {
				out.Time = row.Time
			}
			found = true
		}
	}
	if err := scanner.Err(); err != nil {
		return BuoyReading{}, fmt.Errorf("scanning buoy data: %w", err)
	}
	if !found {
		return BuoyReading{}, errors.New("no buoy readings within the last hour")
	}
	return out, nil
}

// parseBuoyRow reads one realtime2 data line into a BuoyReading.
func parseBuoyRow(fields []string) (BuoyReading, error) {
	// NDBC realtime2 timestamps are UTC; a layout without a zone parses as UTC.
	stamp := fmt.Sprintf("%s-%s-%s %s:%s",
		fields[colYear], fields[colMonth], fields[colDay], fields[colHour], fields[colMinute])
	t, err := time.Parse("2006-01-02 15:04", stamp)
	if err != nil {
		return BuoyReading{}, err
	}
	return BuoyReading{
		Time:          t,
		WindSpeed:     parseBuoyDataPoint(fields[colWindSpeed]),
		WindDirection: parseBuoyDataPoint(fields[colWindDir]),
		WaveHeight:    parseBuoyDataPoint(fields[colWaveHeight]),
		WavePeriod:    parseBuoyDataPoint(fields[colAvgPeriod]),
		WaterTemp:     parseBuoyDataPoint(fields[colWaterTemp]),
		WaveDirection: parseBuoyDataPoint(fields[colMeanWaveDir]),
	}, nil
}

// mergeBuoyReading copies each non-nil field from in into out, but only when in
// is newer than the reading currently supplying that field (tracked by field
// address in fieldTimes). Returns true if at least one field was set.
func mergeBuoyReading(out *BuoyReading, fieldTimes map[**float64]time.Time, in BuoyReading) bool {
	any := false
	set := func(dst **float64, src *float64) {
		if src == nil {
			return
		}
		if last, ok := fieldTimes[dst]; ok && !in.Time.After(last) {
			return // already have a value from a newer (or equal) row
		}
		*dst = src
		fieldTimes[dst] = in.Time
		any = true
	}

	set(&out.WindSpeed, in.WindSpeed)
	set(&out.WindDirection, in.WindDirection)
	set(&out.WaveHeight, in.WaveHeight)
	set(&out.WavePeriod, in.WavePeriod)
	set(&out.WaterTemp, in.WaterTemp)
	set(&out.WaveDirection, in.WaveDirection)

	return any
}

func parseBuoyDataPoint(s string) *float64 {
	if s == "MM" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// Break out full summary into periods
func parseForecastSummary(data string) (forcastSummary, error) {
	headerRe := regexp.MustCompile(`(?m)^\.([A-Z0-9 /]+)\.\.\.`)

	headers := headerRe.FindAllString(data, -1)
	bodies := headerRe.Split(data, -1)[1:] // drop text before first header

	var periods []forecastPeriod
	for i, body := range bodies {
		if idx := strings.Index(body, "\n\n"); idx != -1 {
			body = body[:idx]
		}
		header := strings.TrimSuffix(strings.TrimPrefix(headers[i], "."), "...")
		periods = append(periods, forecastPeriod{
			Header: header,
			Text:   strings.TrimSpace(strings.Join(strings.Fields(body), " ")),
		})
	}

	return forcastSummary{
		Periods: periods,
		Full:    data,
	}, nil
}

// Given the HTML for the NOAA buoy webpage, this will hunt and find the cool images
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
