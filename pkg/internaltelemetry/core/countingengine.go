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
	portState *TelemetryPortState

	HMemo map[MemoKey]int
	FMemo map[MemoKey]int
	RMemo map[MemoKey]reachabilityInfo
}

// arguments are treated as immutable
func newCountingEngine(
	network *Network, 
	nodesWithOppositeDeployment map[string]struct{},
	portState *TelemetryPortState,
) *CountingEngine {
	return &CountingEngine{
		network: network,
		nodesWithOppositeDeployment: nodesWithOppositeDeployment,
		portState: portState,
		HMemo: map[MemoKey]int{},
		FMemo: map[MemoKey]int{},
		RMemo: map[MemoKey]reachabilityInfo{},
	}
}

func (e *CountingEngine) memoKey(u *Vertex, v *Vertex) MemoKey {
	return fmt.Sprintf("%s->%s", u.Name, v.Name)
}

func (e *CountingEngine) isTelemetrySwitch(v *Vertex) bool {
	_, ok := e.network.TelemetryEnabledSwitches[v.Name]	
	return ok
}

// undefined for leaves
func (e *CountingEngine) children(u *Vertex, v *Vertex) []*Vertex {
	if u == v.Parent {
		return v.Children
	}
	numChildren := len(v.Children)
	if v.Parent == nil {
		numChildren--
	}
	res := make([]*Vertex, numChildren)
	j := 0
	if v.Parent != nil {
		res[0] = v.Parent
		j = 1
	}
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
				if _, isTelemetrySwitch := e.network.TelemetryEnabledSwitches[v.Name]; isTelemetrySwitch {
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
	if e.getReachability(u, v).reaches {
		return 1
	}
	return 0
}

func (e *CountingEngine) Q(u *Vertex, v *Vertex) int {
	if _, isTelemetrySwitch := e.network.TelemetryEnabledSwitches[v.Name]; !isTelemetrySwitch {
		return 0
	}
	if _, available := e.portState.PortMetaOf(v).AvailableTelemetryPorts[u.Name]; !available {
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
		res = e.Q(u, v) * e.R(u, v)
		for _, k := range e.children(u, v) {
			res += e.F(v, k) + (e.Q(k, v) * e.R(v, k))
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
				res += e.Q(k, v)
			}
		}
		if someChildReachesOppositePodThroughINTSwitch {
			res += e.Q(u, v)
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
// later queries will return in O(1) without modifying anything (and are thus thread-safe)
func (e *CountingEngine) Precompute(requiredSources []string) {
	for _, src := range requiredSources {
		nodeVertex := e.network.Vertices[src]
		e.H(nodeVertex, nodeVertex.Parent)
	}
}

func (e *CountingEngine) GetNumberOfNewINTPortsCoverableBySchedulingOn(node *Vertex) int {
	if res, memoed := e.HMemo[e.memoKey(node, node.Parent)]; memoed {
		return res
	} else {
		panic(fmt.Sprintf("H not memoized for node %s, recomputing is unsafe in multi-threaded env", node.Name))
	}
}
