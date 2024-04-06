package core

import (
	"context"
	"slices"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)


func TestTopologyEngine(t *testing.T) {
	tests := []struct {
		name              string
		pod 			  *v1.Pod	
		topology		  *shimv1alpha.Topology
		telemetrySwitches []string
		wantError         bool
	}{
		{
			name: "preparation fails when there is no topology",
			pod: st.MakePod().Name("p1").UID("p1").Namespace("ns1").Obj(),
			topology: nil,
			telemetrySwitches: []string{},
			wantError: true,
		},
		{
			name: "preparation succeeds for tree topology",
			pod: st.MakePod().Name("p1").UID("p1").Namespace("ns1").Obj(),
			topology: testTopology("shallow"),
			telemetrySwitches: []string{"r0", "r1", "r2"},
			wantError: false,
		},
		{
			name: "preparation fails for not-tree topology",
			pod: st.MakePod().Name("p1").UID("p1").Namespace("ns1").Obj(),
			topology: withExtraEdges(testTopology("shallow"), edge{"r1", "r2"}),
			telemetrySwitches: []string{"r0", "r1", "r2"},
			wantError: true,
		},
		{
			name: "preparation succeeds when there are no inc switches",
			pod: st.MakePod().Name("p1").UID("p1").Namespace("ns1").Obj(),
			topology: makeTestTopology(
				[]vertex{
					{"external", shimv1alpha.EXTERNAL},
					{"r0", shimv1alpha.NET},
					{"n0", shimv1alpha.NODE},
				},
				[]edge{
					{"external", "r0"},
					{"r0", "n0"},
				},
			),
			telemetrySwitches: []string{},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []runtime.Object{}
			if tt.topology != nil {
				objs = append(objs, tt.topology)
			}
			objs = append(objs, tt.pod)
			if tt.topology != nil {
				objs = append(objs, makeTestIncSwitchesForTopo(tt.topology, tt.telemetrySwitches)...)
			}
			client := newFakeClient(t, objs...)

			topoEngine := NewTopologyEngine(client)
			ctx := context.Background()
			_, err := topoEngine.PrepareForPodScheduling(ctx, tt.pod)
			if tt.wantError && err == nil {
				t.Errorf("expected error but succeeded")
			} else if !tt.wantError && err != nil {
				t.Errorf("expected success, but failed with %e", err)
			}
		})
	}
}

func TestDeploymentManager(t *testing.T) {
	tests := []struct {
		name                string
		pod 			    *v1.Pod
		intdepl 		    *intv1alpha.InternalInNetworkTelemetryDeployment 
		expectedIntDeplName string
		expectedDeplName 	string
		wantError        	bool
	}{
		{
			name: "manager can be successfuly prepared for pod belonging to existing intdepl with proper labels",
			pod: st.MakePod().Name("p1").UID("p1").Namespace("ns1").Labels(map[string]string{
				INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL: "intdepl",
				INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL: "depl",
			}).Obj(),
			intdepl: makeTestIintDepl("intdepl", "ns1", []deploymentInfo{{"depl", 1, "foo"}}),
			expectedIntDeplName: "intdepl",
			expectedDeplName: "depl",
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []runtime.Object{tt.pod, tt.intdepl}
			client := newFakeClient(t, objs...)
			deplMgr := NewDeploymentManager(client)
			ctx := context.Background()
			intdepl, deplName, err := deplMgr.PrepareForPodScheduling(ctx, tt.pod)
			if tt.wantError && err == nil {
				t.Fatalf("expected error but succeeded")
			} else if !tt.wantError && err != nil {
				t.Fatalf("expected success, but failed with %e", err)
			}
			if intdepl.Name != tt.expectedIntDeplName {
				t.Errorf("invalid intdeplname, expected %s got %s", tt.expectedIntDeplName, intdepl.Name)
			}
			if deplName != tt.expectedDeplName { 
				t.Errorf("invalid deplname, expected %s got %s", tt.expectedDeplName, deplName)
			}
		})
	}
}

