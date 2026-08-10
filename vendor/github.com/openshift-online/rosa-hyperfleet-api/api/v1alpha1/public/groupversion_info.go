// +kubebuilder:object:generate=true
// +groupName=hyperfleet.io
package public

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: "hyperfleet.io", Version: "v1alpha1"}

	GroupVersion = SchemeGroupVersion

	SchemeBuilder = runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
		scheme.AddKnownTypes(SchemeGroupVersion,
			&Cluster{}, &ClusterList{},
			&NodePool{}, &NodePoolList{},
			&ManagementCluster{}, &ManagementClusterList{},
			&Manifest{}, &ManifestList{},
			&Placement{}, &PlacementList{},
		)
		return nil
	})

	AddToScheme = SchemeBuilder.AddToScheme
)
