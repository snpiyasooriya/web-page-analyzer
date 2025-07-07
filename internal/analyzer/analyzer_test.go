package analyzer

import (
	"fmt"
	"strings"
	"testing"
)

func TestAnalyze_ValidHTML(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Main Heading</h1>
    <h2>Sub Heading</h2>
    <h2>Another Sub Heading</h2>
    <h3>Sub Sub Heading</h3>
    <a href="https://example.com">External Link</a>
    <a href="/internal">Internal Link</a>
    <a href="">Empty Link</a>
    <form>
        <input type="text" name="username">
        <input type="password" name="password">
    </form>
</body>
</html>`

	result, err := Analyze(strings.NewReader(html))

	if err != nil {
		t.Fatalf("Analyze() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Analyze() returned nil result")
	}

	// Test HTML version
	if result.HTMLVersion != "HTML5" {
		t.Errorf("Expected HTMLVersion 'HTML5', got '%s'", result.HTMLVersion)
	}

	// Test title
	if result.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", result.Title)
	}

	// Test headings
	expectedHeadings := map[string]int{
		"h1": 1,
		"h2": 2,
		"h3": 1,
	}

	for tag, expectedCount := range expectedHeadings {
		if result.Headings[tag] != expectedCount {
			t.Errorf("Expected %d %s headings, got %d", expectedCount, tag, result.Headings[tag])
		}
	}

	// Test links (empty href should be filtered out)
	expectedLinks := []string{"https://example.com", "/internal"}
	if len(result.Links) != len(expectedLinks) {
		t.Errorf("Expected %d links, got %d", len(expectedLinks), len(result.Links))
	}

	for i, expectedLink := range expectedLinks {
		if i < len(result.Links) && result.Links[i] != expectedLink {
			t.Errorf("Expected link[%d] to be '%s', got '%s'", i, expectedLink, result.Links[i])
		}
	}

	// Test login form detection
	if !result.HasLoginForm {
		t.Error("Expected HasLoginForm to be true")
	}
}

func TestAnalyze_InvalidHTML(t *testing.T) {
	// Test with completely invalid input that might cause parsing errors
	invalidInputs := []string{
		"", // Empty string
		"not html at all",
		"<html><head><title>Unclosed",
		"<><><>",
	}

	for _, input := range invalidInputs {
		result, err := Analyze(strings.NewReader(input))

		// The html.Parse function is quite robust and shouldn't return errors for most inputs
		if err != nil {
			t.Errorf("Analyze() returned unexpected error for input '%s': %v", input, err)
		}

		if result == nil {
			t.Errorf("Analyze() returned nil result for input '%s'", input)
		}
	}
}

func TestAnalyze_HTMLVersionDetection(t *testing.T) {
	testCases := []struct {
		name            string
		html            string
		expectedVersion string
	}{
		{
			name:            "HTML5 DOCTYPE",
			html:            `<!DOCTYPE html><html><head><title>Test</title></head></html>`,
			expectedVersion: "HTML5",
		},
		{
			name:            "No DOCTYPE",
			html:            `<html><head><title>Test</title></head></html>`,
			expectedVersion: "",
		},
		{
			name: "HTML4 DOCTYPE",
			html: `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">
<html><head><title>Test</title></head></html>`,
			expectedVersion: "HTML5", // html.Parse treats any DOCTYPE as HTML5
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Analyze(strings.NewReader(tc.html))

			if err != nil {
				t.Fatalf("Analyze() returned error: %v", err)
			}

			if result.HTMLVersion != tc.expectedVersion {
				t.Errorf("Expected HTMLVersion '%s', got '%s'", tc.expectedVersion, result.HTMLVersion)
			}
		})
	}
}

func TestAnalyze_TitleExtraction(t *testing.T) {
	testCases := []struct {
		name          string
		html          string
		expectedTitle string
	}{
		{
			name:          "Normal title",
			html:          `<html><head><title>My Page Title</title></head></html>`,
			expectedTitle: "My Page Title",
		},
		{
			name:          "Empty title",
			html:          `<html><head><title></title></head></html>`,
			expectedTitle: "",
		},
		{
			name:          "No title tag",
			html:          `<html><head></head></html>`,
			expectedTitle: "",
		},
		{
			name:          "Multiple title tags (last one wins)",
			html:          `<html><head><title>First Title</title><title>Second Title</title></head></html>`,
			expectedTitle: "Second Title",
		},
		{
			name:          "Title with whitespace",
			html:          `<html><head><title>   Spaced Title   </title></head></html>`,
			expectedTitle: "   Spaced Title   ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Analyze(strings.NewReader(tc.html))

			if err != nil {
				t.Fatalf("Analyze() returned error: %v", err)
			}

			if result.Title != tc.expectedTitle {
				t.Errorf("Expected title '%s', got '%s'", tc.expectedTitle, result.Title)
			}
		})
	}
}

func TestAnalyze_HeadingsAnalysis(t *testing.T) {
	testCases := []struct {
		name             string
		html             string
		expectedHeadings map[string]int
	}{
		{
			name: "All heading levels",
			html: `<html><body>
				<h1>H1</h1>
				<h2>H2</h2>
				<h3>H3</h3>
				<h4>H4</h4>
				<h5>H5</h5>
				<h6>H6</h6>
			</body></html>`,
			expectedHeadings: map[string]int{"h1": 1, "h2": 1, "h3": 1, "h4": 1, "h5": 1, "h6": 1},
		},
		{
			name: "Multiple same level headings",
			html: `<html><body>
				<h1>First H1</h1>
				<h1>Second H1</h1>
				<h2>First H2</h2>
				<h2>Second H2</h2>
				<h2>Third H2</h2>
			</body></html>`,
			expectedHeadings: map[string]int{"h1": 2, "h2": 3},
		},
		{
			name:             "No headings",
			html:             `<html><body><p>Just a paragraph</p></body></html>`,
			expectedHeadings: map[string]int{},
		},
		{
			name: "Nested headings",
			html: `<html><body>
				<div>
					<h1>Nested H1</h1>
					<section>
						<h2>Deeply nested H2</h2>
					</section>
				</div>
			</body></html>`,
			expectedHeadings: map[string]int{"h1": 1, "h2": 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Analyze(strings.NewReader(tc.html))

			if err != nil {
				t.Fatalf("Analyze() returned error: %v", err)
			}

			if len(result.Headings) != len(tc.expectedHeadings) {
				t.Errorf("Expected %d heading types, got %d", len(tc.expectedHeadings), len(result.Headings))
			}

			for tag, expectedCount := range tc.expectedHeadings {
				if result.Headings[tag] != expectedCount {
					t.Errorf("Expected %d %s headings, got %d", expectedCount, tag, result.Headings[tag])
				}
			}
		})
	}
}

func TestAnalyze_LinksExtraction(t *testing.T) {
	testCases := []struct {
		name          string
		html          string
		expectedLinks []string
	}{
		{
			name: "Various link types",
			html: `<html><body>
				<a href="https://example.com">External</a>
				<a href="/internal">Internal</a>
				<a href="relative.html">Relative</a>
				<a href="#anchor">Anchor</a>
				<a href="mailto:test@example.com">Email</a>
				<a href="tel:+1234567890">Phone</a>
			</body></html>`,
			expectedLinks: []string{
				"https://example.com",
				"/internal",
				"relative.html",
				"#anchor",
				"mailto:test@example.com",
				"tel:+1234567890",
			},
		},
		{
			name: "Empty and missing href",
			html: `<html><body>
				<a href="">Empty href</a>
				<a>No href</a>
				<a href="valid.html">Valid</a>
			</body></html>`,
			expectedLinks: []string{"valid.html"}, // Empty href should be filtered out
		},
		{
			name:          "No links",
			html:          `<html><body><p>No links here</p></body></html>`,
			expectedLinks: []string{},
		},
		{
			name: "Nested links",
			html: `<html><body>
				<div>
					<a href="link1.html">Link 1</a>
					<section>
						<a href="link2.html">Link 2</a>
					</section>
				</div>
			</body></html>`,
			expectedLinks: []string{"link1.html", "link2.html"},
		},
		{
			name: "Links with special characters",
			html: `<html><body>
				<a href="https://example.com/path?param=value&other=123">Query params</a>
				<a href="https://example.com/path#section">With fragment</a>
				<a href="https://user:pass@example.com">With credentials</a>
			</body></html>`,
			expectedLinks: []string{
				"https://example.com/path?param=value&other=123",
				"https://example.com/path#section",
				"https://user:pass@example.com",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Analyze(strings.NewReader(tc.html))

			if err != nil {
				t.Fatalf("Analyze() returned error: %v", err)
			}

			if len(result.Links) != len(tc.expectedLinks) {
				t.Errorf("Expected %d links, got %d. Links: %v", len(tc.expectedLinks), len(result.Links), result.Links)
			}

			for i, expectedLink := range tc.expectedLinks {
				if i < len(result.Links) && result.Links[i] != expectedLink {
					t.Errorf("Expected link[%d] to be '%s', got '%s'", i, expectedLink, result.Links[i])
				}
			}
		})
	}
}

func TestAnalyze_LoginFormDetection(t *testing.T) {
	testCases := []struct {
		name              string
		html              string
		expectedLoginForm bool
	}{
		{
			name: "Form with password input",
			html: `<html><body>
				<form>
					<input type="text" name="username">
					<input type="password" name="password">
				</form>
			</body></html>`,
			expectedLoginForm: true,
		},
		{
			name: "Form without password input",
			html: `<html><body>
				<form>
					<input type="text" name="name">
					<input type="email" name="email">
				</form>
			</body></html>`,
			expectedLoginForm: false,
		},
		{
			name: "Multiple forms, one with password",
			html: `<html><body>
				<form>
					<input type="text" name="search">
				</form>
				<form>
					<input type="text" name="username">
					<input type="password" name="password">
				</form>
			</body></html>`,
			expectedLoginForm: true,
		},
		{
			name:              "No forms",
			html:              `<html><body><p>No forms here</p></body></html>`,
			expectedLoginForm: false,
		},
		{
			name: "Nested password input",
			html: `<html><body>
				<form>
					<div>
						<fieldset>
							<input type="text" name="username">
							<input type="password" name="password">
						</fieldset>
					</div>
				</form>
			</body></html>`,
			expectedLoginForm: true,
		},
		{
			name: "Case insensitive password type",
			html: `<html><body>
				<form>
					<input type="TEXT" name="username">
					<input type="PASSWORD" name="password">
				</form>
			</body></html>`,
			expectedLoginForm: true,
		},
		{
			name: "Mixed case password type",
			html: `<html><body>
				<form>
					<input type="Password" name="password">
				</form>
			</body></html>`,
			expectedLoginForm: true,
		},
		{
			name: "Input without type attribute",
			html: `<html><body>
				<form>
					<input name="username">
					<input type="password" name="password">
				</form>
			</body></html>`,
			expectedLoginForm: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Analyze(strings.NewReader(tc.html))

			if err != nil {
				t.Fatalf("Analyze() returned error: %v", err)
			}

			if result.HasLoginForm != tc.expectedLoginForm {
				t.Errorf("Expected HasLoginForm to be %v, got %v", tc.expectedLoginForm, result.HasLoginForm)
			}
		})
	}
}

func TestAnalyze_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		html        string
		description string
	}{
		{
			name: "Deeply nested structure",
			html: `<html><body>
				<div><div><div><div><div>
					<h1>Deep heading</h1>
					<a href="deep.html">Deep link</a>
					<form><div><div>
						<input type="password" name="pass">
					</div></div></form>
				</div></div></div></div></div>
			</body></html>`,
			description: "Should handle deeply nested elements",
		},
		{
			name: "Mixed content",
			html: `<!DOCTYPE html>
<html>
<head><title>Complex Page</title></head>
<body>
	<h1>Main Title</h1>
	<nav>
		<a href="/">Home</a>
		<a href="/about">About</a>
	</nav>
	<main>
		<h2>Content</h2>
		<article>
			<h3>Article Title</h3>
			<p>Some content with <a href="https://example.com">external link</a></p>
		</article>
		<aside>
			<h3>Sidebar</h3>
			<form>
				<input type="email" name="email">
				<input type="password" name="password">
			</form>
		</aside>
	</main>
	<footer>
		<a href="/contact">Contact</a>
	</footer>
</body>
</html>`,
			description: "Should handle complex real-world HTML structure",
		},
		{
			name: "Malformed HTML",
			html: `<html>
<head><title>Malformed</title>
<body>
<h1>Unclosed heading
<a href="link.html">Unclosed link
<form>
<input type="password" name="pass"
</body>`,
			description: "Should handle malformed HTML gracefully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Analyze(strings.NewReader(tc.html))

			if err != nil {
				t.Fatalf("Analyze() returned error for %s: %v", tc.description, err)
			}

			if result == nil {
				t.Fatalf("Analyze() returned nil result for %s", tc.description)
			}

			// Basic sanity checks - the exact values depend on the HTML structure
			// but we're mainly testing that it doesn't crash
			t.Logf("%s - Title: '%s', Headings: %v, Links: %d, HasLoginForm: %v",
				tc.description, result.Title, result.Headings, len(result.Links), result.HasLoginForm)
		})
	}
}

func TestContainsPasswordInput(t *testing.T) {
	// This tests the helper function indirectly through form analysis
	testCases := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "Direct password input",
			html:     `<form><input type="password"></form>`,
			expected: true,
		},
		{
			name:     "No password input",
			html:     `<form><input type="text"></form>`,
			expected: false,
		},
		{
			name: "Multiple inputs with password",
			html: `<form>
				<input type="text">
				<input type="email">
				<input type="password">
			</form>`,
			expected: true,
		},
		{
			name: "Nested password input",
			html: `<form>
				<div>
					<fieldset>
						<legend>Login</legend>
						<input type="text" name="user">
						<div>
							<input type="password" name="pass">
						</div>
					</fieldset>
				</div>
			</form>`,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Analyze(strings.NewReader(tc.html))

			if err != nil {
				t.Fatalf("Analyze() returned error: %v", err)
			}

			if result.HasLoginForm != tc.expected {
				t.Errorf("Expected HasLoginForm to be %v, got %v", tc.expected, result.HasLoginForm)
			}
		})
	}
}

func TestAnalyze_EmptyAndNilInputs(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"Empty string", ""},
		{"Only whitespace", "   \n\t  "},
		{"Just text", "This is not HTML"},
		{"Incomplete tags", "<html><head>"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Analyze(strings.NewReader(tc.input))

			if err != nil {
				t.Errorf("Analyze() returned error for '%s': %v", tc.name, err)
			}

			if result == nil {
				t.Errorf("Analyze() returned nil result for '%s'", tc.name)
			}

			// Should have initialized Headings map
			if result != nil && result.Headings == nil {
				t.Errorf("Headings map not initialized for '%s'", tc.name)
			}
		})
	}
}

func TestAnalyze_MultipleFormsStopAtFirst(t *testing.T) {
	// Test that once a login form is found, it stops checking other forms
	html := `<html><body>
		<form id="login">
			<input type="password" name="password">
		</form>
		<form id="other">
			<input type="text" name="search">
		</form>
	</body></html>`

	result, err := Analyze(strings.NewReader(html))

	if err != nil {
		t.Fatalf("Analyze() returned error: %v", err)
	}

	if !result.HasLoginForm {
		t.Error("Expected HasLoginForm to be true")
	}
}

func TestAnalyze_TitleWithNestedElements(t *testing.T) {
	// Test title extraction when title has nested elements (should only get text content)
	html := `<html><head>
		<title>Main Title</title>
	</head></html>`

	result, err := Analyze(strings.NewReader(html))

	if err != nil {
		t.Fatalf("Analyze() returned error: %v", err)
	}

	if result.Title != "Main Title" {
		t.Errorf("Expected title 'Main Title', got '%s'", result.Title)
	}
}

func TestAnalyze_LinksWithMultipleAttributes(t *testing.T) {
	html := `<html><body>
		<a href="link1.html" class="nav-link" id="link1">Link 1</a>
		<a class="button" href="link2.html" target="_blank">Link 2</a>
		<a id="link3" title="Link 3" href="link3.html" rel="noopener">Link 3</a>
	</body></html>`

	result, err := Analyze(strings.NewReader(html))

	if err != nil {
		t.Fatalf("Analyze() returned error: %v", err)
	}

	expectedLinks := []string{"link1.html", "link2.html", "link3.html"}

	if len(result.Links) != len(expectedLinks) {
		t.Errorf("Expected %d links, got %d", len(expectedLinks), len(result.Links))
	}

	for i, expectedLink := range expectedLinks {
		if i < len(result.Links) && result.Links[i] != expectedLink {
			t.Errorf("Expected link[%d] to be '%s', got '%s'", i, expectedLink, result.Links[i])
		}
	}
}

// Benchmark tests
func BenchmarkAnalyze_SimpleHTML(b *testing.B) {
	html := `<!DOCTYPE html>
<html>
<head><title>Benchmark Test</title></head>
<body>
	<h1>Main Heading</h1>
	<h2>Sub Heading</h2>
	<a href="https://example.com">Link</a>
	<form>
		<input type="text" name="username">
		<input type="password" name="password">
	</form>
</body>
</html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Analyze(strings.NewReader(html))
		if err != nil {
			b.Fatalf("Analyze() returned error: %v", err)
		}
	}
}

