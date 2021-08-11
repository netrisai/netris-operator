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

func (l *L4LB) GetServiceName() string {
	return l.GetAnnotations()["servicename"]
}

func (l *L4LB) SetImportFlag(s string) {
	anns := l.GetAnnotations()
	anns["resource.k8s.netris.ai/import"] = s
	l.SetAnnotations(anns)
}

func (l *L4LB) SetServiceName(s string) {
	anns := l.GetAnnotations()
	anns["servicename"] = s
	l.SetAnnotations(anns)
}

func (l *L4LB) GetServiceNamespace() string {
	return l.GetAnnotations()["servicenamespace"]
}

func (l *L4LB) SetServiceNamespace(s string) {
	anns := l.GetAnnotations()
	anns["servicenamespace"] = s
	l.SetAnnotations(anns)
}

func (l *L4LB) GetServiceUID() string {
	return l.GetAnnotations()["serviceuid"]
}

func (l *L4LB) SetServiceUID(s string) {
	anns := l.GetAnnotations()
	anns["serviceuid"] = s
	l.SetAnnotations(anns)
}

func (l *L4LB) GetServiceIngressIPs() string {
	return l.GetAnnotations()["serviceingressips"]
}

func (l *L4LB) SetServiceIngressIPs(s string) {
	anns := l.GetAnnotations()
	anns["serviceingressips"] = s
	l.SetAnnotations(anns)
}

func (l *L4LB) IPRole() string {
	return l.GetAnnotations()["resource.k8s.netris.ai/iprole"]
}

func (l *L4LB) SetIPRole(s string) {
	anns := l.GetAnnotations()
	anns["resource.k8s.netris.ai/iprole"] = s
	l.SetAnnotations(anns)
}
