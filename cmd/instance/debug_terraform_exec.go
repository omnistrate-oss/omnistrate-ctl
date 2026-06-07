package instance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/muesli/cancelreader"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// TerraformFileEntry represents a file or directory in the terraform executor pod
type TerraformFileEntry struct {
	Path     string
	RelPath  string
	Name     string
	IsDir    bool
	Depth    int
	Expanded bool
	Children []*TerraformFileEntry
}

// TerraformFileTree holds the fetched file tree and metadata for the executor pod
type TerraformFileTree struct {
	PodName   string
	Namespace string
	BasePath  string
	Root      *TerraformFileEntry
	Flat      []*TerraformFileEntry // flattened visible entries for rendering
	conn      *k8sConnection        // the k8s connection where the executor pod was found
}

// terraformExecutorPodName builds the pod name for the terraform executor
func terraformExecutorPodName(terraformName string) string {
	return "tf-executor-" + terraformName
}

// terraformFilesBasePath builds the base path for terraform files in the executor pod
func terraformFilesBasePath(terraformName, instanceID, operation string) string {
	return fmt.Sprintf("/tmp/tf-%s-%s-%s", terraformName, instanceID, operation)
}

// execInPod runs a command in a pod and returns stdout
func execInPod(ctx context.Context, conn *k8sConnection, namespace, podName string, command []string) (string, error) {
	stdout, _, err := execInPodWithStreams(ctx, conn, namespace, podName, "", command, false, nil, nil, nil)
	return stdout, err
}