func TestSchedulingTracker(t *testing.T) {
	depl1 := "depl1"
	depl2 := "depl2"
	makePod := func(name string, deplName string, nodeName string) *v1.Pod { 
		return st.MakePod().Name(name).UID("p1").Namespace("ns1").Labels(map[string]string{
			INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL: "intdepl",
			INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL: deplName,
		}).Node(nodeName).Obj()
	}
	withoutIntDeplLabel := func (p *v1.Pod) *v1.Pod {
		delete(p.Labels, INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL)
		return p
	}
	intDepl := func(first int32, second int32) *intv1alpha.InternalInNetworkTelemetryDeployment {
		depl1Info := deploymentInfo{name: depl1, replicas: -1, podLabel: "foo1"}
		depl2Info := deploymentInfo{name: depl2, replicas: -1, podLabel: "foo2"}
		intDeplTemplate := makeTestIintDepl("intdepl", "ns1", []deploymentInfo{depl1Info, depl2Info})
		res := intDeplTemplate.DeepCopy()
		res.Spec.DeploymentTemplates[0].Template.Replicas = &first
		res.Spec.DeploymentTemplates[1].Template.Replicas = &second
		return res
	}
	queuedPods := func (depl1Count int, depl2Count int) QueuedPods {
		return QueuedPods{
			PerDeploymentCounts: map[string]int{
				depl1: depl1Count,
				depl2: depl2Count,
			},
		}
	}
	type reservation struct {
		pod *v1.Pod
		deplName string
		nodeName string
	}
	rsv := func (podName, deplName, nodeName string) reservation {
		return reservation{
			pod: makePod(podName, deplName, ""),
			deplName: deplName,
			nodeName: nodeName,
		}
	}

	merge := mergeScheduledNodes
	t.Run("scheduled nodes and queued pods are correctly computed", func(t *testing.T) {
		tests := []struct{
			name 					string
			intdepl 				*intv1alpha.InternalInNetworkTelemetryDeployment
			podsDeploymentName 		string
			scheduledPods           []*v1.Pod
			reservations 			[]reservation
			expectedScheduledNodes  []ScheduledNode
			expectedQueuedPods 		QueuedPods
		}{
			{
				name: "when no pods were schedule scheduled nodes are empty and queued pods have all remaining replicas",
				intdepl: intDepl(2, 2),
				podsDeploymentName: depl1,
				scheduledPods: []*v1.Pod{},
				reservations: []reservation{},
				expectedScheduledNodes: []ScheduledNode{},
				expectedQueuedPods: queuedPods(1, 2),
			},
			{
				name: "scheduled pods of given intdepl contribute to tracked state",
				intdepl: intDepl(3, 3),
				podsDeploymentName: depl1,
				scheduledPods: []*v1.Pod{makePod("p1", depl1, "n1"), makePod("p2", depl2, "n1"), makePod("p3", depl2, "n2")},
				reservations: []reservation{},
				expectedScheduledNodes: merge(schedule(depl1, "n1"), schedule(depl2, "n1", "n2")),
				expectedQueuedPods: queuedPods(1, 1),
			},
			{
				name: "only pods of given intdepl contribute to tracked state",
				intdepl: intDepl(2, 2),
				podsDeploymentName: depl1,
				scheduledPods: []*v1.Pod{makePod("p1", depl1, "n1"), withoutIntDeplLabel(makePod("p2", "x", "n1"))},
				reservations: []reservation{},
				expectedScheduledNodes: schedule(depl1, "n1"),
				expectedQueuedPods: queuedPods(0, 2),
			},
			{
				name: "reservations contribute to tracked state",
				intdepl: intDepl(3, 3),
				podsDeploymentName: depl1,
				scheduledPods: []*v1.Pod{},
				reservations: []reservation{rsv("p1", depl1, "n1"), rsv("p2", depl2, "n1"), rsv("p3", depl2, "n2")},
				expectedScheduledNodes: merge(schedule(depl1, "n1"), schedule(depl2, "n1", "n2")),
				expectedQueuedPods: queuedPods(1, 1),
			},
			{
				name: "reservations are correctly combined with scheduled nodes",
				intdepl: intDepl(5, 5),
				podsDeploymentName: depl1,
				scheduledPods: []*v1.Pod{makePod("p1", depl1, "n1"), makePod("p2", depl2, "n1"), makePod("p3", depl2, "n2")},
				reservations: []reservation{rsv("p4", depl1, "n1"), rsv("p5", depl2, "n3"), rsv("p6", depl1, "n4"), rsv("p7", depl2, "n4")},
				expectedScheduledNodes: merge(schedule(depl1, "n1", "n4"), schedule(depl2, "n1", "n2", "n3", "n4")),
				expectedQueuedPods: queuedPods(1, 1),
			},
		}
		for _, tt := range tests[3:] {
			t.Run(tt.name, func(t *testing.T) {
				objs := []runtime.Object{tt.intdepl}
				for _, pod := range tt.scheduledPods {
					objs = append(objs, pod)
				}
				for _, r := range tt.reservations {
					objs = append(objs, r.pod)
				}
				client := newFakeClient(t, objs...)
				tracker := NewTelemetrySchedulingTracker(client)
				tracker.PrepareForPodScheduling(tt.intdepl)
				for _, r := range tt.reservations {
					tracker.ReserveForScheduling(tt.intdepl, r.nodeName, r.deplName, r.pod.Name)
				}
				ctx := context.Background()
				scheduledNodes, queuedPods, err := tracker.GetSchedulingState(ctx, tt.intdepl, tt.podsDeploymentName)
				if err != nil {
					t.Fatal(err)
				}
				if actual, expected := len(scheduledNodes), len(tt.expectedScheduledNodes); actual != expected {
					t.Errorf("Invalid scheduledNodes length, expected %d, got %d", expected, actual)
				} else {
					for _, esn := range tt.expectedScheduledNodes {
						if idx := slices.IndexFunc(scheduledNodes, func(sn ScheduledNode) bool {
							return esn.Name == sn.Name
						}); idx == -1 {
							t.Errorf("Scheduled node %s not found", esn.Name)
						} else if !esn.Equals(&scheduledNodes[idx]) {
							t.Errorf("Invalid deployments on node %s, expected %v got %v",
								 esn.Name, esn.ScheduledDeployments, scheduledNodes[idx].ScheduledDeployments)
						}
					}
				}
				if actual, expected := len(queuedPods.PerDeploymentCounts),
									   len(tt.expectedQueuedPods.PerDeploymentCounts); actual != expected {
					t.Errorf("Invalid queuedPods length, expected %d, got %d", expected, actual)
				} else {
					for k, expectedCount := range tt.expectedQueuedPods.PerDeploymentCounts {
						if actualCount, ok := queuedPods.PerDeploymentCounts[k]; ok {
							if expectedCount != actualCount {
								t.Errorf("invalid deployment count for %s, expected %d, got %d", k, expectedCount, actualCount)
							}
						} else {
							t.Errorf("deployment's count %s not found in %v", k, queuedPods)
						}
					}
				}
			})
		}
	})
}