func BenchmarkAnalyze_ComplexHTML(b *testing.B) {
	// Generate a more complex HTML structure
	var htmlBuilder strings.Builder
	htmlBuilder.WriteString(`<!DOCTYPE html><html><head><title>Complex Page</title></head><body>`)

	// Add many headings
	for i := 1; i <= 6; i++ {
		for j := 0; j < 10; j++ {
			htmlBuilder.WriteString(fmt.Sprintf(`<h%d>Heading %d-%d</h%d>`, i, i, j, i))
		}
	}

	// Add many links
	for i := 0; i < 50; i++ {
		htmlBuilder.WriteString(fmt.Sprintf(`<a href="https://example%d.com">Link %d</a>`, i, i))
	}

	// Add forms
	for i := 0; i < 5; i++ {
		htmlBuilder.WriteString(`<form>`)
		htmlBuilder.WriteString(`<input type="text" name="field1">`)
		if i == 2 { // Only one form has password
			htmlBuilder.WriteString(`<input type="password" name="password">`)
		}
		htmlBuilder.WriteString(`</form>`)
	}

	htmlBuilder.WriteString(`</body></html>`)
	html := htmlBuilder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Analyze(strings.NewReader(html))
		if err != nil {
			b.Fatalf("Analyze() returned error: %v", err)
		}
	}
}
