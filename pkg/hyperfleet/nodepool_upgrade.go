package hyperfleet

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// NodePoolUpgrade holds the rolling-update settings of a node pool as the CLI
// reports them: an absolute node count or a percentage, empty when unset.
type NodePoolUpgrade struct {
	MaxSurge       string
	MaxUnavailable string
}

// Defaults the Platform API applies to an unset rolling-update value. They are
// part of the field contract rather than stored values: a node pool the
// customer never configured stores nothing and behaves as if these were set.
const (
	defaultMaxSurge       = "1"
	defaultMaxUnavailable = "0"
)

// NodePoolUpgradeSettings returns the upgrade type and the rolling-update
// settings of a node pool: the customer values on spec first, then the
// management block the operator renders from them, the way the autorepair
// formatter does, and finally the API's documented defaults so the effective
// behavior is reported the way the OCM path reports it.
// An InPlace node pool never surges, so maxSurge is left out for one.
func NodePoolUpgradeSettings(np *v1alpha1.NodePool) (string, NodePoolUpgrade) {
	upgrade := NodePoolUpgrade{
		MaxSurge:       FormatUpgradeValue(np.Spec.MaxSurge),
		MaxUnavailable: FormatUpgradeValue(np.Spec.MaxUnavailable),
	}

	management := np.Spec.NodePool.Management
	if management.Replace != nil && management.Replace.RollingUpdate != nil {
		rendered := management.Replace.RollingUpdate
		if upgrade.MaxSurge == "" {
			upgrade.MaxSurge = FormatUpgradeValue(rendered.MaxSurge)
		}
		if upgrade.MaxUnavailable == "" {
			upgrade.MaxUnavailable = FormatUpgradeValue(rendered.MaxUnavailable)
		}
	}
	inPlace := management.UpgradeType == hypershiftv1beta1.UpgradeTypeInPlace
	if management.InPlace != nil && upgrade.MaxUnavailable == "" {
		upgrade.MaxUnavailable = FormatUpgradeValue(management.InPlace.MaxUnavailable)
	}

	if upgrade.MaxSurge == "" && !inPlace {
		upgrade.MaxSurge = defaultMaxSurge
	}
	if upgrade.MaxUnavailable == "" {
		upgrade.MaxUnavailable = defaultMaxUnavailable
	}
	return string(management.UpgradeType), upgrade
}

// FormatUpgradeValue renders an absolute or percentage rolling-update value.
// An unset value reports as empty rather than as a fabricated default.
func FormatUpgradeValue(value *intstr.IntOrString) string {
	if value == nil {
		return ""
	}
	return value.String()
}

// EncodeUpgradeValue renders a rolling-update flag value in the JSON form the
// generated input expects for an int-or-string field: a bare number for an
// absolute node count, a quoted literal for a percentage. Passing "20%"
// through unquoted is not valid JSON, so pathbind cannot decode it.
// An empty value stays empty, which pathbind skips as unset.
func EncodeUpgradeValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if _, err := strconv.Atoi(trimmed); err == nil {
		return trimmed
	}
	return strconv.Quote(trimmed)
}