func TestScoringEngine(t *testing.T) {
	depl1 := "depl1"
	depl2 := "depl2"
	makePod := func(name string, deplName string) *v1.Pod { 
		return st.MakePod().Name(name).UID("p1").Namespace("ns1").Labels(map[string]string{
			INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL: "intdepl",
			INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL: deplName,
		}).Obj()
	}
	intDepl := func(first int32, second int32) *intv1alpha.InternalInNetworkTelemetryDeployment {
		depl1Info := deploymentInfo{name: depl1, replicas: -1, podLabel: "foo1"}
		depl2Info := deploymentInfo{name: depl2, replicas: -1, podLabel: "foo2"}
		intDeplTemplate := makeTestIintDepl("intdepl", "ns1", []deploymentInfo{depl1Info, depl2Info})
		res := intDeplTemplate.DeepCopy()
		res.Spec.DeploymentTemplates[0].Template.Replicas = &first
		res.Spec.DeploymentTemplates[1].Template.Replicas = &second
		return res
	}
	allV4IncSwitches := func() []string {
		return []string{"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7"}
	}
	merge := mergeScheduledNodes

	t.Run("scheduler's network representation correctly tracks utilized telemetry ports", func(t *testing.T) {
		tests := []struct {
			name                string
			topo 				*shimv1alpha.Topology
			telemetrySwitches   []string
			pod 			    *v1.Pod
			intdepl 		    *intv1alpha.InternalInNetworkTelemetryDeployment
			scheduledNodes      []ScheduledNode
			expectedUsedPorts   []switchPort
		}{
			{
				name: "no ports are used when nothing was scheduled yet",
				topo: testTopology("v4-tree"),
				telemetrySwitches: allV4IncSwitches(),
				pod: makePod("p1", depl1),
				intdepl: intDepl(2, 2),
				scheduledNodes: []ScheduledNode{},
				expectedUsedPorts: []switchPort{},
			},
			{
				name: "no ports are used when pods of only 1 deployment were scheduled",
				topo: testTopology("v4-tree"),
				telemetrySwitches: allV4IncSwitches(),
				pod: makePod("p1", depl1),
				intdepl: intDepl(2, 2),
				scheduledNodes: schedule("depl2", "w1", "w4"),
			},
			{	
				name: "all ports on path between between 2 pods of different deployments are used",
				topo: testTopology("v4-tree"),
				telemetrySwitches: allV4IncSwitches(),
				pod: makePod("p1", depl1),
				intdepl: intDepl(2, 2),
				scheduledNodes: merge(schedule("depl1", "w1"), schedule("depl2", "w4")),
				expectedUsedPorts: []switchPort{
					{"r5", "w1", false},
					{"r5", "r1", true},
					{"r1", "r0", true},
					{"r0", "r3", true},
					{"r3", "r7", true},
					{"r7", "w4", false},
				},
			},
			{	
				name: "many-to-many communication paths are used",
				topo: testTopology("v4-tree"),
				telemetrySwitches: allV4IncSwitches(),
				pod: makePod("p1", depl1),
				intdepl: intDepl(3, 2),
				scheduledNodes: merge(schedule("depl1", "w1", "w3"), schedule("depl2", "w4", "w5")),
				expectedUsedPorts: []switchPort{
					{"r5", "w1", false},
					{"r5", "r1", true},
					{"r1", "r0", true},
					{"r0", "r3", true},
					{"r3", "r7", true},
					{"r7", "w4", false},
					{"r0", "r4", true},
					{"r0", "r2", true},
					{"r4", "w5", false},
					{"r6", "w3", false},
					{"r6", "r2", true},
					{"r6", "r0", true},
				},
			},
			{	
				name: "only telemetry-enabled switches have used ports",
				topo: testTopology("v4-tree"),
				telemetrySwitches: []string{"r1", "r3", "r6"},
				pod: makePod("p1", depl1),
				intdepl: intDepl(3, 2),
				scheduledNodes: merge(schedule("depl1", "w1", "w3"), schedule("depl2", "w4")),
				expectedUsedPorts: []switchPort{
					{"r1", "r5", false},
					{"r1", "r0", false},
					{"r3", "r0", false},
					{"r3", "r7", false},
					{"r6", "w3", false},
					{"r6", "r2", false},
				},
			},
			{
				name: "1 telemetry switch on route is not enough to setup telemetry",
				topo: testTopology("v4-tree"),
				telemetrySwitches: []string{"r5"},
				pod: makePod("p1", depl1),
				intdepl: intDepl(2, 2),
				scheduledNodes: merge(schedule("depl1", "w1"), schedule("depl2", "w2")),
				expectedUsedPorts: []switchPort{},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sched := NewTelemetrySchedulingEngine(DefaultTelemetrySchedulingEngineConfig())
				objs := []runtime.Object{tt.topo, tt.pod, tt.intdepl}
				objs = append(objs, makeTestIncSwitchesForTopo(tt.topo, tt.telemetrySwitches)...)
				client := newFakeClient(t, objs...)
				topoEngine := NewTopologyEngine(client)
				ctx := context.Background()
				net, err := topoEngine.PrepareForPodScheduling(ctx, tt.pod)
				if err != nil {
					t.Fatal(err)
				}
				sched.PrepareForPodScheduling(net, tt.intdepl, tt.scheduledNodes)
				schedState := sched.MustGetCachedDeploymentsNetworkView(tt.intdepl)

				shouldBeUsed := func(from string, to string) bool {
					for _, p := range tt.expectedUsedPorts {
						if p.switchName == from && p.peerName == to || (p.bidir && p.switchName == to && p.peerName == from) {
							return true
						}
					}
					return false
				}
				for _, v := range net.Vertices {
					mustHaveNoTelemetryPorts := slices.Index(tt.telemetrySwitches, v.Name) == -1
					portMeta := schedState.PortMetaOf(v)
					if mustHaveNoTelemetryPorts {
						if len(portMeta.AvailableTelemetryPorts) > 0 {
							t.Errorf("Expected %s to have no availabe ports, but got: %v", v.Name, portMeta.AvailableTelemetryPorts)
						}
					} else {
						for _, neigh := range v.Neighbors() {
							is := !portMeta.IsPortUnallocated(neigh.Name)
							if shouldBe := shouldBeUsed(v.Name, neigh.Name); is != shouldBe {
								if shouldBe {
									t.Errorf("Expected port from %s to %s to be used but isn't", v.Name, neigh.Name)
								} else {
									t.Errorf("Expected port from %s to %s to be unused but is", v.Name, neigh.Name)
								}
							}
						}
					}
				}
			})
		}
	})
	t.Run("sched correctly scores nodes", func(t *testing.T) {
		tests := []struct {
			name                string
			topo 				*shimv1alpha.Topology
			telemetrySwitches   []string
			pod 			    *v1.Pod
			podsDeploymentName	string
			intdepl 		    *intv1alpha.InternalInNetworkTelemetryDeployment
			queuedPods 			QueuedPods
			scheduledNodes      []ScheduledNode
			nodesToScore 		[]string
			expectedScores      []int
		}{
			{
				name: "for 2 pods score for nodes on on longest unused path is highest",
				topo: testTopology("v4-tree"),
				telemetrySwitches: []string{"r1", "r0", "r3"},
				pod: makePod("p1", depl1),
				podsDeploymentName: depl1,
				intdepl: intDepl(1, 1),
				queuedPods: QueuedPods{PerDeploymentCounts: map[string]int{depl1: 0, depl2: 1}},
				scheduledNodes: []ScheduledNode{},
				nodesToScore: []string{"w1", "w2", "w3", "w4", "w5"},
				expectedScores: []int{6, 6, 4, 6, 4},
			},
			{
				name: "for second pod score for nodes on longest unused path is highest",
				topo: testTopology("v4-tree"),
				telemetrySwitches: []string{"r1", "r0", "r3"},
				pod: makePod("p1", depl1),
				podsDeploymentName: depl1,
				intdepl: intDepl(1, 1),
				queuedPods: QueuedPods{PerDeploymentCounts: map[string]int{depl1: 0, depl2: 0}},
				scheduledNodes: schedule(depl2, "w4"),
				nodesToScore: []string{"w1", "w2", "w3", "w4", "w5"},
				expectedScores: []int{6, 6, 4, 0, 4},
			},
			{
				name: "score for node where pod of same type is already scheduled is 0",
				topo: testTopology("v4-tree"),
				telemetrySwitches: []string{"r1", "r0", "r3"},
				pod: makePod("p1", depl1),
				podsDeploymentName: depl1,
				intdepl: intDepl(2, 2),
				queuedPods: QueuedPods{PerDeploymentCounts: map[string]int{depl1: 0, depl2: 1}},
				scheduledNodes: merge(schedule(depl1, "w1"), schedule(depl2, "w4")),
				nodesToScore: []string{"w1"},
				expectedScores: []int{0},
			},
			{
				name: "scoring accounts for 1:n communication model",
				topo: testTopology("v4-tree"),
				telemetrySwitches: []string{"r1", "r0", "r3"},
				pod: makePod("p1", depl1),
				podsDeploymentName: depl1,
				intdepl: intDepl(1, 2),
				queuedPods: QueuedPods{PerDeploymentCounts: map[string]int{depl1: 0, depl2: 0}},
				scheduledNodes: schedule(depl2, "w3", "w4"),
				nodesToScore: []string{"w1", "w2", "w3", "w4", "w5"},
				expectedScores: []int{7, 7, 4, 4, 4},
			},
			{
				name: "scoring accounts for n:m communication model",
				topo: testTopology("v4-tree"),
				telemetrySwitches: []string{"r1", "r0", "r3"},
				pod: makePod("p1", depl1),
				podsDeploymentName: depl1,
				intdepl: intDepl(2, 2),
				queuedPods: QueuedPods{PerDeploymentCounts: map[string]int{depl1: 0, depl2: 0}},
				scheduledNodes: merge(schedule(depl1, "w1"), schedule(depl2, "w3", "w4")),
				nodesToScore: []string{"w1", "w2", "w3", "w4", "w5"},
				expectedScores: []int{0, 0, 0, 0, 1},
			},
			{
				name: "score is 0 if there is just 1 telemetry switch on path",
				topo: testTopology("v4-tree"),
				telemetrySwitches: []string{"r0", "r2"},
				pod: makePod("p1", depl1),
				podsDeploymentName: depl1,
				intdepl: intDepl(1, 1),
				queuedPods: QueuedPods{PerDeploymentCounts: map[string]int{depl1: 0, depl2: 0}},
				scheduledNodes: schedule(depl2, "w4"),
				nodesToScore: []string{"w1"},
				expectedScores: []int{0},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				testConfig := TelemetrySchedulingEngineConfig{ImmediateWeight: 1, FutureWeight: 1}
				sched := NewTelemetrySchedulingEngine(testConfig)
				objs := []runtime.Object{tt.topo, tt.pod, tt.intdepl}
				objs = append(objs, makeTestIncSwitchesForTopo(tt.topo, tt.telemetrySwitches)...)
				client := newFakeClient(t, objs...)
				topoEngine := NewTopologyEngine(client)
				ctx := context.Background()
				net, err := topoEngine.PrepareForPodScheduling(ctx, tt.pod)
				if err != nil {
					t.Fatal(err)
				}
				sched.PrepareForPodScheduling(net, tt.intdepl, tt.scheduledNodes)
				for i := range tt.nodesToScore {
					nodeName := tt.nodesToScore[i]
					score := sched.ComputeNodeSchedulingScore(net, nodeName, tt.intdepl, tt.pod,
						tt.podsDeploymentName, tt.scheduledNodes, tt.queuedPods)
					if score != tt.expectedScores[i] {
						t.Errorf("Invalid score for node %s, expected %d got %d", nodeName, tt.expectedScores[i], score)
					}
				}
			})
		}
	})
}

