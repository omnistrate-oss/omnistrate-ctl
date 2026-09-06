package dataaccess

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ValidateServiceSpecPath is the wire path of the read-only build validation
// endpoint. It is declared here (and not only in the generated client) because
// the debug-logging policy below has to recognise the route.
const ValidateServiceSpecPath = "/2022-09-01-00/service/spec/validate"

// redactedHeaderPlaceholder replaces credential-bearing header values in debug
// dumps, and omittedBodyPlaceholder stands in for the body itself.
const (
	redactedHeaderPlaceholder = "[REDACTED]"
	omittedBodyPlaceholder    = "[body omitted: sensitive validation payload]"
)

// sensitiveHeaders are never written to a debug log for any route.
var sensitiveHeaders = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
}

// isSensitiveBodyPath reports whether a request/response body for this route may
// never be written to a debug log.
//
// This is a deliberately narrow policy covering only the read-only validation
// route added by the build --dry-run work. Its body carries the user's raw
// specification, compose configs and secrets, and base64 tar.gz archives of
// local Terraform/Helm/Kustomize directories — none of which may ever reach a
// log file. Every other route keeps the pre-existing dump behaviour.
func isSensitiveBodyPath(u *url.URL) bool {
	if u == nil {
		return false
	}
	return u.Path == ValidateServiceSpecPath
}

// metadataOnlyRequestDump renders request metadata with credential headers
// redacted and the body deliberately omitted. It never touches req.Body, so it
// cannot consume or leak the payload.
func metadataOnlyRequestDump(req *http.Request) string {
	var b strings.Builder
	writeMetadataLine(&b, req.Method+" "+req.URL.RequestURI()+" HTTP/"+
		strconv.Itoa(req.ProtoMajor)+"."+strconv.Itoa(req.ProtoMinor))
	switch {
	case req.Host != "":
		writeMetadataLine(&b, "Host: "+req.Host)
	case req.URL != nil:
		writeMetadataLine(&b, "Host: "+req.URL.Host)
	}
	writeRedactedHeaders(&b, req.Header)
	if req.ContentLength > 0 {
		writeMetadataLine(&b, "Content-Length: "+strconv.FormatInt(req.ContentLength, 10))
	}
	b.WriteString("\r\n")
	b.WriteString(omittedBodyPlaceholder)
	return b.String()
}

// metadataOnlyResponseDump is the response counterpart. The validation result is
// surfaced on stdout by the command itself, so nothing is lost by keeping the
// body out of the log.
func metadataOnlyResponseDump(res *http.Response) string {
	var b strings.Builder
	writeMetadataLine(&b, "HTTP/"+strconv.Itoa(res.ProtoMajor)+"."+strconv.Itoa(res.ProtoMinor)+" "+res.Status)
	writeRedactedHeaders(&b, res.Header)
	if res.ContentLength > 0 {
		writeMetadataLine(&b, "Content-Length: "+strconv.FormatInt(res.ContentLength, 10))
	}
	b.WriteString("\r\n")
	b.WriteString(omittedBodyPlaceholder)
	return b.String()
}

// writeMetadataLine appends one header-shaped line. Values are written with
// WriteString rather than a format verb so no untrusted value can be treated as
// a format string, and CR/LF are stripped so a header value cannot forge extra
// lines in the log.
func writeMetadataLine(b *strings.Builder, line string) {
	b.WriteString(strings.NewReplacer("\r", "", "\n", "").Replace(line))
	b.WriteString("\r\n")
}

func writeRedactedHeaders(b *strings.Builder, header http.Header) {
	names := make([]string, 0, len(header))
	for name := range header {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, sensitive := sensitiveHeaders[strings.ToLower(name)]; sensitive {
			writeMetadataLine(b, name+": "+redactedHeaderPlaceholder)
			continue
		}
		for _, value := range header[name] {
			writeMetadataLine(b, name+": "+value)
		}
	}
}

// Configure registration api client
func getV1Client() *openapiclientv1.APIClient {
	configuration := openapiclientv1.NewConfiguration()
	configuration.Host = config.GetHost()
	configuration.Scheme = config.GetHostScheme()
	configuration.Debug = false                     // We set logging on the retryablehttp client
	configuration.UserAgent = config.GetUserAgent() // Set User-Agent header

	var servers openapiclientv1.ServerConfigurations
	for _, server := range configuration.Servers {
		server.URL = fmt.Sprintf("%s://%s", config.GetHostScheme(), config.GetHost())
		servers = append(servers, server)
	}
	configuration.Servers = servers

	configuration.HTTPClient = getRetryableHttpClient()

	apiClient := openapiclientv1.NewAPIClient(configuration)

	return apiClient
}

