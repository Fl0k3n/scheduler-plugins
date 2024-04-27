package core

import (
	"sync"

	"golang.org/x/exp/slices"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
)

// thread-safe with respect to scheduling framework semantics
type TelemetrySchedulingEngine struct {
	// TODO this should be a thread-safe bounded LRU cache
	deploymentSchedulingCache sync.Map // objectKey(intdepl) -> *DeploymentSchedulingState
}

func NewTelemetrySchedulingEngine() *TelemetrySchedulingEngine {
	return &TelemetrySchedulingEngine{
		deploymentSchedulingCache: sync.Map{},
	}
}

func (t *TelemetrySchedulingEngine) PrepareForPodScheduling(
	network *Network,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	podDeplName string,
	feasibleNodes []*v1.Node,
	feasibleNodesForOppositeDeploymentProvider func() *[]*v1.Node,
	previouslyScheduledNodes []ScheduledNode,
) *DeploymentSchedulingState {
	// TODO cached should be used for performance reasons, but it needs to be carefully managed (e.g. watch terminated pods etc)
	// for now let's recompute it for each pod
	// if _, ok := t.deploymentSchedulingCache.Load(client.ObjectKeyFromObject(intdepl)); ok {
		// TODO check if cached version matches current once network is mutable
		// return
	// }

	oppositeDeployment := getOppositeDeployment(intdepl, podDeplName)
	nodesWithOppositeDeployment := getNodesWithDeployment(previouslyScheduledNodes, oppositeDeployment)
	portMarker := newTelemetryPortMarker()
	portState := portMarker.GetInitialPortState(network)
	var countingEngine *CountingEngine = nil
	if len(nodesWithOppositeDeployment) > 0 {
		countingEngine = newCountingEngine(network, nodesWithOppositeDeployment, portState)
	}
	var feasibleNodesForOppositeDeployment *[]*v1.Node = nil
	if len(nodesWithOppositeDeployment) == 0 {
		feasibleNodesForOppositeDeployment = feasibleNodesForOppositeDeploymentProvider()
	}
	deplSchedulingState := newDeploymentSchedulingState(portState, countingEngine, nodesWithOppositeDeployment, feasibleNodesForOppositeDeployment)
	t.deploymentSchedulingCache.Store(client.ObjectKeyFromObject(intdepl), deplSchedulingState)
	
	if len(previouslyScheduledNodes) > 0 {
		portMarker.FillInitialPortState(network, portState, previouslyScheduledNodes, intdepl)
	}
	if countingEngine != nil {
		countingEngine.Precompute(extractNames(feasibleNodes))
	}
	return deplSchedulingState
}

func (t *TelemetrySchedulingEngine) ComputeNodeSchedulingScore(
	network *Network,
	state *DeploymentSchedulingState,
	nodeName string,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	pod *v1.Pod,
	podsDeploymentName string,
	scheduledNodes []ScheduledNode,
) int {
	if t.isDeploymentMemberAlreadyScheduledOnNode(nodeName, scheduledNodes, podsDeploymentName) {
		return 0
	}
	if len(state.nodesWithOppositeDeployment) > 0 {
		return state.countingEngine.GetNumberOfNewINTPortsCoverableBySchedulingOn(network.Vertices[nodeName])
	} else {
		nodesWithCurDepl := getNodesWithDeployment(scheduledNodes, podsDeploymentName)
		nodesWithCurDepl[nodeName] = struct{}{}
		// as there is no live traffic putting this pod on that node doesn't affect portState
		countingEngine := newCountingEngine(network, nodesWithCurDepl, state.portState)
		feasibleNodeNames := extractNames(*state.feasibleNodesForOppositeDeployment)
		countingEngine.Precompute(feasibleNodeNames)	
		bestScore := 0
		for _, feasibleNode := range feasibleNodeNames {
			score := countingEngine.GetNumberOfNewINTPortsCoverableBySchedulingOn(network.Vertices[feasibleNode])
			if score > bestScore {
				bestScore = score
			}
		}
		return bestScore
	}
}

// func (t *TelemetrySchedulingEngine) markScheduled(
// 	network *Network,
// 	schedulingState *DeploymentSchedulingState,
// 	scheduledNodes []ScheduledNode,
// 	deplName string,
// 	nodeVertex *Vertex,
// ) []ScheduledNode {
// 	visited := make([]bool, len(network.Vertices))
// 	for i := 0; i < len(visited); i++ {
// 		visited[i] = false
// 	}
// 	targets := t.getNodesWithPodsOfOtherDeployment(deplName, scheduledNodes)
// 	var dfs func(*Vertex, *Vertex, bool) (bool, bool)
// 	dfs = func(cur *Vertex, prev *Vertex, foundTelemetrySwitchBefore bool) (leadsToTarget bool, foundTelemetrySwitchAfter bool) {
// 		visited[cur.Ordinal] = true
// 		foundTelemetrySwitchAfter = false
// 		if cur.DeviceType == shimv1alpha.NODE {
// 			_, leadsToTarget = targets[cur.Name]	
// 			return
// 		}
// 		_, hasTelemetryEnabled := network.TelemetryEnabledSwitches[cur.Name]
// 		for _, neigh := range cur.Neighbors() {
// 			if !visited[neigh.Ordinal] {
// 				neighHadTelemetryOnRoute := foundTelemetrySwitchBefore || hasTelemetryEnabled
// 				neighLeadsToTarget, foundTelemetryAfter := dfs(neigh, cur, neighHadTelemetryOnRoute)
// 				if neighLeadsToTarget {
// 					leadsToTarget = true
// 					areOtherSwitchesOnRoute := foundTelemetrySwitchBefore || foundTelemetryAfter
// 					curMeta := schedulingState.PortMetaOf(cur)
// 					if hasTelemetryEnabled && curMeta.IsPortUnallocated(neigh.Name) && areOtherSwitchesOnRoute {
// 						curMeta.AllocatePort(neigh.Name)
// 					}
// 					if hasTelemetryEnabled && curMeta.IsPortUnallocated(prev.Name) && areOtherSwitchesOnRoute {
// 						curMeta.AllocatePort(prev.Name)
// 					}
// 				}
// 				foundTelemetrySwitchAfter = foundTelemetrySwitchAfter || foundTelemetryAfter
// 			}
// 		}
// 		if hasTelemetryEnabled {
// 			foundTelemetrySwitchAfter = true
// 		}
// 		return
// 	}
// 	found := false
// 	for _, node := range scheduledNodes {
// 		if node.Name == nodeVertex.Name {
// 			node.MarkScheduled(deplName)
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		scheduledNodes = append(scheduledNodes, ScheduledNode{
// 			Name: nodeVertex.Name,
// 			ScheduledDeployments: []string{deplName},
// 		})
// 	}
// 	if nodeVertex.Parent == nil {
// 		return scheduledNodes
// 	}
// 	visited[nodeVertex.Ordinal] = true
// 	dfs(nodeVertex.Parent, nodeVertex, false)
// 	return scheduledNodes
// }

func (t *TelemetrySchedulingEngine) isDeploymentMemberAlreadyScheduledOnNode(nodeName string, scheduledNodes []ScheduledNode, podsDeploymentName string) bool {
	var node ScheduledNode
	for _, node = range scheduledNodes {
		if nodeName == node.Name {
			return slices.Contains(node.ScheduledDeployments, podsDeploymentName)
		}
	}
	return false
}
