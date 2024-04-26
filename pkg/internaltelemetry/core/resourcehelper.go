package core

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
)

const CLUSTER_TELEMETRY_DEVICE_RESOURCES_NAME = "telemetry-device-resources"

type ResourceHelper struct {
	client client.Client
}

func NewResourceHelper(client client.Client) *ResourceHelper {
	return &ResourceHelper{
		client: client,
	}
}

func (r *ResourceHelper) GetTelemetrySwitchesThatCantBeSources(ctx context.Context, namespace string) []string {
	// TODO: consider watching changes instead of fetching
	telemetryResources := &intv1alpha.TelemetryDeviceResources{}
	resourceKey := types.NamespacedName{
		Name: CLUSTER_TELEMETRY_DEVICE_RESOURCES_NAME,
		Namespace: namespace,
	}
	res := []string{}
	if err := r.client.Get(ctx, resourceKey, telemetryResources); err != nil {
		klog.Errorf("Failed to get telemetry resources %e", err)
		return res
	}
	if telemetryResources.Status.DeviceResources == nil {
		return res
	}
	for _, deviceResources := range telemetryResources.Status.DeviceResources {
		if !deviceResources.CanBeSource {
			res = append(res, deviceResources.DeviceName)
		}
	}
	return res
}
