package analyzer

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

type AnalysisResult struct {
	HTMLVersion                    string
	Title                          string
	Headings                       map[string]int
	InternalLinksCount             int
	ExternalLinksCount             int
	InaccessibleInternalLinksCount int
	InaccessibleExternalLinksCount int
	HasLoginForm                   bool
	InternalLinks                  []string
	ExternalLinks                  []string
}

func Analyze(body io.Reader) (*AnalysisResult, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	result := &AnalysisResult{
		Headings: make(map[string]int),
	}

	traverseTags(doc, result)

	return result, nil
}

func traverseTags(n *html.Node, result *AnalysisResult) {
	if n.Type == html.DoctypeNode {
		result.HTMLVersion = "HTML5"
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "title":
			if n.FirstChild != nil {
				result.Title = n.FirstChild.Data
			}
		case "h1", "h2", "h3", "h4", "h5", "h6":
			result.Headings[n.Data]++
		case "a":
			for _, attr := range n.Attr {
				if attr.Key == "href" && attr.Val != "" {
					if strings.Contains(attr.Val, "http") || strings.Contains(attr.Val, "https") {
						result.ExternalLinks = append(result.ExternalLinks, attr.Val)
						result.ExternalLinksCount++
					} else {
						result.InternalLinks = append(result.InternalLinks, attr.Val)
						result.InternalLinksCount++
					}

				}
			}
		case "form":
			if !result.HasLoginForm { // Stop checking once one is found
				result.HasLoginForm = containsPasswordInput(n)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		traverseTags(c, result)
	}
}

// containsPasswordInput is a helper to recursively check for a password field within a form.
func containsPasswordInput(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "input" {
		for _, attr := range n.Attr {
			if attr.Key == "type" && strings.ToLower(attr.Val) == "password" {
				return true
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if containsPasswordInput(c) {
			return true
		}
	}
	return false
}