type edge struct {
	u string 
	v string
}

type switchPort struct {
	switchName string
	peerName string
	bidir bool
}

type vertex struct {
	name string
	deviceType shimv1alpha.DeviceType
}

func makeTestTopology(vertices []vertex, edges []edge) *shimv1alpha.Topology {
	graph := []shimv1alpha.NetworkDevice{}
	vertexConnections := map[string][]string{}
	for _, v := range vertices {
		vertexConnections[v.name] = []string{}
	}
	for _, e := range edges {
		s := vertexConnections[e.u]
		vertexConnections[e.u] = append(s, e.v)
		s = vertexConnections[e.v]
		vertexConnections[e.v] = append(s, e.u)
	}
	for _, v := range vertices {
		links := []shimv1alpha.Link{}
		for _, neigh := range vertexConnections[v.name] {
			links = append(links, shimv1alpha.Link{
				PeerName: neigh,
			})
		}
		dev := shimv1alpha.NetworkDevice{
			Name: v.name,
			DeviceType: v.deviceType,
			Links: links,
		}
		graph = append(graph, dev)
	}
	return &shimv1alpha.Topology{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: shimv1alpha.TopologySpec{
			Graph: graph,
		},
	}
}

func makeTestP4Program(name string, arch string, implementsTelemetry bool) *shimv1alpha.P4Program {
	implementedInterfaces := []string{}
	if implementsTelemetry {
		implementedInterfaces = append(implementedInterfaces, TELEMETRY_INTERFACE)
	}
	return &shimv1alpha.P4Program{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: shimv1alpha.P4ProgramSpec{
			Artifacts: []shimv1alpha.ProgramArtifacts{
				{Arch: arch},
			},
			ImplementedInterfaces: implementedInterfaces,
		},
	}
}

