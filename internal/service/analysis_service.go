package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/snpiyasooriya/web-page-analyzer/internal/analyzer"
)

type AnalysisService struct {
	httpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}
}

func NewAnalysisService() *AnalysisService {
	return &AnalysisService{
		httpClient: http.DefaultClient,
	}
}

func (s *AnalysisService) AnalyzePage(ctx context.Context, url string) (*analyzer.AnalysisResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// Check for non-successful status codes after getting the response.
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status code: %d", response.StatusCode)
	}
	result, err := analyzer.Analyze(response.Body)
	if err != nil {
		return nil, err
	}
	return result, nil

}
