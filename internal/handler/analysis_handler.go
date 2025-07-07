package handler

import (
	"html/template"
	"net/http"

	"github.com/snpiyasooriya/web-page-analyzer/internal/logger"
	"github.com/snpiyasooriya/web-page-analyzer/internal/service"
)

var templates = template.Must(template.ParseGlob("template/*.html"))

func HomePageHandler(w http.ResponseWriter, _ *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.WithField("error", err).Error("Failed to execute template")
		return
	}
}

func AnalysisHandler(w http.ResponseWriter, r *http.Request) {
	analysisService := service.NewAnalysisService()
	url := r.FormValue(`url`)
	page, err := analysisService.AnalyzePage(r.Context(), url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.WithField("error", err).Error("Failed to analyze page")
		return
	}
	err = templates.ExecuteTemplate(w, "results.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.WithField("error", err).Error("Failed to execute template")
		return
	}
}

// HealthHandler provides a health check endpoint
func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"status":"healthy","service":"web-page-analyzer"}`))
	if err != nil {
		logger.WithField("error", err).Error("Failed to write response")
		return
	}
}
