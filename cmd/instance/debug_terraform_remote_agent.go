package instance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	dataplaneAgentNamespace = "dataplane-agent"
	dataplaneAgentContainer = "dataplane-agent"
	dataplaneAgentPort      = 80
	dataplaneAgentAPIVer    = "2022-09-01-00"
)

type terraformDebugSession struct {
	Namespace           string   `json:"namespace"`
	PodName             string   `json:"podName"`
	ContainerName       string   `json:"containerName"`
	WorkspacePath       string   `json:"workspacePath"`
	BootstrapScriptPath string   `json:"bootstrapScriptPath"`
	ShellEntrypoint     string   `json:"shellEntrypoint"`
	TerraformName       string   `json:"terraformName"`
	TerraformAction     string   `json:"terraformAction"`
	EnvironmentKeys     []string `json:"environmentKeys,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}

func prepareTerraformDebugSessionRemote(
	ctx context.Context,
	conn *k8sConnection,
	token string,
	instanceID string,
	terraformName string,
	workspacePath string,
) (*terraformDebugSession, error) {
	body := map[string]string{
		"terraformAction": "apply",
		"workspacePath":   workspacePath,
	}
	var session terraformDebugSession
	if err := remoteDataplaneAgentRequest(
		ctx,
		conn,
		http.MethodPost,
		fmt.Sprintf("/terraform/session/%s/%s", instanceID, terraformName),
		token,
		body,
		&session,
	); err != nil {
		return nil, err
	}
	return &session, nil
}

func patchTerraformWorkspaceRemote(
	ctx context.Context,
	conn *k8sConnection,
	token string,
	instanceID string,
	terraformName string,
	files map[string][]byte,
) error {
	body := map[string]any{
		"filesContents":   files,
		"terraformAction": "apply",
	}
	return remoteDataplaneAgentRequest(
		ctx,
		conn,
		http.MethodPatch,
		fmt.Sprintf("/terraform/%s/%s", instanceID, terraformName),
		token,
		body,
		nil,
	)
}

func remoteDataplaneAgentRequest(
	ctx context.Context,
	conn *k8sConnection,
	method string,
	apiPath string,
	token string,
	body any,
	out any,
) error {
	if conn == nil {
		return fmt.Errorf("kubernetes connection is not available")
	}

	podName, err := findDataplaneAgentPod(ctx, conn)
	if err != nil {
		return err
	}

	if err = remoteDataplaneAgentProxyRequest(ctx, conn, podName, method, apiPath, token, body, out); err == nil {
		return nil
	} else {
		proxyErr := err
		if err = remoteDataplaneAgentPortForwardRequest(ctx, conn, podName, method, apiPath, token, body, out); err != nil {
			return fmt.Errorf("failed to call dataplane-agent through Kubernetes proxy (%v) or port-forward (%w)", proxyErr, err)
		}
		return nil
	}
}

func remoteDataplaneAgentProxyRequest(
	ctx context.Context,
	conn *k8sConnection,
	podName string,
	method string,
	apiPath string,
	token string,
	body any,
	out any,
) error {
	transport, err := rest.TransportFor(conn.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes API transport: %w", err)
	}

	requestBody, err := remoteAgentRequestBody(body)
	if err != nil {
		return err
	}

	reqURL := conn.clientset.CoreV1().RESTClient().Verb(method).
		Resource("pods").
		Namespace(dataplaneAgentNamespace).
		Name(fmt.Sprintf("%s:%d", podName, dataplaneAgentPort)).
		SubResource("proxy").
		Suffix(dataplaneAgentAPIVer).
		Suffix(strings.TrimPrefix(apiPath, "/")).
		URL()

	httpReq, err := http.NewRequestWithContext(ctx, method, reqURL.String(), requestBody)
	if err != nil {
		return err
	}
	setRemoteAgentHeaders(httpReq, token, body != nil)

	client := &http.Client{Transport: transport}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call dataplane-agent through Kubernetes proxy: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return handleRemoteAgentResponse(resp, out)
}

func remoteDataplaneAgentPortForwardRequest(
	ctx context.Context,
	conn *k8sConnection,
	podName string,
	method string,
	apiPath string,
	token string,
	body any,
	out any,
) error {
	localPort, err := freeLocalPort()
	if err != nil {
		return err
	}

	stopCh := make(chan struct{}, 1)
	readyCh := make(chan struct{})
	req := conn.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(dataplaneAgentNamespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(conn.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create port-forward transport: %w", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())
	ports := []string{fmt.Sprintf("%d:%d", localPort, dataplaneAgentPort)}
	forwarder, err := portforward.New(dialer, ports, stopCh, readyCh, io.Discard, io.Discard)
	if err != nil {
		return fmt.Errorf("failed to create port-forward to dataplane-agent pod %s: %w", podName, err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- forwarder.ForwardPorts()
	}()

	select {
	case <-readyCh:
	case err := <-errCh:
		return fmt.Errorf("failed to start port-forward to dataplane-agent pod %s: %w", podName, err)
	case <-ctx.Done():
		close(stopCh)
		return ctx.Err()
	case <-time.After(30 * time.Second):
		close(stopCh)
		return fmt.Errorf("timed out starting port-forward to dataplane-agent pod %s", podName)
	}
	defer close(stopCh)

	requestBody, err := remoteAgentRequestBody(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/%s%s", localPort, dataplaneAgentAPIVer, apiPath)
	httpReq, err := http.NewRequestWithContext(ctx, method, url, requestBody)
	if err != nil {
		return err
	}
	setRemoteAgentHeaders(httpReq, token, body != nil)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call dataplane-agent: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return handleRemoteAgentResponse(resp, out)
}

func remoteAgentRequestBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(payload), nil
}

func setRemoteAgentHeaders(req *http.Request, token string, hasBody bool) {
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(token) == "" {
		return
	}
	if strings.Contains(token, " ") {
		req.Header.Set("Authorization", token)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
}

func handleRemoteAgentResponse(resp *http.Response, out any) error {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("dataplane-agent returned %s: %s", resp.Status, extractRemoteAgentError(respBody))
	}
	if out != nil && len(respBody) > 0 {
		if err = json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("failed to decode dataplane-agent response: %w", err)
		}
	}
	return nil
}

func findDataplaneAgentPod(ctx context.Context, conn *k8sConnection) (string, error) {
	selectors := []string{
		"app.kubernetes.io/name=dp-agent,app.kubernetes.io/instance=dp-agent",
		"app.kubernetes.io/name=dp-agent",
		"app.kubernetes.io/name=dataplane-agent",
	}
	for _, selector := range selectors {
		pods, err := conn.clientset.CoreV1().Pods(dataplaneAgentNamespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			continue
		}
		if podName := firstReadyDataplaneAgentPod(pods.Items); podName != "" {
			return podName, nil
		}
	}

	pods, err := conn.clientset.CoreV1().Pods(dataplaneAgentNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list dataplane-agent pods: %w", err)
	}
	if podName := firstReadyDataplaneAgentPod(pods.Items); podName != "" {
		return podName, nil
	}
	return "", fmt.Errorf("no ready dataplane-agent pod found in namespace %s", dataplaneAgentNamespace)
}

func firstReadyDataplaneAgentPod(pods []corev1.Pod) string {
	for _, pod := range pods {
		if pod.DeletionTimestamp != nil || pod.Status.Phase != corev1.PodRunning {
			continue
		}
		hasAgentContainer := false
		for _, container := range pod.Spec.Containers {
			if container.Name == dataplaneAgentContainer {
				hasAgentContainer = true
				break
			}
		}
		if !hasAgentContainer {
			continue
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return pod.Name
			}
		}
	}
	return ""
}

func freeLocalPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected local listener address %s", listener.Addr())
	}
	return addr.Port, nil
}

func extractRemoteAgentError(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		for _, key := range []string{"error", "message"} {
			if value, ok := payload[key].(string); ok && value != "" {
				return value
			}
		}
	}
	return string(body)
}
