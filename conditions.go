// Conditions assembly — pulls each station's live readings and shapes them into
// the unified Vineyard conditions payload. Kept out of the HTTP handlers so the
// handler just reads the BumpyScore store and calls these.
package main

import "log"

// Conditions is the full Vineyard conditions response for one vessel: the vessel
// it's for, that vessel's BumpyScore, and the ocean readings from each station
// we pull.
type Conditions struct {
	Vessel     Vessel            `json:"vessel"`
	BumpyScore bumpyScoreResult  `json:"bumpyScore"`
	MVCO       StationConditions `json:"mvco"`
	Buoy       StationConditions `json:"buoy"`
}

// StationConditions are the ocean readings from a single station (the MVCO
// sensor or the NOAA buoy). Each reading carries its own unit so the value is
// never ambiguous; cardinal direction is a bare label with no unit.
type StationConditions struct {
	WaveHeight            Measurement `json:"waveHeight"`
	WavePeriod            Measurement `json:"wavePeriod"`
	WaveLength            Measurement `json:"waveLength"`
	WindSpeed             Measurement `json:"windSpeed"`
	WindDirection         Measurement `json:"windDirectionDegrees"`
	WindDirectionCardinal *string     `json:"windDirectionCardinal"`
	WaterTemp             Measurement `json:"waterTemp"`
}

// Measurement is a numeric reading paired with its unit. Value is nil when the
// sensor didn't report; Unit is always set since it's a property of the field,
// not the reading.
type Measurement struct {
	Value *float64 `json:"value"`
	Unit  string   `json:"unit"`
}

// Units for station readings.
const (
	unitMeters          = "m"
	unitSeconds         = "s"
	unitMetersPerSecond = "m/s"
	unitDegrees         = "deg"
	unitCelsius         = "C"
)

// measure pairs a value with its unit.
func measure(value *float64, unit string) Measurement {
	return Measurement{Value: value, Unit: unit}
}

// emptyStationConditions is an all-units, no-values station, used when a source
// is unavailable so the response shape (and units) stays consistent.
func emptyStationConditions() StationConditions {
	return StationConditions{
		WaveHeight:    measure(nil, unitMeters),
		WavePeriod:    measure(nil, unitSeconds),
		WaveLength:    measure(nil, unitMeters),
		WindSpeed:     measure(nil, unitMetersPerSecond),
		WindDirection: measure(nil, unitDegrees),
		WaterTemp:     measure(nil, unitCelsius),
	}
}

// buildMvcoStationConditions pulls the latest MVCO reading and shapes it into a
// StationConditions. Any failure is logged and returns empty (all-nil) readings.
func buildMvcoStationConditions() StationConditions {
	mvcoData, err := fetchMvcoData()
	if err != nil || mvcoData == nil {
		log.Printf("conditions: mvco fetch: %v", err)
		return emptyStationConditions()
	}
	reading, err := parseMvcoRecentReading(*mvcoData)
	if err != nil {
		log.Printf("conditions: mvco parse: %v", err)
		return emptyStationConditions()
	}
	return StationConditions{
		WaveHeight:            measure(reading.WaveHeightSig, unitMeters),
		WavePeriod:            measure(reading.WavePeriodDom, unitSeconds),
		WaveLength:            measure(calculateWaveLength(reading.WaveHeightSig, reading.WavePeriodDom, stationDepth(stationMVCO)), unitMeters),
		WindSpeed:             measure(reading.WindSpeedMean, unitMetersPerSecond),
		WindDirection:         measure(reading.WindDirectionMean, unitDegrees),
		WindDirectionCardinal: calculateCardinalDirection(reading.WindDirectionMean),
		WaterTemp:             measure(nil, unitCelsius),
	}
}

// buildBuoyStationConditions pulls the NOAA buoy realtime feed and shapes the
// latest reading into a StationConditions. Any failure is logged and returns
// empty (all-nil) readings.
func buildBuoyStationConditions() StationConditions {
	data, err := fetchRealtimeBuoyData(vineyardBuoyID)
	if err != nil {
		log.Printf("conditions: buoy fetch: %v", err)
		return emptyStationConditions()
	}
	reading, err := parseRecentBuoyReading(data)
	if err != nil {
		log.Printf("conditions: buoy parse: %v", err)
		return emptyStationConditions()
	}
	return StationConditions{
		WaveHeight:            measure(reading.WaveHeight, unitMeters),
		WavePeriod:            measure(reading.WavePeriod, unitSeconds),
		WaveLength:            measure(calculateWaveLength(reading.WaveHeight, reading.WavePeriod, stationDepth(stationBuoy)), unitMeters),
		WindSpeed:             measure(reading.WindSpeed, unitMetersPerSecond),
		WindDirection:         measure(reading.WindDirection, unitDegrees),
		WindDirectionCardinal: calculateCardinalDirection(reading.WindDirection),
		WaterTemp:             measure(reading.WaterTemp, unitCelsius),
	}
}
