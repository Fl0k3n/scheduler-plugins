package core

import (
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)


func TestTopologyEngine(t *testing.T) {
	n := makeTestNetwork(
		[]vertex{
			{"external", shimv1alpha.EXTERNAL},
			{"r0", shimv1alpha.INC_SWITCH},
			{"r1", shimv1alpha.INC_SWITCH},
			{"r2", shimv1alpha.INC_SWITCH},
			{"n0", shimv1alpha.NODE},
			{"n1", shimv1alpha.NODE},
			{"n2", shimv1alpha.NODE},
		},
		[]edge{
			{"external", "r0"},
			{"r0", "r1"},
			{"r0", "r2"},
			{"r1", "n0"},
			{"r1", "n1"},
			{"r2", "n2"},
		},
		[]string{"r0", "r1", "r2"},
	)
	_ = n
}

func TestScoringEngine(t *testing.T) {
	// testNetwork := Network[Nothing]{}
}

type edge struct {
	u string 
	v string
}

type vertex struct {
	name string
	deviceType shimv1alpha.DeviceType
}

// func makeTestNetwork(root string, vertices []vertex, connections []connection, telemetryEnabledSwitches []string) *Network[Nothing] {
// 	nameToVertex := map[string]vertex{}
// 	for _, v := range vertices {
// 		nameToVertex[v.name] = v
// 	}
// 	type nodeWithParent struct {
// 		parent *Vertex[Nothing]
// 		child string
// 		childIdx int
// 	}
// 	telSwitches := map[string]*shimv1alpha.IncSwitch{}
// 	for _, name := range telemetryEnabledSwitches {
// 		telSwitches[name] = makeTestIncSwitch(name)
// 	}
// 	res := newNetwork[Nothing](nil, telSwitches)
	
// 	queue := deque.New[nodeWithParent]()
// 	queue.PushBack(nodeWithParent{parent: nil, child: root, childIdx: -1})
// 	for queue.Len() > 0 {
// 		cur := queue.PopFront()
// 		v := &Vertex[Nothing]{

// 		}
// 	}
// 	return res
// }


func makeTestNetwork(vertices []vertex, edges []edge, telemetryEnabledSwitches []string) *Network[Nothing] {
	telSwitches := map[string]*shimv1alpha.IncSwitch{}
	for _, name := range telemetryEnabledSwitches {
		telSwitches[name] = makeTestIncSwitch(name)
	}
	topoEngine := NewTopologyEngine(nil)
	topoEngine.topo = makeTestTopology(vertices, edges)
	net, er := topoEngine.buildNetworkRepr(telSwitches)
	if er != nil {
		panic(er)
	}
	return net
}

func makeTestTopology(vertices []vertex, edges []edge) *shimv1alpha.Topology {
	graph := []shimv1alpha.NetworkDevice{}
	vertexConnections := map[string][]string{}
	for _, v := range vertices {
		vertexConnections[v.name] = []string{}
	}
	for _, e := range edges {
		s := vertexConnections[e.u]
		vertexConnections[e.u] = append(s, e.v)
		s = vertexConnections[e.v]
		vertexConnections[e.v] = append(s, e.u)
	}
	for _, v := range vertices {
		links := []shimv1alpha.Link{}
		for _, neigh := range vertexConnections[v.name] {
			links = append(links, shimv1alpha.Link{
				PeerName: neigh,
			})
		}
		dev := shimv1alpha.NetworkDevice{
			Name: v.name,
			DeviceType: v.deviceType,
			Links: links,
		}
		graph = append(graph, dev)
	}
	return &shimv1alpha.Topology{
		ObjectMeta: v1.ObjectMeta{
			Name: "test",
		},
		Spec: shimv1alpha.TopologySpec{
			Graph: graph,
		},
	}
}

func makeTestIncSwitch(name string) *shimv1alpha.IncSwitch {
	return &shimv1alpha.IncSwitch{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},		
		Spec: shimv1alpha.IncSwitchSpec{
			Arch: "foo",
			ProgramName: "bar",
		},
	}
}
