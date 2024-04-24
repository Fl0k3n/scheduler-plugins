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

type InternalInNetworkTelemetryEndpointsEntry struct {
	PodReference v1.ObjectReference `json:"podReference"`
	EntryStatus EndpointsEntryStatus `json:"entryStatus"`
	NodeName string `json:"nodeName"`
	PodIp string `json:"podIp"`
}

type DeploymentEndpoints struct {
	DeploymentName string `json:"deploymentName"`
	Entries []InternalInNetworkTelemetryEndpointsEntry `json:"entries"`
}

// InternalInNetworkTelemetryEndpointsSpec defines the desired state of InternalInNetworkTelemetryEndpoints
type InternalInNetworkTelemetryEndpointsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of InternalInNetworkTelemetryEndpoints. Edit internalinnetworktelemetryendpoints_types.go to remove/update
	DeploymentEndpoints []DeploymentEndpoints `json:"deploymentEndpoints"`
	CollectorRef v1.LocalObjectReference `json:"collectorRef"`
}

// InternalInNetworkTelemetryEndpointsStatus defines the observed state of InternalInNetworkTelemetryEndpoints
type InternalInNetworkTelemetryEndpointsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InternalInNetworkTelemetryEndpoints is the Schema for the internalinnetworktelemetryendpoints API
type InternalInNetworkTelemetryEndpoints struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InternalInNetworkTelemetryEndpointsSpec   `json:"spec,omitempty"`
	Status InternalInNetworkTelemetryEndpointsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InternalInNetworkTelemetryEndpointsList contains a list of InternalInNetworkTelemetryEndpoints
type InternalInNetworkTelemetryEndpointsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InternalInNetworkTelemetryEndpoints `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InternalInNetworkTelemetryEndpoints{}, &InternalInNetworkTelemetryEndpointsList{})
}
