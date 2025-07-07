package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// MockHTTPClient is a mock implementation of the HTTP client interface
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// Helper function to create a mock HTTP response
func createMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// Sample HTML content for testing
const sampleHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Main Heading</h1>
    <h2>Sub Heading</h2>
    <a href="/internal-link">Internal Link</a>
    <a href="https://example.com/internal">Internal Full URL</a>
    <a href="https://external.com">External Link</a>
    <form>
        <input type="password" name="password">
    </form>
</body>
</html>`

func TestNewAnalysisService(t *testing.T) {
	service := NewAnalysisService()

	if service == nil {
		t.Fatal("NewAnalysisService() returned nil")
	}

	if service.httpClient == nil {
		t.Fatal("httpClient is nil")
	}

	// Verify that the HTTP client has the expected timeout
	if client, ok := service.httpClient.(*http.Client); ok {
		if client.Timeout != 10*time.Second {
			t.Errorf("Expected timeout of 10s, got %v", client.Timeout)
		}
	}
}

func TestAnalyzePage_Success(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// For the main page request
			if req.Method == http.MethodGet {
				return createMockResponse(200, sampleHTML), nil
			}
			// For link checking (HEAD requests)
			if req.Method == http.MethodHead {
				return createMockResponse(200, ""), nil
			}
			return createMockResponse(404, ""), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	result, err := service.AnalyzePage(ctx, "https://example.com/test")

	if err != nil {
		t.Fatalf("AnalyzePage() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("AnalyzePage() returned nil result")
	}

	// Verify basic analysis results
	if result.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", result.Title)
	}

	if result.HTMLVersion != "HTML5" {
		t.Errorf("Expected HTML version 'HTML5', got '%s'", result.HTMLVersion)
	}

	if !result.HasLoginForm {
		t.Error("Expected HasLoginForm to be true")
	}

	// Verify headings
	expectedHeadings := map[string]int{"h1": 1, "h2": 1}
	for tag, count := range expectedHeadings {
		if result.Headings[tag] != count {
			t.Errorf("Expected %d %s headings, got %d", count, tag, result.Headings[tag])
		}
	}

	// Verify link categorization
	if result.InternalLinksCount != 2 {
		t.Errorf("Expected 2 internal links, got %d", result.InternalLinksCount)
	}

	if result.ExternalLinksCount != 1 {
		t.Errorf("Expected 1 external link, got %d", result.ExternalLinksCount)
	}

	// Verify internal links
	expectedInternalLinks := []string{"/internal-link", "https://example.com/internal"}
	if len(result.InternalLinks) != len(expectedInternalLinks) {
		t.Errorf("Expected %d internal links, got %d", len(expectedInternalLinks), len(result.InternalLinks))
	}

	// Verify external links
	expectedExternalLinks := []string{"https://external.com"}
	if len(result.ExternalLinks) != len(expectedExternalLinks) {
		t.Errorf("Expected %d external links, got %d", len(expectedExternalLinks), len(result.ExternalLinks))
	}
}

func TestAnalyzePage_HTTPClientError(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	result, err := service.AnalyzePage(ctx, "https://example.com/test")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if result != nil {
		t.Fatal("Expected nil result on error")
	}

	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("Expected error to contain 'network error', got: %v", err)
	}
}

func TestAnalyzePage_NonSuccessStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"Not Found", 404},
		{"Internal Server Error", 500},
		{"Bad Request", 400},
		{"Unauthorized", 401},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(_ *http.Request) (*http.Response, error) {
					return createMockResponse(tc.statusCode, "Error"), nil
				},
			}

			service := &AnalysisService{httpClient: mockClient}
			ctx := context.Background()

			result, err := service.AnalyzePage(ctx, "https://example.com/test")

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if result != nil {
				t.Fatal("Expected nil result on error")
			}

			expectedError := fmt.Sprintf("request failed with status code: %d", tc.statusCode)
			if err.Error() != expectedError {
				t.Errorf("Expected error '%s', got '%v'", expectedError, err)
			}
		})
	}
}

func TestAnalyzePage_InvalidURL(t *testing.T) {
	service := NewAnalysisService()
	ctx := context.Background()

	result, err := service.AnalyzePage(ctx, "://invalid-url")

	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}

	if result != nil {
		t.Fatal("Expected nil result on error")
	}

	if !strings.Contains(err.Error(), "failed to create request") {
		t.Errorf("Expected error to contain 'failed to create request', got: %v", err)
	}
}

func TestAnalyzePage_ContextCancellation(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Check if context is already cancelled
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			default:
				// Simulate a slow response
				time.Sleep(100 * time.Millisecond)
				return createMockResponse(200, sampleHTML), nil
			}
		},
	}

	service := &AnalysisService{httpClient: mockClient}

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := service.AnalyzePage(ctx, "https://example.com/test")

	if err == nil {
		t.Fatal("Expected error due to context cancellation, got nil")
	}

	if result != nil {
		t.Fatal("Expected nil result on error")
	}

	// Verify it's a context cancellation error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestCountInaccessibleLinks_AllAccessible(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return createMockResponse(200, ""), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	links := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	count := service.countInaccessibleLinks(ctx, links)

	if count != 0 {
		t.Errorf("Expected 0 inaccessible links, got %d", count)
	}
}

func TestCountInaccessibleLinks_AllInaccessible(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return createMockResponse(404, ""), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	links := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	count := service.countInaccessibleLinks(ctx, links)

	if count != 3 {
		t.Errorf("Expected 3 inaccessible links, got %d", count)
	}
}

func TestCountInaccessibleLinks_Mixed(t *testing.T) {
	var callCount int64
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			count := atomic.AddInt64(&callCount, 1)
			// First two calls return 200, third returns 404
			if count <= 2 {
				return createMockResponse(200, ""), nil
			}
			return createMockResponse(404, ""), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	links := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	count := service.countInaccessibleLinks(ctx, links)

	if count != 1 {
		t.Errorf("Expected 1 inaccessible link, got %d", count)
	}
}

func TestCountInaccessibleLinks_NetworkErrors(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("network timeout")
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	links := []string{
		"https://example.com/page1",
		"https://example.com/page2",
	}

	count := service.countInaccessibleLinks(ctx, links)

	if count != 2 {
		t.Errorf("Expected 2 inaccessible links due to network errors, got %d", count)
	}
}

func TestCountInaccessibleLinks_EmptyList(t *testing.T) {
	service := NewAnalysisService()
	ctx := context.Background()

	count := service.countInaccessibleLinks(ctx, []string{})

	if count != 0 {
		t.Errorf("Expected 0 inaccessible links for empty list, got %d", count)
	}
}

func TestCountInaccessibleLinks_StatusCodeBoundaries(t *testing.T) {
	testCases := []struct {
		name                 string
		statusCode           int
		shouldBeInaccessible bool
	}{
		{"Status 199", 199, true},  // Below 200
		{"Status 200", 200, false}, // OK
		{"Status 299", 299, false}, // Still OK
		{"Status 300", 300, false}, // Redirect (considered accessible)
		{"Status 399", 399, false}, // Still redirect range
		{"Status 400", 400, true},  // Client error
		{"Status 404", 404, true},  // Not found
		{"Status 500", 500, true},  // Server error
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				DoFunc: func(_ *http.Request) (*http.Response, error) {
					return createMockResponse(tc.statusCode, ""), nil
				},
			}

			service := &AnalysisService{httpClient: mockClient}
			ctx := context.Background()

			links := []string{"https://example.com/test"}
			count := service.countInaccessibleLinks(ctx, links)

			expectedCount := 0
			if tc.shouldBeInaccessible {
				expectedCount = 1
			}

			if count != expectedCount {
				t.Errorf("For status %d, expected %d inaccessible links, got %d",
					tc.statusCode, expectedCount, count)
			}
		})
	}
}

func TestAnalyzePage_LinkCategorization(t *testing.T) {
	testHTML := `<!DOCTYPE html>
