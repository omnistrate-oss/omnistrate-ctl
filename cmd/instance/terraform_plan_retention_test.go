package instance

import (
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTerraformPlanRetentionCompatibility(t *testing.T) {
	const (
		instanceID    = "instance-retention"
		resourceID    = "r-abc123"
		terraformName = "tf-r-abc123-instance-retention"
		cellID        = "hc-retention"
		generationID  = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		operationID   = generationID + ".0123456789abcdef"
		planJSON      = `{"format_version":"1.2","terraform_version":"1.11.5","resource_changes":[{"address":"aws_instance.retained","mode":"managed","type":"aws_instance","name":"retained","change":{"actions":["create"],"before":null,"after":{"instance_type":"t3.micro"},"after_unknown":{"id":true}}}]}`
		nativeDiff    = "Terraform will perform the following actions:\n\n  # aws_instance.retained will be created\n  + resource \"aws_instance\" \"retained\" {\n      + instance_type = \"t3.micro\"\n    }\n\nPlan: 1 to add, 0 to change, 0 to destroy."
	)

	for _, testCase := range []struct {
		name        string
		withHistory bool
		withDiff    bool
	}{
		{name: "preview_only_json_fallback"},
		{name: "preview_only_native_diff", withDiff: true},
		{name: "history_json_fallback", withHistory: true},
		{name: "history_native_diff", withHistory: true, withDiff: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			configMaps := []corev1.ConfigMap{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tf-plan-" + terraformName,
					Namespace: "dataplane-agent",
				},
				Data: map[string]string{operationID + "-plan-preview": planJSON},
			}}
			if testCase.withDiff {
				configMaps = append(configMaps, corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tf-plan-diff-" + terraformName,
						Namespace: "dataplane-agent",
					},
					Data: map[string]string{operationID + "-plan-preview": nativeDiff},
				})
			}
			var history []TerraformHistoryEntry
			if testCase.withHistory {
				history = []TerraformHistoryEntry{{
					Operation:   "apply",
					Status:      "completed",
					OperationID: operationID,
				}}
				historyJSON, err := json.Marshal(history)
				require.NoError(t, err)
				configMaps = append(configMaps, corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tf-state-" + terraformName,
						Namespace: "dataplane-agent",
					},
					Data: map[string]string{"history": string(historyJSON)},
				})
			}

			loader := makeStubLoader(map[string][]corev1.ConfigMap{cellID: configMaps})
			connection, err := loader(t.Context(), "", cellID)
			require.NoError(t, err)
			index, err := loadTerraformConfigMapIndex(t.Context(), connection.clientset, instanceID)
			require.NoError(t, err)
			stateData := extractTerraformStateData(index, instanceID, resourceID)
			require.NotNil(t, stateData)
			require.Equal(t, map[string]string{operationID: planJSON}, stateData.PlanPreviews)
			require.Equal(t, history, stateData.History)
			require.Nil(t, stateData.Progress)
			require.Empty(t, stateData.ExecutionState)
			require.Empty(t, stateData.PodName)
			require.Empty(t, stateData.TerraformFilesPath)
			require.Empty(t, stateData.PreviewErrors)
			if testCase.withDiff {
				require.Equal(t, map[string]string{operationID: nativeDiff}, stateData.PlanPreviewDiffs)
			} else {
				require.Empty(t, stateData.PlanPreviewDiffs)
			}
			if !testCase.withHistory {
				require.Empty(t, index.stateByResource)
			}

			kubernetesServer := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet || request.URL.Path != "/api/v1/namespaces/dataplane-agent/configmaps" {
					t.Errorf("unexpected Kubernetes request without executor pod: %s %s", request.Method, request.URL.Path)
					http.NotFound(response, request)
					return
				}
				response.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(response).Encode(corev1.ConfigMapList{
					TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMapList"},
					Items:    configMaps,
				}); err != nil {
					t.Errorf("encode ConfigMaps: %v", err)
				}
			}))
			t.Cleanup(kubernetesServer.Close)
			certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: kubernetesServer.Certificate().Raw})
			instance := resourceInstanceFixture("s-retention", "se-retention", "sub-retention")
			instance.DeploymentCellID = new(cellID)
			startCreateInstanceTestServer(t, func(response http.ResponseWriter, request *http.Request) {
				response.Header().Set("Content-Type", "application/json")
				var result interface{}
				switch {
				case request.Method == http.MethodGet && strings.Contains(request.URL.Path, instanceID):
					result = instance
				case request.Method == http.MethodGet && strings.Contains(request.URL.Path, cellID):
					result = map[string]string{
						"id":                          cellID,
						"userName":                    "retention-test",
						"apiServerEndpoint":           kubernetesServer.URL,
						"caDataBase64":                base64.StdEncoding.EncodeToString(certificatePEM),
						"clientCertificateDataBase64": "",
						"clientKeyDataBase64":         "",
						"serviceAccountToken":         "retention-test",
					}
				default:
					t.Errorf("unexpected API request: %s %s", request.Method, request.URL.Path)
					http.NotFound(response, request)
					return
				}
				if err := json.NewEncoder(response).Encode(result); err != nil {
					t.Errorf("encode API fixture: %v", err)
				}
			})

			model := newTerraformDetailModel(PlanDAGNode{ID: resourceID}, DebugData{
				InstanceID:    instanceID,
				ServiceID:     "s-retention",
				EnvironmentID: "se-retention",
				Token:         "retention-test",
			})
			message, ok := model.fetchData()().(terraformDataMsg)
			require.True(t, ok)
			require.NoError(t, message.err)
			require.Nil(t, message.fileTree)
			require.Nil(t, message.progress)
			require.Empty(t, message.executionState)
			require.Equal(t, history, message.history)
			expectedPreview := planJSON
			if testCase.withDiff {
				expectedPreview = nativeDiff
			}
			require.Equal(t, map[string]string{operationID: expectedPreview}, message.planPreviewByOpID)
			model.planPreviewByOpID = message.planPreviewByOpID
			content, isError, found := model.planPreviewForOpID(operationID)
			require.True(t, found)
			require.False(t, isError)
			require.Equal(t, expectedPreview, content)
			model.previewModalText = content
			rendered := model.previewModalFormattedText()
			if testCase.withDiff {
				require.Equal(t, nativeDiff, rendered)
			} else {
				require.Contains(t, rendered, "Terraform v1.11.5")
				require.Contains(t, rendered, "# aws_instance.retained will be created")
				require.Contains(t, rendered, "Plan: 1 to add, 0 to change, 0 to destroy.")
				require.NotContains(t, rendered, `"format_version"`)
			}
		})
	}
}
