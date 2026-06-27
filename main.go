package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hmacht/jeby-go/middleware"
)

func main() {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")
	v1.Use(middleware.APIKeyAuth(apiKey))
	{
		v1.GET("/marine/buoys/:buoyId/conditions", getMarineConditions)
		v1.GET("/marine/buoys/:buoyId/images", getBuoyImages)
		v1.GET("/marine/zones/:zoneId/forecast/summary", getMarineForcastSummary)
		v1.GET("/marine/zones/:zoneId/alerts/active", getActiveAlerts)
	}

	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: router,
	}

	go func() {
		log.Printf("listening on :%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown:", err)
	}
	log.Println("server exited")
}
