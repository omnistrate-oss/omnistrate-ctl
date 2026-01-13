package dataaccess

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

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

		// Check for authentication/authorization errors based on HTTP status codes
		// Extract status code from error body if available
		errorBody := string(serviceErr.Body())
		if strings.Contains(errorBody, "401") || strings.Contains(err.Error(), "401") ||
			strings.Contains(errorBody, "Unauthorized") || strings.Contains(err.Error(), "Unauthorized") {
			return config.ErrTokenExpired
		}
		if strings.Contains(errorBody, "403") || strings.Contains(err.Error(), "403") ||
			strings.Contains(errorBody, "Forbidden") || strings.Contains(err.Error(), "Forbidden") {
			return config.ErrUnauthorized
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

		// Check for authentication/authorization errors based on HTTP status codes
		errorBody := string(serviceErr.Body())
		if strings.Contains(errorBody, "401") || strings.Contains(err.Error(), "401") ||
			strings.Contains(errorBody, "Unauthorized") || strings.Contains(err.Error(), "Unauthorized") {
			return config.ErrTokenExpired
		}
		if strings.Contains(errorBody, "403") || strings.Contains(err.Error(), "403") ||
			strings.Contains(errorBody, "Forbidden") || strings.Contains(err.Error(), "Forbidden") {
			return config.ErrUnauthorized
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
	shouldRetry, err := retryablehttp.ErrorPropagatedRetryPolicy(ctx, resp, err)
	// Do not retry POST requests on error
	if err != nil && resp != nil && resp.Request != nil && resp.Request.Method == http.MethodPost {
		shouldRetry = false
	}
	return shouldRetry, nil
}
