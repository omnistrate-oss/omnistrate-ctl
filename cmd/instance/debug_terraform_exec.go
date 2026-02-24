package instance

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
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
	req := conn.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: command,
			Stdout:  true,
			Stderr:  true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(conn.restConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", fmt.Errorf("exec error: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// fetchTerraformFileTree lists files from the terraform executor pod and builds a tree
func fetchTerraformFileTree(ctx context.Context, conn *k8sConnection, namespace, podName, basePath string) (*TerraformFileTree, error) {
	output, err := execInPod(ctx, conn, namespace, podName, []string{
		"find", basePath, "-type", "f", "-o", "-type", "d",
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

	// Collect relative paths
	dirs := make(map[string]bool)
	var relPaths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == basePath {
			continue
		}
		rel := strings.TrimPrefix(line, basePath+"/")
		if rel == "" {
			continue
		}
		relPaths = append(relPaths, rel)
	}
	sort.Strings(relPaths)

	// Determine directories (paths that have children)
	for _, rel := range relPaths {
		parts := strings.Split(rel, "/")
		for i := 1; i < len(parts); i++ {
			dirs[strings.Join(parts[:i], "/")] = true
		}
	}

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
