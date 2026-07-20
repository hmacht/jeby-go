package main

import "math"

// BumpyScore Tuning Scales
const (
	steepnessMultiplier    = 400.0
	windMultiplier         = 1.1
	heightMultiplier       = 30.0
	heightExponent         = 2.0
	northerWavesMultiplier = 1.3
)

// Calculates the BumpyScore™.
// What we need is real world data to refine this
// We are doing the calculation tailored for Vineyard Sound right now.
// In the future we will take into account locations
//
// TODO: This is the manual calculation. As of today we only surface the AI calculation,
// but I would like to fine tune this more, maybe with the help of a physicist or mathematician.
func calculateDaBumpyScore(waveHeight, wavelength, waveDirection, windDirection, windSpeed *float64) bumpyScoreResult {
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

	motionScore := heightScore + steepnessScore
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
		Score:       ptr(int(round(bumpyScore, 0))),
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
