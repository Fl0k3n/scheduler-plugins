package core

import (
	"fmt"
)

type reachabilityInfo struct {
	reaches bool
	throughAtLeastOneINTswitch bool
}

type MemoKey = string

// implements H function (and other related) as defined in the thesis
type CountingEngine struct {
	network *Network
	nodesWithOppositeDeployment map[string]struct{}
	telemetryEnabledSwitches map[string]struct{}
	schedulingState *DeploymentSchedulingState

	HMemo map[MemoKey]int
	FMemo map[MemoKey]int
	RMemo map[MemoKey]reachabilityInfo
}

func newCountingEngine(
	network *Network, 
	nodesWithOppositeDeployment map[string]struct{},
	telemetryEnabledSwitches map[string]struct{},
	schedulingState *DeploymentSchedulingState,
) *CountingEngine {
	return &CountingEngine{
		network: network,
		nodesWithOppositeDeployment: nodesWithOppositeDeployment,
		telemetryEnabledSwitches: telemetryEnabledSwitches,
		schedulingState: schedulingState,
		HMemo: map[MemoKey]int{},
		FMemo: map[MemoKey]int{},
		RMemo: map[MemoKey]reachabilityInfo{},
	}
}

func (e *CountingEngine) memoKey(u *Vertex, v *Vertex) MemoKey {
	return fmt.Sprintf("%s->%s", u.Name, v.Name)
}

func (e *CountingEngine) runsOppositeDeploymentPod(v *Vertex) bool {
	_, ok := e.nodesWithOppositeDeployment[v.Name]
	return ok
}

func (e *CountingEngine) isTelemetrySwitch(v *Vertex) bool {
	_, ok := e.telemetryEnabledSwitches[v.Name]	
	return ok
}

// undefined for leaves
func (e *CountingEngine) children(u *Vertex, v *Vertex) []*Vertex {
	if u == v.Parent {
		return v.Children
	}
	res := make([]*Vertex, len(v.Children))
	res[0] = v.Parent
	j := 1
	for _, child := range v.Children {
		if child != u {
			res[j] = child
			j++
		}
	}
	return res
}

func (e *CountingEngine) getReachability(u *Vertex, v *Vertex) reachabilityInfo {
	key := e.memoKey(u, v)
	if res, memoed := e.RMemo[key]; memoed {
		return res
	}
	res := reachabilityInfo{reaches: false, throughAtLeastOneINTswitch: false}
	if v.IsLeaf() {
		_, runsOppositeDeployment := e.nodesWithOppositeDeployment[v.Name]
		res.reaches = runsOppositeDeployment
	} else {
		for _, k := range e.children(u, v) {
			r := e.getReachability(v, k)
			if r.reaches {
				res.reaches = true
				res.throughAtLeastOneINTswitch = r.throughAtLeastOneINTswitch
				if _, isTelemetrySwitch := e.telemetryEnabledSwitches[v.Name]; isTelemetrySwitch {
					res.throughAtLeastOneINTswitch = true
				}
				break
			}
		}
	}
	e.RMemo[key] = res
	return res
}

func (e *CountingEngine) R(u *Vertex, v *Vertex) int {
	if !e.getReachability(u, v).reaches {
		return 0
	}
	return 1
}

// is v a telemetry switch and it's port to u is available
func (e *CountingEngine) AllocGain(u *Vertex, v *Vertex) int {
	if _, isTelemetrySwitch := e.telemetryEnabledSwitches[v.Name]; !isTelemetrySwitch {
		return 0
	}
	if _, available := e.schedulingState.PortMetaOf(v).AvailableTelemetryPorts[u.Name]; !available {
		return 0
	}
	return 1
}

func (e *CountingEngine) F(u *Vertex, v *Vertex) int {
	key := e.memoKey(u, v)
	if res, memoed := e.FMemo[key]; memoed {
		return res
	}
	res := 0
	if v.IsLeaf() {
		res = 0
	} else {
		res = e.AllocGain(u, v) * e.R(u, v)
		for _, k := range e.children(u, v) {
			res += e.F(v, k) + (e.AllocGain(k, v) * e.R(v, k))
		}
	}
	e.FMemo[key] = res
	return res
}

func (e *CountingEngine) H(u *Vertex, v *Vertex) int {
	key := e.memoKey(u, v)
	if res, memoed := e.HMemo[key]; memoed {
		return res
	}
	res := 0
	if v.IsLeaf() {
		res = 0
	} else if e.isTelemetrySwitch(v) {
		someChildReachesOppositePodThroughINTSwitch := false
		for _, k := range e.children(u, v) {
			res += e.F(v, k)
			r := e.getReachability(v, k)
			if r.reaches && r.throughAtLeastOneINTswitch {
				someChildReachesOppositePodThroughINTSwitch = true
				res += e.AllocGain(k, v)
			}
		}
		if someChildReachesOppositePodThroughINTSwitch {
			res += e.AllocGain(u, v)
		}
	} else {
		for _, k := range e.children(u, v) {
			res += e.H(v, k)
		}
	}
	e.HMemo[key] = res
	return res
}

// calling it from required sources guarantees that memos will be filled and 
// later queries will return in O(1) without modifying anything
func (e *CountingEngine) ComputeHfunc(requiredSources []string) {
	for _, src := range requiredSources {
		nodeVertex := e.network.Vertices[src]
		e.H(nodeVertex, nodeVertex.Parent)
	}
}
