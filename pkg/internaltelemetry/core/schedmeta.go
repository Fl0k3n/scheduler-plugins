package core

import (
	"sync"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/core/v1"
)

type ScheduledNode struct {
	Name string
	ScheduledDeployments []string
}

func (s *ScheduledNode) DeepCopy() ScheduledNode {
	return ScheduledNode{
		Name: s.Name,
		ScheduledDeployments: slices.Clone(s.ScheduledDeployments),
	}
}

func (s *ScheduledNode) MarkScheduled(deplName string) {
	for _, sd := range s.ScheduledDeployments {
		if sd == deplName {
			return
		}
	}
	s.ScheduledDeployments = append(s.ScheduledDeployments, deplName)
}

// order of scheduled deployments is ignored
func (s *ScheduledNode) Equals(other *ScheduledNode) bool {
	if s.Name != other.Name || len(s.ScheduledDeployments) != len(other.ScheduledDeployments) {
		return false
	}
	for _, sd := range s.ScheduledDeployments {
		if slices.Index(other.ScheduledDeployments, sd) == -1 {
			return false
		}
	}
	return true
}

func (s *ScheduledNode) HasScheduled(deplName string) bool {
	return slices.Contains(s.ScheduledDeployments, deplName)
}

// must exclude pod that is currently being scheduled
type QueuedPods struct {
	PerDeploymentCounts map[string]int // deploymentName -> number of remaining pods
}

func newQueuedPods() QueuedPods {
	return QueuedPods{
		PerDeploymentCounts: map[string]int{},
	}
}

func (q *QueuedPods) AllQueued() bool {
	for _, count := range q.PerDeploymentCounts {
		if count > 0 {
			return false
		}
	}
	return true
}

// Tracks per-deployment counts of numbers of scheduled (or reserved) pods
type ScheduluedCounters struct {
	counter map[string]int
}

func newScheduledCounters() *ScheduluedCounters {
	return &ScheduluedCounters{
		counter: map[string]int{},
	}
}

func (s *ScheduluedCounters) Increment(deplName string) {
	if count, ok := s.counter[deplName]; ok {
		s.counter[deplName] = count + 1
	} else {
		s.counter[deplName] = 1
	}
}

// copy is returned, neither this nor other is modified
func (s *ScheduluedCounters) CombinedWith(other *ScheduluedCounters) *ScheduluedCounters {
	res := newScheduledCounters()
	res.counter = maps.Clone(s.counter)
	for k, v := range other.counter {
		if count, ok := res.counter[k]; ok {
			res.counter[k] = count + v
		} else {
			res.counter[k] = v
		}
	}
	return res
}

func (s *ScheduluedCounters) Get(deplName string) int {
	if count, ok := s.counter[deplName]; ok {
		return count 
	}
	return 0
}

type ReservationMeta struct {
	NodeName string
	PodsDeploymentName string
}

type ReservationState struct {
	StateLock sync.Mutex
	Reservations map[string]ReservationMeta // key is podName 
}

func newReservationState () *ReservationState {
	return &ReservationState{
		StateLock: sync.Mutex{},
		Reservations: map[string]ReservationMeta{},
	}
}

type TelemetryPortState struct {
	portMeta []*TelemetryPortsMeta
}

func newTelemetryPortState(network *Network) *TelemetryPortState {
	portMeta := make([]*TelemetryPortsMeta, len(network.Vertices))
	for i := range portMeta {
		portMeta[i] = nil
	}
	return &TelemetryPortState{
		portMeta: portMeta,
	}
}

func (t *TelemetryPortState) SetPortMetaOf(v *Vertex, meta *TelemetryPortsMeta) {
	t.portMeta[v.Ordinal] = meta
}

func (t *TelemetryPortState) PortMetaOf(v *Vertex) *TelemetryPortsMeta {
	return t.portMeta[v.Ordinal]
}

func (t *TelemetryPortState) DeepCopy() *TelemetryPortState {
	portMeta := make([]*TelemetryPortsMeta, len(t.portMeta))
	for i, p := range t.portMeta {
		if p != nil {
			portMeta[i] = p.DeepCopy()
		} else {
			portMeta[i] = nil
		}
	}
	return &TelemetryPortState{
		portMeta: portMeta,
	}
}

type DeploymentSchedulingState struct {
	portState *TelemetryPortState
	countingEngine *CountingEngine
	nodesWithOppositeDeployment map[string]struct{}
	feasibleNodesForOppositeDeployment *[]*v1.Node
}

func newDeploymentSchedulingState(
	portState *TelemetryPortState,
	countingEngine *CountingEngine,
	nodesWithOppositeDeployment map[string]struct{},
	feasibleNodesForOppositeDeployment *[]*v1.Node,
) *DeploymentSchedulingState {
	return &DeploymentSchedulingState{
		portState: portState,	
		countingEngine: countingEngine,
		nodesWithOppositeDeployment: nodesWithOppositeDeployment,
		feasibleNodesForOppositeDeployment: feasibleNodesForOppositeDeployment,
	}
}

func (d *DeploymentSchedulingState) PortMetaOf(v *Vertex) *TelemetryPortsMeta {
	return d.portState.PortMetaOf(v)
}
