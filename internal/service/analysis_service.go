package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/snpiyasooriya/web-page-analyzer/internal/analyzer"
	"github.com/snpiyasooriya/web-page-analyzer/internal/logger"
)

type AnalysisService struct {
	httpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}
}

func NewAnalysisService() *AnalysisService {
	return &AnalysisService{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type AnalysisServiceResultDTO struct {
	analyzer.AnalysisResult
	InternalLinksCount             int
	ExternalLinksCount             int
	InaccessibleExternalLinksCount int
	InaccessibleInternalLinksCount int
	InternalLinks                  []string
	ExternalLinks                  []string
	InaccessibleInternalLinks      []string
	InaccessibleExternalLinks      []string
}

func (s *AnalysisService) AnalyzePage(ctx context.Context, pageURL string) (*AnalysisServiceResultDTO, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		logger.WithField("error", err).Error("Failed to create request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := s.httpClient.Do(req)
	if err != nil {
		logger.WithField("error", err).Error("Failed to execute request")
		return nil, err
	}
	defer response.Body.Close()

	// Check for non-successful status codes after getting the response.
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status code: %d", response.StatusCode)
	}
	result, err := analyzer.Analyze(response.Body)
	if err != nil {
		logger.WithField("error", err).Error("Failed to analyze page")
		return nil, err
	}

	URL, _ := url.Parse(pageURL)
	baseScheme := URL.Scheme
	baseHost := URL.Host
	baseURL := baseScheme + "://" + baseHost

	var internalLinks, externalLinks, internalLinksFormatted []string
	for _, link := range result.Links {
		if strings.HasPrefix(link, "/") {
			internalLinks = append(internalLinks, link)
			internalLinksFormatted = append(internalLinksFormatted, baseURL+link)
		} else if strings.Contains(link, baseURL) {
			internalLinks = append(internalLinks, link)
		} else {
			externalLinks = append(externalLinks, link)
		}
	}
	inaccessibleExternalLinksCount := s.countInaccessibleLinks(ctx, externalLinks)
	inaccessibleInternalLinksCount := s.countInaccessibleLinks(ctx, internalLinksFormatted)

	dto := &AnalysisServiceResultDTO{
		AnalysisResult:                 *result,
		InternalLinksCount:             len(internalLinks),
		ExternalLinksCount:             len(externalLinks),
		InaccessibleExternalLinksCount: inaccessibleExternalLinksCount,
		InaccessibleInternalLinksCount: inaccessibleInternalLinksCount,
		InternalLinks:                  internalLinks,
		ExternalLinks:                  externalLinks,
	}

	return dto, nil
}

func (s *AnalysisService) countInaccessibleLinks(ctx context.Context, links []string) int {
	jobs := make(chan string, len(links))
	results := make(chan bool, len(links))
	var wg sync.WaitGroup
	inaccessibleCount := 0

	// Start workers
	for w := 0; w < 10; w++ { // Cap at 10 concurrent checks
		wg.Add(1)
		go s.linkCheckerWorker(ctx, &wg, jobs, results)
	}

	// Send jobs
	for _, link := range links {
		jobs <- link
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Collect results
	for isAccessible := range results {
		if !isAccessible {
			inaccessibleCount++
		}
	}

	return inaccessibleCount
}

func (s *AnalysisService) linkCheckerWorker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan string, results chan<- bool) {
	defer wg.Done()
	for link := range jobs {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
		if err != nil {
			results <- false
			continue
		}
		resp, err := s.httpClient.Do(req)
		if err != nil || (resp.StatusCode < 200 || resp.StatusCode >= 400) {
			results <- false
		} else {
			results <- true
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}
