package core

import (
	"context"
	"errors"
	"fmt"

	deques "github.com/gammazero/deque"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

type TopologyEngine struct {
	client client.Client
	topoReady bool
	topo *shimv1alpha.Topology 
}

func NewTopologyEngine(client client.Client) *TopologyEngine {
	return &TopologyEngine{
		client: client,
		topoReady: false,
		topo: nil,
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

// we require every device of type incswitch declared in Topology resource to have 
// its incSwitch resource for this to succeed
func (t *TopologyEngine) loadIncSwitches(ctx context.Context) (map[string]*shimv1alpha.IncSwitch, error){
	incswitches := &shimv1alpha.IncSwitchList{}
	if err := t.client.List(ctx, incswitches); err != nil {
		return nil, err
	}
	incSwitchesMap := make(map[string]*shimv1alpha.IncSwitch, len(incswitches.Items))
	for _, incsw := range incswitches.Items {
		incSwitchesMap[incsw.Name] = &incsw
	}
	for _, dev := range t.topo.Spec.Graph {
		if _, ok := incSwitchesMap[dev.Name]; !ok {
			return nil, fmt.Errorf("Incswitch %s was defined in topology graph but doesn't have a resource", dev.Name)
		}
	}
	return incSwitchesMap, nil
}

func (t *TopologyEngine) buildNetworkRepr(incswitches map[string]*shimv1alpha.IncSwitch) *Network {
	// pick any non-k8s device for root
	root := t.topo.Spec.Graph[0]
	for _, dev := range t.topo.Spec.Graph {
		if dev.DeviceType != shimv1alpha.NODE {
			root = dev
			break
		}
	}
	queue := deques.New()
	_ = queue
	_ = root
	return nil
}

func (t *TopologyEngine) InitTopologyState(ctx context.Context) error {
	if err := t.loadTopology(ctx); err != nil {
		return err
	}
	incswitches, err := t.loadIncSwitches(ctx)
	if err != nil {
		return err
	}
	_ = t.buildNetworkRepr(incswitches)

	t.topoReady = true
	return nil
}

// starts a new goroutine to watch for topology changes, ctx should be canceled to stop watching
func (t *TopologyEngine) WatchTopologyChanges(ctx context.Context) error {
	// TODO
	return nil
}

func (t *TopologyEngine) PrepareForScheduling(ctx context.Context, pod *v1.Pod) error {
	if !t.topoReady {
		if err := t.InitTopologyState(ctx); err != nil {
			return err
		}
	}
	return nil
}

