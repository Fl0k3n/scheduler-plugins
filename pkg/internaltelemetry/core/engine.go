package core

import (
	"slices"

	v1 "k8s.io/api/core/v1"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

type TelemetrySchedulingEngine struct {
	deploymentNetworkCache map[string]*Network[TelemetryPortsMeta] // iintdeplName -> meta
}

func NewTelemetrySchedulingEngine() *TelemetrySchedulingEngine {
	return &TelemetrySchedulingEngine{
		deploymentNetworkCache: map[string]*Network[TelemetryPortsMeta]{},
	}
}

func (t *TelemetrySchedulingEngine) PrepareForScheduling(network *Network[any], intdeplName string) {
	if _, ok := t.deploymentNetworkCache[intdeplName]; ok {
		// TODO check if cached version matches current once network is mutable
		return
	}
	deploymentsNetworkView := mapDeepCopy[any, TelemetryPortsMeta](network, func(v *Vertex[any]) TelemetryPortsMeta {
		if _, ok := network.TelemetryEnabledSwitches[v.Name]; ok {
			neighbors := v.Neighbors()
			res := make(map[string]struct{}, len(neighbors))
			for _, neighbor := range neighbors {
				res[neighbor.Name] = struct{}{}
			}
		}
		return TelemetryPortsMeta{AvailableTelemetryPorts: map[string]struct{}{}}
	})
	// TODO check if some pods have already been scheduled (e.g. if sched crashed) and remove ports
	t.deploymentNetworkCache[intdeplName] = deploymentsNetworkView
}

func (t *TelemetrySchedulingEngine) doTheMath() int {
	return 1
}

func (t *TelemetrySchedulingEngine) greedyGetBestNodes() (nodes []string, usedDeplName string) {
	return []string{}, ""
}

func (t *TelemetrySchedulingEngine) computeNumberOfNewTelemetryEnabledPorts(
	scheduledNodes []ScheduledNode,
	queuedPods QueuedPods,
	depth int,
) int {
	if queuedPods.AllQueued() {
		return t.doTheMath()
	}
	MAX_DEPTH_TO_CONSIDER_MORE_THAN_ONE := 1
	maxNodesToConsider := 3
	if depth > MAX_DEPTH_TO_CONSIDER_MORE_THAN_ONE {
		maxNodesToConsider = 1
	}
	// TODO check mntc < len(nodes)
	bestNodes, selectedDeplName := t.greedyGetBestNodes()
	for i := 0; i < maxNodesToConsider; i++ {
		candidate := bestNodes[i]
	}
}

func (t *TelemetrySchedulingEngine) approximateFuturePortsGainedByScheduling(
	queuedPods QueuedPods,
	scheduledNodes []ScheduledNode,
	previouslyScheduledDeplName string,
) int {
	// Greedy simpliefied approach where we don't account for nodes that don't have
	// enough resources to schedule one type of pod, futhermore, after a pod of each of these deployments
	// have beed scheduled we don't care about scheduling order of remaining pods and just perform
	// greedy selection to put pods to places where most new ports can be covered
	// this probably isn't an optimal strategy
	var otherDeplName string
	for k := range queuedPods.PerDeploymentCounts {
		if k != previouslyScheduledDeplName {
			otherDeplName = k	
			break
		}
	}
	if queuedPods.PerDeploymentCounts[otherDeplName] > 0 {
		
	}
	return 0
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
	visited[nodeVertex.Ordinal] = true
	targetNodes := t.getNodesWithPodsOfOtherDeployment(podsDeploymentName, previouslyScheduledNodes)
	
	var dfs func(*Vertex[TelemetryPortsMeta], bool) (bool, int, bool)
	dfs = func(
		cur *Vertex[TelemetryPortsMeta],
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
				if neighLeadsToTarget, descendands, foundTelemetrySwitchAfter := dfs(neigh, neighHadTelemetryOnRoute); neighLeadsToTarget {
					leadsToTarget = true
					numPortsInDescendats += descendands
					areOtherSwitchesOnRoute := foundTelemetrySwitchBefore || foundTelemetrySwitchAfter
					if hasTelemetryEnabled && cur.Meta.IsPortUnallocated(neigh.Name) && areOtherSwitchesOnRoute {
						numPortsInDescendats++
					}
				}
			}
		}
		return
	}
	visited[nodeVertex.Ordinal] = true
	if nodeVertex.Parent == nil {
		return 0
	}
	_, newTelemetryPortsCoveredBySchedulingOnNode, _ := dfs(nodeVertex.Parent, false)
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

func (t *TelemetrySchedulingEngine) ComputeNodeSchedulingScore(
	nodeName string,
	intdeplName string,
	pod *v1.Pod,
	podsDeploymentName string,
	scheduledNodes []ScheduledNode,
	queuedPods QueuedPods,
) int {
	var node ScheduledNode
	for _, node = range scheduledNodes {
		if nodeName == node.Name {
			break
		}
	}
	if slices.Contains(node.ScheduledDeployments, podsDeploymentName) {
		return 0
	}
	network := t.deploymentNetworkCache[intdeplName]
	nodeVertex := network.Vertices[nodeName]
	immediateGain := t.computeImmediatePortsGainedByScheduling(network, podsDeploymentName, nodeVertex, scheduledNodes)
	node.MarkScheduled(podsDeploymentName)
	futureGain := t.approximateFuturePortsGainedByScheduling(queuedPods, scheduledNodes)
	immediateWeight := 0.75
	futureWeight := 1 - immediateWeight
	return int(float64(immediateGain) * immediateWeight + float64(futureGain) * futureWeight)
}
