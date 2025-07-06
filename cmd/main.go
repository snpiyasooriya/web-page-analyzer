package main

import (
	"log"
	"net/http"

	"github.com/snpiyasooriya/web-page-analyzer/internal/handler"
)

func main() {

	router := http.NewServeMux()

	router.HandleFunc("GET /", handler.HomePageHandler)
	router.HandleFunc("POST /analyze", handler.AnalysisHandler)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
