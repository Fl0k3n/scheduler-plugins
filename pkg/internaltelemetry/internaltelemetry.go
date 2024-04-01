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
	"sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
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
	deplMgr *core.DeploymentManager
}

var _ framework.ScorePlugin = &InternalTelemetry{}
var _ framework.FilterPlugin = &InternalTelemetry{}
var _ framework.PreFilterPlugin = &InternalTelemetry{}

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	klog.Info("initializing internal telemetry plugin: v0.0.1")

	scheme := runtime.NewScheme()
	_ = clientscheme.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = shimv1alpha.AddToScheme(scheme)
	_ = intv1alpha.AddToScheme(scheme)
	client, err := client.New(handle.KubeConfig(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	topoEngine := core.NewTopologyEngine(client)
	deplMgr := core.NewDeploymentManager(client)
	schedEngine := core.NewTelemetrySchedulingEngine()

	return &InternalTelemetry{
		Client: client,
		handle: handle,	
		topoEngine: topoEngine,
		schedEngine: schedEngine,
		deplMgr: deplMgr,
	}, nil
}


func (ts *InternalTelemetry) Name() string {
	return Name
}

func (ts *InternalTelemetry) PreFilter(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
) (*framework.PreFilterResult, *framework.Status) {
	var err error
	if err = ts.topoEngine.PrepareForScheduling(ctx, pod); err != nil {
		goto fail
	}
	if err = ts.deplMgr.PrepareForPodScheduling(ctx, pod); err != nil {
		goto fail
	}
	ts.schedEngine.PrepareForScheduling()
	return nil, nil
fail:
	klog.Errorf("Failed to prepare for scheduling %e", err)
	return nil, framework.AsStatus(err)
}

func (ts *InternalTelemetry) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (ts *InternalTelemetry) Filter(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeInfo *framework.NodeInfo,
) *framework.Status {
	// topologies := &shimv1alpha.TopologyList{}
	// if err := ts.List(ctx, topologies); err != nil {
	// 	klog.Errorf("Failed to fetch topologies %e", err)
	// }
	// incSwitches := &shimv1alpha.IncSwitchList{}
	// if err := ts.List(ctx, incSwitches); err != nil {
	// 	klog.Errorf("Failed to fetch incswitches %e", err)
	// }
	// klog.Infof("Filtering node %s for pod %s", nodeInfo.Node().Name, pod.Name)
	// // if nodeInfo.Node().Name == "test-worker2" {
	// if nodeInfo.Node().Name == "" { // TODO clear this
	// 	klog.Infof("Node %s didn't pass filter", nodeInfo.Node().Name)
	// 	return framework.NewStatus(framework.UnschedulableAndUnresolvable, "not here ;/")
	// }

	// intdepl := &intv1alpha.InternalInNetworkTelemetryDeployment{}
	// ownerName := pod.Labels[INTERNAL_TELEMETRY_POD_DEPLOYMENT_OWNER_LABEL]
	// resourceKey := types.NamespacedName{Name: ownerName, Namespace: pod.Namespace}
	// if err := ts.Get(ctx, resourceKey, intdepl); err != nil {
	// 	return framework.AsStatus(err)
	// }

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
