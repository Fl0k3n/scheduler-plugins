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


// TODO this is for evaluation purposes and should be deleted
var DoneChan = make(chan struct{})
var PodsToSchedule = 0
var ScheduledPods = 0
var IsEvaluatingDefaultScheduler = false

type TelemetryCycleState struct {
	Network *core.Network
	QueuedPods core.QueuedPods
	Intdepl *intv1alpha.InternalInNetworkTelemetryDeployment
	PodsDeploymentName string
	TelemetrySwitchesThatCantBeSources []string
	ScoreProvider core.NodeScoreProvider
}

// cycle state is treated as immutable and hence full shallow copy suffices
func (t *TelemetryCycleState) Clone() framework.StateData {
	return &TelemetryCycleState{
		Network: t.Network,
		QueuedPods: t.QueuedPods,
		Intdepl: t.Intdepl,
		PodsDeploymentName: t.PodsDeploymentName,
		TelemetrySwitchesThatCantBeSources: t.TelemetrySwitchesThatCantBeSources,
		ScoreProvider: t.ScoreProvider,
	}
}

type InternalTelemetry struct {
	client.Client
	handle     framework.Handle
	topoEngine *core.TopologyEngine
	schedEngine *core.TelemetrySchedulingEngine
	tracker *core.TelemetrySchedulingTracker
	deplMgr *core.DeploymentManager
	resourceHelper *core.ResourceHelper
}

var _ framework.PreScorePlugin = &InternalTelemetry{}
var _ framework.ScorePlugin = &InternalTelemetry{}
var _ framework.ReservePlugin = &InternalTelemetry{}
var _ framework.PostBindPlugin = &InternalTelemetry{}

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	klog.Info("initializing internal telemetry plugin: v0.0.7")

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
	resourceHelper := core.NewResourceHelper(client)
	schedEngine := core.NewTelemetrySchedulingEngine()

	return &InternalTelemetry{
		Client: client,
		handle: handle,	
		topoEngine: topoEngine,
		schedEngine: schedEngine,
		tracker: tracker,
		deplMgr: deplMgr,
		resourceHelper: resourceHelper,
	}, nil
}

func (t *InternalTelemetry) Name() string {
	return Name
}

func (t *InternalTelemetry) getTelemetryCycleState(cycleState *framework.CycleState) (*TelemetryCycleState, error) {
	ts, err := cycleState.Read(TELEMETRY_CYCLE_STATE_KEY)
	if err != nil {
		err = errors.New("assertion failed, telemetry cycle state is not set")
		klog.Error(err)
		return nil, err
	}
	return ts.(*TelemetryCycleState), nil
}

func (t *InternalTelemetry) getFeasibleNodesForOppositeDeploymentProvider(
	feasibleNodesForThisDeployment []*v1.Node,	
) func() []*v1.Node {
	// TODO: find a way to reuse some of K8s internal mechanisms to figure this out
	// for now we assume that the same nodes are feasible for both deployments
	return func() []*v1.Node {
		return feasibleNodesForThisDeployment
	}
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
	var switchesThatCantBeSources []string
	var scoreProvider core.NodeScoreProvider

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
	scoreProvider, _ = t.schedEngine.Prescore(
		network,
		intdepl,
		podsDeplName,
		nodes,
		t.getFeasibleNodesForOppositeDeploymentProvider(nodes),
		scheduledNodes,
	)
	switchesThatCantBeSources = t.resourceHelper.GetTelemetrySwitchesThatCantBeSources(ctx, intdepl.Namespace)
	state.Write(TELEMETRY_CYCLE_STATE_KEY, &TelemetryCycleState{
		Network: network,
		QueuedPods: queuedPods,
		Intdepl: intdepl,
		PodsDeploymentName: podsDeplName,
		TelemetrySwitchesThatCantBeSources: switchesThatCantBeSources,
		ScoreProvider: scoreProvider,
	})
	return nil
fail:
	klog.Errorf("Failed to prepare for scheduling %e", err)
	return framework.AsStatus(err)
}

func (t *InternalTelemetry) Score(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeName string,
) (int64, *framework.Status) {
	klog.Infof("Scoring node %s for pod %s", nodeName, pod.Name)
	telemetryState, err := t.getTelemetryCycleState(state)
	if err != nil {
		return 0, framework.AsStatus(err)
	}
	score := t.schedEngine.Score(nodeName, telemetryState.ScoreProvider)
	klog.Infof("Node %s score for pod %s = %d", nodeName, pod.Name, score)
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
			scores[node].Score = framework.MaxNodeScore
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
	ts, err := t.getTelemetryCycleState(state)
	if err != nil {
		return framework.AsStatus(err)
	}
	t.tracker.ReserveForScheduling(ts.Intdepl, nodeName, ts.PodsDeploymentName, p.Name)
	return nil
}

func (t *InternalTelemetry) Unreserve(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	nodeName string,
) {
	ts, err := t.getTelemetryCycleState(state)
	if err != nil {
		klog.Errorf("failed to unreserve pod because telemetry cycle state wasn't found: %e", err)
		return
	}
	t.tracker.RemoveSchedulingReservation(ts.Intdepl, p.Name)	
}

func (t *InternalTelemetry) PostBind(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	nodeName string,
) {
	// TODO this is for evaluation purposes and should be deleted
	if IsEvaluatingDefaultScheduler {
		ScheduledPods++
		if ScheduledPods == PodsToSchedule {
			close(DoneChan)
		}
		return
	}
	ts, err := t.getTelemetryCycleState(state)
	if err != nil {
		klog.Errorf("failed to unreserve pod because telemetry cycle state wasn't found: %e", err)
		return
	}
	t.tracker.RemoveSchedulingReservation(ts.Intdepl, p.Name)	
	// TODO this is for evaluation purposes and should be deleted
	if ts.QueuedPods.Empty() {
		close(DoneChan)
	}
}
