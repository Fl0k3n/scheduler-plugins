package integration

import (
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/scheduler-plugins/pkg/internaltelemetry/core"
	shimv1alpha "sigs.k8s.io/scheduler-plugins/pkg/shimv1alpha"
)

const (
	RESULTS_DIR = "/home/flok3n/develop/k8s_inc_analysis/data"
)

type scheduligTimeMeasurement struct {
	cluster *cluster
	numPods int
	startTime time.Time
	stopTime time.Time
	telemetry bool
}

func (s scheduligTimeMeasurement) CsvHeader() string {
	return "sched_type,cluster_key,num_nodes,num_inc_switches,num_graph_vertices,num_graph_edges,num_pods,total_scheduling_time_ms\n"
}

func (s scheduligTimeMeasurement) Csv() string {
	schedType := "default"
	if s.telemetry {
		schedType = "telemetry"
	}
	return fmt.Sprintf("%s,%s,%d,%d,%d,%d,%d,%d\n",
		schedType, makeClusterDescriptor(s.cluster), len(s.cluster.nodes), len(s.cluster.incSwitches),
		s.cluster.totalVertices, s.cluster.totalEdges, s.numPods, s.stopTime.Sub(s.startTime).Milliseconds())
}

type MeasurementHelper struct {
	t *testing.T
	experimentDir string
	timeFile *os.File

	scheduligTimeMeasurement scheduligTimeMeasurement	
}

func newMeasurementHelper(t *testing.T, experimentName string) *MeasurementHelper {
	return &MeasurementHelper{
		t: t,
		experimentDir: fmt.Sprintf("%s/%s", RESULTS_DIR, experimentName),
		timeFile: nil,
	}	
}

func (m *MeasurementHelper) Init() {
	if err := os.Mkdir(m.experimentDir, 0774); err != nil {
		m.t.Fatalf("Failed to create directory for experiment at %s; %v", m.experimentDir, err)
	}
	f, err := os.Create(fmt.Sprintf("%s/time_measurements.csv", m.experimentDir))
	if err != nil {
		m.t.Fatalf("Failed to create time measurements file: %v", err)
	}
	m.timeFile = f
	if _, err = m.timeFile.WriteString(scheduligTimeMeasurement{}.CsvHeader()); err != nil {
		m.t.Fatalf("Failed to write header")
	}
}

func (m *MeasurementHelper) Close() {
	m.timeFile.Close()
}

func (m *MeasurementHelper) StartMeasurement(cluster *cluster, numPods int, telemetry bool) {
	m.scheduligTimeMeasurement = scheduligTimeMeasurement{
		cluster: cluster,
		numPods: numPods,
		startTime: time.Now(),
		telemetry: telemetry,
	}
}

func (m *MeasurementHelper) StopMeasurement() {
	m.scheduligTimeMeasurement.stopTime = time.Now()
	if _, err := m.timeFile.WriteString(m.scheduligTimeMeasurement.Csv()); err != nil {
		m.t.Fatalf("Failed to record measurement: %v", err)
	}
}

func (m *MeasurementHelper) SetupMemoryProfiling() {
	core.TakeMemorySnapshot = true
	core.MemorySnapshotFilePath = fmt.Sprintf("%s/mem.prof", m.experimentDir)
}

func (m *MeasurementHelper) SetupTimeProfiling() {
	f, err := os.Create(fmt.Sprintf("%s/cpu.prof", m.experimentDir))
	if err != nil {
		m.t.Fatalf("Failed to create CPU profiling file: %v", err)
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		m.t.Fatalf("Failed to start CPU profiling: %v", err)
	}
}

func (m *MeasurementHelper) StopTimeProfiling() {
	pprof.StopCPUProfile()
}

func makeClusterDescriptor(cluster *cluster) string {
	return fmt.Sprintf("tree_%d_%d_%d_%d_%f", 
			cluster.numRackNodes, cluster.numRacks,
			cluster.numFakeRackExtenders, cluster.numl0ToL1Connections,
			cluster.telemetrySwitchFraction,
	)
}


type cluster struct {
	numRackNodes int
	numRacks int
	numFakeRackExtenders int
	numl0ToL1Connections int
	telemetrySwitchFraction float32

	nodes []*v1.Node
	incSwitches []*shimv1alpha.IncSwitch
	topology *shimv1alpha.Topology
	programs []*shimv1alpha.P4Program
	totalVertices int
	totalEdges int
}

func (c *cluster) DeepCopy() *cluster {
	nodesDeepCopy := make([]*v1.Node, len(c.nodes))
	for i := range c.nodes {
		nodesDeepCopy[i] = c.nodes[i].DeepCopy()
	}
	incSwitchesDeepCopy := make([]*shimv1alpha.IncSwitch, len(c.incSwitches))
	for i := range c.incSwitches {
		incSwitchesDeepCopy[i] = c.incSwitches[i].DeepCopy()
	}
	programsDeepCopy := make([]*shimv1alpha.P4Program, len(c.programs))
	for i := range c.programs {
		programsDeepCopy[i] = c.programs[i].DeepCopy()
	}
	return &cluster{
		numRackNodes: c.numRackNodes,
		numRacks: c.numRacks,
		numFakeRackExtenders: c.numFakeRackExtenders,
		numl0ToL1Connections: c.numl0ToL1Connections,
		telemetrySwitchFraction: c.telemetrySwitchFraction,
		nodes: nodesDeepCopy,
		incSwitches: incSwitchesDeepCopy,
		programs: programsDeepCopy,
		topology: c.topology.DeepCopy(),
		totalVertices: c.totalVertices,
		totalEdges: c.totalEdges,
	}
}

type edge struct {
	u string 
	v string
}

type vertex struct {
	name string
	deviceType shimv1alpha.DeviceType
}