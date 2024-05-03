package integration

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/scheduler"
	schedapi "k8s.io/kubernetes/pkg/scheduler/apis/config"
	fwkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/scheduler-plugins/pkg/internaltelemetry"
	"sigs.k8s.io/scheduler-plugins/pkg/internaltelemetry/core"
	intv1alpha "sigs.k8s.io/scheduler-plugins/pkg/intv1alpha"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
	"sigs.k8s.io/scheduler-plugins/test/util"
)


func TestStuff(t *testing.T) {
	testCtx := &testContext{}
	testCtx.Ctx, testCtx.CancelFn = context.WithCancel(context.Background())

	cs := clientset.NewForConfigOrDie(globalKubeConfig)

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(intv1alpha.AddToScheme(scheme))
	utilruntime.Must(shimv1alpha.AddToScheme(scheme))
	client, err := client.New(globalKubeConfig, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}
	testCtx.ClientSet = cs
	testCtx.KubeConfig = globalKubeConfig

	if err := wait.Poll(200*time.Millisecond, 3*time.Second, func() (done bool, err error) {
		groupList, _, err := cs.ServerGroupsAndResources()
		if err != nil {
			return false, nil
		}
		for _, group := range groupList {
			if group.Name == "inc.kntp.com" {
				t.Log("The CRD is ready to serve")
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		t.Fatalf("Timed out waiting for CRD to be ready: %v", err)
	}

	cfg, err := util.NewDefaultSchedulerComponentConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Profiles[0].Plugins.PreScore.Enabled = append(cfg.Profiles[0].Plugins.PreScore.Enabled, schedapi.Plugin{Name: internaltelemetry.Name})
	cfg.Profiles[0].Plugins.Score = schedapi.PluginSet{
		Enabled:  []schedapi.Plugin{{Name: internaltelemetry.Name}},
		Disabled: []schedapi.Plugin{{Name: "*"}},
	}
	cfg.Profiles[0].Plugins.Reserve.Enabled = append(cfg.Profiles[0].Plugins.Reserve.Enabled, schedapi.Plugin{Name: internaltelemetry.Name})
	cfg.Profiles[0].Plugins.PostBind.Enabled = append(cfg.Profiles[0].Plugins.PostBind.Enabled, schedapi.Plugin{Name: internaltelemetry.Name})


	testCtx = initTestSchedulerWithOptions(
		t,
		testCtx,
		scheduler.WithProfiles(cfg.Profiles...),
		scheduler.WithFrameworkOutOfTreeRegistry(fwkruntime.Registry{internaltelemetry.Name: internaltelemetry.New}),
	)

	syncInformerFactory(testCtx)
	go testCtx.Scheduler.Run(testCtx.Ctx)
	t.Log("Init scheduler success")
	defer cleanupTest(t, testCtx)

	cluster := makeCluster(2, 4, 1, 2, 0.4)
	for _, node := range cluster.nodes {
		_, err := cs.CoreV1().Nodes().Create(testCtx.Ctx, node, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create Node %q: %v", node.Name, err)
		}
	}
	
	if err := createSdnClusterResources(testCtx.Ctx, client, cluster); err != nil {
		t.Fatalf("Failed to create cluster resources: %v", err)
	}

	depl1 := makeDeployment("depl1", "intdepl", 1)
	depl2 := makeDeployment("depl2", "intdepl", 1)
	intdepl := makeInternalTelemetryDeployment("intdepl", depl1, depl2)
	p1 := makeIntDeplPod("p1", "intdepl", "depl1")
	if err := createIntDeplResources(testCtx.Ctx, client, intdepl, depl1, depl2); err != nil {
		t.Fatalf("Failed to create telemetry resources: %v", err)
	}
	if err := client.Create(testCtx.Ctx, p1); err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}
	if err := wait.Poll(1*time.Second, 20*time.Second, func() (bool, error) {
		return podScheduled(cs, "", p1.Name), nil

	}); err != nil {
		t.Errorf("wanted pod %q to be scheduled, error: %v", p1.Name, err)
	}
}

func createIntDeplResources(
	ctx context.Context,
	c client.Client,
	intdepl *intv1alpha.InternalInNetworkTelemetryDeployment,
	depl1 *appsv1.Deployment,
	depl2 *appsv1.Deployment,
) error {
	if err := c.Create(ctx, intdepl); err != nil {
		return err
	}
	if err := c.Create(ctx, depl1); err != nil {
		return err
	}
	if err := c.Create(ctx, depl2); err != nil {
		return err
	}
	return nil 
}

func createIncSwitches(ctx context.Context, c client.Client, incSwitches []*shimv1alpha.IncSwitch) error {
	for _, isw := range incSwitches {
		if err := c.Create(ctx, isw); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func createP4Programs(ctx context.Context, c client.Client, p4Programs []*shimv1alpha.P4Program) error {
	for _, prog := range p4Programs {
		if err := c.Create(ctx, prog); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func createTopology(ctx context.Context, c client.Client, topo *shimv1alpha.Topology) error { 
	if err := c.Create(ctx, topo); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func createSdnClusterResources(ctx context.Context, c client.Client, cluster *cluster) error {
	err := createTopology(ctx, c, cluster.topology)
	if err == nil {
		err = createP4Programs(ctx, c, cluster.programs)
	}
	if err == nil {
		err = createIncSwitches(ctx, c, cluster.incSwitches)
	}
	return err
}

type cluster struct {
	nodes []*v1.Node
	incSwitches []*shimv1alpha.IncSwitch
	topology *shimv1alpha.Topology
	programs []*shimv1alpha.P4Program
}

func makeCluster(
	numRackNodes int,
	numRacks int,
	numFakeRackExtenders int,
	numl0ToL1Connections int,
	telemetrySwitchFraction float32,
) *cluster {
	rng := rand.New(rand.NewSource(42))
	telemetryProgramName := "telemetry"
	intProgram := makeTelemetryProgram(telemetryProgramName)
	
	c := &cluster{
		nodes: []*v1.Node{},
		incSwitches: []*shimv1alpha.IncSwitch{},
		topology: nil,
		programs: []*shimv1alpha.P4Program{intProgram},
	}
	racks := [][]*v1.Node{}
	vertices := []vertex{}
	edges := []edge{}

	// create racks of nodes
	for i := 0; i < numRacks; i++ {
		rack := []*v1.Node{}
		for j := 0; j < numRackNodes; j++ {
			name := fmt.Sprintf("n-%d-%d", i, j)
			node := makeNode(name)
			rack = append(rack, node)
			c.nodes = append(c.nodes, node)
			vertices = append(vertices, vertex{
				name: name,
				deviceType: shimv1alpha.NODE,
			})
		}
		racks = append(racks, rack)
	}

	// create network devices to connect these nodes, shape it in a tree
	// fakeExtenders increases number of TOR like switches (just path of switches instead of 1)
	// then after number of fakeExtenders we do normal branching, this is just to increase network size
	rackNetDevices := [][]vertex{}
	for i := 0; i < numRacks; i++ {
		rack := []vertex{}
		for j := 0; j < numFakeRackExtenders + 1; j++ {
			devName := fmt.Sprintf("s-%d-%d", j, i) // s-lvl-id_on_lvl
			v, sw := makeNetDevice(devName, rng, telemetrySwitchFraction, telemetryProgramName)
			rack = append(rack, v)
			vertices = append(vertices, v)
			if sw != nil {
				c.incSwitches = append(c.incSwitches, sw)
			}
		}
		rackNetDevices = append(rackNetDevices, rack)
	}

	// setup connections for racks
	for i := 0; i < numRacks; i++ {
		torDev := rackNetDevices[i][0]
		for j := 0; j < numRackNodes; j++ {
			edges = append(edges, edge{torDev.name, racks[i][j].Name})
		}
		for j := 1; j < numFakeRackExtenders + 1; j++ {
			edges = append(edges, edge{rackNetDevices[i][j-1].name, rackNetDevices[i][j].name})
		}
	}

	// build higher level tree topology
	perDeviceInterLevelConnections := numl0ToL1Connections
	previousLevelDevices := []vertex{}
	lvl := numFakeRackExtenders + 1
	for i := 0; i < numRacks; i++ {
		previousLevelDevices = append(previousLevelDevices, rackNetDevices[i][numFakeRackExtenders])
	}
	for len(previousLevelDevices) > 1 {
		curLevelDevices := []vertex{}
		for i := 0; i < len(previousLevelDevices); i += perDeviceInterLevelConnections {
			numDevicesToConnectWithCur := perDeviceInterLevelConnections
			if numDevicesToConnectWithCur > len(previousLevelDevices) - i {
				numDevicesToConnectWithCur = len(previousLevelDevices) - i
			}
			name := fmt.Sprintf("s-%d-%d", lvl, i)
			v, sw := makeNetDevice(name, rng, telemetrySwitchFraction, telemetryProgramName)
			curLevelDevices = append(curLevelDevices, v)
			vertices = append(vertices, v)
			if sw != nil {
				c.incSwitches = append(c.incSwitches, sw)
			}
			for j := 0; j < numDevicesToConnectWithCur; j++ {
				edges = append(edges, edge{name, previousLevelDevices[i+j].name})
			}
		}
		perDeviceInterLevelConnections = 2
		previousLevelDevices = curLevelDevices
		lvl++
	}
	c.topology = makeTestTopology(vertices, edges)
	return c
}

// incSwitch is nil if rng returns > telemetryProbability
func makeNetDevice(
	name string,
	rng *rand.Rand,
	telemetryProbability float32,
	telemetryProgram string,
) (vertex, *shimv1alpha.IncSwitch) {
	isINTswitch := rng.Float32() < telemetryProbability
	if isINTswitch {
		sw := makeIncSwitch(name, telemetryProgram)
		return vertex{name: name, deviceType: shimv1alpha.INC_SWITCH}, sw
	} 
	return vertex{name: name, deviceType: shimv1alpha.NET}, nil
}

func makeNode(name string) *v1.Node {
	return st.MakeNode().Name(name).Obj()
}

type edge struct {
	u string 
	v string
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

func makeIncSwitch(name string, programName string) *shimv1alpha.IncSwitch {
	return &shimv1alpha.IncSwitch{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: shimv1alpha.IncSwitchSpec{
			Arch: "bmv2",
			ProgramName: programName,
		},
	}
}

func makeTelemetryProgram(name string) *shimv1alpha.P4Program {
	return &shimv1alpha.P4Program{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: shimv1alpha.P4ProgramSpec{
			Artifacts: []shimv1alpha.ProgramArtifacts{},
			ImplementedInterfaces: []string{
				core.TELEMETRY_INTERFACE,
			},
		},
	}
}

func makeInternalTelemetryDeployment(
	name string,
	depl1 *appsv1.Deployment,
	depl2 *appsv1.Deployment,
) *intv1alpha.InternalInNetworkTelemetryDeployment {
	return &intv1alpha.InternalInNetworkTelemetryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: intv1alpha.InternalInNetworkTelemetryDeploymentSpec{
			CollectorRef: v1.LocalObjectReference{
				Name: "collector",
			},
			CollectionId: "collection-id",
			DeploymentTemplates: []intv1alpha.NamedDeploymentSpec{
				{
					Name: depl1.Name,
					Template: depl1.Spec,
				},
				{
					Name: depl2.Name,
					Template: depl2.Spec,
				},
			},
		},
	}
}

func intPodLabels(
	intdeplName string,
	deplName string,
) map[string]string {
	return map[string]string{
		"inc.kntp.com/owned-by-iintdepl": intdeplName,
		"inc.kntp.com/part-of-deployment": deplName,
	}
}

func makeIntDeplPod(
	name string,
	intdeplName string,
	deplName string,
) *v1.Pod {
	return st.MakePod().Name(name).Namespace("").
			Labels(intPodLabels(intdeplName, deplName)).
			SchedulerName(internaltelemetry.Name).
			Obj()
}

func makeDeployment(
	name string,
	intdeplName string,
	replicas int,
) *appsv1.Deployment {
	labels := intPodLabels(intdeplName, name)
	replicas32 := int32(replicas)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: &replicas32,
			Template: v1.PodTemplateSpec{
				Spec: st.MakePod().Spec,
			},
		},
	}	
}
