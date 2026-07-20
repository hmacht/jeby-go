// El Parsor
// Parse some tings
package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type MvcoReading struct {
	Time time.Time

	// asit.mininode.Riegl — wave sensor
	WaveHeightSig   *float64 // wave_height_sig
	WavePeriodDom   *float64 // wave_period_dom
	WaveHeightSwell *float64 // wave_height_swell
	WavePeriodSwell *float64 // wave_period_swell
	WaveHeightWind  *float64 // wave_height_wind
	WavePeriodWind  *float64 // wave_period_wind
	WaterLevel      *float64 // water_level

	// asit.mininode.CLRohn — pressure / temp / humidity
	PressureMean   *float64 // pressure_mean
	PressureMedian *float64 // pressure_median
	PressureStd    *float64 // pressure_std
	TempMean       *float64 // temperature_mean
	TempMedian     *float64 // temperature_median
	TempStd        *float64 // temperature_std
	HumidityMean   *float64 // humidity_mean
	HumidityMedian *float64 // humidity_median
	HumidityStd    *float64 // humidity_std

	// asit.mininode.Sonic1 — anemometer
	Lrec              *float64 // lrec
	WindSpeedMean     *float64 // wind_speed_mean
	WindDirectionMean *float64 // wind_direction_mean
	WindSpeedWMean    *float64 // wind_speed_w_mean
	SonicTempMean     *float64 // temperature_mean
	WindSpeedMedian   *float64 // wind_speed_median
	SonicTempMedian   *float64 // temperature_median
	WindSpeedUStd     *float64 // wind_speed_u_std
	WindSpeedVStd     *float64 // wind_speed_v_std
	WindSpeedWStd     *float64 // wind_speed_w_std
}

// parseRecentReading returns a reading where each field holds its most recent
// non-nil value, ignoring anything older than one hour before `now`.
// Sensors drop out independently, so fields may come from different timestamps
// (all guaranteed within the last hour). Time is the newest contributing row.
//
// Rows may arrive in any order: each field keeps the value from its newest row.
func parseMvcoRecentReading(data string) (MvcoReading, error) {
	cutoff := time.Now().Add(-time.Hour)

	cr := csv.NewReader(strings.NewReader(data))
	cr.FieldsPerRecord = -1

	if _, err := cr.Read(); err != nil { // skip header
		return MvcoReading{}, fmt.Errorf("reading header: %w", err)
	}

	var out MvcoReading
	// fieldTimes records the source timestamp of each field currently in out, so a
	// field is only replaced by a strictly newer reading — independent of the
	// order rows arrive in.
	fieldTimes := make(map[**float64]time.Time)
	found := false
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return MvcoReading{}, fmt.Errorf("reading record: %w", err)
		}
		row, err := parseRow(rec)
		if err != nil {
			continue // unparseable timestamp
		}
		if row.Time.Before(cutoff) {
			continue // older than an hour → ignore
		}
		if mergeReading(&out, fieldTimes, row) {
			if row.Time.After(out.Time) {
				out.Time = row.Time
			}
			found = true
		}
	}
	if !found {
		return MvcoReading{}, errors.New("no readings within the last hour")
	}
	return out, nil
}

// mergeReading copies each non-nil field from in into out, but only when in is
// newer than the reading currently supplying that field (tracked by field
// address in fieldTimes). Returns true if at least one field was set. This makes
// the result independent of the order rows are fed in.
func mergeReading(out *MvcoReading, fieldTimes map[**float64]time.Time, in MvcoReading) bool {
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

	set(&out.WaveHeightSig, in.WaveHeightSig)
	set(&out.WavePeriodDom, in.WavePeriodDom)
	set(&out.WaveHeightSwell, in.WaveHeightSwell)
	set(&out.WavePeriodSwell, in.WavePeriodSwell)
	set(&out.WaveHeightWind, in.WaveHeightWind)
	set(&out.WavePeriodWind, in.WavePeriodWind)
	set(&out.WaterLevel, in.WaterLevel)

	set(&out.PressureMean, in.PressureMean)
	set(&out.PressureMedian, in.PressureMedian)
	set(&out.PressureStd, in.PressureStd)
	set(&out.TempMean, in.TempMean)
	set(&out.TempMedian, in.TempMedian)
	set(&out.TempStd, in.TempStd)
	set(&out.HumidityMean, in.HumidityMean)
	set(&out.HumidityMedian, in.HumidityMedian)
	set(&out.HumidityStd, in.HumidityStd)

	set(&out.Lrec, in.Lrec)
	set(&out.WindSpeedMean, in.WindSpeedMean)
	set(&out.WindDirectionMean, in.WindDirectionMean)
	set(&out.WindSpeedWMean, in.WindSpeedWMean)
	set(&out.SonicTempMean, in.SonicTempMean)
	set(&out.WindSpeedMedian, in.WindSpeedMedian)
	set(&out.SonicTempMedian, in.SonicTempMedian)
	set(&out.WindSpeedUStd, in.WindSpeedUStd)
	set(&out.WindSpeedVStd, in.WindSpeedVStd)
	set(&out.WindSpeedWStd, in.WindSpeedWStd)

	return any
}

func parseRow(rec []string) (MvcoReading, error) {
	t, err := time.Parse("2006-01-02T15:04-0700", rec[0])
	if err != nil {
		return MvcoReading{}, err
	}
	return MvcoReading{
		Time:              t,
		WaveHeightSig:     parseFloat(rec[1]),
		WavePeriodDom:     parseFloat(rec[2]),
		WaveHeightSwell:   parseFloat(rec[3]),
		WavePeriodSwell:   parseFloat(rec[4]),
		WaveHeightWind:    parseFloat(rec[5]),
		WavePeriodWind:    parseFloat(rec[6]),
		WaterLevel:        parseFloat(rec[7]),
		PressureMean:      parseFloat(rec[8]),
		PressureMedian:    parseFloat(rec[9]),
		PressureStd:       parseFloat(rec[10]),
		TempMean:          parseFloat(rec[11]),
		TempMedian:        parseFloat(rec[12]),
		TempStd:           parseFloat(rec[13]),
		HumidityMean:      parseFloat(rec[14]),
		HumidityMedian:    parseFloat(rec[15]),
		HumidityStd:       parseFloat(rec[16]),
		Lrec:              parseFloat(rec[17]),
		WindSpeedMean:     parseFloat(rec[18]),
		WindDirectionMean: parseFloat(rec[19]),
		WindSpeedWMean:    parseFloat(rec[20]),
		SonicTempMean:     parseFloat(rec[21]),
		WindSpeedMedian:   parseFloat(rec[22]),
		SonicTempMedian:   parseFloat(rec[23]),
		WindSpeedUStd:     parseFloat(rec[24]),
		WindSpeedVStd:     parseFloat(rec[25]),
		WindSpeedWStd:     parseFloat(rec[26]),
	}, nil
}
