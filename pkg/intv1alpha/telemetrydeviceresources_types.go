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

type TelemetryDeviceResource struct {
	DeviceName string `json:"deviceName"`
	CanBeSource bool `json:"canBeSource"`
}


const (
	TypeAvailableTelemetryDeviceResources = "Available"
)


type TelemetryDeviceResourcesSpec struct {
}

type TelemetryDeviceResourcesStatus struct {
	DeviceResources []TelemetryDeviceResource `json:"deviceResources"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TelemetryDeviceResources is the Schema for the telemetrydeviceresources API
type TelemetryDeviceResources struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TelemetryDeviceResourcesSpec   `json:"spec,omitempty"`
	Status TelemetryDeviceResourcesStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TelemetryDeviceResourcesList contains a list of TelemetryDeviceResources
type TelemetryDeviceResourcesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TelemetryDeviceResources `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TelemetryDeviceResources{}, &TelemetryDeviceResourcesList{})
}
