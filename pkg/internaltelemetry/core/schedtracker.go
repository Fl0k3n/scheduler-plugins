package core

import (
	"context"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
)

type TelemetrySchedulingTracker struct {
	client client.Client

	// types.NamespacedName -> *ReservationState, key is namespaced name of intdepl
	// reservation handling must be thread-safe as it may be called from different contexts
	reservations sync.Map 
}

func NewTelemetrySchedulingTracker(client client.Client) *TelemetrySchedulingTracker {
	return &TelemetrySchedulingTracker{
		client: client,
		reservations: sync.Map{},
	}	
}

func (t *TelemetrySchedulingTracker) PrepareForPodScheduling(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) {
	t.reservations.LoadOrStore(client.ObjectKeyFromObject(intdepl), newReservationState())
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
	reservedNodes, reservedCounters := t.getReservedSchedulingState(intdepl)
	scheduledNodes := mergeScheduledNodes(scheduledAndBound, reservedNodes)
	counters := scheduledCounters.CombinedWith(reservedCounters)

	queuedPods := newQueuedPods()
	for _, depl := range intdepl.Spec.DeploymentTemplates {
		alreadyScheduled := counters.Get(depl.Name)
		if currentPodsDeploymentName == depl.Name {
			alreadyScheduled++
		}
		if depl.Template.Replicas == nil {
			return nil, QueuedPods{}, fmt.Errorf("deployment template %s doesn't have number of replicas set", depl.Name)
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

func (t *TelemetrySchedulingTracker) getReservedSchedulingState(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
) ([]ScheduledNode, *ScheduluedCounters) {
	key := client.ObjectKeyFromObject(intdepl)
	reservationState, ok := t.reservations.Load(key)
	scheduledNodes := []ScheduledNode{}
	counters := newScheduledCounters()
	if !ok {
		klog.Errorf("Reservation state for %v not found", key)
		return scheduledNodes, counters
	}
	rs := reservationState.(*ReservationState)
	rs.StateLock.Lock()
	defer rs.StateLock.Unlock()
	for _, v := range rs.Reservations {
		scheduledNodes = markScheduled(scheduledNodes, v.NodeName, v.PodsDeploymentName)
		counters.Increment(v.PodsDeploymentName)
	}
	return scheduledNodes, counters
}

func (t *TelemetrySchedulingTracker) ReserveForScheduling(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	nodeName string,
	podsDeploymentName string,
	podName string,
) {
	key := client.ObjectKeyFromObject(intdepl)
	reservationState, ok := t.reservations.Load(key)
	if !ok {
		klog.Errorf("Reservation state for %v not found", key)
		return
	}
	rs := reservationState.(*ReservationState)
	rs.StateLock.Lock()
	defer rs.StateLock.Unlock()
	rs.Reservations[podName] = ReservationMeta{
		NodeName: nodeName,
		PodsDeploymentName: podsDeploymentName,
	}
}

// idempotent, if reservation was not present function doesn't do anything
func (t *TelemetrySchedulingTracker) RemoveSchedulingReservation(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	podName string,
) {
	key := client.ObjectKeyFromObject(intdepl)
	if reservationState, ok := t.reservations.Load(key); ok {
		// TODO we also need to delete t.reservations[key] when last reservation was removed
		rs := reservationState.(*ReservationState)
		rs.StateLock.Lock()
		defer rs.StateLock.Unlock()
		delete(rs.Reservations, podName)
	}
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
