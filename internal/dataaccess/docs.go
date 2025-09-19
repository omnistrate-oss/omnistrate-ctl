package dataaccess

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	stdregexp "regexp"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/regexp"
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
	Subtitle    string  `json:"subtitle,omitempty"`
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
	Subtitle    string `json:"subtitle"`
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
func initializeSearchIndex() (err error) {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	// If index is already initialized, return
	if searchIndex != nil {
		return nil
	}

	// Create new in-memory index
	indexMapping, err := createIndexMapping()
	if err != nil {
		return fmt.Errorf("failed to create index mapping: %w", err)
	}
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
func createIndexMapping() (mapping.IndexMapping, error) {
	// Create a new index mapping
	indexMapping := bleve.NewIndexMapping()

	customWhitespaceTokenizer := map[string]interface{}{
		"type":   regexp.Name,
		"regexp": `[\p{L}\p{N}_-]+`, // Unicode letters, numbers, underscore, hyphen
	}

	err := indexMapping.AddCustomTokenizer("word_with_hyphen", customWhitespaceTokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to add custom tokenizer: %w", err)
	}

	// Define custom whitespace analyzer for English language only
	customWhitespaceAnalyzer := map[string]interface{}{
		"type":      custom.Name,
		"tokenizer": "word_with_hyphen",
		"token_filters": []string{
			"to_lower", // Convert to lowercase
			"stop_en",  // Remove English stop words
		},
	}
	// Add the custom analyzer to index mapping
	err = indexMapping.AddCustomAnalyzer("hyphen_preserving", customWhitespaceAnalyzer)
	if err != nil {
		return nil, fmt.Errorf("failed to add custom analyzer: %w", err)
	}

	// Create field mappings
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Store = true
	textFieldMapping.Index = true
	textFieldMapping.IncludeTermVectors = false
	textFieldMapping.Analyzer = "hyphen_preserving"

	// Create document mapping
	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("title", textFieldMapping)
	docMapping.AddFieldMappingsAt("description", textFieldMapping)
	docMapping.AddFieldMappingsAt("section", textFieldMapping)
	docMapping.AddFieldMappingsAt("subtitle", textFieldMapping)
	docMapping.AddFieldMappingsAt("content", textFieldMapping)

	indexMapping.AddDocumentMapping("_default", docMapping)

	return indexMapping, nil
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
					log.Warn().Err(err).Str("url", url).Msg("Failed to fetch content for indexing")
				} else {
					// Parse H2 sections from the content and create multiple documents
					h2Sections := parseH2Sections(content)

					if len(h2Sections) == 0 {
						// No H2 sections found, create a single document with all content
						doc := Document{
							ID:          fmt.Sprintf("doc_%d", docID),
							Section:     currentSection,
							Title:       title,
							Description: description,
							URL:         strings.TrimSuffix(url, "index.md"),
							Content:     content,
						}
						documents = append(documents, doc)
						docID++
					} else {
						// Create separate documents for each H2 section
						for _, h2section := range h2Sections {
							doc := Document{
								ID:          fmt.Sprintf("doc_%d", docID),
								Section:     currentSection,
								Title:       title,
								Description: description,
								URL:         strings.TrimSuffix(url, "index.md") + "#" + strings.ReplaceAll(strings.ToLower(h2section.Title), " ", "-"),
								Subtitle:    h2section.Title,
								Content:     h2section.Content,
							}
							documents = append(documents, doc)
							docID++
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading documentation: %w", err)
	}

	return documents, nil
}

// H2Section represents a section of content under an H2 heading
type H2Section struct {
	Title   string
	Content string
}

// parseH2Sections parses markdown content and splits it by H2 headings (##)
func parseH2Sections(content string) []H2Section {
	var sections []H2Section

	// Use regex to find H2 headings
	h2Regex := stdregexp.MustCompile(`(?m)^## (.+)$`)

	// Find all H2 headings and their positions
	matches := h2Regex.FindAllStringSubmatchIndex(content, -1)

	if len(matches) == 0 {
		// No H2 headings found
		return sections
	}

	// Process each H2 section
	for i, match := range matches {
		// Extract the H2 title (first capture group)
		titleStart := match[2]
		titleEnd := match[3]
		title := strings.TrimSpace(content[titleStart:titleEnd])

		// Determine the content boundaries
		contentStart := match[1] // End of the H2 line
		var contentEnd int

		if i+1 < len(matches) {
			// Content ends at the start of the next H2 heading
			contentEnd = matches[i+1][0]
		} else {
			// This is the last section, content goes to the end
			contentEnd = len(content)
		}

		// Extract and clean the section content
		sectionContent := strings.TrimSpace(content[contentStart:contentEnd])

		// Skip empty sections
		if len(sectionContent) > 0 {
			sections = append(sections, H2Section{
				Title:   title,
				Content: sectionContent,
			})
		}
	}

	return sections
}

// searchDocuments performs a search query against the indexed documents
func searchDocuments(query string, limit int) ([]DocumentationResult, error) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	if searchIndex == nil {
		return nil, fmt.Errorf("search index not initialized")
	}

	// Create a more sophisticated query strategy for better scoring

	// 1. First try exact phrase queries for each field with reasonable boost
	titlePhraseQuery := bleve.NewMatchPhraseQuery(query)
	titlePhraseQuery.SetField("title")
	titlePhraseQuery.SetBoost(8.0) // High boost for exact phrase in title

	sectionPhraseQuery := bleve.NewMatchPhraseQuery(query)
	sectionPhraseQuery.SetField("section")
	sectionPhraseQuery.SetBoost(6.0) // Very high boost for exact phrase in section

	subtitlePhraseQuery := bleve.NewMatchPhraseQuery(query)
	subtitlePhraseQuery.SetField("subtitle")
	subtitlePhraseQuery.SetBoost(9.0) // Very high boost for exact phrase in subtitle (H2 titles)

	descriptionPhraseQuery := bleve.NewMatchPhraseQuery(query)
	descriptionPhraseQuery.SetField("description")
	descriptionPhraseQuery.SetBoost(4.0) // Good boost for exact phrase in description

	contentPhraseQuery := bleve.NewMatchPhraseQuery(query)
	contentPhraseQuery.SetField("content")
	contentPhraseQuery.SetBoost(1.0) // Normal boost for exact phrase in content

	// 2. Then add individual word queries for broader matching
	titleQuery := bleve.NewMatchQuery(query)
	titleQuery.SetField("title")
	titleQuery.SetBoost(5.0) // High boost for title matches

	subtitleQuery := bleve.NewMatchQuery(query)
	subtitleQuery.SetField("subtitle")
	subtitleQuery.SetBoost(8.0) // Very high boost for subtitle matches (H2 titles)

	descriptionQuery := bleve.NewMatchQuery(query)
	descriptionQuery.SetField("description")
	descriptionQuery.SetBoost(3.0) // Boost description matches

	contentQuery := bleve.NewMatchQuery(query)
	contentQuery.SetField("content")
	contentQuery.SetBoost(1.0) // Normal boost for content matches

	sectionQuery := bleve.NewMatchQuery(query)
	sectionQuery.SetField("section")
	sectionQuery.SetBoost(7.0) // Very high boost for section matches

	// 3. Use BooleanQuery with SHOULD clauses for better scoring accumulation
	// This allows scores to accumulate when multiple fields match
	combinedQuery := bleve.NewBooleanQuery()

	// Add phrase queries first (highest priority)
	combinedQuery.AddShould(titlePhraseQuery)
	combinedQuery.AddShould(sectionPhraseQuery)
	combinedQuery.AddShould(subtitlePhraseQuery)
	combinedQuery.AddShould(descriptionPhraseQuery)
	combinedQuery.AddShould(contentPhraseQuery)

	// Add individual word queries
	combinedQuery.AddShould(titleQuery)
	combinedQuery.AddShould(subtitleQuery)
	combinedQuery.AddShould(descriptionQuery)
	combinedQuery.AddShould(contentQuery)
	combinedQuery.AddShould(sectionQuery)

	// Set minimum should match to 1 (at least one field must match)
	combinedQuery.SetMinShould(1)

	// Create search request
	searchRequest := bleve.NewSearchRequest(combinedQuery)
	searchRequest.Fields = []string{"title", "url", "description", "section", "subtitle", "content"}

	// Set the size to the requested limit
	searchRequest.Size = limit

	// Ensure results are sorted by score (highest to lowest) - this is default but explicit
	searchRequest.SortBy([]string{"-_score"})

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
		// Only process up to the limit
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
		if subtitle, ok := hit.Fields["subtitle"].(string); ok {
			result.Subtitle = subtitle
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
