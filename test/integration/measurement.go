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
type SchedulerType string 

const (
	SCHED_DEFAULT 		   SchedulerType = "default"
	SCHED_NETWORK_OVERHEAD SchedulerType = "network-overhead"
	SCHED_TELEMETRY    	   SchedulerType = "telemetry"
)

type ClusterType string 

const (
	TREE ClusterType = "tree"
	FAT_TREE ClusterType = "fat-tree"
)

type scheduligTimeMeasurement struct {
	cluster *cluster
	numPods int
	startTime time.Time
	stopTime time.Time
	schedType SchedulerType
}

func (s scheduligTimeMeasurement) CsvHeader(clusterType ClusterType) string {
	if clusterType == TREE {
		return "sched_type,cluster_key,num_nodes,num_inc_switches,num_graph_vertices,num_graph_edges,num_pods,total_scheduling_time_ms\n"
	}
	return "sched_type,cluster_key,k,num_nodes,num_inc_switches,num_graph_vertices,num_graph_edges,num_pods,total_scheduling_time_ms\n"
}

func (s scheduligTimeMeasurement) Csv() string {
	if s.cluster.T == TREE {
		return fmt.Sprintf("%s,%s,%d,%d,%d,%d,%d,%d\n",
			string(s.schedType), makeClusterDescriptor(s.cluster), len(s.cluster.nodes), len(s.cluster.incSwitches),
			s.cluster.totalVertices, s.cluster.totalEdges, s.numPods, s.stopTime.Sub(s.startTime).Milliseconds(),
		)
	}
	return fmt.Sprintf("%s,%s,%d,%d,%d,%d,%d,%d,%d\n",
		string(s.schedType), makeClusterDescriptor(s.cluster), s.cluster.params.(*FatTreeClusterGenParams).k,
		len(s.cluster.nodes), len(s.cluster.incSwitches), s.cluster.totalVertices, s.cluster.totalEdges,
		s.numPods, s.stopTime.Sub(s.startTime).Milliseconds(),
	)
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

func (m *MeasurementHelper) Init(clusterType ClusterType) {
	if err := os.Mkdir(m.experimentDir, 0774); err != nil {
		m.t.Fatalf("Failed to create directory for experiment at %s; %v", m.experimentDir, err)
	}
	f, err := os.Create(fmt.Sprintf("%s/time_measurements.csv", m.experimentDir))
	if err != nil {
		m.t.Fatalf("Failed to create time measurements file: %v", err)
	}
	m.timeFile = f
	if _, err = m.timeFile.WriteString(scheduligTimeMeasurement{}.CsvHeader(clusterType)); err != nil {
		m.t.Fatalf("Failed to write header")
	}
}

func (m *MeasurementHelper) Close() {
	m.timeFile.Close()
}

func (m *MeasurementHelper) StartMeasurement(cluster *cluster, numPods int, schedType SchedulerType) {
	m.scheduligTimeMeasurement = scheduligTimeMeasurement{
		cluster: cluster,
		numPods: numPods,
		startTime: time.Now(),
		schedType: schedType,
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
	if cluster.T == TREE {
		params := cluster.params.(*TreeClusterGenParams)
		return fmt.Sprintf("tree_%d_%d_%d_%d_%f", 
				params.numRackNodes, params.numRacks,
				params.numFakeRackExtenders, params.numl0ToL1Connections,
				params.telemetrySwitchFraction,
		)
	}
	params := cluster.params.(*FatTreeClusterGenParams)
	return fmt.Sprintf("fat_tree_%d", params.k)
}

type ClusterGenParams interface {
	DeepCopy() ClusterGenParams
}

type TreeClusterGenParams struct {
	numRackNodes int
	numRacks int
	numFakeRackExtenders int
	numl0ToL1Connections int
	telemetrySwitchFraction float32
}

func (t *TreeClusterGenParams) DeepCopy() ClusterGenParams {
	return &TreeClusterGenParams{
		numRackNodes: t.numRackNodes,
		numRacks: t.numRacks,
		numFakeRackExtenders: t.numFakeRackExtenders,
		numl0ToL1Connections: t.numl0ToL1Connections,
		telemetrySwitchFraction: t.telemetrySwitchFraction,
	}
}

type FatTreeClusterGenParams struct {
	k int
	telemetrySwitchFraction float32
}

func (t *FatTreeClusterGenParams) DeepCopy() ClusterGenParams {
	return &FatTreeClusterGenParams{
		k: t.k,
		telemetrySwitchFraction: t.telemetrySwitchFraction,
	}
}

type cluster struct {
	params ClusterGenParams
	T ClusterType

	nodes []*v1.Node
	incSwitches []*shimv1alpha.IncSwitch
	topology *shimv1alpha.Topology
	programs []*shimv1alpha.P4Program
	totalVertices int
	totalEdges int

	zones []string
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
		params: c.params.DeepCopy(),
		nodes: nodesDeepCopy,
		incSwitches: incSwitchesDeepCopy,
		programs: programsDeepCopy,
		topology: c.topology.DeepCopy(),
		totalVertices: c.totalVertices,
		totalEdges: c.totalEdges,
		zones: c.zones[:],
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
