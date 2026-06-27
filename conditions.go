package main

import (
	"math"
)

// These are the Realtime2 columns
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

// The number of columns we got
const columnLength = 18

// TODO: tune this
const weightDensityConstant = 5.0

// BumpyScore Tuning Scales
const (
	steepnessMultiplier    = 400.0
	windMultiplier         = 1.1
	heightMultiplier       = 30.0
	heightExponent         = 2.0
	northerWavesMultiplier = 1.3
	ratioMultiplierX1      = 30.0
	ratioMultiplierX2      = 15.0
	ratioMultiplierX3      = 0.0
)

// Given the NOAA realtime file dataset, lets calculate some conditions
// This is all tailored to Marthas Vineyard, MA
//
// Realtime2 data header
// YY  MM DD hh mm WDIR WSPD GST  WVHT   DPD   APD MWD   PRES  ATMP  WTMP  DEWP  VIS PTDY  TIDE
// yr  mo dy hr mn degT m/s  m/s     m   sec   sec degT   hPa  degC  degC  degC  nmi  hPa    ft
func calculateMarineConditions(data string, boat boat, avgOceanDepth *float64) (marineConditions, error) {
	readings, err := parseRealtime2(data, 5)
	if err != nil {
		// TODO: handle error
	}
	windSpeedAvg := average(readings, func(r BuoyReading) *float64 { return r.WindSpeed })
	windDirectionAvg := average(readings, func(r BuoyReading) *float64 { return r.WindDirection })
	waveHeightAvg := average(readings, func(r BuoyReading) *float64 { return r.WaveHeight })
	wavePeriodAvg := average(readings, func(r BuoyReading) *float64 { return r.WavePeriod })
	waterTempAvg := average(readings, func(r BuoyReading) *float64 { return r.WaterTemp })
	waveDirectionAvg := average(readings, func(r BuoyReading) *float64 { return r.WaveDirection })

	waveLengthAvg := calculateWaveLength(waveHeightAvg, wavePeriodAvg, avgOceanDepth)

	return marineConditions{
		WaveHeight:            waveHeightAvg,
		WavePeriod:            wavePeriodAvg,
		WaveLength:            waveLengthAvg,
		WindSpeed:             windSpeedAvg,
		WindDirection:         windDirectionAvg,
		WindDirectionCardinal: calculateCardinalDirection(windDirectionAvg),
		BumpyScore:            calculateDaBumpyScore(waveHeightAvg, waveLengthAvg, waveDirectionAvg, windDirectionAvg, windSpeedAvg, boat),
		WaterTemp:             waterTempAvg,
	}, nil
}

// Given a list of buoy readings, this will some he average of a field
func average(readings []BuoyReading, getField func(BuoyReading) *float64) *float64 {
	var sum float64
	var count int
	for _, reading := range readings {
		if v := getField(reading); v != nil {
			sum += *v
			count++
		}
	}
	if count == 0 {
		return nil
	}
	result := round(sum/float64(count), 2)
	return &result
}

// Calculates the BumpyScore™.
// What we need is real world data to refine this
// We are doing the calculation tailored for Vineyard Sound right now.
// In the future we will take into account locations
func calculateDaBumpyScore(waveHeight, wavelength, waveDirection, windDirection, windSpeed *float64, boat boat) bumpyScoreResult {
	disclaimers := []string{"This score is tailored to Northern Atlantic waters"}

	// Validate core data
	if waveHeight == nil || wavelength == nil || windSpeed == nil {
		disclaimers = append(disclaimers, "Could not pull one of the core metrics from buoy: Wave Height, Wave Length, Wind Speed.")
		return bumpyScoreResult{
			Score:       nil,
			Disclaimers: disclaimers,
		}
	}

	// Validate non essensial data
	if waveDirection == nil && windDirection != nil {
		disclaimers = append(disclaimers, "Used wind direction as substitute for wave direction.")
		waveDirection = windDirection
	} else {
		disclaimers = append(disclaimers, "Wave direction is not accounted for.")
	}

	// Steepness
	steepness := *waveHeight / *wavelength
	steepnessDamping := math.Min(*waveHeight/1.0, 1.0)
	steepnessScore := steepness * steepnessMultiplier * steepnessDamping

	// Wind
	windScore := *windSpeed * windMultiplier

	// Heigh
	//
	// Expenationa from 1-5 ft waves
	// The wave direction is a multipliter on the hight
	// Norther waves are much rougher, 30% bang
	heightScore := math.Pow(*waveHeight, heightExponent) * heightMultiplier
	if waveDirection != nil && (*waveDirection >= 315 || *waveDirection <= 45) {
		heightScore *= northerWavesMultiplier
	}

	// Wavelength
	//
	// Wave length is super importat lets see if this thang has hobby-horsing
	// There is a sweetspot on a boat where if wave length is in, the boat keeps smashing into waves.
	// But tighter waves ride smoothly under boat.
	// Ratio < 1 means wavelength shorter than the boat itself — worst case.
	// Ratio > 2-3 means the boat rides over smoothly — minimal penalty.
	lengthRatio := *wavelength / boat.Length
	var ratioMultiplier float64
	switch {
	case lengthRatio < 1:
		ratioMultiplier = ratioMultiplierX1
	case lengthRatio < 2:
		ratioMultiplier = ratioMultiplierX2
	default:
		ratioMultiplier = ratioMultiplierX3
	}

	// Scale the ratio penalty by wave height
	ratioScore := ratioMultiplier * math.Min(*waveHeight/1.0, 1.0)

	// We dampen the rockyness if the boat is heavier
	// The 1.5 is just a cap for lighter boats
	expectedWeight := weightDensityConstant * boat.Length * boat.Length * boat.Length
	weightFactor := math.Min(expectedWeight/boat.Weight, 1.5)

	motionScore := (heightScore + steepnessScore + ratioScore) * weightFactor
	bumpyScore := motionScore + windScore

	// Score cant exceed 100
	if bumpyScore > 100 {
		bumpyScore = 100
	}

	// Score cant be below 0
	if bumpyScore < 0 {
		bumpyScore = 0
	}

	return bumpyScoreResult{
		Score:       new(int(round(bumpyScore, 0))),
		Disclaimers: disclaimers,
	}
}

func calculateCardinalDirection(degrees *float64) *string {
	if degrees == nil {
		return nil
	}

	directions := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	index := int(math.Round(*degrees/45)) % 8
	result := directions[index]
	return &result
}

// Calculate the legth of the wave using the wave dispersion theory
// L = (g·T² / 2π) × tanh(2π·d / L
// Reference: https://www.oas.org/cdcm_train/courses/course21/chap_05.pdf
func calculateWaveLength(height, period, depth *float64) *float64 {
	if height == nil || period == nil {
		return nil
	}

	g := 9.81
	deepL := (g * (*period) * (*period)) / (2 * math.Pi)

	// If we dont have a depth then just return deepsea calculation
	if depth == nil {
		roundedDeepL := round(deepL, 2)
		return &roundedDeepL
	}

	L := deepL
	for range 50 { // iterate to solve the dispersion relation
		L = deepL * math.Tanh(2*math.Pi*(*depth)/L)
	}
	roundedL := round(L, 2)
	return &roundedL
}
