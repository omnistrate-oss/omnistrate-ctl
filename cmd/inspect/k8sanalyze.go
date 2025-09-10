package inspect

import (
	"github.com/spf13/cobra"
)

// Cmd is the main inspect command
var Cmd = &cobra.Command{
	Use:   "inspect [instance-id]",
	Short: "Interactive TUI to inspect Kubernetes resources",
	Long: `This command provides an interactive Terminal UI to inspect resources in a Kubernetes namespace.
	
The command connects to a Kubernetes cluster using your kubeconfig file and displays resources 
in the specified namespace. The instance-id parameter is used as the namespace name.

Three main views are provided:
1. Workload View - Shows StatefulSets and Deployments with their pods grouped by Availability Zone
2. Infrastructure View - Shows cluster infrastructure organized by Availability Zone, VMs, and pods
3. Storage View - Shows StatefulSets and Deployments with their pods, PVCs, and PVs in a hierarchy. 
   Click on a PV to show a detailed pop-up with storage class information.

Navigation:
- TAB: Switch between views
- ↑/↓: Navigate through the tree
- ENTER: Expand/collapse nodes
- q: Quit the TUI

Connection to Kubernetes:
- Uses your local kubeconfig file (default: ~/.kube/config)
- Can specify alternate kubeconfig with --kubeconfig flag
- Can specify Kubernetes context with --context flag`,
	Args:         cobra.ExactArgs(1),
	RunE:         runInspect,
	SilenceUsage: true,
}
