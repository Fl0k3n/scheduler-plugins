package core

import shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"

func toMapByDeviceName(topo *shimv1alpha.Topology) map[string]*shimv1alpha.NetworkDevice {
	res := make(map[string]*shimv1alpha.NetworkDevice, len(topo.Spec.Graph))
	for _, dev := range topo.Spec.Graph {
		res[dev.Name] = dev.DeepCopy()
	}
	return res
}

func toOrdinalsByDeviceName(topo *shimv1alpha.Topology) map[string]int {
	res := make(map[string]int, len(topo.Spec.Graph))
	for i, dev := range topo.Spec.Graph {
		res[dev.Name] = i
	}
	return res
}
