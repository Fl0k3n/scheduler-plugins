package core

import (
	"context"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
)

// TODO refactor this
const INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL = "inc.kntp.com/owned-by-iintdepl"
const INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL = "inc.kntp.com/part-of-deployment"

type DeploymentManager struct {
	client client.Client
	iintdeplCache sync.Map // iintdeplName: string -> *intv1alpha.iintdepl, it probably doesn't have to be a sync.Map
}

func NewDeploymentManager(client client.Client) *DeploymentManager {
	return &DeploymentManager{
		client: client,
	}
}

func (d *DeploymentManager) PrepareForPodScheduling(ctx context.Context, pod *v1.Pod) error {
	iintdeplName, okIintdepl := pod.Labels[INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL]
	deploymentName, okDepl := pod.Labels[INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL]
	_ = deploymentName
	if !okIintdepl || !okDepl {
		return fmt.Errorf("pod %s is not a part of iintdeployment", pod.Name)
	}
	var intdepl *intv1alpha.InternalInNetworkTelemetryDeployment
	iintdeplKey := types.NamespacedName{Name: iintdeplName, Namespace: pod.Namespace}
	if idepl, ok := d.iintdeplCache.Load(iintdeplKey); ok {
		intdepl = idepl.(*intv1alpha.InternalInNetworkTelemetryDeployment)
	} else {
		intdepl = &intv1alpha.InternalInNetworkTelemetryDeployment{}
		if err := d.client.Get(ctx, iintdeplKey, intdepl); err != nil {
			return err
		}
		d.iintdeplCache.Store(iintdeplKey, intdepl)
	}
	
	_ = intdepl
	return nil
}


func (d *DeploymentManager) onAllScheduled() {
	// clear cache
}
