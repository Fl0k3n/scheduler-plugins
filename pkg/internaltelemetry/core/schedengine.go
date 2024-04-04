package core

import (
	"maps"
	"slices"

	v1 "k8s.io/api/core/v1"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

type TelemetrySchedulingEngineConfig struct {
	ImmediateWeight float64
	FutureWeight float64
}

func DefaultTelemetrySchedulingEngineConfig() TelemetrySchedulingEngineConfig {
	immediateWeight := 0.75
	return TelemetrySchedulingEngineConfig{
		ImmediateWeight: immediateWeight,
		FutureWeight: 1 - immediateWeight,
	}
}

type TelemetrySchedulingEngine struct {
	deploymentNetworkCache map[string]*Network[TelemetryPortsMeta] // iintdeplName -> meta
	config TelemetrySchedulingEngineConfig
}

func NewTelemetrySchedulingEngine(config TelemetrySchedulingEngineConfig) *TelemetrySchedulingEngine {
	return &TelemetrySchedulingEngine{
		deploymentNetworkCache: map[string]*Network[TelemetryPortsMeta]{},
		config: config,
	}
}

func (t *TelemetrySchedulingEngine) PrepareForScheduling(
	network *Network[Nothing],
	intdeplName string,
	previouslyScheduleNodes []ScheduledNode,
) {
	if _, ok := t.deploymentNetworkCache[intdeplName]; ok {
		// TODO check if cached version matches current once network is mutable
		return
	}
	deploymentsNetworkView := mapDeepCopy(network, func(v *Vertex[Nothing]) TelemetryPortsMeta {
		if _, ok := network.TelemetryEnabledSwitches[v.Name]; ok {
			neighbors := v.Neighbors()
			res := make(map[string]struct{}, len(neighbors))
			for _, neighbor := range neighbors {
				res[neighbor.Name] = struct{}{}
			}
			return TelemetryPortsMeta{AvailableTelemetryPorts: res}
		}
		return TelemetryPortsMeta{AvailableTelemetryPorts: map[string]struct{}{}}
	})
	t.deploymentNetworkCache[intdeplName] = deploymentsNetworkView
	if len(previouslyScheduleNodes) > 0 {
		// this can happen if sched crashed or in tests where we don't have cached 
		// network state but some pods of this deployment have already been scheduled
		virtualScheduledNodes := []ScheduledNode{}
		for _, sn := range previouslyScheduleNodes {
			for _, depl := range sn.ScheduledDeployments {
				v := deploymentsNetworkView.Vertices[sn.Name]
				virtualScheduledNodes = t.markScheduled(deploymentsNetworkView, virtualScheduledNodes, depl, v)
			}
		}
	}
}

// count is performed based on network metadata, not actual flow tracing
// func (t *TelemetrySchedulingEngine) countCoveredPorts(network *Network[TelemetryPortsMeta]) int {
// 	res := 0
// 	network.IterVerticesOfType(shimv1alpha.INC_SWITCH, func(v *Vertex[TelemetryPortsMeta]) {
// 		if _, hasTelemetryEnabled := network.TelemetryEnabledSwitches[v.Name]; hasTelemetryEnabled {
// 			res += len(v.Neighbors()) - len(v.Meta.AvailableTelemetryPorts)
// 		}
// 	})
// 	return res
// }

func (t *TelemetrySchedulingEngine) getCachedDeploymentsNetworkView(intdeplName string) *Network[TelemetryPortsMeta] {
	return t.deploymentNetworkCache[intdeplName]	
}

func (t *TelemetrySchedulingEngine) getGreedyBestNodeToSchedule(
	network *Network[TelemetryPortsMeta],
	scheduledNodes []ScheduledNode,
	deplName string,
) (*Vertex[TelemetryPortsMeta], int) {
	bestGain := -1
	var bestNode *Vertex[TelemetryPortsMeta] = nil 
	// TODO optimize this
	network.IterVerticesOfType(shimv1alpha.NODE, func(v *Vertex[TelemetryPortsMeta]) {
		gain := t.computeImmediatePortsGainedByScheduling(network, deplName, v, scheduledNodes)
		if gain > bestGain {
			bestNode = v
			bestGain = gain
		}
	})
	return bestNode, bestGain
}

func (t *TelemetrySchedulingEngine) markScheduled(
	network *Network[TelemetryPortsMeta],
	scheduledNodes []ScheduledNode,
	deplName string,
	nodeVertex *Vertex[TelemetryPortsMeta],
) []ScheduledNode {
	visited := make([]bool, len(network.Vertices))
	for i := 0; i < len(visited); i++ {
		visited[i] = false
	}
	targets := t.getNodesWithPodsOfOtherDeployment(deplName, scheduledNodes)
	var dfs func(*Vertex[TelemetryPortsMeta], *Vertex[TelemetryPortsMeta], bool) (bool, bool)
	dfs = func(cur *Vertex[TelemetryPortsMeta], prev *Vertex[TelemetryPortsMeta], foundTelemetrySwitchBefore bool) (leadsToTarget bool, foundTelemetrySwitchAfter bool) {
		visited[cur.Ordinal] = true
		foundTelemetrySwitchAfter = false
		if cur.DeviceType == shimv1alpha.NODE {
			_, leadsToTarget = targets[cur.Name]	
			return
		}
		_, hasTelemetryEnabled := network.TelemetryEnabledSwitches[cur.Name]
		for _, neigh := range cur.Neighbors() {
			if !visited[neigh.Ordinal] {
				neighHadTelemetryOnRoute := foundTelemetrySwitchBefore || hasTelemetryEnabled
				neighLeadsToTarget, foundTelemetryAfter := dfs(neigh, cur, neighHadTelemetryOnRoute)
				if neighLeadsToTarget {
					leadsToTarget = true
					areOtherSwitchesOnRoute := foundTelemetrySwitchBefore || foundTelemetryAfter
					if hasTelemetryEnabled && cur.Meta.IsPortUnallocated(neigh.Name) && areOtherSwitchesOnRoute {
						cur.Meta.AllocatePort(neigh.Name)
					}
					if hasTelemetryEnabled && cur.Meta.IsPortUnallocated(prev.Name) && areOtherSwitchesOnRoute {
						cur.Meta.AllocatePort(prev.Name)
					}
				}
				foundTelemetrySwitchAfter = foundTelemetrySwitchAfter || foundTelemetryAfter
			}
		}
		if hasTelemetryEnabled {
			foundTelemetrySwitchAfter = true
		}
		return
	}
	found := false
	for _, node := range scheduledNodes {
		if node.Name == nodeVertex.Name {
			node.MarkScheduled(deplName)
			found = true
			break
		}
	}
	if !found {
		scheduledNodes = append(scheduledNodes, ScheduledNode{
			Name: nodeVertex.Name,
			ScheduledDeployments: []string{deplName},
		})
	}
	if nodeVertex.Parent == nil {
		return scheduledNodes
	}
	visited[nodeVertex.Ordinal] = true
	dfs(nodeVertex.Parent, nodeVertex, false)
	return scheduledNodes
}

func (t *TelemetrySchedulingEngine) approximateFuturePortsGainedByScheduling(
	network *Network[TelemetryPortsMeta],
	queuedPods QueuedPods,
	scheduledNodes []ScheduledNode,
	previouslyScheduledDeplName string,
) int {
	// Greedy simpliefied approach where we don't account for nodes that don't have
	// enough resources to schedule one type of pod, futhermore, after a pod of each of these deployments
	// have beed scheduled we don't care about scheduling order of remaining pods and just perform
	// greedy selection to put pods to places where most new ports can be covered
	// this probably isn't an optimal strategy
	otherDeplNames := []string{}
	for k := range queuedPods.PerDeploymentCounts {
		if k != previouslyScheduledDeplName {
			otherDeplNames = append(otherDeplNames, k)	
		}
	}
	totalGain := 0
	for _, otherDeplName := range otherDeplNames {
		for i := 0; i < queuedPods.PerDeploymentCounts[otherDeplName]; i++ {
			node, gain := t.getGreedyBestNodeToSchedule(network, scheduledNodes, otherDeplName)
			totalGain += gain
			scheduledNodes = t.markScheduled(network, scheduledNodes, otherDeplName, node)
		}
	}
	return totalGain
}

func (t *TelemetrySchedulingEngine) computeImmediatePortsGainedByScheduling(
	network *Network[TelemetryPortsMeta],
	podsDeploymentName string,
	nodeVertex *Vertex[TelemetryPortsMeta],
	previouslyScheduledNodes []ScheduledNode,
) int {
	visited := make([]bool, len(network.Vertices))
	for i := 0; i < len(visited); i++ {
		visited[i] = false
	}
	targetNodes := t.getNodesWithPodsOfOtherDeployment(podsDeploymentName, previouslyScheduledNodes)
	
	var dfs func(*Vertex[TelemetryPortsMeta], *Vertex[TelemetryPortsMeta], bool) (bool, int, bool)
	dfs = func(
		cur *Vertex[TelemetryPortsMeta],
		prev *Vertex[TelemetryPortsMeta],
		foundTelemetrySwitchBefore bool,
	) (leadsToTarget bool, numPortsInDescendats int, foundTelemetrySwitchAfter bool) {
		visited[cur.Ordinal] = true
		numPortsInDescendats = 0
		foundTelemetrySwitchAfter = false
		// we assume that k8s node is always a leaf
		if cur.DeviceType == shimv1alpha.NODE {
			_, leadsToTarget = targetNodes[cur.Name]
			return
		}
		_, hasTelemetryEnabled := network.TelemetryEnabledSwitches[cur.Name]
		if hasTelemetryEnabled {
			foundTelemetrySwitchAfter = true
		}
		for _, neigh := range cur.Neighbors() {
			if !visited[neigh.Ordinal] {
				neighHadTelemetryOnRoute := foundTelemetrySwitchBefore || hasTelemetryEnabled
				if neighLeadsToTarget, numPortsFromNeigh, foundTelemetryAfter := dfs(neigh, cur, neighHadTelemetryOnRoute); neighLeadsToTarget {
					foundTelemetrySwitchAfter = foundTelemetrySwitchAfter || foundTelemetryAfter
					leadsToTarget = true
					numPortsInDescendats += numPortsFromNeigh
					areOtherSwitchesOnRoute := foundTelemetrySwitchBefore || foundTelemetrySwitchAfter
					if hasTelemetryEnabled && cur.Meta.IsPortUnallocated(neigh.Name) && areOtherSwitchesOnRoute {
						numPortsInDescendats++
					}
					if hasTelemetryEnabled && cur.Meta.IsPortUnallocated(prev.Name) && areOtherSwitchesOnRoute {
						numPortsInDescendats++
					}
				}
			}
		}
		return
	}
	if nodeVertex.Parent == nil {
		return 0
	}
	visited[nodeVertex.Ordinal] = true
	_, newTelemetryPortsCoveredBySchedulingOnNode, _ := dfs(nodeVertex.Parent, nodeVertex, false)
	return newTelemetryPortsCoveredBySchedulingOnNode
}

func (t *TelemetrySchedulingEngine) getNodesWithPodsOfOtherDeployment(
	deployName string,
	scheduledNodes []ScheduledNode,
) map[string]struct{} {
	res := map[string]struct{}{}
	for _, node := range scheduledNodes {
		if len(node.ScheduledDeployments) > 1 || (
		 	  len(node.ScheduledDeployments) == 1 && node.ScheduledDeployments[0] != deployName) {
			res[node.Name] = struct{}{}
		}
	}
	return res
}

func (t *TelemetrySchedulingEngine) isDeploymentMemberAlreadyScheduledOnNode(nodeName string, scheduledNodes []ScheduledNode, podsDeploymentName string) bool {
	var node ScheduledNode
	for _, node = range scheduledNodes {
		if nodeName == node.Name {
			return slices.Contains(node.ScheduledDeployments, podsDeploymentName)
		}
	}
	return false
}

func (t *TelemetrySchedulingEngine) ComputeNodeSchedulingScore(
	nodeName string,
	intdeplName string,
	pod *v1.Pod,
	podsDeploymentName string,
	scheduledNodes []ScheduledNode,
	queuedPods QueuedPods,
) int {
	if t.isDeploymentMemberAlreadyScheduledOnNode(nodeName, scheduledNodes, podsDeploymentName) {
		return 0
	}
	network := t.deploymentNetworkCache[intdeplName]
	nodeVertex := network.Vertices[nodeName]

	immediateGain := t.computeImmediatePortsGainedByScheduling(network, podsDeploymentName, nodeVertex, scheduledNodes)

	// TODO COW
	snapshot := network.TakeMetadataSnapshot(func(tpm TelemetryPortsMeta) TelemetryPortsMeta {
		return TelemetryPortsMeta{AvailableTelemetryPorts: maps.Clone(tpm.AvailableTelemetryPorts)}
	})
	scheduledNodes = deepCopyScheduledNodes(scheduledNodes)
	defer network.RestoreMetadataFromSnapshot(snapshot)

	scheduledNodes = t.markScheduled(network, scheduledNodes, podsDeploymentName, nodeVertex)
	futureGain := t.approximateFuturePortsGainedByScheduling(network, queuedPods, scheduledNodes, podsDeploymentName)

	return int(float64(immediateGain) * t.config.ImmediateWeight + float64(futureGain) * t.config.FutureWeight)
}
