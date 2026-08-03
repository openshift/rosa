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

// ManifestPhase represents the lifecycle phase of a Manifest.
// +kubebuilder:validation:Enum=Syncing;Applied;Deleting
type ManifestPhase string

const (
	ManifestPhaseSyncing  ManifestPhase = "Syncing"
	ManifestPhaseApplied  ManifestPhase = "Applied"
	ManifestPhaseDeleting ManifestPhase = "Deleting"
)

// ManifestSpec defines a set of Kubernetes resources to apply on a management cluster.
// +kubebuilder:validation:XValidation:rule="self.managementCluster == oldSelf.managementCluster",message="spec.managementCluster is immutable"
type ManifestSpec struct {
	// ManagementCluster is the target MC ID (e.g. mc01).
	// +kubebuilder:validation:MinLength=1
	ManagementCluster string `json:"managementCluster"`

	// Resources is the list of Kubernetes resources to apply on the MC.
	// +kubebuilder:validation:MinItems=1
	Resources []ResourceTemplate `json:"resources"`
}

// ResourceTemplate describes a single Kubernetes resource to apply.
// The controller extracts apiVersion, metadata.name, and metadata.namespace
// from Content automatically. Only the plural resource name is required
// separately because kind-to-resource conversion needs a RESTMapper
// that the operator doesn't have for MC-side resources. kube-applier-aws
// uses the plural to build Kubernetes REST paths; if it derived the plural
// via discovery instead, this field could be dropped.
type ResourceTemplate struct {
	// Resource is the plural resource name (e.g. "configmaps", "deployments").
	// +kubebuilder:validation:MinLength=1
	Resource string `json:"resource"`

	// Content is the full Kubernetes resource manifest.
	// Must include apiVersion, kind, and metadata.name at minimum.
	Content runtime.RawExtension `json:"content"`

	// Watch creates a ReadDesire for this resource, mirroring its live state
	// from the MC back into status.resourceStatuses via kube-applier-aws.
	// +optional
	Watch bool `json:"watch,omitempty"`
}

// ResourceStatus holds the mirrored status of a watched resource from the MC.
type ResourceStatus struct {
	// Resource is the plural resource name (e.g. "jobs").
	Resource string `json:"resource"`
	// Name is metadata.name of the watched resource.
	Name string `json:"name"`
	// Namespace is metadata.namespace (empty for cluster-scoped resources).
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Group is the API group (e.g. "batch", "" for core).
	// +optional
	Group string `json:"group,omitempty"`
	// Version is the API version (e.g. "v1").
	// +optional
	Version string `json:"version,omitempty"`
	// Status is the .status sub-object of the live resource mirrored from the MC.
	// Only the status is stored to avoid duplicating the full spec in etcd.
	// +optional
	Status runtime.RawExtension `json:"status,omitempty"`
}

// ManifestStatus defines the observed state of a Manifest.
type ManifestStatus struct {
	// Conditions represent the latest observations of the manifest's state.
	// Known condition types: Synced.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase summarizes the manifest's lifecycle state.
	// +optional
	Phase ManifestPhase `json:"phase,omitempty"`

	// AppliedResources is the number of resources written as ApplyDesires.
	// +optional
	AppliedResources int32 `json:"appliedResources,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ResourceStatuses contains the mirrored live state of watched resources.
	// +optional
	ResourceStatuses []ResourceStatus `json:"resourceStatuses,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=hfm
// +kubebuilder:printcolumn:name="MC",type=string,JSONPath=".spec.managementCluster"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Resources",type=integer,JSONPath=".status.appliedResources"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// Manifest is the Schema for the manifests API.
// It deploys arbitrary Kubernetes resources to a management cluster via DynamoDB desires.
type Manifest struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ManifestSpec `json:"spec"`

	// +optional
	Status ManifestStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ManifestList contains a list of Manifest.
type ManifestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Manifest `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Manifest{}, &ManifestList{})
		return nil
	})
}
