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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type IngressType string

const (
	NODE_PORT IngressType = "NodePort"
)

type MonitoringPolicy string

const (
	MONITOR_EXTERNAL_TO_PODS MonitoringPolicy = "external-to-pods"
	MONITOR_PODS_TO_EXTERNAL MonitoringPolicy = "pods-to-external"
	MONITOR_ALL				 MonitoringPolicy = "all"
)

type IngressInfo struct {
	IngressType IngressType `json:"type"`

	// these fields should be included in a dynamic subtype based on IngressType,
	// but since we probably won't support anything except simple NodePort let's keep it this way
	NodeNames []string `json:"nodeNames"` 
	NodePortServiceName string `json:"serviceName"`
	NodePortServiceNamespace string `json:"serviceNamespace"`
}

// ExternalInNetworkTelemetryDeploymentSpec defines the desired state of ExternalInNetworkTelemetryDeployment
type ExternalInNetworkTelemetryDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DeploymentTemplate appsv1.DeploymentSpec `json:"deploymentTemplate"`
	RequiredProgram string `json:"requiredProgram"`
	IngressInfo IngressInfo `json:"ingressInfo"`
	MonitoringPolicy MonitoringPolicy `json:"monitoringPolicy"`
	RequireAtLeastIntDevices *string `json:"requireAtLeastIntDevices,omitempty"`
}

// ExternalInNetworkTelemetryDeploymentStatus defines the observed state of ExternalInNetworkTelemetryDeployment
type ExternalInNetworkTelemetryDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// pod name -> status, TODO make it better
	BasicStatus string `json:"basicStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ExternalInNetworkTelemetryDeployment is the Schema for the externalinnetworktelemetrydeployments API
type ExternalInNetworkTelemetryDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalInNetworkTelemetryDeploymentSpec   `json:"spec,omitempty"`
	Status ExternalInNetworkTelemetryDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExternalInNetworkTelemetryDeploymentList contains a list of ExternalInNetworkTelemetryDeployment
type ExternalInNetworkTelemetryDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalInNetworkTelemetryDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalInNetworkTelemetryDeployment{}, &ExternalInNetworkTelemetryDeploymentList{})
}
