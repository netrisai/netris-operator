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

package v1alpha1

// GetServiceName gets the service name from annotations.
func (l *L4LB) GetServiceName() string {
	return l.GetAnnotations()["servicename"]
}

// SetImportFlag set import flags into annotations.
func (l *L4LB) SetImportFlag(s string) {
	anns := l.GetAnnotations()
	anns["resource.k8s.netris.ai/import"] = s
	l.SetAnnotations(anns)
}

// SetServiceName set service name into annotations.
func (l *L4LB) SetServiceName(s string) {
	anns := l.GetAnnotations()
	anns["servicename"] = s
	l.SetAnnotations(anns)
}

// GetServiceNamespace gets the service namespace from annotations.
func (l *L4LB) GetServiceNamespace() string {
	return l.GetAnnotations()["servicenamespace"]
}

// SetServiceNamespace set service namespace into annotations.
func (l *L4LB) SetServiceNamespace(s string) {
	anns := l.GetAnnotations()
	anns["servicenamespace"] = s
	l.SetAnnotations(anns)
}

// GetServiceUID gets the service uuid from annotations.
func (l *L4LB) GetServiceUID() string {
	return l.GetAnnotations()["serviceuid"]
}

// SetServiceUID set service uuid into annotations.
func (l *L4LB) SetServiceUID(s string) {
	anns := l.GetAnnotations()
	anns["serviceuid"] = s
	l.SetAnnotations(anns)
}

// GetServiceIngressIPs gets the ingress ips from annotations.
func (l *L4LB) GetServiceIngressIPs() string {
	return l.GetAnnotations()["serviceingressips"]
}

// SetServiceIngressIPs set ingress ips into annotations.
func (l *L4LB) SetServiceIngressIPs(s string) {
	anns := l.GetAnnotations()
	anns["serviceingressips"] = s
	l.SetAnnotations(anns)
}
