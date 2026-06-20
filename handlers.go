package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Conditions about the mighty ocean
// Bumpy Score in a proprietary number to represent how rough it will be for your boat.
// Requires boat length and weight to calculate.
type conditions struct {
	StationId          string  `json:"stationId"`
	WaveHeight         string  `json:"waveHeight"`
	WavePeriod         string  `json:"wavePeriod"`
	WaveLength         string  `json:"waveLength"`
	WindSpeed          float64 `json:"windSpeed"`
	WindSpeedUnits     string  `json:"windSpeedUnits"`
	Precipitation      float64 `json:"precipitation"`
	Humidity           float64 `json:"humidity"`
	BumpyScore         float64 `json:"bumpyScore"`
	SmallCraftAdvisory bool    `json:"isSmallCraftAdvisory"`
}

// Alerts are marine alerts such as small craft warning
type alert struct {
	Event       string `json:"event"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// getConditions responds with the current NOAA conditions as JSON.
func getConditions(c *gin.Context) {
	// c.IndentedJSON(http.StatusOK, albums)
}

func getAlerts(c *gin.Context) {
	alerts, err := fetchActiveAlerts("ANZ233")
	if err != nil {
		return
	}
	c.IndentedJSON(http.StatusOK, alerts)
}

func getTides(c *gin.Context) {
	// TODO: Implement
}
