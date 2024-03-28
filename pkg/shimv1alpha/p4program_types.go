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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ProgramArtifacts struct {
	Arch string `json:"arch"`
	P4InfoURL string `json:"p4InfoUrl"`
	P4PipelineURL string `json:"P4PipelineUrl"`
}

// P4ProgramSpec defines the desired state of P4Program
type P4ProgramSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Artifacts []ProgramArtifacts `json:"artifacts"`
	ImplementedInterfaces []string `json:"implements"`
}

// P4ProgramStatus defines the observed state of P4Program
type P4ProgramStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// P4Program is the Schema for the p4programs API
type P4Program struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   P4ProgramSpec   `json:"spec,omitempty"`
	Status P4ProgramStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// P4ProgramList contains a list of P4Program
type P4ProgramList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []P4Program `json:"items"`
}

func init() {
	SchemeBuilder.Register(&P4Program{}, &P4ProgramList{})
}
