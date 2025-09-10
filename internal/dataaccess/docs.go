package dataaccess

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/rs/zerolog/log"
)

// userAgentTransport wraps an http.RoundTripper to add a User-Agent header
type userAgentTransport struct {
	Transport http.RoundTripper
	UserAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.UserAgent)
	return t.Transport.RoundTrip(req)
}

// DocumentationResult represents a search result
type DocumentationResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Section     string `json:"section"`
	Content     string `json:"content"`
}

func PerformDocumentationSearch(query string, limit int) ([]DocumentationResult, error) {
	// Fetch documentation from llms.txt
	contentReader, err := fetchContentFromURL(config.GetLlmsTxtURL())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documentation: %w", err)
	}

	// Parse the documentation content
	results, err := parseDocumentationContent(contentReader)
	if err != nil {
		return nil, err
	}

	// TODO: Filter results based on query
	filteredResults := results

	// Apply limit
	if limit > 0 && len(filteredResults) > limit {
		filteredResults = filteredResults[:limit]
	}

	return filteredResults, nil
}

// parseDocumentationContent parses the llms.txt content and extracts documentation entries
func parseDocumentationContent(body string) ([]DocumentationResult, error) {
	var results []DocumentationResult
	scanner := bufio.NewScanner(strings.NewReader(body))

	var currentSection string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if line starts with "## " - this indicates a section name
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.TrimPrefix(line, "## ")
			continue
		}

		// Parse markdown links format: - [Title](URL): Description
		if strings.HasPrefix(line, "- [") && strings.Contains(line, "](") {
			// Parse the current line
			// Format: - [Title](URL): Description or - [Title](URL)
			parts := strings.SplitN(line, "](", 2)
			if len(parts) == 2 {
				title := strings.TrimPrefix(parts[0], "- [")

				// Handle both formats: with and without description
				var url, description string
				if strings.Contains(parts[1], "): ") {
					// Format: - [Title](URL): Description
					urlAndDesc := strings.SplitN(parts[1], "): ", 2)
					url = urlAndDesc[0]
					if len(urlAndDesc) == 2 {
						description = urlAndDesc[1]
					}
				} else {
					// Format: - [Title](URL)
					url = strings.TrimSuffix(parts[1], ")")
					description = title // Use title as description if no separate description
				}

				// Fetch content from the URL
				content, err := fetchContentFromURL(url)
				if err != nil {
					content = err.Error()
				}

				// Create a result entry
				result := DocumentationResult{
					Title:       title,
					URL:         strings.TrimSuffix(url, "index.md"), // Remove index.md from URLs
					Description: description,
					Section:     currentSection,
					Content:     content,
				}

				results = append(results, result)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading documentation: %w", err)
	}

	return results, nil
}

func fetchContentFromURL(url string) (string, error) {
	// Make HTTP request to fetch the content
	// retryablehttp gives us automatic retries with exponential backoff.
	httpClient := retryablehttp.NewClient()
	httpClient.HTTPClient.Transport = &http.Transport{}

	// Set User-Agent header for all requests
	originalTransport := httpClient.HTTPClient.Transport
	httpClient.HTTPClient.Transport = &userAgentTransport{
		Transport: originalTransport,
		UserAgent: config.GetUserAgent(),
	}

	// HTTP requests are logged at DEBUG level.
	httpClient.ErrorHandler = retryablehttp.PassthroughErrorHandler
	httpClient.CheckRetry = retryablehttp.DefaultRetryPolicy
	httpClient.HTTPClient.Timeout = config.GetClientTimeout()
	httpClient.Logger = NewLeveledLogger()
	httpClient.RequestLogHook = func(logger retryablehttp.Logger, req *http.Request, retryNumber int) {
		if config.IsDebugLogLevel() {
			dump, err := httputil.DumpRequestOut(req, true)
			if err != nil {
				log.Err(err).Msg("Failed to dump request")
			}
			log.Debug().Msgf("Request %s %s\n%s", req.Method, req.URL, dump)
		}
	}
	httpClient.ResponseLogHook = func(logger retryablehttp.Logger, res *http.Response) {
		if config.IsDebugLogLevel() {
			dump, err := httputil.DumpResponse(res, true)
			if err != nil {
				log.Err(err).Msg("Failed to dump response")
			}
			log.Debug().Msgf("Response %s\n%s", res.Status, dump)
		}
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch content from %s: HTTP %d", url, resp.StatusCode)
	}
	if resp.Body == nil {
		return "", fmt.Errorf("no content found at %s", url)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %w", url, err)
	}

	return string(bodyBytes), nil
}
