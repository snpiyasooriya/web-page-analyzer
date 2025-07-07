package main

import (
	"net/http"

	"github.com/snpiyasooriya/web-page-analyzer/internal/handler"
	"github.com/snpiyasooriya/web-page-analyzer/internal/logger"
)

func main() {
	// Initialize logger
	logger.Init()

	logger.Info("Starting web page analyzer server...")

	router := http.NewServeMux()

	router.HandleFunc("GET /", handler.HomePageHandler)
	router.HandleFunc("POST /analyze", handler.AnalysisHandler)
	router.HandleFunc("GET /health", handler.HealthHandler)

	logger.WithField("port", 8080).Info("Server starting on port 8080")

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		logger.WithField("error", err).Fatal("Failed to start server")
	}
}
