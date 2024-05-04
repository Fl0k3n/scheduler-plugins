package core

import (
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

type TelemetryPortMarker struct {
	traversedWithPreviousINTMemo map[MemoKey]reachabilityInfo
	traversedWithoutPreviousINTMemo map[MemoKey]reachabilityInfo
}

func newTelemetryPortMarker() *TelemetryPortMarker {
	return &TelemetryPortMarker{
		traversedWithPreviousINTMemo: map[MemoKey]reachabilityInfo{},
		traversedWithoutPreviousINTMemo: map[MemoKey]reachabilityInfo{},
	}
}

func (t *TelemetryPortMarker) memoKey(u *Vertex, v *Vertex) MemoKey {
	return MemoKey{src: u.Ordinal, dst: v.Ordinal}
}

func (t *TelemetryPortMarker) GetInitialPortState(network *Network) *TelemetryPortState {
	portState := newTelemetryPortState(network)
	network.IterVertices(func(v *Vertex) {
		if _, ok := network.TelemetryEnabledSwitches[v.Name]; ok {
			neighbors := v.Neighbors()
			res := make(map[string]struct{}, len(neighbors))
			for _, neighbor := range neighbors {
				res[neighbor.Name] = struct{}{}
			}
			portState.SetPortMetaOf(v, &TelemetryPortsMeta{AvailableTelemetryPorts: res})
		} else {
			portState.SetPortMetaOf(v, &TelemetryPortsMeta{AvailableTelemetryPorts: map[string]struct{}{}})
		}
	})
	return portState
}

func (t *TelemetryPortMarker) FillInitialPortState(
	network *Network,
	portState *TelemetryPortState,
	previouslyScheduledNodes []ScheduledNode,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) {
	deplNames := make([]string, 2)
	nodesWithDeployment := make([]map[string]struct{}, 2)
	for i, depl := range intdepl.Spec.DeploymentTemplates {
		deplNames[i] = depl.Name
		nodesWithDeployment[i] = getNodesWithDeployment(previouslyScheduledNodes, depl.Name)
	}
	for _, sn := range previouslyScheduledNodes {
		for _, depl := range sn.ScheduledDeployments {
			v := network.Vertices[sn.Name]
			nodesWithOppositeDeployment := nodesWithDeployment[0]
			if depl == deplNames[0] {
				nodesWithOppositeDeployment = nodesWithDeployment[1]
			}
			t.MarkScheduled(network, portState, depl, v, nodesWithOppositeDeployment)
		}
	}
}

func (t *TelemetryPortMarker) MarkScheduled(
	network *Network,
	portState *TelemetryPortState,
	deplName string,
	nodeVertex *Vertex,
	nodesWithOppositeDeployment map[string]struct{},
) {
	var dfs func(*Vertex, *Vertex, bool) (bool, bool)
	dfs = func(cur *Vertex, prev *Vertex, foundTelemetrySwitchBefore bool) (leadsToTarget bool, foundTelemetrySwitchAfter bool) {
		shouldContinueTraversal, r := t.shouldContinueTraversal(prev, cur, foundTelemetrySwitchBefore)
		if !shouldContinueTraversal {
			return r.reaches, r.throughAtLeastOneINTswitch
		}
		foundTelemetrySwitchAfter = false
		if cur.DeviceType == shimv1alpha.NODE {
			_, leadsToTarget = nodesWithOppositeDeployment[cur.Name]	
			return
		}
		_, hasTelemetryEnabled := network.TelemetryEnabledSwitches[cur.Name]
		for _, neigh := range cur.Neighbors() {
			if neigh != prev {
				neighHadTelemetryOnRoute := foundTelemetrySwitchBefore || hasTelemetryEnabled
				neighLeadsToTarget, foundTelemetryAfter := dfs(neigh, cur, neighHadTelemetryOnRoute)
				if neighLeadsToTarget {
					leadsToTarget = true
					areOtherSwitchesOnRoute := foundTelemetrySwitchBefore || foundTelemetryAfter
					curMeta := portState.PortMetaOf(cur)
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
		r = reachabilityInfo{reaches: leadsToTarget, throughAtLeastOneINTswitch: foundTelemetrySwitchAfter}
		t.memoizePathResults(prev, cur, r, foundTelemetrySwitchBefore)
		return
	}
	dfs(nodeVertex.Parent, nodeVertex, false)
}

func (t *TelemetryPortMarker) shouldContinueTraversal(
	prev *Vertex,
	cur *Vertex,
	foundTelemetrySwitchBefore bool,
) (bool, reachabilityInfo) {
	key := t.memoKey(prev, cur)
	r, hasTraversedWithINT := t.traversedWithPreviousINTMemo[key]
	if hasTraversedWithINT {
		return false, r
	}
	if foundTelemetrySwitchBefore {
		return true, reachabilityInfo{}
	} 
	r, hasTraversedWithoutINT := t.traversedWithoutPreviousINTMemo[key]
	return !hasTraversedWithoutINT, r
}

func (t *TelemetryPortMarker) memoizePathResults(
	prev *Vertex,
	cur *Vertex,
	r reachabilityInfo,
	foundTelemetrySwitchBefore bool,
) {
	key := t.memoKey(prev, cur)
	if foundTelemetrySwitchBefore {
		t.traversedWithPreviousINTMemo[key] = r
	} else {
		t.traversedWithoutPreviousINTMemo[key] = r
	}
}