func execInPodWithStreams(
	ctx context.Context,
	conn *k8sConnection,
	namespace string,
	podName string,
	container string,
	command []string,
	tty bool,
	stdin io.Reader,
	stdoutWriter io.Writer,
	stderrWriter io.Writer,
) (string, string, error) {
	if conn == nil {
		return "", "", fmt.Errorf("kubernetes connection is not available")
	}
	if container == "" {
		container = "terraform-executor"
	}
	req := conn.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     stdin != nil,
			Stdout:    true,
			Stderr:    !tty,
			TTY:       tty,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(conn.restConfig, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	streamOptions := remotecommand.StreamOptions{}
	if stdoutWriter != nil {
		streamOptions.Stdout = stdoutWriter
	} else {
		streamOptions.Stdout = &stdout
	}
	if stderrWriter != nil {
		streamOptions.Stderr = stderrWriter
	} else if !tty {
		streamOptions.Stderr = &stderr
	}
	if stdin != nil {
		streamOptions.Stdin = stdin
	}
	streamOptions.Tty = tty
	if tty {
		streamOptions.TerminalSizeQueue = currentTerminalSizeQueue(stdoutWriter)
	}

	err = exec.StreamWithContext(ctx, streamOptions)
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("exec error: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), stderr.String(), nil
}

// fetchTerraformFileTree lists files from the terraform executor pod and builds a tree
func fetchTerraformFileTree(ctx context.Context, conn *k8sConnection, namespace, podName, basePath string) (*TerraformFileTree, error) {
	// List files and directories, marking dirs with trailing /
	output, err := execInPod(ctx, conn, namespace, podName, []string{
		"sh", "-c", fmt.Sprintf("find %s -type d -exec sh -c 'echo \"$1/\"' _ {} \\; -o -type f -print", basePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no files found in %s", basePath)
	}

	root := &TerraformFileEntry{
		Path:     basePath,
		RelPath:  ".",
		Name:     path.Base(basePath),
		IsDir:    true,
		Expanded: true,
	}

	// Collect relative paths, tracking which are directories
	dirs := make(map[string]bool)
	var relPaths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		isDir := strings.HasSuffix(line, "/")
		if isDir {
			line = strings.TrimSuffix(line, "/")
		}
		if line == basePath {
			continue
		}
		rel := strings.TrimPrefix(line, basePath+"/")
		if rel == "" {
			continue
		}
		if isDir {
			dirs[rel] = true
		}
		relPaths = append(relPaths, rel)
		// Also mark all parent paths as directories
		parts := strings.Split(rel, "/")
		for i := 1; i < len(parts); i++ {
			dirs[strings.Join(parts[:i], "/")] = true
		}
	}
	sort.Strings(relPaths)
	// Deduplicate (a dir may appear both from -type d and as a parent)
	deduped := relPaths[:0]
	seen := make(map[string]bool)
	for _, rel := range relPaths {
		if !seen[rel] {
			seen[rel] = true
			deduped = append(deduped, rel)
		}
	}
	relPaths = deduped

	// Build tree structure
	nodeMap := map[string]*TerraformFileEntry{".": root}

	for _, rel := range relPaths {
		isDir := dirs[rel]
		parts := strings.Split(rel, "/")
		parentPath := "."
		if len(parts) > 1 {
			parentPath = strings.Join(parts[:len(parts)-1], "/")
		}

		entry := &TerraformFileEntry{
			Path:     basePath + "/" + rel,
			RelPath:  rel,
			Name:     parts[len(parts)-1],
			IsDir:    isDir,
			Depth:    len(parts),
			Expanded: false,
		}
		nodeMap[rel] = entry

		// Ensure parent chain exists
		if _, ok := nodeMap[parentPath]; !ok {
			for i := 1; i <= len(parts)-1; i++ {
				dirPath := strings.Join(parts[:i], "/")
				if _, exists := nodeMap[dirPath]; !exists {
					dir := &TerraformFileEntry{
						Path:     basePath + "/" + dirPath,
						RelPath:  dirPath,
						Name:     parts[i-1],
						IsDir:    true,
						Depth:    i,
						Expanded: false,
					}
					nodeMap[dirPath] = dir
					gpPath := "."
					if i > 1 {
						gpPath = strings.Join(parts[:i-1], "/")
					}
					if gp, exists := nodeMap[gpPath]; exists {
						gp.Children = append(gp.Children, dir)
					}
				}
			}
		}

		if parent, ok := nodeMap[parentPath]; ok {
			parent.Children = append(parent.Children, entry)
		}
	}

	sortFileTree(root)

	tree := &TerraformFileTree{
		PodName:   podName,
		Namespace: namespace,
		BasePath:  basePath,
		Root:      root,
	}
	tree.rebuildFlat()
	return tree, nil
}

func sortFileTree(entry *TerraformFileEntry) {
	sort.Slice(entry.Children, func(i, j int) bool {
		ci, cj := entry.Children[i], entry.Children[j]
		if ci.IsDir != cj.IsDir {
			return ci.IsDir
		}
		return ci.Name < cj.Name
	})
	for _, child := range entry.Children {
		if child.IsDir {
			sortFileTree(child)
		}
	}
}

// rebuildFlat builds the flat list of visible entries for rendering
func (t *TerraformFileTree) rebuildFlat() {
	t.Flat = nil
	t.flattenEntry(t.Root)
}

func (t *TerraformFileTree) flattenEntry(entry *TerraformFileEntry) {
	if entry != t.Root {
		t.Flat = append(t.Flat, entry)
	}
	if entry.IsDir && (entry.Expanded || entry == t.Root) {
		for _, child := range entry.Children {
			t.flattenEntry(child)
		}
	}
}

// fetchFileContent reads a file's content from the executor pod
func fetchFileContentFromPod(ctx context.Context, conn *k8sConnection, namespace, podName, filePath string) (string, error) {
	return execInPod(ctx, conn, namespace, podName, []string{"cat", filePath})
}

func writeFileContentToPod(ctx context.Context, conn *k8sConnection, namespace, podName, filePath string, content []byte) error {
	cleanPath := path.Clean(filePath)
	command := []string{
		"sh",
		"-c",
		fmt.Sprintf("mkdir -p -- %s && cat > %s", shellQuote(path.Dir(cleanPath)), shellQuote(cleanPath)),
	}
	_, _, err := execInPodWithStreams(ctx, conn, namespace, podName, "", command, false, bytes.NewReader(content), nil, nil)
	return err
}

func readWorkspaceFilesFromPod(ctx context.Context, conn *k8sConnection, namespace, podName, workspacePath string) (map[string][]byte, error) {
	basePath := path.Clean(workspacePath)
	output, err := execInPod(ctx, conn, namespace, podName, []string{
		"sh",
		"-c",
		fmt.Sprintf("find %s -type f -print", shellQuote(basePath)),
	})
	if err != nil {
		return nil, err
	}

	files := make(map[string][]byte)
	for _, line := range strings.Split(output, "\n") {
		absPath := strings.TrimSpace(line)
		if absPath == "" {
			continue
		}
		relPath := strings.TrimPrefix(absPath, basePath+"/")
		if !isPatchableTerraformFile(relPath) {
			continue
		}
		content, readErr := fetchFileContentFromPod(ctx, conn, namespace, podName, absPath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read %s: %w", relPath, readErr)
		}
		files[relPath] = []byte(content)
	}
	return files, nil
}

func isPatchableTerraformFile(relPath string) bool {
	relPath = path.Clean(strings.TrimSpace(relPath))
	if relPath == "." || relPath == "" || path.IsAbs(relPath) || strings.HasPrefix(relPath, "../") {
		return false
	}
	if strings.HasPrefix(relPath, ".terraform/") || strings.HasPrefix(relPath, ".omnistrate/") {
		return false
	}
	base := path.Base(relPath)
	if base == ".workspace_hash" ||
		base == "terraform.tfstate" ||
		base == "terraform.tfstate.backup" ||
		strings.HasSuffix(base, ".tfstate") ||
		strings.HasSuffix(base, ".tfstate.backup") ||
		strings.HasSuffix(base, ".tfplan") ||
		strings.HasSuffix(base, ".log") {
		return false
	}
	return true
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

type singleTerminalSizeQueue struct {
	size *remotecommand.TerminalSize
	sent bool
}

func (q *singleTerminalSizeQueue) Next() *remotecommand.TerminalSize {
	if q == nil || q.sent {
		return nil
	}
	q.sent = true
	return q.size
}

func currentTerminalSizeQueue(stdoutWriter io.Writer) remotecommand.TerminalSizeQueue {
	if size := terminalSizeFromWriter(stdoutWriter); size != nil {
		return &singleTerminalSizeQueue{size: size}
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 && height > 0 {
			return &singleTerminalSizeQueue{size: &remotecommand.TerminalSize{Width: uint16(width), Height: uint16(height)}}
		}
	}
	return nil
}

func terminalSizeFromWriter(writer io.Writer) *remotecommand.TerminalSize {
	file, ok := writer.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return nil
	}
	width, height, err := term.GetSize(int(file.Fd()))
	if err != nil || width <= 0 || height <= 0 {
		return nil
	}
	return &remotecommand.TerminalSize{Width: uint16(width), Height: uint16(height)}
}

func makeTerminalRaw(reader io.Reader) func() {
	file, ok := reader.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		file = os.Stdin
	}
	if file == nil || !term.IsTerminal(int(file.Fd())) {
		return func() {}
	}
	fd := int(file.Fd())
	previousState, err := term.MakeRaw(fd)
	if err != nil {
		return func() {}
	}
	return func() {
		_ = term.Restore(fd, previousState)
	}
}

type cancelableInteractiveStdin struct {
	reader cancelreader.CancelReader
	done   chan struct{}
	once   sync.Once
}

func newCancelableInteractiveStdin(stdin io.Reader) (io.Reader, func()) {
	if stdin == nil {
		return nil, func() {}
	}
	reader, err := cancelreader.NewReader(stdin)
	if err != nil {
		return stdin, func() {}
	}
	cancelable := &cancelableInteractiveStdin{
		reader: reader,
		done:   make(chan struct{}),
	}
	return cancelable, cancelable.close
}

func (r *cancelableInteractiveStdin) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err != nil {
		r.once.Do(func() { close(r.done) })
	}
	return n, err
}

func (r *cancelableInteractiveStdin) close() {
	r.reader.Cancel()
	select {
	case <-r.done:
	case <-time.After(200 * time.Millisecond):
	}
	_ = r.reader.Close()
}

var interactiveExecErrorHandlersMu sync.Mutex

func suppressInteractiveExecTeardownErrors() func() {
	interactiveExecErrorHandlersMu.Lock()
	previousHandlers := append([]k8sruntime.ErrorHandler(nil), k8sruntime.ErrorHandlers...)
	k8sruntime.ErrorHandlers = []k8sruntime.ErrorHandler{
		func(ctx context.Context, err error, msg string, keysAndValues ...interface{}) {
			if isInteractiveExecTeardownError(err) {
				return
			}
			for _, handler := range previousHandlers {
				handler(ctx, err, msg, keysAndValues...)
			}
		},
	}
	return func() {
		// remotecommand reports teardown errors from goroutines that are not
		// joined by StreamWithContext, so keep the filter installed briefly
		// after stdin cancellation.
		time.Sleep(150 * time.Millisecond)
		k8sruntime.ErrorHandlers = previousHandlers
		interactiveExecErrorHandlersMu.Unlock()
	}
}

func isInteractiveExecTeardownError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed) ||
		errors.Is(err, cancelreader.ErrCanceled) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return msg == "eof" ||
		strings.Contains(msg, "read canceled") ||
		strings.Contains(msg, "stream closed") ||
		strings.Contains(msg, "use of closed network connection")
}

