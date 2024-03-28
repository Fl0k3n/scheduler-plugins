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

type EndpointsEntryStatus string

const (
	EP_PENDING EndpointsEntryStatus = "pending"
	EP_READY EndpointsEntryStatus = "ready"
	EP_FAILED EndpointsEntryStatus = "failed"
	EP_TERMINATING EndpointsEntryStatus = "terminating"
)

type ExternalInNetworkTelemetryEndpointsEntry struct {
	PodReference v1.ObjectReference `json:"podReference"`
	EntryStatus EndpointsEntryStatus `json:"entryStatus"`
	NodeName string `json:"nodeName"`
}

// ExternalInNetworkTelemetryEndpointsSpec defines the desired state of ExternalInNetworkTelemetryEndpoints
type ExternalInNetworkTelemetryEndpointsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Entries []ExternalInNetworkTelemetryEndpointsEntry `json:"entries"`
	CollectorNodeName string `json:"collectorNodeName"`
	ProgramName string `json:"programName"`
	IngressInfo IngressInfo `json:"ingressInfo"`
	MonitoringPolicy MonitoringPolicy `json:"monitoringPolicy"`
}

// ExternalInNetworkTelemetryEndpointsStatus defines the observed state of ExternalInNetworkTelemetryEndpoints
type ExternalInNetworkTelemetryEndpointsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ExternalIngressInfo *IngressInfo `json:"externalIngressInfo,omitempty"`
	ExternalMonitoringPolicy *MonitoringPolicy `json:"externalMonitoringPolicy,omitempty"`
	InternalIngressInfo *IngressInfo `json:"internalIngressInfo,omitempty"`
	InternalMonitoringPolicy *MonitoringPolicy `json:"internalMonitoringPolicy,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ExternalInNetworkTelemetryEndpoints is the Schema for the externalinnetworktelemetryendpoints API
type ExternalInNetworkTelemetryEndpoints struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalInNetworkTelemetryEndpointsSpec   `json:"spec,omitempty"`
	Status ExternalInNetworkTelemetryEndpointsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExternalInNetworkTelemetryEndpointsList contains a list of ExternalInNetworkTelemetryEndpoints
type ExternalInNetworkTelemetryEndpointsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalInNetworkTelemetryEndpoints `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalInNetworkTelemetryEndpoints{}, &ExternalInNetworkTelemetryEndpointsList{})
}
