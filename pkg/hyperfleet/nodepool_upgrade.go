package hyperfleet

import (
	"k8s.io/apimachinery/pkg/util/intstr"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
)

// NodePoolUpgrade models the customer rolling-update settings absent from the
// pinned SDK. The Platform API renders them into the service-set
// spec.nodePool.management.replace.rollingUpdate, so they have no writable
// field on v1alpha1.NodePoolSpec and have to be encoded by hand.
type NodePoolUpgrade struct {
	MaxSurge       string `json:"maxSurge,omitempty"`
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
}

// NodePoolUpgradeSettings returns the upgrade type and the rolling-update
// values the Platform API reports for a node pool. The upgrade is nil when the
// API has not rendered any rolling-update settings yet.
func NodePoolUpgradeSettings(np *v1alpha1.NodePool) (string, *NodePoolUpgrade) {
	management := np.Spec.NodePool.Management
	if management.Replace == nil || management.Replace.RollingUpdate == nil {
		return string(management.UpgradeType), nil
	}
	rollingUpdate := management.Replace.RollingUpdate
	return string(management.UpgradeType), &NodePoolUpgrade{
		MaxSurge:       formatUpgradeValue(rollingUpdate.MaxSurge),
		MaxUnavailable: formatUpgradeValue(rollingUpdate.MaxUnavailable),
	}
}

// formatUpgradeValue renders an absolute or percentage rolling-update value.
// An unset value reports as empty rather than as a fabricated default.
func formatUpgradeValue(value *intstr.IntOrString) string {
	if value == nil {
		return ""
	}
	return value.String()
}
