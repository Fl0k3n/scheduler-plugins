package core

import (
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

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

func deepCopyScheduledNodes(sns []ScheduledNode) []ScheduledNode {
	res := make([]ScheduledNode, len(sns))
	for i, sn := range sns {
		res[i] = sn.DeepCopy()
	}
	return res
}

func markScheduled(sns []ScheduledNode, nodeName string, deplName string) []ScheduledNode {
	for i, sn := range sns {
		if sn.Name == nodeName {
			sns[i].MarkScheduled(deplName)
			return sns
		}
	}
	return append(sns, ScheduledNode{Name: nodeName, ScheduledDeployments: []string{deplName}})
}

func mergeScheduledNodes (sn1 []ScheduledNode, sn2 []ScheduledNode) []ScheduledNode {
	res := []ScheduledNode{}
	joined := map[string]map[string]struct{}{}
	for _, sn := range append(sn1, sn2...) {
		var deplsSet map[string]struct{}
		if depls, ok := joined[sn.Name]; ok {
			deplsSet = depls
		} else {
			deplsSet = map[string]struct{}{}
		}
		for _, depl := range sn.ScheduledDeployments {
			deplsSet[depl] = struct{}{}
		}
		joined[sn.Name] = deplsSet
	}
	for nodeName, deplsSet := range joined {
		deplsSlice := make([]string, len(deplsSet))
		i := 0
		for depl := range deplsSet {
			deplsSlice[i] = depl
			i++
		}
		res = append(res, ScheduledNode{nodeName, deplsSlice})
	}
	return res
}