func makeTestIncSwitch(name string, arch string, programName string) *shimv1alpha.IncSwitch {
	return &shimv1alpha.IncSwitch{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},		
		Spec: shimv1alpha.IncSwitchSpec{
			Arch: arch,
			ProgramName: programName,
		},
	}
}

type deploymentInfo struct {
	name string
	replicas int32
	podLabel string
}

func makeTestIintDepl(name string, namespace string, deplyoments []deploymentInfo) *intv1alpha.InternalInNetworkTelemetryDeployment {
	templates := []intv1alpha.NamedDeploymentSpec{}
	for _, depl := range deplyoments {
		templates = append(templates, intv1alpha.NamedDeploymentSpec{
			Name: depl.name,
			Template: appsv1.DeploymentSpec{
				Replicas: &depl.replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"foo": depl.podLabel,
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"foo": depl.podLabel,
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name: "foo",
								Image: "foo",
							},
						},
					},
				},
			},
		})
	}
	return &intv1alpha.InternalInNetworkTelemetryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Namespace: namespace,
		},
		Spec: intv1alpha.InternalInNetworkTelemetryDeploymentSpec{
			RequiredProgram: "foo",
			DeploymentTemplates: templates,
		},
	}
}

func newFakeClient(t *testing.T, objs ...runtime.Object) client.Client {
	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := intv1alpha.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := shimv1alpha.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()
}

