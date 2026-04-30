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

// ServerClusterMetaSpec defines the desired state of ServerClusterMeta
type ServerClusterMetaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Imported                  bool     `json:"imported"`
	Reclaim                   bool     `json:"reclaimPolicy"`
	ServerClusterCRGeneration int64    `json:"serverClusterGeneration"`
	ID                        int      `json:"id"`
	Name                      string   `json:"name"`
	ServerClusterName         string   `json:"serverClusterName"`
	AdminID                   int      `json:"adminId"`
	Admin                     string   `json:"admin"`
	SiteID                    int      `json:"siteId"`
	Site                      string   `json:"site"`
	VPCID                     int      `json:"vpcId"`
	VPC                       string   `json:"vpc"`
	TemplateID                int      `json:"templateId"`
	Template                  string   `json:"template"`
	Tags                      []string `json:"tags"`
}

// ServerClusterMetaStatus defines the observed state of ServerClusterMeta
type ServerClusterMetaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServerClusterMeta is the Schema for the serverclustermeta API
type ServerClusterMeta struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerClusterMetaSpec   `json:"spec,omitempty"`
	Status ServerClusterMetaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServerClusterMetaList contains a list of ServerClusterMeta
type ServerClusterMetaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServerClusterMeta `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServerClusterMeta{}, &ServerClusterMetaList{})
}