func handleV1Error(err error) error {
	if err != nil {
		var serviceErr *openapiclientv1.GenericOpenAPIError
		ok := errors.As(err, &serviceErr)
		if !ok {
			return err
		}
		apiError, ok := serviceErr.Model().(openapiclientv1.Error)
		if !ok {
			return fmt.Errorf("%s\nDetail: %s", serviceErr.Error(), string(serviceErr.Body()))
		}
		return fmt.Errorf("%s\nDetail: %s", apiError.Name, apiError.Message)
	}
	return err
}

// Configure fleet api client
func getFleetClient() *openapiclientfleet.APIClient {
	configuration := openapiclientfleet.NewConfiguration()
	configuration.Host = config.GetHost()
	configuration.Scheme = config.GetHostScheme()
	configuration.Debug = false                     // We set logging on the retryablehttp client
	configuration.UserAgent = config.GetUserAgent() // Set User-Agent header

	var servers openapiclientfleet.ServerConfigurations
	for _, server := range configuration.Servers {
		server.URL = fmt.Sprintf("%s://%s", config.GetHostScheme(), config.GetHost())
		servers = append(servers, server)
	}
	configuration.Servers = servers

	configuration.HTTPClient = getRetryableHttpClient()

	apiClient := openapiclientfleet.NewAPIClient(configuration)
	return apiClient
}

func handleFleetError(err error) error {
	if err != nil {
		var serviceErr *openapiclientfleet.GenericOpenAPIError
		ok := errors.As(err, &serviceErr)
		if !ok {
			return err
		}
		apiError, ok := serviceErr.Model().(openapiclientfleet.Error)
		if !ok {
			return fmt.Errorf("%s\nDetail: %s", serviceErr.Error(), string(serviceErr.Body()))
		}
		return fmt.Errorf("%s\nDetail: %s", apiError.Name, apiError.Message)
	}
	return err
}

// Configure retryable http client
// retryablehttp gives us automatic retries with exponential backoff.
func getRetryableHttpClient() *http.Client {
	// retryablehttp gives us automatic retries with exponential backoff.
	httpClient := retryablehttp.NewClient()
	// HTTP requests are logged at DEBUG level.
	httpClient.ErrorHandler = retryablehttp.PassthroughErrorHandler
	httpClient.CheckRetry = retryPolicy
	httpClient.Backoff = retryablehttp.DefaultBackoff
	httpClient.RetryWaitMin = config.GetRetryWaitMin()
	httpClient.RetryWaitMax = config.GetRetryWaitMax()
	httpClient.RetryMax = config.GetRetryMax()
	httpClient.HTTPClient.Timeout = config.GetClientTimeout()
	httpClient.Logger = NewLeveledLogger()
	httpClient.RequestLogHook = func(logger retryablehttp.Logger, req *http.Request, retryNumber int) {
		if config.IsDebugLogLevel() {
			// Narrow sensitive-body policy: the read-only validation route
			// carries specification bytes, compose configs/secrets and local
			// artifact archives, so only its metadata is ever logged.
			if isSensitiveBodyPath(req.URL) {
				log.Debug().Msgf("Request %s %s\n%s", req.Method, req.URL, metadataOnlyRequestDump(req))
				return
			}
			dump, err := httputil.DumpRequestOut(req, true)
			if err != nil {
				log.Err(err).Msg("Failed to dump request")
			}
			log.Debug().Msgf("Request %s %s\n%s", req.Method, req.URL, dump)
		}
	}
	httpClient.ResponseLogHook = func(logger retryablehttp.Logger, res *http.Response) {
		if config.IsDebugLogLevel() {
			if res != nil && res.Request != nil && isSensitiveBodyPath(res.Request.URL) {
				log.Debug().Msgf("Response %s\n%s", res.Status, metadataOnlyResponseDump(res))
				return
			}
			dump, err := httputil.DumpResponse(res, true)
			if err != nil {
				log.Err(err).Msg("Failed to dump response")
			}
			log.Debug().Msgf("Response %s\n%s", res.Status, dump)
		}
	}
	return httpClient.StandardClient()
}

// Used to transform the retryablehttp logger to a zerolog logger
type LeveledLogger struct {
	retryablehttp.LeveledLogger
}

func NewLeveledLogger() *LeveledLogger {
	return &LeveledLogger{}
}

func (l *LeveledLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Error().Msgf(msg, keysAndValues...)
}

func (l *LeveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	log.Debug().Msgf(msg, keysAndValues...)
}

func (l *LeveledLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Info().Msgf(msg, keysAndValues...)
}

func (l *LeveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.Warn().Msgf(msg, keysAndValues...)
}

func retryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	shouldRetry, _ := retryablehttp.ErrorPropagatedRetryPolicy(ctx, resp, err)
	// Do not retry POST requests unless the response is a known transient gateway/rate-limit failure.
	if shouldRetry && resp != nil && resp.Request != nil && resp.Request.Method == http.MethodPost {
		if !isRetriablePostStatus(resp.StatusCode) {
			shouldRetry = false
		}
	}
	return shouldRetry, nil
}

func isRetriablePostStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
