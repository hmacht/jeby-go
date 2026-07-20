package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

// getVessels serves the vessel registry: every vessel's code, name, and specs.
func getVessels(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, vessels)
}

// getStations serves the station registry: every station's code, name, location,
// and links.
func getStations(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, stations)
}

// getConditions serves the Vineyard conditions for one vessel, named by the
// required `vessel` query parameter (its code). The response carries that
// vessel's AI BumpyScore plus the ocean readings from both the MVCO sensor and
// the NOAA buoy. BumpyScores are computed by the background worker (see
// mvco_worker.go) and read from the store here, so a request never blocks on an
// AI call.
func getConditions(store *bumpyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := strings.ToUpper(strings.TrimSpace(c.Query("vessel")))
		if code == "" {
			c.IndentedJSON(http.StatusBadRequest, errorResponse("query parameter 'vessel' is required"))
			return
		}
		vessel, ok := vesselByCode(code)
		if !ok {
			c.IndentedJSON(http.StatusBadRequest, errorResponse(fmt.Sprintf("unknown vessel code %q", code)))
			return
		}

		bumpyScore, ok := store.get(code)
		if !ok {
			bumpyScore = bumpyScoreResult{
				Disclaimers: []string{"BumpyScore not computed yet — check back shortly."},
				Analysis:    ptr("BumpyScore not computed yet — check back shortly."),
			}
		}

		c.IndentedJSON(http.StatusOK, Conditions{
			Vessel:     vessel,
			BumpyScore: bumpyScore,
			MVCO:       buildMvcoStationConditions(),
			Buoy:       buildBuoyStationConditions(),
		})
	}
}

func errorResponse(message string) gin.H {
	return gin.H{"error": message}
}
