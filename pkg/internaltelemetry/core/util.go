package core

import (
	v1 "k8s.io/api/core/v1"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
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

func getOppositeDeployment(
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	podDeplName string,
) string {
	for _, depl := range intdepl.Spec.DeploymentTemplates {
		if depl.Name != podDeplName {
			return depl.Name
		}
	}
	panic("opposite deployment not found")
}


func getNodesWithDeployment(sn []ScheduledNode, deplName string) map[string]struct {} {
	res := map[string]struct{}{}
	for _, node := range sn {
		if node.HasScheduled(deplName) {
			res[node.Name] = struct{}{}
		}
	}
	return res
}

func extractNames(nodes []*v1.Node) []string {
	res := make([]string, len(nodes))
	for i, node := range nodes {
		res[i] = node.Name
	}
	return res
}

func toNodeNameSet(nodes []*v1.Node) map[string]struct{} {
	res := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		res[n.Name] = struct{}{}
	}	
	return res
}