<html>
<head><title>Link Test</title></head>
<body>
    <a href="/relative-path">Relative Link</a>
    <a href="https://example.com/internal-full">Internal Full URL</a>
    <a href="https://example.com/another-internal">Another Internal</a>
    <a href="https://external.com/page">External Link</a>
    <a href="https://another-external.org">Another External</a>
    <a href="mailto:test@example.com">Email Link</a>
    <a href="#anchor">Anchor Link</a>
    <a href="">Empty Link</a>
</body>
</html>`

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet {
				return createMockResponse(200, testHTML), nil
			}
			// For link checking, return 200 for all
			return createMockResponse(200, ""), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	result, err := service.AnalyzePage(ctx, "https://example.com/test-page")

	if err != nil {
		t.Fatalf("AnalyzePage() returned error: %v", err)
	}

	// Should have 3 internal links: /relative-path, https://example.com/internal-full, https://example.com/another-internal
	if result.InternalLinksCount != 3 {
		t.Errorf("Expected 3 internal links, got %d. Links: %v", result.InternalLinksCount, result.InternalLinks)
	}

	// Should have 4 external links: https://external.com/page, https://another-external.org, mailto:test@example.com, #anchor
	// Note: empty href is filtered out by the analyzer
	if result.ExternalLinksCount != 4 {
		t.Errorf("Expected 4 external links, got %d. Links: %v", result.ExternalLinksCount, result.ExternalLinks)
	}
}

func TestAnalyzePage_NoLinks(t *testing.T) {
	testHTML := `<!DOCTYPE html>
