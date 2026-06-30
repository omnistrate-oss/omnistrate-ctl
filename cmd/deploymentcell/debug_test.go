package deploymentcell

import (
	"strings"
	"testing"
)

func TestToPrettyJSON(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantOK   bool
		contains []string
	}{
		{
			name:     "json object is pretty printed",
			raw:      `{"replicaCount":3,"image":{"tag":"latest"}}`,
			wantOK:   true,
			contains: []string{"\"replicaCount\": 3", "\"image\": {", "\"tag\": \"latest\""},
		},
		{
			name:     "yaml values are converted to json",
			raw:      "replicaCount: 3\nimage:\n  tag: latest\n",
			wantOK:   true,
			contains: []string{"\"replicaCount\": 3", "\"tag\": \"latest\""},
		},
		{
			name:   "json array is pretty printed",
			raw:    `[1,2,3]`,
			wantOK: true,
		},
		{
			name:   "bare scalar string is left untouched",
			raw:    "No payload found.",
			wantOK: false,
		},
		{
			name:   "placeholder message is left untouched",
			raw:    "No HELM_VALUES_RENDERED artifact found.",
			wantOK: false,
		},
		{
			name:   "go template values are left untouched",
			raw:    "{{ .Values.foo }}",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toPrettyJSON(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("toPrettyJSON(%q) ok = %v, want %v (got %q)", tt.raw, ok, tt.wantOK, got)
			}
			if !tt.wantOK {
				if got != tt.raw {
					t.Errorf("expected raw passthrough, got %q want %q", got, tt.raw)
				}
				return
			}
			for _, sub := range tt.contains {
				if !strings.Contains(got, sub) {
					t.Errorf("expected output to contain %q, got:\n%s", sub, got)
				}
			}
		})
	}
}

func TestWorkflowLine(t *testing.T) {
	wf := "wf-123"
	run := "run-456"
	empty := ""

	tests := []struct {
		name   string
		status deploymentCellAmenityStatus
		want   string
	}{
		{
			name:   "both present",
			status: deploymentCellAmenityStatus{WorkflowID: &wf, WorkflowRunID: &run},
			want:   "workflow=wf-123 run=run-456",
		},
		{
			name:   "only workflow id",
			status: deploymentCellAmenityStatus{WorkflowID: &wf},
			want:   "workflow=wf-123",
		},
		{
			name:   "only run id",
			status: deploymentCellAmenityStatus{WorkflowRunID: &run},
			want:   "run=run-456",
		},
		{
			name:   "neither present",
			status: deploymentCellAmenityStatus{},
			want:   "",
		},
		{
			name:   "empty strings treated as absent",
			status: deploymentCellAmenityStatus{WorkflowID: &empty, WorkflowRunID: &empty},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := workflowLine(tt.status); got != tt.want {
				t.Errorf("workflowLine() = %q, want %q", got, tt.want)
			}
		})
	}
}
