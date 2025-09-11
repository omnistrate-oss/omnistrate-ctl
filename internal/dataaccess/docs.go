package dataaccess

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
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
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	Section     string  `json:"section"`
	Content     string  `json:"content"`
	Score       float64 `json:"score,omitempty"`
}

// Document represents a document to be indexed by Bleve
type Document struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Section     string `json:"section"`
	Content     string `json:"content"`
}

var (
	searchIndex bleve.Index
	indexMutex  sync.RWMutex
)

func PerformDocumentationSearch(query string, limit int) ([]DocumentationResult, error) {
	// Initialize the search index if not already done
	if err := initializeSearchIndex(); err != nil {
		return nil, fmt.Errorf("failed to initialize search index: %w", err)
	}

	// Perform the search
	results, err := searchDocuments(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return results, nil
}

// initializeSearchIndex creates and populates the Bleve search index in memory
func initializeSearchIndex() error {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	// If index is already initialized, return
	if searchIndex != nil {
		return nil
	}

	// Create new in-memory index
	indexMapping := createIndexMapping()
	var err error
	searchIndex, err = bleve.NewMemOnly(indexMapping)
	if err != nil {
		return fmt.Errorf("failed to create in-memory search index: %w", err)
	}

	log.Debug().Msg("Created new in-memory search index")

	// Populate the index with documentation
	if err := populateIndex(); err != nil {
		return fmt.Errorf("failed to populate search index: %w", err)
	}

	return nil
}

// createIndexMapping creates the mapping for the search index
func createIndexMapping() mapping.IndexMapping {
	// Create a new index mapping
	indexMapping := bleve.NewIndexMapping()

	// Create field mappings
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Store = true
	textFieldMapping.Index = true
	textFieldMapping.IncludeTermVectors = true

	keywordFieldMapping := bleve.NewKeywordFieldMapping()
	keywordFieldMapping.Store = true
	keywordFieldMapping.Index = true

	// Create document mapping
	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("title", textFieldMapping)
	docMapping.AddFieldMappingsAt("url", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("description", textFieldMapping)
	docMapping.AddFieldMappingsAt("section", keywordFieldMapping)
	docMapping.AddFieldMappingsAt("content", textFieldMapping)

	indexMapping.AddDocumentMapping("_default", docMapping)

	return indexMapping
}

// populateIndex fetches documentation and adds it to the search index
func populateIndex() error {
	// Fetch documentation from llms.txt
	contentReader, err := fetchContentFromURL(config.GetLlmsTxtURL())
	if err != nil {
		return fmt.Errorf("failed to fetch documentation: %w", err)
	}

	// Parse the documentation content
	documents, err := parseDocumentationContentForIndexing(contentReader)
	if err != nil {
		return err
	}

	// Add documents to the index
	batch := searchIndex.NewBatch()
	for _, doc := range documents {
		if err := batch.Index(doc.ID, doc); err != nil {
			log.Warn().Err(err).Str("docID", doc.ID).Msg("Failed to add document to batch")
			continue
		}
	}

	if err := searchIndex.Batch(batch); err != nil {
		return fmt.Errorf("failed to index documents: %w", err)
	}

	log.Debug().Msgf("Indexed %d documents", len(documents))
	return nil
}

// parseDocumentationContentForIndexing parses the llms.txt content and creates documents for indexing
func parseDocumentationContentForIndexing(body string) ([]Document, error) {
	var documents []Document
	scanner := bufio.NewScanner(strings.NewReader(body))

	var currentSection string
	docID := 0

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
					content = fmt.Sprintf("Error fetching content: %s", err.Error())
					log.Warn().Err(err).Str("url", url).Msg("Failed to fetch content for indexing")
				}

				// Create a document for indexing
				doc := Document{
					ID:          fmt.Sprintf("doc_%d", docID),
					Title:       title,
					URL:         strings.TrimSuffix(url, "index.md"), // Remove index.md from URLs
					Description: description,
					Section:     currentSection,
					Content:     content,
				}

				documents = append(documents, doc)
				docID++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading documentation: %w", err)
	}

	return documents, nil
}

// searchDocuments performs a search query against the indexed documents
func searchDocuments(query string, limit int) ([]DocumentationResult, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if searchIndex == nil {
		return nil, fmt.Errorf("search index not initialized")
	}

	// Create a query that searches across title, description, and content fields
	titleQuery := bleve.NewMatchQuery(query)
	titleQuery.SetField("title")
	titleQuery.SetBoost(8.0) // Boost title matches

	descriptionQuery := bleve.NewMatchQuery(query)
	descriptionQuery.SetField("description")
	descriptionQuery.SetBoost(5.0) // Boost description matches

	contentQuery := bleve.NewMatchPhraseQuery(query)
	contentQuery.SetField("content")
	contentQuery.SetBoost(1.0) // Normal boost for content matches

	sectionQuery := bleve.NewMatchQuery(query)
	sectionQuery.SetField("section")
	sectionQuery.SetBoost(10.0) // Boost section matches

	// Combine queries with OR
	combinedQuery := bleve.NewDisjunctionQuery(titleQuery, descriptionQuery, contentQuery, sectionQuery)

	// Create search request
	searchRequest := bleve.NewSearchRequest(combinedQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"title", "url", "description", "section", "content"}
	searchRequest.IncludeLocations = false

	// Ensure results are sorted by score (highest to lowest) - this is default but explicit
	searchRequest.SortBy([]string{"-_score"})

	// Execute search
	searchResult, err := searchIndex.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}

	// Convert search results to DocumentationResult, results are already ordered by score
	var results []DocumentationResult
	for i, hit := range searchResult.Hits {
		// Only process up to the limit (extra safety check)
		if i >= limit {
			break
		}

		result := DocumentationResult{
			Score: hit.Score,
		}

		// Extract fields from the hit
		if title, ok := hit.Fields["title"].(string); ok {
			result.Title = title
		}
		if url, ok := hit.Fields["url"].(string); ok {
			result.URL = url
		}
		if description, ok := hit.Fields["description"].(string); ok {
			result.Description = description
		}
		if section, ok := hit.Fields["section"].(string); ok {
			result.Section = section
		}
		if content, ok := hit.Fields["content"].(string); ok {
			result.Content = content
		}

		results = append(results, result)
	}

	log.Debug().Msgf("Search for '%s' returned %d results (ordered by relevance score)", query, len(results))
	return results, nil
}

// CleanupSearchIndex closes the in-memory search index
func cleanupSearchIndex() error {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if searchIndex != nil {
		if err := searchIndex.Close(); err != nil {
			log.Warn().Err(err).Msg("Failed to close search index")
		}
		searchIndex = nil
		log.Debug().Msg("Closed in-memory search index")
	}

	return nil
}

// refreshSearchIndex rebuilds the search index with fresh data
func refreshSearchIndex() error {
	// Clean up existing index
	if err := cleanupSearchIndex(); err != nil {
		return fmt.Errorf("failed to cleanup existing index: %w", err)
	}

	// Reinitialize the index
	return initializeSearchIndex()
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
