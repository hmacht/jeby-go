package main

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
func calculateNoaaConditions(data string, boat boat, avgOceanDepth *float64) (Conditions, error) {
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

	return Conditions{
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
