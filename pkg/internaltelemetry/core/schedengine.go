package core

import (
	"sync"

	"golang.org/x/exp/slices"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

type TelemetrySchedulingEngineConfig struct {
	ImmediateWeight float64
	FutureWeight float64
}

func DefaultTelemetrySchedulingEngineConfig() TelemetrySchedulingEngineConfig {
	immediateWeight := 0.75 * 100
	return TelemetrySchedulingEngineConfig{
		ImmediateWeight: immediateWeight,
		FutureWeight: 100 - immediateWeight,
	}
}

type TelemetrySchedulingEngine struct {
	// TODO this should be a thread-safe bounded LRU cache
	deploymentSchedulingCache sync.Map // objectKey(intdepl) -> *DeploymentSchedulingState
	config TelemetrySchedulingEngineConfig
}

func NewTelemetrySchedulingEngine(config TelemetrySchedulingEngineConfig) *TelemetrySchedulingEngine {
	return &TelemetrySchedulingEngine{
		deploymentSchedulingCache: sync.Map{},
		config: config,
	}
}

func (t *TelemetrySchedulingEngine) PrepareForPodScheduling(
	network *Network,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	previouslyScheduledNodes []ScheduledNode,
) {
	// TODO cached should used for performance reasons, but it needs to be carefully managed (e.g. watch terminated pods etc)
	// for now let's recompute it for each pod
	// if _, ok := t.deploymentSchedulingCache.Load(client.ObjectKeyFromObject(intdepl)); ok {
		// TODO check if cached version matches current once network is mutable
		// return
	// }
	state := newDeploymentSchedulingState(network)
	network.IterVertices(func(v *Vertex) {
		if _, ok := network.TelemetryEnabledSwitches[v.Name]; ok {
			neighbors := v.Neighbors()
			res := make(map[string]struct{}, len(neighbors))
			for _, neighbor := range neighbors {
				res[neighbor.Name] = struct{}{}
			}
			state.SetPortMetaOf(v, &TelemetryPortsMeta{AvailableTelemetryPorts: res})
		} else {
			state.SetPortMetaOf(v, &TelemetryPortsMeta{AvailableTelemetryPorts: map[string]struct{}{}})
		}
	})
	t.deploymentSchedulingCache.Store(client.ObjectKeyFromObject(intdepl), state)
	if len(previouslyScheduledNodes) > 0 {
		// this can happen if sched crashed or in tests where we don't have cached 
		// network state but some pods of this deployment have already been scheduled
		virtualScheduledNodes := []ScheduledNode{}
		for _, sn := range previouslyScheduledNodes {
			for _, depl := range sn.ScheduledDeployments {
				v := network.Vertices[sn.Name]
				virtualScheduledNodes = t.markScheduled(network, state, virtualScheduledNodes, depl, v)
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

func (t *TelemetrySchedulingEngine) GetCachedDeploymentsNetworkView(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) (*DeploymentSchedulingState, bool) {
	state, ok := t.deploymentSchedulingCache.Load(client.ObjectKeyFromObject(intdepl))
	if !ok {
		return nil, false
	}
	return state.(*DeploymentSchedulingState), true
}

func (t *TelemetrySchedulingEngine) MustGetCachedDeploymentsNetworkView(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) *DeploymentSchedulingState {
	if state, ok := t.GetCachedDeploymentsNetworkView(intdepl); ok {
		return state
	}
	panic("assertion failed")
}

func (t *TelemetrySchedulingEngine) getGreedyBestNodeToSchedule(
	network *Network,
	schedulingState *DeploymentSchedulingState,
	scheduledNodes []ScheduledNode,
	deplName string,
) (*Vertex, int) {
	bestGain := -1
	var bestNode *Vertex = nil 
	// TODO optimize this
	network.IterVerticesOfType(shimv1alpha.NODE, func(v *Vertex) {
		gain := t.computeImmediatePortsGainedByScheduling(network, schedulingState, deplName, v, scheduledNodes)
		if gain > bestGain {
			bestNode = v
			bestGain = gain
		}
	})
	return bestNode, bestGain
}

func (t *TelemetrySchedulingEngine) markScheduled(
	network *Network,
	schedulingState *DeploymentSchedulingState,
	scheduledNodes []ScheduledNode,
	deplName string,
	nodeVertex *Vertex,
) []ScheduledNode {
	visited := make([]bool, len(network.Vertices))
	for i := 0; i < len(visited); i++ {
		visited[i] = false
	}
	targets := t.getNodesWithPodsOfOtherDeployment(deplName, scheduledNodes)
	var dfs func(*Vertex, *Vertex, bool) (bool, bool)
	dfs = func(cur *Vertex, prev *Vertex, foundTelemetrySwitchBefore bool) (leadsToTarget bool, foundTelemetrySwitchAfter bool) {
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
					curMeta := schedulingState.PortMetaOf(cur)
					if hasTelemetryEnabled && curMeta.IsPortUnallocated(neigh.Name) && areOtherSwitchesOnRoute {
						curMeta.AllocatePort(neigh.Name)
					}
					if hasTelemetryEnabled && curMeta.IsPortUnallocated(prev.Name) && areOtherSwitchesOnRoute {
						curMeta.AllocatePort(prev.Name)
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
	network *Network,
	schedulingState *DeploymentSchedulingState,
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
			node, gain := t.getGreedyBestNodeToSchedule(network, schedulingState, scheduledNodes, otherDeplName)
			totalGain += gain
			scheduledNodes = t.markScheduled(network, schedulingState, scheduledNodes, otherDeplName, node)
		}
	}
	return totalGain
}

func (t *TelemetrySchedulingEngine) computeImmediatePortsGainedByScheduling(
	network *Network,
	schedulingState *DeploymentSchedulingState,
	podsDeploymentName string,
	nodeVertex *Vertex,
	previouslyScheduledNodes []ScheduledNode,
) int {
	visited := make([]bool, len(network.Vertices))
	for i := 0; i < len(visited); i++ {
		visited[i] = false
	}
	targetNodes := t.getNodesWithPodsOfOtherDeployment(podsDeploymentName, previouslyScheduledNodes)
	
	var dfs func(*Vertex, *Vertex, bool) (bool, int, bool)
	dfs = func(
		cur *Vertex,
		prev *Vertex,
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
		curMeta := schedulingState.PortMetaOf(cur)
		for _, neigh := range cur.Neighbors() {
			if !visited[neigh.Ordinal] {
				neighHadTelemetryOnRoute := foundTelemetrySwitchBefore || hasTelemetryEnabled
				if neighLeadsToTarget, numPortsFromNeigh, foundTelemetryAfter := dfs(neigh, cur, neighHadTelemetryOnRoute); neighLeadsToTarget {
					foundTelemetrySwitchAfter = foundTelemetrySwitchAfter || foundTelemetryAfter
					leadsToTarget = true
					numPortsInDescendats += numPortsFromNeigh
					areOtherSwitchesOnRoute := foundTelemetrySwitchBefore || foundTelemetryAfter
					if hasTelemetryEnabled && curMeta.IsPortUnallocated(neigh.Name) && areOtherSwitchesOnRoute {
						numPortsInDescendats++
					}
				}
			}
		}
		if hasTelemetryEnabled && leadsToTarget && 
				(foundTelemetrySwitchBefore || foundTelemetrySwitchAfter) &&
				curMeta.IsPortUnallocated(prev.Name) {
			numPortsInDescendats++
		}
		if hasTelemetryEnabled {
			foundTelemetrySwitchAfter = true
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

func (t *TelemetrySchedulingEngine) mustLoadStateCopy(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) *DeploymentSchedulingState {
	s, ok := t.deploymentSchedulingCache.Load(client.ObjectKeyFromObject(intdepl))
	if !ok {
		klog.Errorf("failed to load scheduled state for %v", client.ObjectKeyFromObject(intdepl))
		panic("assertion failed")
	}
	return s.(*DeploymentSchedulingState).DeepCopy()
}

func (t *TelemetrySchedulingEngine) ComputeNodeSchedulingScore(
	network *Network,
	nodeName string,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	pod *v1.Pod,
	podsDeploymentName string,
	scheduledNodes []ScheduledNode,
	queuedPods QueuedPods,
) int {
	if t.isDeploymentMemberAlreadyScheduledOnNode(nodeName, scheduledNodes, podsDeploymentName) {
		return 0
	}
	nodeVertex := network.Vertices[nodeName]
	state := t.mustLoadStateCopy(intdepl)

	immediateGain := t.computeImmediatePortsGainedByScheduling(network, state, podsDeploymentName, nodeVertex, scheduledNodes)
	scheduledNodes = deepCopyScheduledNodes(scheduledNodes)

	scheduledNodes = t.markScheduled(network, state, scheduledNodes, podsDeploymentName, nodeVertex)
	futureGain := t.approximateFuturePortsGainedByScheduling(network, state, queuedPods, scheduledNodes, podsDeploymentName)

	return int(float64(immediateGain) * t.config.ImmediateWeight + float64(futureGain) * t.config.FutureWeight)
}
