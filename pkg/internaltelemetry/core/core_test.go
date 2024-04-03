package core

import (
	"context"
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
		telemetrySwitches []*shimv1alpha.IncSwitch
		wantError              bool
	}{
		{
			name: "preparation fails when there is no topology",
			pod: st.MakePod().Name("p1").UID("p1").Namespace("ns1").Obj(),
			topology: nil,
			telemetrySwitches: []*shimv1alpha.IncSwitch{},
			wantError: true,
		},
		{
			name: "preparation succeeds for tree topology",
			pod: st.MakePod().Name("p1").UID("p1").Namespace("ns1").Obj(),
			topology: testTopology("shallow"),
			telemetrySwitches: makeTestIncSwitches("r0", "r1", "r2"),
			wantError: false,
		},
		{
			name: "preparation fails for not-tree topology",
			pod: st.MakePod().Name("p1").UID("p1").Namespace("ns1").Obj(),
			topology: withExtraEdges(testTopology("shallow"), edge{"r1", "r2"}),
			telemetrySwitches: makeTestIncSwitches("r0", "r1", "r2"),
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
			telemetrySwitches: []*shimv1alpha.IncSwitch{},
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
			for _, sw := range tt.telemetrySwitches {
				objs = append(objs, sw)
			}
			client := newFakeClient(t, objs...)

			topoEngine := NewTopologyEngine(client)
			ctx := context.Background()
			_, err := topoEngine.PrepareForScheduling(ctx, tt.pod)
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

func TestScoringEngine(t *testing.T) {
	t.Run("scheduler's network representation correctly tracks utilized telemetry ports", func(t *testing.T) {
		depl1Pod := st.MakePod().Name("p1").UID("p1").Namespace("ns1").Labels(map[string]string{
			INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL: "intdepl",
			INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL: "depl1",
		}).Obj()
		depl1 := deploymentInfo{name: "depl1", replicas: -1, podLabel: "foo1"}
		depl2 := deploymentInfo{name: "depl2", replicas: -1, podLabel: "foo2"}
		intDeplTemplate := makeTestIintDepl("intdepl", "ns1", []deploymentInfo{depl1, depl2})
		intDepl := func(first int32, second int32) *intv1alpha.InternalInNetworkTelemetryDeployment {
			res := intDeplTemplate.DeepCopy()
			res.Spec.DeploymentTemplates[0].Template.Replicas = &first
			res.Spec.DeploymentTemplates[1].Template.Replicas = &second
			return res
		}
		allV4IncSwitches := func() []*shimv1alpha.IncSwitch {
			return makeTestIncSwitches("r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7")
		}
		schedule := func (what string, where ...string) []ScheduledNode {
			res := []ScheduledNode{}
			for _, node := range where {
				res = append(res, ScheduledNode{node, []string{what}})
			}
			return res
		}
		merge := func (sn1 []ScheduledNode, sn2 []ScheduledNode) []ScheduledNode {
			res := []ScheduledNode{}
			joined := map[string][]string{}
			for _, sn := range append(sn1, sn2...) {
				if depls, ok := joined[sn.Name]; ok {
					joined[sn.Name] = append(depls, sn.ScheduledDeployments...)
				} else {
					joined[sn.Name] = sn.ScheduledDeployments
				}
			}
			for k, v := range joined {
				res = append(res, ScheduledNode{k, v})
			}
			return res
		}
		

		tests := []struct {
			name                string
			topo 				*shimv1alpha.Topology
			telemetrySwitches   []*shimv1alpha.IncSwitch
			pod 			    *v1.Pod
			intdepl 		    *intv1alpha.InternalInNetworkTelemetryDeployment
			scheduledNodes      []ScheduledNode
			expectedUsedPorts   []switchPort
		}{
			{
				name: "no ports are used when nothing was scheduled yet",
				topo: testTopology("v4-tree"),
				telemetrySwitches: allV4IncSwitches(),
				pod: depl1Pod,
				intdepl: intDepl(2, 2),
				scheduledNodes: []ScheduledNode{},
				expectedUsedPorts: []switchPort{},
			},
			{	
				name: "all ports on path between only path between 2 pods of different deployments are used",
				topo: testTopology("v4-tree"),
				telemetrySwitches: allV4IncSwitches(),
				pod: depl1Pod,
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
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sched := NewTelemetrySchedulingEngine(DefaultTelemetrySchedulingEngineConfig())
				objs := []runtime.Object{tt.topo, tt.pod, tt.intdepl}
				for _, sw := range tt.telemetrySwitches {
					objs = append(objs, sw)
				}
				client := newFakeClient(t, objs...)
				topoEngine := NewTopologyEngine(client)
				ctx := context.Background()
				net, err := topoEngine.PrepareForScheduling(ctx, tt.pod)
				if err != nil {
					t.Fatal(err)
				}
				sched.PrepareForScheduling(net, tt.intdepl.Name, tt.scheduledNodes)
				repr := sched.getCachedDeploymentsNetworkView(tt.intdepl.Name)

				shouldBeUsed := func(from string, to string) bool {
					for _, p := range tt.expectedUsedPorts {
						if p.switchName == from && p.peerName == to || (p.bidir && p.switchName == to && p.peerName == from) {
							return true
						}
					}
					return false
				}
				for _, v := range repr.Vertices {
					mustHaveNoTelemetryPorts := true
					if v.DeviceType == shimv1alpha.INC_SWITCH {
						for _, telemetrySw := range tt.telemetrySwitches {
							if telemetrySw.Name == v.Name {
								mustHaveNoTelemetryPorts = false
								break
							}
						}
					}
					if mustHaveNoTelemetryPorts {
						if len(v.Meta.AvailableTelemetryPorts) > 0 {
							t.Errorf("Expected %s to have no availabe ports, but got: %v", v.Name, v.Meta.AvailableTelemetryPorts)
						}
					} else {
						for _, neigh := range v.Neighbors() {
							is := !v.Meta.IsPortUnallocated(neigh.Name)
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

	t.Run("", func(t *testing.T) {
		tests := []struct {
			name                string
			topo 				*shimv1alpha.Topology
			telemetrySwitches   []*shimv1alpha.IncSwitch
			pod 			    *v1.Pod
			intdepl 		    *intv1alpha.InternalInNetworkTelemetryDeployment
			queuedPods 			QueuedPods
			scheduledNodes      []ScheduledNode
		}{
			{
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {

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

func makeTestIncSwitch(name string) *shimv1alpha.IncSwitch {
	return &shimv1alpha.IncSwitch{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},		
		Spec: shimv1alpha.IncSwitchSpec{
			Arch: "foo",
			ProgramName: "bar",
		},
	}
}

func makeTestIncSwitches(names ...string) []*shimv1alpha.IncSwitch {
	res := []*shimv1alpha.IncSwitch{}
	for _, name := range names {
		res = append(res, makeTestIncSwitch(name))
	}
	return res
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
		for _, v := range t.Spec.Graph {
			if v.Name == e.u {
				v.Links = append(v.Links, shimv1alpha.Link{PeerName: e.v})
			} else if v.Name == e.v {
				v.Links = append(v.Links, shimv1alpha.Link{PeerName: e.u})
			}
		}
	}
	return t
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
		// matches kinda-sdn v4 topo, but without control plane node
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
			},
		)
	default:
		panic("no such topology")
	}
}
