package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/conditions", getConditions)
	router.GET("/alerts", getAlerts)
	router.GET("/tides", getTides)

	router.Run("localhost:8080")
}
