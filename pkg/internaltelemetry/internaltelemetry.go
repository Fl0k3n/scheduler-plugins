package internaltelemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
)

const Name = "InternalTelemetry"

type InternalTelemetry struct {
	client.Client
	handle     framework.Handle
}

var _ framework.ScorePlugin = &InternalTelemetry{}
var _ framework.FilterPlugin = &InternalTelemetry{}

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	scheme := runtime.NewScheme()
	_ = clientscheme.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	client, err := client.New(handle.KubeConfig(), client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &InternalTelemetry{
		Client: client,
		handle: handle,	
	}, nil
}


func (ts *InternalTelemetry) Name() string {
	return Name
}

func (ts *InternalTelemetry) Filter(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeInfo *framework.NodeInfo,
) *framework.Status {
	klog.Infof("Filtering node %s for pod %s", nodeInfo.Node().Name, pod.Name)
	if nodeInfo.Node().Name != "test-cluster-worker" {
		klog.Infof("Node %s didn't pass filter", nodeInfo.Node().Name)
		return framework.AsStatus(errors.New("unshedulable here"))
	}
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

func (ts *InternalTelemetry) ScoreExtensions() framework.ScoreExtensions {
	return nil
}
