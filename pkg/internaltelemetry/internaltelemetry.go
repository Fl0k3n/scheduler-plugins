package internaltelemetry

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/scheduler-plugins/pkg/internaltelemetry/core"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

const (
	Name = "InternalTelemetry"
	INTERNAL_TELEMETRY_POD_DEPLOYMENT_OWNER_LABEL = "inc.kntp.com/owned-by-iintdepl"
	INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL = "inc.kntp.com/part-of-deployment"
	TELEMETRY_CYCLE_STATE_KEY = Name
)

type TelemetryCycleState struct {
	Network *core.Network
	ScheduledNodes []core.ScheduledNode
	QueuedPods core.QueuedPods
	Intdepl *intv1alpha.InternalInNetworkTelemetryDeployment
	PodsDeploymentName string
}

// cycle state is treated as immutable and hence full shallow copy suffices
func (t *TelemetryCycleState) Clone() framework.StateData {
	return &TelemetryCycleState{
		Network: t.Network,
		ScheduledNodes: t.ScheduledNodes,
		QueuedPods: t.QueuedPods,
		Intdepl: t.Intdepl,
		PodsDeploymentName: t.PodsDeploymentName,
	}
}

type InternalTelemetry struct {
	client.Client
	handle     framework.Handle
	topoEngine *core.TopologyEngine
	schedEngine *core.TelemetrySchedulingEngine
	tracker *core.TelemetrySchedulingTracker
	deplMgr *core.DeploymentManager
}

var _ framework.PreScorePlugin = &InternalTelemetry{}
var _ framework.ScorePlugin = &InternalTelemetry{}
var _ framework.ReservePlugin = &InternalTelemetry{}
var _ framework.PostBindPlugin = &InternalTelemetry{}

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	klog.Info("initializing internal telemetry plugin: v0.0.1")

	scheme := runtime.NewScheme()
	_ = clientscheme.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = shimv1alpha.AddToScheme(scheme)
	_ = intv1alpha.AddToScheme(scheme)
	client, err := client.New(handle.KubeConfig(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	topoEngine := core.NewTopologyEngine(client)
	deplMgr := core.NewDeploymentManager(client)
	tracker := core.NewTelemetrySchedulingTracker(client)
	schedEngine := core.NewTelemetrySchedulingEngine(core.DefaultTelemetrySchedulingEngineConfig())

	return &InternalTelemetry{
		Client: client,
		handle: handle,	
		topoEngine: topoEngine,
		schedEngine: schedEngine,
		tracker: tracker,
		deplMgr: deplMgr,
	}, nil
}

func (t *InternalTelemetry) Name() string {
	return Name
}

func (t *InternalTelemetry) mustGetTelemetryCycleState(cycleState *framework.CycleState) *TelemetryCycleState {
	ts, err := cycleState.Read(TELEMETRY_CYCLE_STATE_KEY)
	if err != nil {
		er := errors.New("assertion failed, telemetry cycle state is not set")
		klog.Error(er)
		panic(er)
	}
	return ts.(*TelemetryCycleState)
}

func (t *InternalTelemetry) PreScore(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodes []*v1.Node,
) *framework.Status {
	var err error
	var network *core.Network
	var intdepl *intv1alpha.InternalInNetworkTelemetryDeployment
	var podsDeplName string
	var scheduledNodes []core.ScheduledNode
	var queuedPods core.QueuedPods

	if network, err = t.topoEngine.PrepareForPodScheduling(ctx, pod); err != nil {
		goto fail
	}
	if intdepl, podsDeplName, err = t.deplMgr.PrepareForPodScheduling(ctx, pod); err != nil {
		goto fail
	}
	t.tracker.PrepareForPodScheduling(intdepl)
	if scheduledNodes, queuedPods, err = t.tracker.GetSchedulingState(ctx, intdepl, podsDeplName); err != nil {
		goto fail
	}
	t.schedEngine.PrepareForPodScheduling(network, intdepl, scheduledNodes)
	state.Write(TELEMETRY_CYCLE_STATE_KEY, &TelemetryCycleState{
		Network: network,
		ScheduledNodes: scheduledNodes,
		QueuedPods: queuedPods,
		Intdepl: intdepl,
		PodsDeploymentName: podsDeplName,
	})
	return nil
fail:
	klog.Errorf("Failed to prepare for scheduling %e", err)
	return framework.AsStatus(err)
}

func (t *InternalTelemetry) Score(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	nodeName string,
) (int64, *framework.Status) {
	klog.Infof("Scoring node %s for pod %s", nodeName, p.Name)
	telemetryState := t.mustGetTelemetryCycleState(state)
	score := t.schedEngine.ComputeNodeSchedulingScore(
		telemetryState.Network,
		nodeName,
		telemetryState.Intdepl,
		p,
		telemetryState.PodsDeploymentName,
		telemetryState.ScheduledNodes,
		telemetryState.QueuedPods,
	)
	klog.Infof("Node %s score for pod %s = %d", nodeName, p.Name, score)
	return int64(score), nil
}

func (t *InternalTelemetry) NormalizeScore(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	scores framework.NodeScoreList,
) *framework.Status {
	maxScore := scores[0].Score
	minScore := scores[0].Score
	for _, nodeScore := range scores {
		if nodeScore.Score > maxScore {
			maxScore = nodeScore.Score
		}
		if nodeScore.Score < minScore {
			minScore = nodeScore.Score
		}
	}
	klog.Infof("min score: %d, max score: %d", minScore, maxScore)
	if minScore == maxScore {
		for node := range scores {
			scores[node].Score = framework.MinNodeScore
		}
	} else {
		for node, score := range scores {
			newScore := float64(score.Score - minScore) / float64(maxScore - minScore) // [0, 1]
			newScore = newScore * float64(framework.MaxNodeScore - framework.MinNodeScore) // [0, M - m]
			newScore += float64(framework.MinNodeScore) // [m, M]
			scores[node].Score = int64(newScore)
		}
	}
	return nil
}

func (t *InternalTelemetry) ScoreExtensions() framework.ScoreExtensions {
	return t
}

func (t *InternalTelemetry) Reserve(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	nodeName string,
) *framework.Status {
	ts := t.mustGetTelemetryCycleState(state)
	t.tracker.ReserveForScheduling(ts.Intdepl, nodeName, ts.PodsDeploymentName, p.Name)
	return nil
}

func (t *InternalTelemetry) Unreserve(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	nodeName string,
) {
	ts := t.mustGetTelemetryCycleState(state)
	t.tracker.RemoveSchedulingReservation(ts.Intdepl, p.Name)	
}

func (t *InternalTelemetry) PostBind(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	nodeName string,
) {
	ts := t.mustGetTelemetryCycleState(state)
	t.tracker.RemoveSchedulingReservation(ts.Intdepl, p.Name)	
	// TODO: check if this was the last pod and cleanup
}