func withExtraEdges(topo *shimv1alpha.Topology, edges ...edge) *shimv1alpha.Topology {
	t := topo.DeepCopy()
	for _, e := range edges {
		for i, v := range t.Spec.Graph {
			if v.Name == e.u {
				t.Spec.Graph[i].Links = append(t.Spec.Graph[i].Links, shimv1alpha.Link{PeerName: e.v})
			} else if v.Name == e.v {
				t.Spec.Graph[i].Links = append(t.Spec.Graph[i].Links, shimv1alpha.Link{PeerName: e.u})
			}
		}
	}
	return t
}

func schedule (what string, where ...string) []ScheduledNode {
	res := []ScheduledNode{}
	for _, node := range where {
		res = append(res, ScheduledNode{node, []string{what}})
	}
	return res
}

func makeTestIncSwitchesForTopo(topo *shimv1alpha.Topology, telemetrySwitches []string) []runtime.Object {
	telemetryProgram := makeTestP4Program("telemetry", "bmv2", true)
	nonTelemetryProgram := makeTestP4Program("foo", "bmv2", false)
	res := makeTestIncSwitchesForTopoWithCustomPrograms(topo, nonTelemetryProgram, telemetryProgram, telemetrySwitches)
	res = append(res, telemetryProgram)
	res = append(res, nonTelemetryProgram)
	return res
}

