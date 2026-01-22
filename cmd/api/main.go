package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status string `json:"status,omitempty"`
}

func main() {
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
	}

	router := gin.Default()
	router.GET("/health", healthHandler)
	log.Fatal(router.Run())
}
