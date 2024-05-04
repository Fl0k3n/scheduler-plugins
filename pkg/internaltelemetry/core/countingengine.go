package core

import (
	"fmt"
)

type ChildAggregationOption string

const (
	CHILD_AGGREGATION_SUM ChildAggregationOption = "sum"
	CHILD_AGGREGATION_MAX ChildAggregationOption = "max"
)

type reachabilityInfo struct {
	reaches bool
	throughAtLeastOneINTswitch bool
}

type MemoKey struct {
	src int
	dst int
}

type childAggregator = func (previousVal int, childVal int) int

// implements H function (and other related) as defined in the thesis
type CountingEngine struct {
	network *Network
	targetNodes map[string]struct{}
	portState *TelemetryPortState

	HMemo map[MemoKey]int
	FMemo map[MemoKey]int
	RMemo map[MemoKey]reachabilityInfo
	aggregate childAggregator
}

// arguments are treated as immutable
func newCountingEngine(
	network *Network, 
	targetNodes map[string]struct{},
	portState *TelemetryPortState,
	aggOp ChildAggregationOption,
) *CountingEngine {
	aggSum := func(prev int, childVal int) int {
		return prev + childVal
	}
	aggMax := func(prev int, childVal int) int {
		if childVal > prev {
			return childVal
		}
		return prev
	}
	aggregate := aggSum
	if aggOp == CHILD_AGGREGATION_MAX {
		aggregate = aggMax
	}
	return &CountingEngine{
		network: network,
		targetNodes: targetNodes,
		portState: portState,
		HMemo: map[MemoKey]int{},
		FMemo: map[MemoKey]int{},
		RMemo: map[MemoKey]reachabilityInfo{},
		aggregate: aggregate,
	}
}

func (e *CountingEngine) memoKey(u *Vertex, v *Vertex) MemoKey {
	return MemoKey{src: u.Ordinal, dst: v.Ordinal}
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
		_, isTarget := e.targetNodes[v.Name]
		res.reaches = isTarget
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
		childVal := 0
		for _, k := range e.children(u, v) {
			childVal = e.aggregate(childVal, e.F(v, k) + (e.Q(k, v) * e.R(v, k)))
		}
		res += childVal
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
		someChildReachesTargetThroughINTSwitch := false
		childVal := 0
		for _, k := range e.children(u, v) {
			cur := e.F(v, k)
			r := e.getReachability(v, k)
			if r.reaches && r.throughAtLeastOneINTswitch {
				someChildReachesTargetThroughINTSwitch = true
				cur += e.Q(k, v)
			}
			childVal = e.aggregate(childVal, cur)
		}
		res += childVal
		if someChildReachesTargetThroughINTSwitch {
			res += e.Q(u, v)
		}
	} else {
		for _, k := range e.children(u, v) {
			res = e.aggregate(res, e.H(v, k))
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
