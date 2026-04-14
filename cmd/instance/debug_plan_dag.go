package instance

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

type PlanDAG struct {
	Nodes            map[string]PlanDAGNode      `json:"nodes"`
	Edges            []PlanDAGEdge               `json:"edges"`
	Levels           [][]string                  `json:"levels"`
	Errors           []string                    `json:"errors,omitempty"`
	HasCycle         bool                        `json:"hasCycle"`
	WorkflowID       string                      `json:"workflowId,omitempty"`
	ProgressByID     map[string]ResourceProgress `json:"progressById,omitempty"`
	ProgressByKey    map[string]ResourceProgress `json:"progressByKey,omitempty"`
	ProgressByName   map[string]ResourceProgress `json:"progressByName,omitempty"`
	BreakpointByID   map[string]string           `json:"breakpointById,omitempty"`
	BreakpointByKey  map[string]string           `json:"breakpointByKey,omitempty"`
	BreakpointByName map[string]string           `json:"breakpointByName,omitempty"`
	ProgressLoading  bool                        `json:"-"`
	SpinnerTick      int                         `json:"-"`
	// Per-resource workflow step summaries keyed by resource key
	WorkflowStepsByKey map[string]*ResourceWorkflowSteps `json:"workflowStepsByKey,omitempty"`
}

type PlanDAGNode struct {
	ID   string `json:"id"`
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type PlanDAGEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func buildPlanDAG(ctx context.Context, token, serviceID string, instanceData *openapiclientfleet.ResourceInstance) (*PlanDAG, error) {
	if instanceData == nil {
		return nil, fmt.Errorf("instance data is nil")
	}

	productTierID := instanceData.ProductTierId
	tierVersion := instanceData.TierVersion
	if productTierID == "" || tierVersion == "" {
		return nil, fmt.Errorf("missing product tier information for instance")
	}

	versionSet, err := dataaccess.DescribeVersionSet(ctx, token, serviceID, productTierID, tierVersion)
	if err != nil {
		return nil, err
	}

	plan := &PlanDAG{
		Nodes:  make(map[string]PlanDAGNode),
		Edges:  []PlanDAGEdge{},
		Errors: []string{},
	}

	for _, resource := range versionSet.Resources {
		node := PlanDAGNode{
			ID:   resource.Id,
			Name: resource.Name,
		}
		if resource.UrlKey != nil {
			node.Key = *resource.UrlKey
		}
		if resource.ManagedResourceType != nil {
			node.Type = *resource.ManagedResourceType
		}
		plan.Nodes[resource.Id] = node
	}

	for resourceID, node := range plan.Nodes {
		resourceDetails, err := dataaccess.DescribeResource(ctx, token, serviceID, resourceID, &productTierID, &tierVersion)
		if err != nil {
			plan.Errors = append(plan.Errors, fmt.Sprintf("resource %s: %v", nodeLabel(node), err))
			continue
		}

		node.Name = resourceDetails.Name
		node.Key = resourceDetails.Key
		node.Type = resourceDetails.ResourceType
		plan.Nodes[resourceID] = node

		for _, dependency := range resourceDetails.Dependencies {
			depID := dependency.ResourceId
			if depID == "" {
				continue
			}
			if _, ok := plan.Nodes[depID]; !ok {
				plan.Nodes[depID] = PlanDAGNode{
					ID:   depID,
					Name: depID,
				}
			}
			plan.Edges = append(plan.Edges, PlanDAGEdge{
				From: depID,
				To:   resourceID,
			})
		}
	}

	filterPlanDAG(plan)
	plan.Levels, plan.HasCycle = computePlanLevels(plan.Nodes, plan.Edges)
	attachBreakpointStatuses(plan, instanceData)
	return plan, nil
}

func filterPlanDAG(plan *PlanDAG) {
	if plan == nil {
		return
	}

	hidden := map[string]struct{}{}
	for id, node := range plan.Nodes {
		if shouldHidePlanNode(node) {
			hidden[id] = struct{}{}
			delete(plan.Nodes, id)
		}
	}

	if len(hidden) == 0 {
		return
	}

	filtered := plan.Edges[:0]
	for _, edge := range plan.Edges {
		if _, ok := hidden[edge.From]; ok {
			continue
		}
		if _, ok := hidden[edge.To]; ok {
			continue
		}
		filtered = append(filtered, edge)
	}
	plan.Edges = filtered
}

func shouldHidePlanNode(node PlanDAGNode) bool {
	labels := []string{node.Key, node.Name, node.ID}
	for _, label := range labels {
		lower := strings.ToLower(label)
		if strings.Contains(lower, "omnistratecloudaccountconfig") {
			return true
		}
		if strings.Contains(lower, "cloudaccountconfig") {
			return true
		}
		if strings.Contains(lower, "cloud-account-config") {
			return true
		}
		if strings.Contains(lower, "omnistrateobserv") {
			return true
		}
	}
	return false
}

func computePlanLevels(nodes map[string]PlanDAGNode, edges []PlanDAGEdge) ([][]string, bool) {
	indegree := make(map[string]int)
	adjacency := make(map[string][]string)

	for id := range nodes {
		indegree[id] = 0
		adjacency[id] = []string{}
	}

	for _, edge := range edges {
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
		indegree[edge.To]++
	}

	var levels [][]string
	processed := 0
	ready := collectZeroIndegree(indegree)

	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool {
			return ready[i] < ready[j]
		})
		levels = append(levels, ready)
		processed += len(ready)

		next := []string{}
		for _, nodeID := range ready {
			for _, dependent := range adjacency[nodeID] {
				indegree[dependent]--
				if indegree[dependent] == 0 {
					next = append(next, dependent)
				}
			}
		}
		ready = next
	}

	if processed != len(nodes) {
		var remaining []string
		for id, degree := range indegree {
			if degree > 0 {
				remaining = append(remaining, id)
			}
		}
		sort.Strings(remaining)
		if len(remaining) > 0 {
			levels = append(levels, remaining)
		}
		return levels, true
	}

	return levels, false
}

