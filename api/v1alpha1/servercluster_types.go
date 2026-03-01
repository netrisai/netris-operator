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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServerClusterSpec defines the desired state of ServerCluster
type ServerClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Admin    string   `json:"admin"`
	Site     string   `json:"site"`
	VPC      string   `json:"vpc"`
	Template string   `json:"template"`
	Tags     []string `json:"tags,omitempty"`
}

// ServerClusterStatus defines the observed state of ServerCluster
type ServerClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status       string      `json:"status,omitempty"`
	Message      string      `json:"message,omitempty"`
	ModifiedDate metav1.Time `json:"modified,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Admin",type=string,JSONPath=`.spec.admin`
// +kubebuilder:printcolumn:name="Site",type=string,JSONPath=`.spec.site`
// +kubebuilder:printcolumn:name="VPC",type=string,JSONPath=`.spec.vpc`
// +kubebuilder:printcolumn:name="Template",type=string,JSONPath=`.spec.template`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Modified",type=date,JSONPath=`.status.modified`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ServerCluster is the Schema for the serverclusters API
type ServerCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerClusterSpec   `json:"spec"`
	Status ServerClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServerClusterList contains a list of ServerCluster
type ServerClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServerCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServerCluster{}, &ServerClusterList{})
}