func interactiveTerraformShellEntrypoint(session *terraformDebugSession) string {
	if session == nil {
		return "exec sh -i"
	}
	if session.WorkspacePath == "" || session.BootstrapScriptPath == "" {
		if session.ShellEntrypoint != "" {
			return session.ShellEntrypoint
		}
		return "exec sh -i"
	}

	historyPath := path.Join(session.WorkspacePath, ".omnistrate", "shell_history")
	historyDir := path.Dir(historyPath)
	var b strings.Builder
	b.WriteString("export TERM=\"${TERM:-xterm-256color}\"\n")
	b.WriteString("export COLORTERM=\"${COLORTERM:-truecolor}\"\n")
	b.WriteString("export AWS_PAGER=\"${AWS_PAGER:-}\"\n")
	b.WriteString("stty sane 2>/dev/null || true\n")
	b.WriteString("cd -- ")
	b.WriteString(shellQuote(session.WorkspacePath))
	b.WriteString("\n")
	b.WriteString(". ")
	b.WriteString(shellQuote(session.BootstrapScriptPath))
	b.WriteString("\n")
	b.WriteString("mkdir -p -- ")
	b.WriteString(shellQuote(historyDir))
	b.WriteString("\n")
	b.WriteString("export HISTFILE=")
	b.WriteString(shellQuote(historyPath))
	b.WriteString("\n")
	b.WriteString("export PS1='\\w # '\n")
	b.WriteString("if command -v bash >/dev/null 2>&1; then exec bash --noprofile --norc -i; fi\n")
	b.WriteString("if [ -n \"${SHELL:-}\" ] && command -v \"$SHELL\" >/dev/null 2>&1; then exec \"$SHELL\" -i; fi\n")
	b.WriteString("exec sh -i\n")
	return b.String()
}

