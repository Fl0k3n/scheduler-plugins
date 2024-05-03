package core

import (
	"context"
	"errors"
	"fmt"

	deques "github.com/gammazero/deque"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

const TELEMETRY_INTERFACE = "inc.kntp.com/v1alpha1/telemetry"

type TopologyEngine struct {
	client client.Client
	topoReady bool
	topo *shimv1alpha.Topology 
	network *Network
}

func NewTopologyEngine(client client.Client) *TopologyEngine {
	return &TopologyEngine{
		client: client,
		topoReady: false,
		topo: nil,
		network: nil,
	}
}

func (t *TopologyEngine) loadTopology(ctx context.Context) error {
	topologies := &shimv1alpha.TopologyList{}
	if err := t.client.List(ctx, topologies); err != nil {
		return err	
	}
	if len(topologies.Items) == 0 {
		return errors.New("topology is not initiated")
	}
	t.topo = &topologies.Items[0]
	return nil
}

func (t *TopologyEngine) hasTelemetryEnabled(_ *shimv1alpha.IncSwitch, program *shimv1alpha.P4Program) bool {
	// canBeInstalled := false
	// for _, artifact := range program.Spec.Artifacts {
	// 	if artifact.Arch == sw.Spec.Arch {
	// 		canBeInstalled = true
	// 		break
	// 	}
	// }
	// if !canBeInstalled {
	// 	return false
	// }
	for _, implementedInterface := range program.Spec.ImplementedInterfaces {
		if implementedInterface == TELEMETRY_INTERFACE {
			return true
		}
	}
	return false
}

// we require every device of type incswitch declared in Topology resource to have 
// its incSwitch resource for this to succeed, also all declared programs must be created
func (t *TopologyEngine) loadIncSwitches(ctx context.Context) (telemetryEnabledSwitches map[string]*shimv1alpha.IncSwitch, erro error){
	incswitches := &shimv1alpha.IncSwitchList{}
	if err := t.client.List(ctx, incswitches); err != nil {
		return nil, err
	}
	incSwitchesMap := make(map[string]*shimv1alpha.IncSwitch, len(incswitches.Items))
	for i, incsw := range incswitches.Items {
		incSwitchesMap[incsw.Name] = &incswitches.Items[i]
	}
	for _, dev := range t.topo.Spec.Graph {
		if dev.DeviceType != shimv1alpha.INC_SWITCH {
			continue
		}
		if _, ok := incSwitchesMap[dev.Name]; !ok {
			return nil, fmt.Errorf("incswitch %s was defined in topology graph but doesn't have a resource", dev.Name)
		}
	}
	telemetryEnabledSwitches = make(map[string]*shimv1alpha.IncSwitch)
	programs := map[string]*shimv1alpha.P4Program{}
	// TODO parallelize
	for _, incswitch := range incSwitchesMap {
		programName := incswitch.Spec.ProgramName
		if programName == "" {
			continue
		}
		var program *shimv1alpha.P4Program
		var loaded bool
		program, loaded = programs[programName]
		if !loaded {
			program = &shimv1alpha.P4Program{}
			key := types.NamespacedName{Name: programName, Namespace: t.topo.Namespace}
			if err := t.client.Get(ctx, key, program); err != nil {
				return nil, err
			}
			programs[programName] = program
		}
		if t.hasTelemetryEnabled(incswitch, program) {
			telemetryEnabledSwitches[incswitch.Name] = incswitch
		}
	}
	return telemetryEnabledSwitches, nil
}

// build tree representation of topology graph
// returns error if topology is not a fully-connected tree
func (t *TopologyEngine) buildNetworkRepr(incswitches map[string]*shimv1alpha.IncSwitch) (*Network, error) {
	// pick any non-k8s device for root if one is available
	rootName := t.topo.Spec.Graph[0].Name
	for _, dev := range t.topo.Spec.Graph {
		if dev.DeviceType != shimv1alpha.NODE {
			rootName = dev.Name
			break
		}
	}
	type devNameWithParent struct {
		parent *Vertex
		childName string
		childIdx int
	}
	queue := deques.New[devNameWithParent]()
	visitedSet := make(map[string]struct{}, len(t.topo.Spec.Graph))
	queue.PushBack(devNameWithParent{parent: nil, childName: rootName, childIdx: -1})
	visitedSet[rootName] = struct{}{}
	topoMap := toMapByDeviceName(t.topo)
	ordinalsMap := toOrdinalsByDeviceName(t.topo)
	var root *Vertex = nil

	for queue.Len() > 0 {
		cur := queue.PopFront()
		curDev := topoMap[cur.childName]
		vertex := &Vertex{
			Name: curDev.Name,
			Ordinal: ordinalsMap[curDev.Name],
			DeviceType: curDev.DeviceType,
			Children: nil,
			Parent: cur.parent,
		}
		numChildren := len(curDev.Links)
		if cur.parent == nil {
			root = vertex
		} else {
			cur.parent.Children[cur.childIdx] = vertex
			numChildren--
		}
		if numChildren > 0 {
			childCounter := 0
			vertex.Children = make([]*Vertex, numChildren)
			for _, link := range curDev.Links {
				if cur.parent == nil || link.PeerName != cur.parent.Name {
					if _, ok := visitedSet[link.PeerName]; ok {
						return nil, fmt.Errorf("topology is not a tree, %s", link.PeerName)
					}
					visitedSet[link.PeerName] = struct{}{}
					queue.PushBack(devNameWithParent{
						parent: vertex,
						childName: link.PeerName,
						childIdx: childCounter,
					})
					childCounter++
				}
			}
		}
	}
	if len(visitedSet) != len(t.topo.Spec.Graph) {
		return nil, fmt.Errorf("topology is not fully connected")
	}
	res := newNetwork(root, incswitches)
	res.Vertices = map[string]*Vertex{}
	res.IterVertices(func(v *Vertex) {res.Vertices[v.Name] = v})
	return res, nil
}

func (t *TopologyEngine) initTopologyState(ctx context.Context) error {
	if err := t.loadTopology(ctx); err != nil {
		return err
	}
	incswitches, err := t.loadIncSwitches(ctx)
	if err != nil {
		return err
	}
	net, err := t.buildNetworkRepr(incswitches)
	if err != nil {
		return err
	}
	t.network = net
	t.topoReady = true
	return nil
}

// starts a new goroutine to watch for topology changes, ctx should be canceled to stop watching
func (t *TopologyEngine) WatchTopologyChanges(ctx context.Context) error {
	// TODO
	return nil
}

func (t *TopologyEngine) PrepareForPodScheduling(ctx context.Context, pod *v1.Pod) (*Network, error) {
	if !t.topoReady {
		if err := t.initTopologyState(ctx); err != nil {
			return nil, err
		}
	}
	return t.network, nil
}

