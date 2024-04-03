package core

import (
	"github.com/gammazero/deque"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

type Vertex [T any] struct {
	Name string
	// index of vertex in network, guaranteed to be unique and in range [0, #vertices - 1]
	Ordinal int 
	DeviceType shimv1alpha.DeviceType
	Parent *Vertex[T]
	Children []*Vertex[T]
	Meta T
}

func (v *Vertex[T]) IsLeaf() bool {
	return v.Children == nil
}

func (v *Vertex[T]) Neighbors() []*Vertex[T] {
	neighbors := []*Vertex[T]{}
	if v.Parent != nil {
		neighbors = append(neighbors, v.Parent)
	}
	if !v.IsLeaf() {
		neighbors = append(neighbors, v.Children...)
	}
	return neighbors
}

// we assume that network is logically a tree
type Network [T any] struct {
	// root is an arbitrary vertex from the undirected acyclic graph
	// the only constraint is that every K8s node is a leaf
	Root *Vertex[T]
	// key is the name of IncSwitch
	TelemetryEnabledSwitches map[string]*shimv1alpha.IncSwitch
	// key is the name of device from topology
	Vertices map[string]*Vertex[T]
}

func newNetwork[T any](
	root *Vertex[T],
	telemetryEnabledSwitches map[string]*shimv1alpha.IncSwitch,
) *Network[T] {
	return &Network[T]{
		Root: root,
		TelemetryEnabledSwitches: telemetryEnabledSwitches,
	}
}

func (n *Network[T]) IterVertices(consumer func(v *Vertex[T])) {
	if n.Root == nil {
		return
	}
	queue := deque.New[*Vertex[T]]()
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

func (n *Network[T]) IterVerticesOfType(deviceType shimv1alpha.DeviceType, consumer func (v *Vertex[T])) {
	n.IterVertices(func(v *Vertex[T]) {
		if v.DeviceType == deviceType {
			consumer(v)
		}
	})
}

// only tree is deepcopied
func mapDeepCopy[From any, To any](network *Network[From], metaProvider func(v *Vertex[From]) To) *Network[To] {
	if network.Root == nil {
		return newNetwork[To](nil, nil)
	}
	type childWithParent struct {
		copiedParent *Vertex[To] 
		originalChild *Vertex[From]
		childIdx int
	}
	queue := deque.New[childWithParent]()
	queue.PushBack(childWithParent{copiedParent: nil, originalChild: network.Root, childIdx: -1})
	res := newNetwork[To](nil, network.TelemetryEnabledSwitches)
	for queue.Len() > 0 {
		cur := queue.PopFront()
		copied := &Vertex[To]{
			Name: cur.originalChild.Name,
			Ordinal: cur.originalChild.Ordinal,
			DeviceType: cur.originalChild.DeviceType,
			Parent: cur.copiedParent,
			Children: nil,
			Meta: metaProvider(cur.originalChild),
		}
		if cur.copiedParent != nil {
			cur.copiedParent.Children[cur.childIdx] = copied
		} else {
			res.Root = copied
		}
		if !cur.originalChild.IsLeaf() {
			copied.Children = make([]*Vertex[To], len(cur.originalChild.Children))
			for i, child := range cur.originalChild.Children {
				queue.PushBack(childWithParent{
					copiedParent: copied,
					originalChild: child,
					childIdx: i,
				})
			}
		}
	}
	res.Vertices = make(map[string]*Vertex[To], len(network.Vertices))
	res.IterVertices(func(v *Vertex[To]) {res.Vertices[v.Name] = v})
	return res
}

type NetworkMetadataSnapshot [T any] struct {
	snapshot []T
}

func (n *Network[T]) TakeMetadataSnapshot(copier func(T) T) NetworkMetadataSnapshot[T] {
	snapshot := make([]T, len(n.Vertices))
	n.IterVertices(func(v *Vertex[T]) {
		snapshot[v.Ordinal] = copier(v.Meta)
	})
	return NetworkMetadataSnapshot[T]{snapshot}
}

func (n *Network[T]) RestoreMetadataFromSnapshot(snapshot NetworkMetadataSnapshot[T]) {
	n.IterVertices(func(v *Vertex[T]) {
		v.Meta = snapshot.snapshot[v.Ordinal]
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