type kubernetesExecCommand struct {
	ctx       context.Context
	conn      *k8sConnection
	namespace string
	podName   string
	container string
	command   []string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

func (c *kubernetesExecCommand) SetStdin(r io.Reader)  { c.stdin = r }
func (c *kubernetesExecCommand) SetStdout(w io.Writer) { c.stdout = w }
func (c *kubernetesExecCommand) SetStderr(w io.Writer) { c.stderr = w }

func (c *kubernetesExecCommand) Run() error {
	if c.stdout != nil {
		_, _ = fmt.Fprint(c.stdout, "\r\n\x1b[2J\x1b[H")
		defer func() {
			_, _ = fmt.Fprint(c.stdout, "\r\n")
		}()
	}
	restoreErrorHandlers := suppressInteractiveExecTeardownErrors()
	defer restoreErrorHandlers()
	restoreTerminal := makeTerminalRaw(c.stdin)
	defer restoreTerminal()
	stdin, closeStdin := newCancelableInteractiveStdin(c.stdin)
	defer closeStdin()

	_, _, err := execInPodWithStreams(
		c.ctx,
		c.conn,
		c.namespace,
		c.podName,
		c.container,
		c.command,
		true,
		stdin,
		c.stdout,
		c.stderr,
	)
	return err
}
