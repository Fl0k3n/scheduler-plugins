package internaltelemetry

import (
	"context"
	"fmt"
	"strings"

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

const Name = "InternalTelemetry"
const INTERNAL_TELEMETRY_POD_DEPLOYMENT_OWNER_LABEL = "inc.kntp.com/owned-by-iintdepl"
const INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL = "inc.kntp.com/part-of-deployment"

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


func (ts *InternalTelemetry) Name() string {
	return Name
}

func (ts *InternalTelemetry) PreScore(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodes []*v1.Node,
) *framework.Status {
	var err error
	var network *core.Network
	var intdepl *intv1alpha.InternalInNetworkTelemetryDeployment
	var podsDeplName string
	if network, err = ts.topoEngine.PrepareForPodScheduling(ctx, pod); err != nil {
		goto fail
	}
	if intdepl, podsDeplName, err = ts.deplMgr.PrepareForPodScheduling(ctx, pod); err != nil {
		goto fail
	}
	ts.tracker.PrepareForPodScheduling(intdepl)

	_ = podsDeplName // TOOD
	ts.schedEngine.PrepareForPodScheduling(network, intdepl, []core.ScheduledNode{}) // TODO we need tracker of scheduled nodes
	return nil
fail:
	klog.Errorf("Failed to prepare for scheduling %e", err)
	return framework.AsStatus(err)
}

func (ts *InternalTelemetry) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (ts *InternalTelemetry) Score(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	nodeName string,
) (int64, *framework.Status) {
	klog.Infof("Scoring node %s for pod %s", nodeName, p.Name)
	fmt.Println("Scoring pod")
	nodeList := &v1.NodeList{}
	if err := ts.Client.List(ctx, nodeList); err != nil {
		fmt.Printf("Failed to fetch nodes: %e\n", err)	
	}
	nodeNames := []string{}
	for _, node := range nodeList.Items {
		nodeNames = append(nodeNames, node.Name)
	}
	fmt.Printf("Nodes: %s\n", strings.Join(nodeNames, ", "))
	return 50, nil
}

func (ts *InternalTelemetry) NormalizeScore(
	ctx context.Context,
	state *framework.CycleState,
	p *v1.Pod,
	scores framework.NodeScoreList,
) *framework.Status {
	maxScore := scores[0].Score
	for i := range scores {
		if scores[i].Score > maxScore {
			maxScore = scores[0].Score
		}
	}
	klog.Infof("max score: %d")
	return nil
}

func (ts *InternalTelemetry) ScoreExtensions() framework.ScoreExtensions {
	return ts
}
