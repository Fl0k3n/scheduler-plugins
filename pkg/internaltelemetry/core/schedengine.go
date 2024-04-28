package core

import (
	v1 "k8s.io/api/core/v1"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
)

// thread-safe with respect to scheduling framework semantics
type TelemetrySchedulingEngine struct {
	// TODO this should be a thread-safe bounded LRU cache
	// deploymentSchedulingCache sync.Map // objectKey(intdepl) -> *DeploymentSchedulingState
}

func NewTelemetrySchedulingEngine() *TelemetrySchedulingEngine {
	return &TelemetrySchedulingEngine{
		// deploymentSchedulingCache: sync.Map{}, // let's not complicate for now
	}
}

func (t *TelemetrySchedulingEngine) Prescore(
	network *Network,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	podDeplName string,
	feasibleNodes []*v1.Node,
	feasibleNodesForOppositeDeploymentProvider func() []*v1.Node,
	previouslyScheduledNodes []ScheduledNode,
) (NodeScoreProvider, *TelemetryPortState) {
	oppositeDeployment := getOppositeDeployment(intdepl, podDeplName)
	nodesWithOppositeDeployment := getNodesWithDeployment(previouslyScheduledNodes, oppositeDeployment)
	portMarker := newTelemetryPortMarker()
	portState := portMarker.GetInitialPortState(network)
	var scoreProvider NodeScoreProvider

	if len(nodesWithOppositeDeployment) > 0 {
		portMarker.FillInitialPortState(network, portState, previouslyScheduledNodes, intdepl)
		countingEngine := newCountingEngine(network, nodesWithOppositeDeployment, portState, CHILD_AGGREGATION_SUM)
		countingEngine.Precompute(extractNames(feasibleNodes))
		scoreProvider = func(nodeName string) int {
			return countingEngine.GetNumberOfNewINTPortsCoverableBySchedulingOn(network.Vertices[nodeName])
		}
	} else {
		feasibleNodesForOppositeDeployment := feasibleNodesForOppositeDeploymentProvider()
		if len(feasibleNodesForOppositeDeployment) == 0 {
			return func(nodeName string) int {
				return 0
			}, portState
		}
		if len(previouslyScheduledNodes) == 0 {
			targetNodes := toNodeNameSet(feasibleNodesForOppositeDeployment)
			countingEngine := newCountingEngine(network, targetNodes, portState, CHILD_AGGREGATION_MAX)
			countingEngine.Precompute(extractNames(feasibleNodesForOppositeDeployment))
			scoreProvider = func (nodeName string) int {
				return countingEngine.GetNumberOfNewINTPortsCoverableBySchedulingOn(network.Vertices[nodeName])	
			}
		} else {
			nodesWithSameDeployment := getNodesWithDeployment(previouslyScheduledNodes, podDeplName)
			countingEngine := newCountingEngine(network, nodesWithSameDeployment, portState, CHILD_AGGREGATION_SUM)
			countingEngine.Precompute(extractNames(feasibleNodesForOppositeDeployment))
			u, maxScore := feasibleNodesForOppositeDeployment[0].Name, 0
			for _, node := range extractNames(feasibleNodesForOppositeDeployment) {
				score := countingEngine.GetNumberOfNewINTPortsCoverableBySchedulingOn(network.Vertices[node])
				if score > maxScore {
					u, maxScore = node, score
				}
			}
			nodesWithOppositeDeployment[u] = struct{}{}
			portMarker.MarkScheduled(network, portState, oppositeDeployment, network.Vertices[u], nodesWithSameDeployment)
			// sum and max doesn't make any difference when there is just one target
			countingEngine = newCountingEngine(network, nodesWithSameDeployment, portState, CHILD_AGGREGATION_SUM)
			countingEngine.Precompute(extractNames(feasibleNodes))
			scoreProvider = func(nodeName string) int {
				return countingEngine.GetNumberOfNewINTPortsCoverableBySchedulingOn(network.Vertices[nodeName])
			}
		}
	}
	return scoreProvider, portState
}

func (t *TelemetrySchedulingEngine) Score(nodeName string, scoreProvider NodeScoreProvider) int {
	return scoreProvider(nodeName)
}
