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
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterPhase represents the lifecycle phase of a Cluster.
// +kubebuilder:validation:Enum=WaitingForPlacement;Provisioning;Ready;Deleting
type ClusterPhase string

const (
	ClusterPhaseWaitingForPlacement ClusterPhase = "WaitingForPlacement"
	ClusterPhaseProvisioning        ClusterPhase = "Provisioning"
	ClusterPhaseReady               ClusterPhase = "Ready"
	ClusterPhaseDeleting            ClusterPhase = "Deleting"
)

// ClusterSpec defines the desired state of a ROSA HCP cluster.
// metadata.Name is the human-readable cluster name; metadata.Namespace is the cluster UUID.
// The owning AWS account is stored as the label hyperfleet.io/account-id.
type ClusterSpec struct {
	// CreatorARN is the IAM ARN of the user who created this cluster.
	// Used to bootstrap the initial cluster-admin RBAC mapping.
	// +optional
	// +kubebuilder:validation:Pattern=`^arn:aws:`
	CreatorARN string `json:"creatorARN,omitempty"`

	// ExpirationTimestamp is the time after which the cluster will be
	// automatically deleted. If nil, the cluster has no expiration.
	// +optional
	ExpirationTimestamp *metav1.Time `json:"expirationTimestamp,omitempty"`

	// HostedCluster is the full HyperShift HostedClusterSpec. The customer provides
	// the fields they care about; the operator overrides platform-managed fields
	// (InfraID, DNS, PullSecret, Services, etc.) at render time.
	// +kubebuilder:validation:Required
	HostedCluster hypershiftv1beta1.HostedClusterSpec `json:"hostedCluster"`
}

// ClusterStatus defines the observed state of a Cluster.
type ClusterStatus struct {
	// Conditions represent the latest observations of the cluster's state.
	// Known condition types: Synced, Available, Degraded.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase summarizes the cluster's lifecycle state.
	// +optional
	Phase ClusterPhase `json:"phase,omitempty"`

	// ControlPlaneEndpoint is the API server endpoint for the hosted cluster.
	// +optional
	ControlPlaneEndpoint hypershiftv1beta1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`

	// Version is the running OpenShift version.
	// +optional
	Version string `json:"version,omitempty"`

	// PlacementRef references the Placement that assigned this cluster to a management cluster.
	// +optional
	PlacementRef *PlacementReference `json:"placementRef,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// PlacementReference identifies the management cluster assignment.
type PlacementReference struct {
	// Name is the Placement CR name.
	Name string `json:"name"`

	// ManagementCluster is the target management cluster ID (e.g. mc01).
	ManagementCluster string `json:"managementCluster"`
}

// +genclient
// +wire:field=name,meta=name
// +wire:field=id,meta=uid
// +wire:field=resource_version,meta=resourceVersion
// +wire:field=generation,meta=generation
// +wire:watch=disabled
// +wire:wait
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=hfc
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="MC",type=string,JSONPath=".status.placementRef.managementCluster"
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=".status.controlPlaneEndpoint.host",priority=1
// +kubebuilder:printcolumn:name="Expires",type=date,JSONPath=".spec.expirationTimestamp",priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// Cluster is the Schema for the clusters API.
// It represents a ROSA HCP cluster whose lifecycle is managed by the hyperfleet-operator.
// metadata.Name is the human-readable cluster name; metadata.Namespace is the cluster UUID.
// The owning account is the label hyperfleet.io/account-id.
type Cluster struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ClusterSpec `json:"spec"`

	// +optional
	Status ClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Cluster{}, &ClusterList{})
		return nil
	})
}
