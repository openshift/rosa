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

// PlacementPhase represents the lifecycle phase of a Placement.
// +kubebuilder:validation:Enum=Pending;Bound
type PlacementPhase string

const (
	PlacementPhasePending PlacementPhase = "Pending"
	PlacementPhaseBound   PlacementPhase = "Bound"
)

// PlacementSpec defines the desired placement of a Cluster on a management cluster.
// +kubebuilder:validation:XValidation:rule="self.clusterName == oldSelf.clusterName",message="spec.clusterName is immutable"
type PlacementSpec struct {
	// ClusterName is the name of the Cluster CR this placement is for.
	// +kubebuilder:validation:MinLength=1
	ClusterName string `json:"clusterName"`

	// ManagementCluster is the target management cluster ID (e.g. mc01).
	// +kubebuilder:validation:MinLength=1
	ManagementCluster string `json:"managementCluster"`
}

// PlacementStatus defines the observed state of a Placement.
type PlacementStatus struct {
	// Conditions represent the latest observations of the placement's state.
	// Known condition types: Bound.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase summarizes the placement's state: Pending or Bound.
	// +optional
	Phase PlacementPhase `json:"phase,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=hfp
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=".spec.clusterName"
// +kubebuilder:printcolumn:name="MC",type=string,JSONPath=".spec.managementCluster"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// Placement is the Schema for the placements API.
// It assigns a Cluster to a management cluster.
type Placement struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec PlacementSpec `json:"spec"`

	// +optional
	Status PlacementStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PlacementList contains a list of Placement.
type PlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Placement `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Placement{}, &PlacementList{})
		return nil
	})
}