func makeTestIncSwitchesForTopoWithCustomPrograms(
	topo *shimv1alpha.Topology,
	defaultProgram *shimv1alpha.P4Program, 
	telemetryProgram *shimv1alpha.P4Program,
	telemetrySwiches []string,
) []runtime.Object {
	res := []runtime.Object{}
	for _, dev := range topo.Spec.Graph {
		if dev.DeviceType == shimv1alpha.INC_SWITCH {
			var incSwitch *shimv1alpha.IncSwitch
			if slices.Index(telemetrySwiches, dev.Name) == -1 {
				incSwitch = makeTestIncSwitch(dev.Name, defaultProgram.Spec.Artifacts[0].Arch, defaultProgram.Name)
			} else {
				incSwitch = makeTestIncSwitch(dev.Name, telemetryProgram.Spec.Artifacts[0].Arch, telemetryProgram.Name)
			}
			res = append(res, incSwitch)
		}
	}
	return res
}

func testTopology(variant string) *shimv1alpha.Topology {
	switch variant {
	case "shallow":
		/*
			external
			   |
			   r0	
			___|___
			|      | 
			r1     r2
		  __|__    |
		  |   |	   n2	
	      n0  n1
		*/
		return makeTestTopology(
			[]vertex{
				{"external", shimv1alpha.EXTERNAL},
				{"r0", shimv1alpha.INC_SWITCH},
				{"r1", shimv1alpha.INC_SWITCH},
				{"r2", shimv1alpha.INC_SWITCH},
				{"n0", shimv1alpha.NODE},
				{"n1", shimv1alpha.NODE},
				{"n2", shimv1alpha.NODE},
			},
			[]edge{
				{"external", "r0"},
				{"r0", "r1"},
				{"r0", "r2"},
				{"r1", "n0"},
				{"r1", "n1"},
				{"r2", "n2"},
			},
		)
	case "v4-tree":
		// matches kinda-sdn v4 topo, but worker node w5 is used in place of control plane node c1
		return makeTestTopology(
			[]vertex{
				{"external", shimv1alpha.EXTERNAL},
				{"r0", shimv1alpha.INC_SWITCH},
				{"r1", shimv1alpha.INC_SWITCH},
				{"r2", shimv1alpha.INC_SWITCH},
				{"r3", shimv1alpha.INC_SWITCH},
				{"r4", shimv1alpha.INC_SWITCH},
				{"r5", shimv1alpha.INC_SWITCH},
				{"r6", shimv1alpha.INC_SWITCH},
				{"r7", shimv1alpha.INC_SWITCH},
				{"w1", shimv1alpha.NODE},
				{"w2", shimv1alpha.NODE},
				{"w3", shimv1alpha.NODE},
				{"w4", shimv1alpha.NODE},
				{"w5", shimv1alpha.NODE},
			},
			[]edge{
				{"external", "r0"},
				{"r0", "r1"},
				{"r0", "r2"},
				{"r0", "r3"},
				{"r0", "r4"},
				{"r1", "r5"},
				{"r2", "r6"},
				{"r3", "r7"},
				{"r5", "w1"},
				{"r5", "w2"},
				{"r6", "w3"},
				{"r7", "w4"},
				{"r4", "w5"},
			},
		)
	default:
		panic("no such topology")
	}
}
