/*
Copyright 2026.

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
	"k8s.io/apimachinery/pkg/runtime"
)

// ManagementClusterSpec defines the desired state of a ManagementCluster.
type ManagementClusterSpec struct {
	// Region is the AWS region (e.g. us-east-1).
	// +kubebuilder:validation:MinLength=1
	Region string `json:"region"`

	// AccountID is the AWS account ID hosting this management cluster.
	// +kubebuilder:validation:Pattern=`^\d{12}$`
	AccountID string `json:"accountId"`
}

// ManagementClusterStatus defines the observed state of a ManagementCluster.
type ManagementClusterStatus struct {
	// Conditions represent the latest observations of the management cluster's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=hfmc
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=".spec.region"
// +kubebuilder:printcolumn:name="Account",type=string,JSONPath=".spec.accountId"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// ManagementCluster is the Schema for the managementclusters API.
// It represents a management cluster in the fleet.
type ManagementCluster struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ManagementClusterSpec `json:"spec"`

	// +optional
	Status ManagementClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ManagementClusterList contains a list of ManagementCluster.
type ManagementClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ManagementCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ManagementCluster{}, &ManagementClusterList{})
		return nil
	})
}
