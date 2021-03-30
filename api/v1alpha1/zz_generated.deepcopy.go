// +build !ignore_autogenerated

/*
Copyright 2021. Netris, Inc.

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
func (in *EBGP) DeepCopyInto(out *EBGP) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGP.
func (in *EBGP) DeepCopy() *EBGP {
	if in == nil {
		return nil
	}
	out := new(EBGP)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EBGP) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPList) DeepCopyInto(out *EBGPList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EBGP, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPList.
func (in *EBGPList) DeepCopy() *EBGPList {
	if in == nil {
		return nil
	}
	out := new(EBGPList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EBGPList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPMeta) DeepCopyInto(out *EBGPMeta) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPMeta.
func (in *EBGPMeta) DeepCopy() *EBGPMeta {
	if in == nil {
		return nil
	}
	out := new(EBGPMeta)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EBGPMeta) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPMetaList) DeepCopyInto(out *EBGPMetaList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EBGPMeta, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPMetaList.
func (in *EBGPMetaList) DeepCopy() *EBGPMetaList {
	if in == nil {
		return nil
	}
	out := new(EBGPMetaList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EBGPMetaList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPMetaSpec) DeepCopyInto(out *EBGPMetaSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPMetaSpec.
func (in *EBGPMetaSpec) DeepCopy() *EBGPMetaSpec {
	if in == nil {
		return nil
	}
	out := new(EBGPMetaSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPMetaStatus) DeepCopyInto(out *EBGPMetaStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPMetaStatus.
func (in *EBGPMetaStatus) DeepCopy() *EBGPMetaStatus {
	if in == nil {
		return nil
	}
	out := new(EBGPMetaStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPMultihop) DeepCopyInto(out *EBGPMultihop) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPMultihop.
func (in *EBGPMultihop) DeepCopy() *EBGPMultihop {
	if in == nil {
		return nil
	}
	out := new(EBGPMultihop)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPSpec) DeepCopyInto(out *EBGPSpec) {
	*out = *in
	out.Transport = in.Transport
	out.Multihop = in.Multihop
	if in.PrefixListInbound != nil {
		in, out := &in.PrefixListInbound, &out.PrefixListInbound
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.PrefixListOutbound != nil {
		in, out := &in.PrefixListOutbound, &out.PrefixListOutbound
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.SendBGPCommunity != nil {
		in, out := &in.SendBGPCommunity, &out.SendBGPCommunity
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPSpec.
func (in *EBGPSpec) DeepCopy() *EBGPSpec {
	if in == nil {
		return nil
	}
	out := new(EBGPSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPStatus) DeepCopyInto(out *EBGPStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPStatus.
func (in *EBGPStatus) DeepCopy() *EBGPStatus {
	if in == nil {
		return nil
	}
	out := new(EBGPStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EBGPTransport) DeepCopyInto(out *EBGPTransport) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EBGPTransport.
func (in *EBGPTransport) DeepCopy() *EBGPTransport {
	if in == nil {
		return nil
	}
	out := new(EBGPTransport)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *L4LB) DeepCopyInto(out *L4LB) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new L4LB.
func (in *L4LB) DeepCopy() *L4LB {
	if in == nil {
		return nil
	}
	out := new(L4LB)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *L4LB) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *L4LBCheck) DeepCopyInto(out *L4LBCheck) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new L4LBCheck.
func (in *L4LBCheck) DeepCopy() *L4LBCheck {
	if in == nil {
		return nil
	}
	out := new(L4LBCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *L4LBFrontend) DeepCopyInto(out *L4LBFrontend) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new L4LBFrontend.
func (in *L4LBFrontend) DeepCopy() *L4LBFrontend {
	if in == nil {
		return nil
	}
	out := new(L4LBFrontend)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *L4LBList) DeepCopyInto(out *L4LBList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]L4LB, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new L4LBList.
func (in *L4LBList) DeepCopy() *L4LBList {
	if in == nil {
		return nil
	}
	out := new(L4LBList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *L4LBList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *L4LBSpec) DeepCopyInto(out *L4LBSpec) {
	*out = *in
	out.Check = in.Check
	out.Frontend = in.Frontend
	if in.Backend != nil {
		in, out := &in.Backend, &out.Backend
		*out = make([]L4LBBackend, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new L4LBSpec.
func (in *L4LBSpec) DeepCopy() *L4LBSpec {
	if in == nil {
		return nil
	}
	out := new(L4LBSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *L4LBStatus) DeepCopyInto(out *L4LBStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new L4LBStatus.
func (in *L4LBStatus) DeepCopy() *L4LBStatus {
	if in == nil {
		return nil
	}
	out := new(L4LBStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNet) DeepCopyInto(out *VNet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNet.
func (in *VNet) DeepCopy() *VNet {
	if in == nil {
		return nil
	}
	out := new(VNet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VNet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetList) DeepCopyInto(out *VNetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VNet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetList.
func (in *VNetList) DeepCopy() *VNetList {
	if in == nil {
		return nil
	}
	out := new(VNetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VNetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetMeta) DeepCopyInto(out *VNetMeta) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetMeta.
func (in *VNetMeta) DeepCopy() *VNetMeta {
	if in == nil {
		return nil
	}
	out := new(VNetMeta)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VNetMeta) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetMetaGateway) DeepCopyInto(out *VNetMetaGateway) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetMetaGateway.
func (in *VNetMetaGateway) DeepCopy() *VNetMetaGateway {
	if in == nil {
		return nil
	}
	out := new(VNetMetaGateway)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetMetaList) DeepCopyInto(out *VNetMetaList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VNetMeta, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetMetaList.
func (in *VNetMetaList) DeepCopy() *VNetMetaList {
	if in == nil {
		return nil
	}
	out := new(VNetMetaList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VNetMetaList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetMetaMember) DeepCopyInto(out *VNetMetaMember) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetMetaMember.
func (in *VNetMetaMember) DeepCopy() *VNetMetaMember {
	if in == nil {
		return nil
	}
	out := new(VNetMetaMember)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetMetaSite) DeepCopyInto(out *VNetMetaSite) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetMetaSite.
func (in *VNetMetaSite) DeepCopy() *VNetMetaSite {
	if in == nil {
		return nil
	}
	out := new(VNetMetaSite)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetMetaSpec) DeepCopyInto(out *VNetMetaSpec) {
	*out = *in
	if in.Gateways != nil {
		in, out := &in.Gateways, &out.Gateways
		*out = make([]VNetMetaGateway, len(*in))
		copy(*out, *in)
	}
	if in.Members != nil {
		in, out := &in.Members, &out.Members
		*out = make([]VNetMetaMember, len(*in))
		copy(*out, *in)
	}
	if in.Sites != nil {
		in, out := &in.Sites, &out.Sites
		*out = make([]VNetMetaSite, len(*in))
		copy(*out, *in)
	}
	if in.Tenants != nil {
		in, out := &in.Tenants, &out.Tenants
		*out = make([]int, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetMetaSpec.
func (in *VNetMetaSpec) DeepCopy() *VNetMetaSpec {
	if in == nil {
		return nil
	}
	out := new(VNetMetaSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetMetaStatus) DeepCopyInto(out *VNetMetaStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetMetaStatus.
func (in *VNetMetaStatus) DeepCopy() *VNetMetaStatus {
	if in == nil {
		return nil
	}
	out := new(VNetMetaStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetSite) DeepCopyInto(out *VNetSite) {
	*out = *in
	if in.Gateways != nil {
		in, out := &in.Gateways, &out.Gateways
		*out = make([]VNetGateway, len(*in))
		copy(*out, *in)
	}
	if in.SwitchPorts != nil {
		in, out := &in.SwitchPorts, &out.SwitchPorts
		*out = make([]VNetSwitchPort, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetSite.
func (in *VNetSite) DeepCopy() *VNetSite {
	if in == nil {
		return nil
	}
	out := new(VNetSite)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetSpec) DeepCopyInto(out *VNetSpec) {
	*out = *in
	if in.GuestTenants != nil {
		in, out := &in.GuestTenants, &out.GuestTenants
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Sites != nil {
		in, out := &in.Sites, &out.Sites
		*out = make([]VNetSite, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetSpec.
func (in *VNetSpec) DeepCopy() *VNetSpec {
	if in == nil {
		return nil
	}
	out := new(VNetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetStatus) DeepCopyInto(out *VNetStatus) {
	*out = *in
	in.ModifiedDate.DeepCopyInto(&out.ModifiedDate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetStatus.
func (in *VNetStatus) DeepCopy() *VNetStatus {
	if in == nil {
		return nil
	}
	out := new(VNetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VNetSwitchPort) DeepCopyInto(out *VNetSwitchPort) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VNetSwitchPort.
func (in *VNetSwitchPort) DeepCopy() *VNetSwitchPort {
	if in == nil {
		return nil
	}
	out := new(VNetSwitchPort)
	in.DeepCopyInto(out)
	return out
}
