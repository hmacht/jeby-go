package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type mvcoImageData struct {
	Asitcam2Url string `json:"asitcam2Url"`
}

const MarthasVineyardAverageDepthMeters = 13.0

func getMvcoImageData(c *gin.Context) {
	images := mvcoImageData{
		Asitcam2Url: "https://mvco.whoi.edu/aux/webcam/asitcam2.jpg",
	}

	c.IndentedJSON(http.StatusOK, images)
}

// getMvcoConditions serves live MVCO conditions. The BumpyScore is computed by
// the background worker (see mvco_worker.go) and read from the store here, so a
// request never blocks on an AI call.
func getMvcoConditions(store *bumpyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		mvcoData, err := fetchMvcoData()
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, errorResponse("Failed to fetch MVCO Data"))
			return
		}

		if mvcoData == nil {
			c.IndentedJSON(http.StatusInternalServerError, errorResponse("No MVCO Data"))
			return
		}
		reading, err := parseMvcoRecentReading(*mvcoData)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, errorResponse(fmt.Sprintf("parsing mvco reading: %v", err)))
			return
		}

		bumpyScore, _, ok := store.get()
		if !ok {
			bumpyScore = bumpyScoreResult{
				Disclaimers: []string{"BumpyScore not computed yet — check back shortly."},
			}
		}

		conditions := Conditions{
			WaveHeight:            reading.WaveHeightSig,
			WavePeriod:            reading.WavePeriodDom,
			WaveLength:            calculateWaveLength(reading.WaveHeightSig, reading.WavePeriodDom, ptr(MarthasVineyardAverageDepthMeters)),
			WindSpeed:             reading.WindSpeedMean,
			WindDirection:         reading.WindDirectionMean,
			WindDirectionCardinal: calculateCardinalDirection(reading.WindDirectionMean),
			BumpyScore:            bumpyScore,
			WaterTemp:             nil,
		}

		c.IndentedJSON(http.StatusOK, conditions)
	}
}
