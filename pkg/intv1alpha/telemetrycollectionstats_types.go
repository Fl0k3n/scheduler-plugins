/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TelemetryCollectionStatsSpec defines the desired state of TelemetryCollectionStats
type TelemetryCollectionStatsSpec struct {
	CollectorRef v1.LocalObjectReference `json:"collectorRef"`
	CollectionId string `json:"collectionId"`
	RefreshPeriodMillis int `json:"refreshPeriodMillis"`
}

type PortMetrics struct {
	TargetDeviceName string `json:"toDevice"`
	NumberPackets int `json:"packets"`
	AverageLatencyMicroS int `json:"averageLatencyMicroS"`
	AverageQueueFillState int `json:"averageQueueFillState"`
	OnePercentileSlowestLatencyMicroS int `json:"onePercentileSlowestLatencyMicroS"`
	OnePercentileLargestQueueFillState int `json:"onePercentileLargestQueueFillState"`
}

type SwitchMetrics struct {
	DeviceName string `json:"deviceName"`
	PortMetrics []PortMetrics `json:"portMetrics"`
}

type Metrics struct {
	NumberCollectedReports int `json:"collectedReports"`
	AveragePathLatencyMicroS int `json:"averagePathLatencyMicroS"`
	OnePercentileSlowestPathLatencyMicroS int `json:"onePercentileSlowestPathLatencyMicroS"`
	DeviceMetrics []SwitchMetrics `json:"deviceMetrics"`
}

type MetricsSummary struct {
	TotalReports int `json:"totalReports"`
	TimeWindowsSeconds int `json:"timeWindowsSeconds"`
    WindowMetrics Metrics `json:"windowMetrics"`
}

// TelemetryCollectionStatsStatus defines the observed state of TelemetryCollectionStats
type TelemetryCollectionStatsStatus struct {
	MetricsSummary *MetricsSummary `json:"summary,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TelemetryCollectionStats is the Schema for the telemetrycollectionstats API
type TelemetryCollectionStats struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TelemetryCollectionStatsSpec   `json:"spec,omitempty"`
	Status TelemetryCollectionStatsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TelemetryCollectionStatsList contains a list of TelemetryCollectionStats
type TelemetryCollectionStatsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TelemetryCollectionStats `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TelemetryCollectionStats{}, &TelemetryCollectionStatsList{})
}
