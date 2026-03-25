package dataaccess

import (
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

// LogsStream represents a log stream configuration
type LogsStream struct {
	PodName string `json:"podName"`
	LogsURL string `json:"logsUrl"`
}

// LogsConfig holds the configuration for log streaming
type LogsConfig struct {
	BaseURL  string
	Username string
	Password string
}

// LogsService provides methods for log-related operations
type LogsService struct {
}

// NewLogsService creates a new LogsService instance
func NewLogsService() *LogsService {
	return &LogsService{}
}

// IsLogsEnabled checks if logs are enabled for the given resource instance
func (ls *LogsService) IsLogsEnabled(instance *openapiclientfleet.ResourceInstance) bool {
	if instance == nil {
		return false
	}

	features := instance.ConsumptionResourceInstanceResult.ProductTierFeatures
	if features == nil {
		return false
	}

	if featRaw, ok := features["LOGS#INTERNAL"]; ok {
		// featRaw is interface{}, so cast to ProductTierFeature
		// Try concrete type first
		if feat, ok := featRaw.(map[string]interface{}); ok {
			if enabled, ok := feat["enabled"].(bool); ok && enabled {
				return true
			}
		}
	}
	return false
}

// GetLogsConfig extracts the logs configuration from the resource instance topology
func (ls *LogsService) GetLogsConfig(instance *openapiclientfleet.ResourceInstance) (*LogsConfig, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	topology := instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology
	if topology == nil {
		return nil, fmt.Errorf("no network topology available")
	}

	// Find omnistrateobserv resource for log endpoint
	for _, entry := range *topology {
		if entry.ResourceKey == "omnistrateobserv" {
			if entry.ClusterEndpoint != "" {
				parts := strings.SplitN(entry.ClusterEndpoint, "@", 2)
				if len(parts) == 2 {
					userPass := parts[0]
					baseURL := parts[1]
					creds := strings.SplitN(userPass, ":", 2)
					if len(creds) == 2 {
						return &LogsConfig{
							BaseURL:  baseURL,
							Username: creds[0],
							Password: creds[1],
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("omnistrateobserv resource not found or missing credentials")
}

// BuildLogStreams creates log stream configurations for a specific resource
func (ls *LogsService) BuildLogStreams(instance *openapiclientfleet.ResourceInstance, instanceID string, resourceKey string) ([]LogsStream, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	if !ls.IsLogsEnabled(instance) {
		return nil, fmt.Errorf("logs are not enabled for this instance")
	}

	logsConfig, err := ls.GetLogsConfig(instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs config: %w", err)
	}

	topology := instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology
	if topology == nil {
		return nil, fmt.Errorf("no network topology available")
	}

	var logStreams []LogsStream

	// Find the topology entry matching the resourceKey and build log URLs for its nodes
	for _, entry := range *topology {
		if entry.ResourceKey != resourceKey {
			continue
		}

		for _, n := range entry.Nodes {
			if n.Id == nil || *n.Id == "" {
				continue
			}
			logsURL := fmt.Sprintf("wss://%s/logs?username=%s&password=%s&podName=%s&instanceId=%s",
				logsConfig.BaseURL, logsConfig.Username, logsConfig.Password, *n.Id, instanceID)
			logStreams = append(logStreams, LogsStream{
				PodName: *n.Id,
				LogsURL: logsURL,
			})
		}
	}

	if len(logStreams) == 0 {
		return nil, fmt.Errorf("no log streams found for resource %s", resourceKey)
	}

	return logStreams, nil
}

// LogStreamConnection represents an active log stream connection
type LogStreamConnection struct {
	conn *websocket.Conn
	done chan struct{}
}

// ConnectToLogStream establishes a websocket connection to stream logs
func (ls *LogsService) ConnectToLogStream(logsURL string) (*LogStreamConnection, error) {
	if logsURL == "" {
		return nil, fmt.Errorf("logs URL is empty")
	}

	// Set up websocket dialer with proper headers and timeouts
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
	}

	conn, resp, err := dialer.Dial(logsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to log stream: %w", err)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Set read deadline for the connection
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Minute)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	return &LogStreamConnection{
		conn: conn,
		done: make(chan struct{}),
	}, nil
}

// ReadLogs reads log messages from the websocket connection
func (lsc *LogStreamConnection) ReadLogs() (string, error) {
	if lsc.conn == nil {
		return "", fmt.Errorf("connection is nil")
	}

	_, data, err := lsc.conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("failed to read message: %w", err)
	}

	return string(data), nil
}

// Close closes the log stream connection
func (lsc *LogStreamConnection) Close() error {
	if lsc.conn != nil {
		close(lsc.done)
		return lsc.conn.Close()
	}
	return nil
}

// Done returns a channel that is closed when the connection should be terminated
func (lsc *LogStreamConnection) Done() <-chan struct{} {
	return lsc.done
}

// GetAllLogStreamsForInstance gets all available log streams for an instance
func (ls *LogsService) GetAllLogStreamsForInstance(instance *openapiclientfleet.ResourceInstance, instanceID string) (map[string][]LogsStream, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	if !ls.IsLogsEnabled(instance) {
		return nil, fmt.Errorf("logs are not enabled for this instance")
	}

	topology := instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology
	if topology == nil {
		return nil, fmt.Errorf("no network topology available")
	}

	result := make(map[string][]LogsStream)

	// Get all resource keys from topology (except omnistrateobserv)
	for _, entry := range *topology {
		if entry.ResourceKey == "omnistrateobserv" {
			continue
		}

		logStreams, err := ls.BuildLogStreams(instance, instanceID, entry.ResourceKey)
		if err == nil && len(logStreams) > 0 {
			result[entry.ResourceKey] = logStreams
		}
	}

	return result, nil
}
