package core

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
)

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
		t.reservations[client.ObjectKeyFromObject(intdepl)] = &ReservationState{
			Reservations: []ScheduledNode{},
			ScheduledCounters: newScheduledCounters(),
		}
	}
}

func (t *TelemetrySchedulingTracker) GetSchedulingState(
	ctx context.Context,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	currentPodsDeploymentName string,
) ([]ScheduledNode, QueuedPods, error) {
	scheduledAndBound, scheduledCounters, err := t.loadScheduledNodesForBoundPods(ctx, intdepl)
	if err != nil {
		return nil, QueuedPods{}, err
	}
	scheduledAndReserved := t.reservations[client.ObjectKeyFromObject(intdepl)]
	scheduledNodes := mergeScheduledNodes(scheduledAndBound, scheduledAndReserved.Reservations)
	counters := scheduledCounters.CombinedWith(scheduledAndReserved.ScheduledCounters)

	queuedPods := newQueuedPods()
	for _, depl := range intdepl.Spec.DeploymentTemplates {
		alreadyScheduled := counters.Get(depl.Name)
		if currentPodsDeploymentName == depl.Name {
			alreadyScheduled++
		}
		if depl.Template.Replicas == nil {
			return nil, QueuedPods{}, fmt.Errorf("deployment template %s doesn't have number of replicas", depl.Name)
		}
		remainingPods := int(*depl.Template.Replicas - int32(alreadyScheduled))
		if remainingPods < 0 {
			// this can happen if spec changed but hasn't been reconciled yet or if we fd up reservation tracking
			klog.Info("more pods were scheduled and reserved than stated in deployment")
			remainingPods = 0
		}
		queuedPods.PerDeploymentCounts[depl.Name] = remainingPods
	}
	return scheduledNodes, queuedPods, nil
}

func (t *TelemetrySchedulingTracker) ReserveForScheduling(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	nodeName string,
	podsDeploymentNames string,
) {
	reservations := t.reservations[client.ObjectKeyFromObject(intdepl)]
	reservations.Reservations = markScheduled(reservations.Reservations, nodeName, podsDeploymentNames)
	reservations.ScheduledCounters.Increment(podsDeploymentNames)
	t.reservations[client.ObjectKeyFromObject(intdepl)] = reservations
}

func (t *TelemetrySchedulingTracker) loadScheduledNodesForBoundPods(
	ctx context.Context,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) ([]ScheduledNode, *ScheduluedCounters, error) {
	pods := &v1.PodList{}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL: intdepl.Name,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	listOptions := &client.ListOptions{
		LabelSelector: selector,
		Namespace: intdepl.Namespace,
	}
	if err := t.client.List(ctx, pods, listOptions); err != nil {
		return nil, nil, err
	}
	res := []ScheduledNode{}
	counters := newScheduledCounters()
	for _, pod := range pods.Items {
		if pod.Spec.NodeName != "" {
			if deplName, ok := pod.Labels[INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL]; ok {
				res = markScheduled(res, pod.Spec.NodeName, deplName)	
				counters.Increment(deplName)
			} else {
				klog.Infof("Found pod %s belonging to intdepl: %s without deplname label", pod.Name, intdepl.Name)
			}
		}
	}
	return res, counters, nil
}
