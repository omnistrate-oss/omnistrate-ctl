package instance

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/muesli/cancelreader"
	"k8s.io/client-go/tools/remotecommand"
)

func TestIsPatchableTerraformFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "main tf", path: "main.tf", want: true},
		{name: "nested tfvars", path: "modules/db/vars.tfvars", want: true},
		{name: "lock file is source", path: ".terraform.lock.hcl", want: true},
		{name: "terraform dir", path: ".terraform/providers/index.json", want: false},
		{name: "debug script", path: ".omnistrate/terraform-debug-session.sh", want: false},
		{name: "workspace hash", path: ".workspace_hash", want: false},
		{name: "state file", path: "terraform.tfstate", want: false},
		{name: "plan file", path: "out.tfplan", want: false},
		{name: "log file", path: "apply.log", want: false},
		{name: "absolute path", path: "/tmp/workspace/main.tf", want: false},
		{name: "parent traversal", path: "../main.tf", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPatchableTerraformFile(tt.path); got != tt.want {
				t.Fatalf("isPatchableTerraformFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote("a'b c")
	want := `'a'"'"'b c'`
	if got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}

func TestInteractiveTerraformShellEntrypoint(t *testing.T) {
	got := interactiveTerraformShellEntrypoint(&terraformDebugSession{
		WorkspacePath:       "/tmp/tf-workspace",
		BootstrapScriptPath: "/tmp/tf-workspace/.omnistrate/terraform-debug-session.sh",
	})

	for _, want := range []string{
		`export TERM="${TERM:-xterm-256color}"`,
		`export AWS_PAGER="${AWS_PAGER:-}"`,
		`stty sane 2>/dev/null || true`,
		`cd -- '/tmp/tf-workspace'`,
		`. '/tmp/tf-workspace/.omnistrate/terraform-debug-session.sh'`,
		`export PS1='\w # '`,
		`exec bash --noprofile --norc -i`,
		`exec sh -i`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("interactiveTerraformShellEntrypoint() missing %q in:\n%s", want, got)
		}
	}
}

func TestSingleTerminalSizeQueue(t *testing.T) {
	queue := &singleTerminalSizeQueue{size: &remotecommand.TerminalSize{Width: 120, Height: 40}}
	got := queue.Next()
	if got == nil || got.Width != 120 || got.Height != 40 {
		t.Fatalf("first size = %+v, want 120x40", got)
	}
	if got := queue.Next(); got != nil {
		t.Fatalf("second size = %+v, want nil", got)
	}
}

func TestIsInteractiveExecTeardownError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "eof", err: io.EOF, want: true},
		{name: "wrapped eof", err: fmt.Errorf("copy stdin: %w", io.EOF), want: true},
		{name: "closed pipe", err: io.ErrClosedPipe, want: true},
		{name: "read canceled", err: cancelreader.ErrCanceled, want: true},
		{name: "http2 stream closed", err: fmt.Errorf("http2: stream closed"), want: true},
		{name: "real error", err: fmt.Errorf("permission denied"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInteractiveExecTeardownError(tt.err); got != tt.want {
				t.Fatalf("isInteractiveExecTeardownError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
