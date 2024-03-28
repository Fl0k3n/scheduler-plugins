//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryDeployment) DeepCopyInto(out *ExternalInNetworkTelemetryDeployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryDeployment.
func (in *ExternalInNetworkTelemetryDeployment) DeepCopy() *ExternalInNetworkTelemetryDeployment {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExternalInNetworkTelemetryDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryDeploymentList) DeepCopyInto(out *ExternalInNetworkTelemetryDeploymentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExternalInNetworkTelemetryDeployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryDeploymentList.
func (in *ExternalInNetworkTelemetryDeploymentList) DeepCopy() *ExternalInNetworkTelemetryDeploymentList {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryDeploymentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExternalInNetworkTelemetryDeploymentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryDeploymentSpec) DeepCopyInto(out *ExternalInNetworkTelemetryDeploymentSpec) {
	*out = *in
	in.DeploymentTemplate.DeepCopyInto(&out.DeploymentTemplate)
	in.IngressInfo.DeepCopyInto(&out.IngressInfo)
	if in.RequireAtLeastIntDevices != nil {
		in, out := &in.RequireAtLeastIntDevices, &out.RequireAtLeastIntDevices
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryDeploymentSpec.
func (in *ExternalInNetworkTelemetryDeploymentSpec) DeepCopy() *ExternalInNetworkTelemetryDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryDeploymentStatus) DeepCopyInto(out *ExternalInNetworkTelemetryDeploymentStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryDeploymentStatus.
func (in *ExternalInNetworkTelemetryDeploymentStatus) DeepCopy() *ExternalInNetworkTelemetryDeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryDeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpoints) DeepCopyInto(out *ExternalInNetworkTelemetryEndpoints) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpoints.
func (in *ExternalInNetworkTelemetryEndpoints) DeepCopy() *ExternalInNetworkTelemetryEndpoints {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpoints)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExternalInNetworkTelemetryEndpoints) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpointsEntry) DeepCopyInto(out *ExternalInNetworkTelemetryEndpointsEntry) {
	*out = *in
	out.PodReference = in.PodReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpointsEntry.
func (in *ExternalInNetworkTelemetryEndpointsEntry) DeepCopy() *ExternalInNetworkTelemetryEndpointsEntry {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpointsEntry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpointsList) DeepCopyInto(out *ExternalInNetworkTelemetryEndpointsList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExternalInNetworkTelemetryEndpoints, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpointsList.
func (in *ExternalInNetworkTelemetryEndpointsList) DeepCopy() *ExternalInNetworkTelemetryEndpointsList {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpointsList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExternalInNetworkTelemetryEndpointsList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpointsSpec) DeepCopyInto(out *ExternalInNetworkTelemetryEndpointsSpec) {
	*out = *in
	if in.Entries != nil {
		in, out := &in.Entries, &out.Entries
		*out = make([]ExternalInNetworkTelemetryEndpointsEntry, len(*in))
		copy(*out, *in)
	}
	in.IngressInfo.DeepCopyInto(&out.IngressInfo)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpointsSpec.
func (in *ExternalInNetworkTelemetryEndpointsSpec) DeepCopy() *ExternalInNetworkTelemetryEndpointsSpec {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpointsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpointsStatus) DeepCopyInto(out *ExternalInNetworkTelemetryEndpointsStatus) {
	*out = *in
	if in.ExternalIngressInfo != nil {
		in, out := &in.ExternalIngressInfo, &out.ExternalIngressInfo
		*out = new(IngressInfo)
		(*in).DeepCopyInto(*out)
	}
	if in.ExternalMonitoringPolicy != nil {
		in, out := &in.ExternalMonitoringPolicy, &out.ExternalMonitoringPolicy
		*out = new(MonitoringPolicy)
		**out = **in
	}
	if in.InternalIngressInfo != nil {
		in, out := &in.InternalIngressInfo, &out.InternalIngressInfo
		*out = new(IngressInfo)
		(*in).DeepCopyInto(*out)
	}
	if in.InternalMonitoringPolicy != nil {
		in, out := &in.InternalMonitoringPolicy, &out.InternalMonitoringPolicy
		*out = new(MonitoringPolicy)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpointsStatus.
func (in *ExternalInNetworkTelemetryEndpointsStatus) DeepCopy() *ExternalInNetworkTelemetryEndpointsStatus {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpointsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressInfo) DeepCopyInto(out *IngressInfo) {
	*out = *in
	if in.NodeNames != nil {
		in, out := &in.NodeNames, &out.NodeNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressInfo.
func (in *IngressInfo) DeepCopy() *IngressInfo {
	if in == nil {
		return nil
	}
	out := new(IngressInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryDeployment) DeepCopyInto(out *InternalInNetworkTelemetryDeployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryDeployment.
func (in *InternalInNetworkTelemetryDeployment) DeepCopy() *InternalInNetworkTelemetryDeployment {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InternalInNetworkTelemetryDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryDeploymentList) DeepCopyInto(out *InternalInNetworkTelemetryDeploymentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InternalInNetworkTelemetryDeployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryDeploymentList.
func (in *InternalInNetworkTelemetryDeploymentList) DeepCopy() *InternalInNetworkTelemetryDeploymentList {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryDeploymentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InternalInNetworkTelemetryDeploymentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryDeploymentSpec) DeepCopyInto(out *InternalInNetworkTelemetryDeploymentSpec) {
	*out = *in
	if in.DeploymentTemplates != nil {
		in, out := &in.DeploymentTemplates, &out.DeploymentTemplates
		*out = make([]NamedDeploymentSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryDeploymentSpec.
func (in *InternalInNetworkTelemetryDeploymentSpec) DeepCopy() *InternalInNetworkTelemetryDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryDeploymentStatus) DeepCopyInto(out *InternalInNetworkTelemetryDeploymentStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryDeploymentStatus.
func (in *InternalInNetworkTelemetryDeploymentStatus) DeepCopy() *InternalInNetworkTelemetryDeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryDeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpoints) DeepCopyInto(out *InternalInNetworkTelemetryEndpoints) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpoints.
func (in *InternalInNetworkTelemetryEndpoints) DeepCopy() *InternalInNetworkTelemetryEndpoints {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpoints)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InternalInNetworkTelemetryEndpoints) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpointsEntry) DeepCopyInto(out *InternalInNetworkTelemetryEndpointsEntry) {
	*out = *in
	out.PodReference = in.PodReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpointsEntry.
func (in *InternalInNetworkTelemetryEndpointsEntry) DeepCopy() *InternalInNetworkTelemetryEndpointsEntry {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpointsEntry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpointsList) DeepCopyInto(out *InternalInNetworkTelemetryEndpointsList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InternalInNetworkTelemetryEndpoints, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpointsList.
func (in *InternalInNetworkTelemetryEndpointsList) DeepCopy() *InternalInNetworkTelemetryEndpointsList {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpointsList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InternalInNetworkTelemetryEndpointsList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpointsSpec) DeepCopyInto(out *InternalInNetworkTelemetryEndpointsSpec) {
	*out = *in
	if in.Entries != nil {
		in, out := &in.Entries, &out.Entries
		*out = make([]InternalInNetworkTelemetryEndpointsEntry, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpointsSpec.
func (in *InternalInNetworkTelemetryEndpointsSpec) DeepCopy() *InternalInNetworkTelemetryEndpointsSpec {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpointsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpointsStatus) DeepCopyInto(out *InternalInNetworkTelemetryEndpointsStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpointsStatus.
func (in *InternalInNetworkTelemetryEndpointsStatus) DeepCopy() *InternalInNetworkTelemetryEndpointsStatus {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpointsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamedDeploymentSpec) DeepCopyInto(out *NamedDeploymentSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamedDeploymentSpec.
func (in *NamedDeploymentSpec) DeepCopy() *NamedDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(NamedDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}