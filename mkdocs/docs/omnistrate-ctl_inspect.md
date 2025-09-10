## omnistrate-ctl inspect

Interactive TUI to inspect Kubernetes resources

### Synopsis

This command provides an interactive Terminal UI to inspect resources in a Kubernetes namespace.
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
- Can specify Kubernetes context with --context flag

```
omnistrate-ctl inspect [instance-id] [flags]
```

### Options

```
      --context string      Kubernetes context to use
  -h, --help                help for inspect
      --kubeconfig string   Path to the kubeconfig file (default "~/.kube/config")
  -o, --output string       Output format (table|text|json) (default "table")
      --text                Output text representation (shorthand for --output=text)
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl](omnistrate-ctl.md) - Manage your Omnistrate SaaS from the command line