<html>
<head><title>No Links</title></head>
<body>
    <p>This page has no links.</p>
</body>
</html>`

	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return createMockResponse(200, testHTML), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	result, err := service.AnalyzePage(ctx, "https://example.com/no-links")

	if err != nil {
		t.Fatalf("AnalyzePage() returned error: %v", err)
	}

	if result.InternalLinksCount != 0 {
		t.Errorf("Expected 0 internal links, got %d", result.InternalLinksCount)
	}

	if result.ExternalLinksCount != 0 {
		t.Errorf("Expected 0 external links, got %d", result.ExternalLinksCount)
	}

	if result.InaccessibleInternalLinksCount != 0 {
		t.Errorf("Expected 0 inaccessible internal links, got %d", result.InaccessibleInternalLinksCount)
	}

	if result.InaccessibleExternalLinksCount != 0 {
		t.Errorf("Expected 0 inaccessible external links, got %d", result.InaccessibleExternalLinksCount)
	}
}

func TestAnalyzePage_AnalyzerError(t *testing.T) {
	// Invalid HTML that might cause analyzer to fail
	invalidHTML := `<html><head><title>Test</title></head><body><p>Unclosed paragraph`

	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return createMockResponse(200, invalidHTML), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	// Note: The analyzer is quite robust and handles malformed HTML well,
	// so this test mainly ensures the error handling path works
	result, err := service.AnalyzePage(ctx, "https://example.com/invalid")

	// The analyzer should still work with malformed HTML
	if err != nil {
		t.Fatalf("AnalyzePage() returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result even with malformed HTML")
	}
}

func TestLinkCheckerWorker_InvalidURL(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("invalid URL")
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	// Test with a single invalid link
	count := service.countInaccessibleLinks(ctx, []string{"://invalid-url"})

	if count != 1 {
		t.Errorf("Expected 1 inaccessible link for invalid URL, got %d", count)
	}
}

func TestAnalyzePage_ConcurrentLinkChecking(t *testing.T) {
	// Test with many links to ensure concurrent processing works
	var links []string
	for i := 0; i < 25; i++ { // More than the 10 worker limit
		links = append(links, fmt.Sprintf(`<a href="https://example%d.com">Link %d</a>`, i, i))
	}

	testHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Many Links</title></head>
<body>%s</body>
</html>`, strings.Join(links, "\n"))

	// Use atomic counter to avoid race conditions
	var callCount int64
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet {
				return createMockResponse(200, testHTML), nil
			}
			// For HEAD requests (link checking)
			count := atomic.AddInt64(&callCount, 1)
			// Make some links inaccessible
			if count%3 == 0 {
				return createMockResponse(404, ""), nil
			}
			return createMockResponse(200, ""), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	result, err := service.AnalyzePage(ctx, "https://example.com/many-links")

	if err != nil {
		t.Fatalf("AnalyzePage() returned error: %v", err)
	}

	if result.ExternalLinksCount != 25 {
		t.Errorf("Expected 25 external links, got %d", result.ExternalLinksCount)
	}

	// Should have some inaccessible links (every 3rd one)
	expectedInaccessible := 25 / 3
	if result.InaccessibleExternalLinksCount < expectedInaccessible {
		t.Errorf("Expected at least %d inaccessible external links, got %d",
			expectedInaccessible, result.InaccessibleExternalLinksCount)
	}
}

func TestAnalyzePage_ContextTimeout(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Simulate slow response
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(200 * time.Millisecond):
				return createMockResponse(200, sampleHTML), nil
			}
		},
	}

	service := &AnalysisService{httpClient: mockClient}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := service.AnalyzePage(ctx, "https://example.com/slow")

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if result != nil {
		t.Fatal("Expected nil result on timeout")
	}
}

// Benchmark tests
func BenchmarkAnalyzePage(b *testing.B) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet {
				return createMockResponse(200, sampleHTML), nil
			}
			return createMockResponse(200, ""), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.AnalyzePage(ctx, "https://example.com/test")
		if err != nil {
			b.Fatalf("AnalyzePage() returned error: %v", err)
		}
	}
}

func BenchmarkCountInaccessibleLinks(b *testing.B) {
	mockClient := &MockHTTPClient{
		DoFunc: func(_ *http.Request) (*http.Response, error) {
			return createMockResponse(200, ""), nil
		},
	}

	service := &AnalysisService{httpClient: mockClient}
	ctx := context.Background()

	links := make([]string, 20)
	for i := 0; i < 20; i++ {
		links[i] = fmt.Sprintf("https://example%d.com", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.countInaccessibleLinks(ctx, links)
	}
}
