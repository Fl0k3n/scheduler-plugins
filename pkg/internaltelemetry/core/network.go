package core

import (
	"maps"

	"github.com/gammazero/deque"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

type Vertex struct {
	Name string
	// index of vertex in network, guaranteed to be unique and in range [0, #vertices - 1]
	Ordinal int 
	DeviceType shimv1alpha.DeviceType
	Parent *Vertex
	Children []*Vertex
}

func (v *Vertex) IsLeaf() bool {
	return v.Children == nil
}

func (v *Vertex) Neighbors() []*Vertex {
	neighbors := []*Vertex{}
	if v.Parent != nil {
		neighbors = append(neighbors, v.Parent)
	}
	if !v.IsLeaf() {
		neighbors = append(neighbors, v.Children...)
	}
	return neighbors
}

// we assume that network is logically a tree
type Network struct {
	// root is an arbitrary vertex from the undirected acyclic graph
	// the only constraint is that every K8s node is a leaf
	Root *Vertex
	// key is the name of IncSwitch
	TelemetryEnabledSwitches map[string]*shimv1alpha.IncSwitch
	// key is the name of device from topology
	Vertices map[string]*Vertex
}

func newNetwork(
	root *Vertex,
	telemetryEnabledSwitches map[string]*shimv1alpha.IncSwitch,
) *Network {
	return &Network{
		Root: root,
		TelemetryEnabledSwitches: telemetryEnabledSwitches,
	}
}

func (n *Network) IterVertices(consumer func(v *Vertex)) {
	if n.Root == nil {
		return
	}
	queue := deque.New[*Vertex]()
	queue.PushBack(n.Root)
	for queue.Len() > 0 {
		cur := queue.PopFront()
		consumer(cur)
		if !cur.IsLeaf() {
			for _, child := range cur.Children {
				queue.PushBack(child)
			}
		}
	}
}

func (n *Network) IterVerticesOfType(deviceType shimv1alpha.DeviceType, consumer func (v *Vertex)) {
	n.IterVertices(func(v *Vertex) {
		if v.DeviceType == deviceType {
			consumer(v)
		}
	})
}

type TelemetryPortsMeta struct {
	AvailableTelemetryPorts map[string]struct{}
}

func (t *TelemetryPortsMeta) IsPortUnallocated(peerName string) bool {
	_, unallocated := t.AvailableTelemetryPorts[peerName]
	return unallocated
}

func (t *TelemetryPortsMeta) AllocatePort(peerName string) {
	delete(t.AvailableTelemetryPorts, peerName)
}

func (t *TelemetryPortsMeta) DeepCopy() *TelemetryPortsMeta {
	return &TelemetryPortsMeta{
		AvailableTelemetryPorts: maps.Clone(t.AvailableTelemetryPorts),
	}
}
