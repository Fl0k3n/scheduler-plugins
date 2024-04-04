package core

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
)

type ReservationState struct {
	Reservations []ScheduledNode
}

type TelemetrySchedulingTracker struct {
	client client.Client
	reservations map[types.NamespacedName]*ReservationState // key is namespaced name of intdepl 
}

func NewTelemetrySchedulingTracker(client client.Client) *TelemetrySchedulingTracker {
	return &TelemetrySchedulingTracker{
		client: client,
		reservations: map[types.NamespacedName]*ReservationState{},
	}	
}

func (t *TelemetrySchedulingTracker) PrepareForPodScheduling(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) {
	if _, ok := t.reservations[client.ObjectKeyFromObject(intdepl)]; !ok {
		t.reservations[client.ObjectKeyFromObject(intdepl)] = &ReservationState{Reservations: []ScheduledNode{}}
	}
}

func (t *TelemetrySchedulingTracker) GetSchedulingState(
	ctx context.Context,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	currentPodsDeploymentName string,
) ([]ScheduledNode, QueuedPods, error) {
	scheduledAndBound, err := t.loadScheduledNodesForBoundPods(ctx, intdepl)
	if err != nil {
		return nil, QueuedPods{}, err
	}
	scheduledAndReserved := t.reservations[client.ObjectKeyFromObject(intdepl)].Reservations
	scheduledNodes := mergeScheduledNodes(scheduledAndBound, scheduledAndReserved)
	// TODO need counters in reservations and return counter in loadSchedu..ForBoundPods
	return scheduledNodes, QueuedPods{}, nil
}

func (t *TelemetrySchedulingTracker) ReserveForScheduling(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	nodeName string,
	podsDeploymentNames string,
) {
	reservations := t.reservations[client.ObjectKeyFromObject(intdepl)]
	reservations.Reservations = markScheduled(reservations.Reservations, nodeName, podsDeploymentNames)
	t.reservations[client.ObjectKeyFromObject(intdepl)] = reservations
}

func (t *TelemetrySchedulingTracker) loadScheduledNodesForBoundPods(
	ctx context.Context,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) ([]ScheduledNode, error) {
	pods := &v1.PodList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL: intdepl.Name,
		},
	})
	if err != nil {
		return nil, err
	}
	listOptions := &client.ListOptions{
		LabelSelector: selector,
		Namespace: intdepl.Namespace,
	}
	if err := t.client.List(ctx, pods, listOptions); err != nil {
		return nil, err
	}
	res := []ScheduledNode{}
	for _, pod := range pods.Items {
		if pod.Spec.NodeName != "" {
			if deplName, ok := pod.Labels[INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL]; ok {
				res = markScheduled(res, pod.Spec.NodeName, deplName)	
			} else {
				klog.Infof("Found pod %s belonging to intdepl: %s without deplname label", pod.Name, intdepl.Name)
			}
		}
	}
	return res, nil
}
