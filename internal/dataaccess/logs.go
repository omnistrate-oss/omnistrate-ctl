package dataaccess

import (
	"context"
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
	ctx context.Context
}

// NewLogsService creates a new LogsService instance
func NewLogsService(ctx context.Context) *LogsService {
	return &LogsService{
		ctx: ctx,
	}
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
	for _, entry := range topology {
		if entry == nil {
			continue
		}
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if rk, ok := entryMap["resourceKey"].(string); ok && rk == "omnistrateobserv" {
			if ce, ok := entryMap["clusterEndpoint"].(string); ok && ce != "" {
				parts := strings.SplitN(ce, "@", 2)
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
	for _, entry := range topology {
		if entry == nil {
			continue
		}
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		rk, ok := entryMap["resourceKey"].(string)
		if !ok || rk != resourceKey {
			continue
		}
		nodes, ok := entryMap["nodes"].([]interface{})
		if !ok {
			continue
		}
		for _, n := range nodes {
			node, ok := n.(map[string]interface{})
			if !ok {
				continue
			}
			podName, ok := node["id"].(string)
			if !ok || podName == "" {
				continue
			}
			logsURL := fmt.Sprintf("wss://%s/logs?username=%s&password=%s&podName=%s&instanceId=%s", 
				logsConfig.BaseURL, logsConfig.Username, logsConfig.Password, podName, instanceID)
			logStreams = append(logStreams, LogsStream{
				PodName: podName, 
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
	conn       *websocket.Conn
	podName    string
	instanceID string
	done       chan struct{}
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

	conn, _, err := dialer.Dial(logsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to log stream: %w", err)
	}

	// Set read deadline for the connection
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

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
	for _, entry := range topology {
		if entry == nil {
			continue
		}
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if rk, ok := entryMap["resourceKey"].(string); ok && rk != "omnistrateobserv" {
			logStreams, err := ls.BuildLogStreams(instance, instanceID, rk)
			if err == nil && len(logStreams) > 0 {
				result[rk] = logStreams
			}
		}
	}

	return result, nil
}
