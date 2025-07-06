package handler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/snpiyasooriya/web-page-analyzer/internal/service"
)

var templates = template.Must(template.ParseGlob("template/*.html"))

func HomePageHandler(w http.ResponseWriter, _ *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AnalysisHandler(w http.ResponseWriter, r *http.Request) {
	analysisService := service.NewAnalysisService()
	url := r.FormValue(`url`)
	page, err := analysisService.AnalyzePage(r.Context(), url)
	fmt.Println(page)
	if err != nil {
		return
	}
	err = templates.ExecuteTemplate(w, "results.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
