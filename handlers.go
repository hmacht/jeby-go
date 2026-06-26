package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Conditions about the mighty ocean
// Bumpy Score in a proprietary number to represent how rough it will be for your boat.
// Requires boat length and weight to calculate.
type marineConditions struct {
	WaveHeight            *float64 `json:"waveHeight"`
	WavePeriod            *float64 `json:"wavePeriod"`
	WaveLength            *float64 `json:"waveLength"`
	WindSpeed             *float64 `json:"windSpeed"`
	WindDirection         *float64 `json:"windDirectionDegrees"`
	WindDirectionCardinal *string  `json:"windDirectionCardinal"`
	WaterTemp             *float64 `json:"waterTemp"`
	BumpyScore            *int     `json:"bumpyScore"`
}

// Alerts are marine alerts such as small craft warning
type alert struct {
	Event       string `json:"event"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

type boat struct {
	Length float64 // meters
	Weight float64 // kg
}

type forcastSummary struct {
	Today string `json:"today"`
	Full  string `json:"full"`
}

type buoyImageData struct {
	Image360Url *string `json:"image360"`
}

// getMarineConditions responds with the current NOAA conditions as JSON.
// Nantucket Sound Buoy is 44020
func getMarineConditions(c *gin.Context) {
	buoyId := c.Param("buoyId")
	boatLengthStr := c.Query("boatLength")
	boatWeightStr := c.Query("boatWeight")
	avgOceanDepthStr := c.Query("avgOceanDepth")

	if buoyId == "" {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse("Missing buoyId"))
		return
	}

	if boatLengthStr == "" || boatWeightStr == "" {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse("Missing boatLength or boatWeight"))
		return
	}

	boatLength, err := strconv.ParseFloat(boatLengthStr, 64)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, errorResponse("Invalid boatLength"))
		return
	}

	boatWeight, err := strconv.ParseFloat(boatWeightStr, 64)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, errorResponse("Invalid boatWeight"))
		return
	}

	var avgOceanDepth *float64

	if avgOceanDepthStr != "" {
		parsed, err := strconv.ParseFloat(avgOceanDepthStr, 64)
		if err != nil {
			c.IndentedJSON(http.StatusBadRequest, errorResponse("Invalid avgOceanDepth"))
			return
		}
		avgOceanDepth = &parsed
	}

	boatDetails := boat{
		Length: boatLength,
		Weight: boatWeight,
	}

	data, err := fetchRealtimeBuoyData(buoyId)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	condizioni, err := calculateMarineConditions(data, boatDetails, avgOceanDepth)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	c.IndentedJSON(http.StatusOK, condizioni)
}

func getMarineForcastSummary(c *gin.Context) {
	forcastZoneId := c.Query("zone")

	if forcastZoneId == "" {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse("Missing forcastZoneId"))
		return
	}

	forecast, err := fetchMarineForcastSummary(forcastZoneId)

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	c.IndentedJSON(http.StatusOK, forecast)
}

// Reuqires a forecast zone which can be search using station endpoint
// Vineyard sound is under: ANZ233
func getActiveAlerts(c *gin.Context) {
	zone := c.Query("zone")

	alerts, err := fetchActiveAlerts(zone)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	c.IndentedJSON(http.StatusOK, alerts)
}

func getBuoyImages(c *gin.Context) {
	buoyId := c.Param("buoyId")

	if buoyId == "" {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse("Missing buoyId"))
		return
	}

	images, err := fetchBuoyImages(buoyId)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, errorResponse(err.Error()))
		return
	}

	c.IndentedJSON(http.StatusOK, images)
}

func getTides(c *gin.Context) {
	// TODO: Implement
}

func errorResponse(message string) gin.H {
	return gin.H{"error": message}
}