func collectZeroIndegree(indegree map[string]int) []string {
	ready := []string{}
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	return ready
}

func nodeLabel(node PlanDAGNode) string {
	switch {
	case node.Key != "":
		return node.Key
	case node.Name != "":
		return node.Name
	default:
		return node.ID
	}
}

func attachBreakpointStatuses(plan *PlanDAG, instanceData *openapiclientfleet.ResourceInstance) {
	if plan == nil || instanceData == nil {
		return
	}

	byID := make(map[string]string)
	byKey := make(map[string]string)
	byName := make(map[string]string)

	activeBreakpoints := instanceData.GetActiveBreakpoints()
	if len(activeBreakpoints) == 0 {
		plan.BreakpointByID = byID
		plan.BreakpointByKey = byKey
		plan.BreakpointByName = byName
		return
	}

	for _, breakpoint := range activeBreakpoints {
		idOrKey := strings.TrimSpace(breakpoint.GetId())
		if idOrKey == "" {
			continue
		}

		status := normalizeBreakpointStatus(breakpoint.GetStatus())
		for _, node := range plan.Nodes {
			if !strings.EqualFold(node.ID, idOrKey) &&
				!strings.EqualFold(node.Key, idOrKey) &&
				!strings.EqualFold(node.Name, idOrKey) {
				continue
			}

			byID[node.ID] = status
			if node.Key != "" {
				byKey[node.Key] = status
			}
			if node.Name != "" {
				byName[node.Name] = status
			}
		}
	}

	plan.BreakpointByID = byID
	plan.BreakpointByKey = byKey
	plan.BreakpointByName = byName
}

func breakpointStatusForNode(plan *PlanDAG, node PlanDAGNode) (string, bool) {
	if plan == nil {
		return "", false
	}

	if plan.BreakpointByID != nil {
		if status, ok := plan.BreakpointByID[node.ID]; ok {
			return status, true
		}
	}

	if node.Key != "" && plan.BreakpointByKey != nil {
		if status, ok := plan.BreakpointByKey[node.Key]; ok {
			return status, true
		}
	}

	if node.Name != "" && plan.BreakpointByName != nil {
		if status, ok := plan.BreakpointByName[node.Name]; ok {
			return status, true
		}
	}

	return "", false
}

func normalizeBreakpointStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return "pending"
	}
	return strings.ToLower(trimmed)
}

func hasHitBreakpoint(status string) bool {
	return normalizeBreakpointStatus(status) == "hit"
}

func planHasHitBreakpoint(plan *PlanDAG) bool {
	if plan == nil {
		return false
	}

	for _, status := range plan.BreakpointByID {
		if hasHitBreakpoint(status) {
			return true
		}
	}

	for _, status := range plan.BreakpointByKey {
		if hasHitBreakpoint(status) {
			return true
		}
	}

	for _, status := range plan.BreakpointByName {
		if hasHitBreakpoint(status) {
			return true
		}
	}

	return false
}
