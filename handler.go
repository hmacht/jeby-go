package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// This API is Vineyard-only, so the NOAA identifiers are fixed.
const (
	vineyardBuoyID = "44020"  // Nantucket Sound buoy
	vineyardZoneID = "ANZ233" // Vineyard Sound marine zone
)

type bumpyScoreResult struct {
	Score       *int     `json:"score"`
	Disclaimers []string `json:"disclaimers"`
	Analysis    *string  `json:"analysis"`
}

// Alerts are marine alerts such as small craft warning
type alert struct {
	Event       string `json:"event"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

type forcastSummary struct {
	Periods []forecastPeriod `json:"periods"`
	Full    string           `json:"full"`
}

type forecastPeriod struct {
	Header string `json:"header"`
	Text   string `json:"text"`
}

type buoyImageData struct {
	Image360Url *string `json:"image360"`
}

// Images is the unified Vineyard image payload: the NOAA buoy's 360 cam plus the
// MVCO ASIT webcam.
type Images struct {
	Buoy360Url *string `json:"buoy360"`
	Asitcam2   string  `json:"asitcam2"`
}

const asitcam2Url = "https://mvco.whoi.edu/aux/webcam/asitcam2.jpg"

func getMarineForcastSummary(c *gin.Context) {
	forecast, err := fetchMarineForcastSummary(vineyardZoneID)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	c.IndentedJSON(http.StatusOK, forecast)
}

func getActiveAlerts(c *gin.Context) {
	alerts, err := fetchActiveAlerts(vineyardZoneID)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	c.IndentedJSON(http.StatusOK, alerts)
}

// getImages serves the unified Vineyard image payload. The MVCO webcam is a
// fixed URL; the buoy 360 cam is scraped and degrades to null if unavailable.
func getImages(c *gin.Context) {
	images := Images{Asitcam2: asitcam2Url}

	buoy, err := fetchBuoyImages(vineyardBuoyID)
	if err != nil {
		log.Printf("images: buoy fetch: %v", err)
	} else {
		images.Buoy360Url = buoy.Image360Url
	}

	c.IndentedJSON(http.StatusOK, images)
}

// getConditions serves the unified Vineyard conditions: a single AI BumpyScore
// plus the ocean readings from both the MVCO sensor and the NOAA buoy. The
// BumpyScore is computed by the background worker (see mvco_worker.go) and read
// from the store here, so a request never blocks on an AI call.
func getConditions(store *bumpyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		bumpyScore, _, ok := store.get()
		if !ok {
			bumpyScore = bumpyScoreResult{
				Disclaimers: []string{"BumpyScore not computed yet — check back shortly."},
				Analysis:    ptr("BumpyScore not computed yet — check back shortly."),
			}
		}

		c.IndentedJSON(http.StatusOK, Conditions{
			BumpyScore: bumpyScore,
			MVCO:       buildMvcoStationConditions(),
			Buoy:       buildBuoyStationConditions(),
		})
	}
}

func errorResponse(message string) gin.H {
	return gin.H{"error": message}
}
