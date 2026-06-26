package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/marine/:buoyId/conditions", getMarineConditions)
	router.GET("/marine/:buoyId/images", getBuoyImages)
	router.GET("/marine/forecast/summary", getMarineForcastSummary)

	router.GET("/alerts/active", getActiveAlerts)
	router.GET("/tides", getTides)

	router.Run("localhost:8080")
}
