package core

import shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"

type Vertex struct {
	Name string
	DeviceType shimv1alpha.DeviceType
	Children []*Vertex
}

// we assume that network is logically a tree
type Network struct {
	// root is an arbitrary vertex from the undirected acyclic graph
	// the only constraint is that every K8s node is a leaf
	Root *Vertex
}

