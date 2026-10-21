// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
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
// describe path decided to report defaults rather than an empty line.
//
// A node pool without an upgrade strategy reports documented defaults rather
// than empty strings, matching the OCM describe output. InPlace pools report
// only maxUnavailable; the maxSurge returned is an empty string even when the
// defaults are reported, since an InPlace upgrade cannot surge.
func NodePoolUpgradeSettings(np *v1alpha1.NodePool) (string, NodePoolUpgrade) {
	upgradeType := "Replace"
	if np.Spec.NodePool.Management.UpgradeType == hypershiftv1beta1.UpgradeTypeInPlace {
		upgradeType = "InPlace"
	}

	settings := NodePoolUpgrade{}

	// Customer-set values on spec take precedence.
	if np.Spec.MaxSurge != nil {
		settings.MaxSurge = FormatUpgradeValue(np.Spec.MaxSurge)
	}
	if np.Spec.MaxUnavailable != nil {
		settings.MaxUnavailable = FormatUpgradeValue(np.Spec.MaxUnavailable)
	}

	// If spec held nothing, read the rendered management upgrade the operator
	// writes back. The Platform API echoes the customer spec here, so empty
	// means the customer never set it and the API default is in effect.
	if settings.MaxSurge == "" &&
		np.Spec.NodePool.Management.Replace != nil &&
		np.Spec.NodePool.Management.Replace.RollingUpdate != nil {
		if np.Spec.NodePool.Management.Replace.RollingUpdate.MaxSurge != nil {
			settings.MaxSurge = FormatUpgradeValue(np.Spec.NodePool.Management.Replace.RollingUpdate.MaxSurge)
		}
		if np.Spec.NodePool.Management.Replace.RollingUpdate.MaxUnavailable != nil {
			settings.MaxUnavailable = FormatUpgradeValue(np.Spec.NodePool.Management.Replace.RollingUpdate.MaxUnavailable)
		}
	}

	// Report documented defaults when the API stored nothing. InPlace pools
	// omit maxSurge even when reporting defaults, since an InPlace upgrade
	// cannot surge.
	if settings.MaxSurge == "" && upgradeType == "Replace" {
		settings.MaxSurge = defaultMaxSurge
	}
	if settings.MaxUnavailable == "" {
		settings.MaxUnavailable = defaultMaxUnavailable
	}

	return upgradeType, settings
}

// FormatUpgradeValue renders an IntOrString from the API in the CLI shape:
// an absolute node count or a percentage, matching the flag the user passes.
func FormatUpgradeValue(val *intstr.IntOrString) string {
	if val == nil {
		return ""
	}
	if val.Type == intstr.Int {
		return strconv.Itoa(int(val.IntVal))
	}
	return val.StrVal
}

// EncodeUpgradeValue renders a rolling-update flag value in the JSON form the
// pathbind expander expects: a bare integer for absolute counts, a quoted
// string for percentages. The caller writes this to the pathbind input; Expand
// unmarshals it into the IntOrString field the API expects.
func EncodeUpgradeValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	// If it's a plain integer, return it bare. Otherwise quote it so JSON
	// unmarshal reads it as a string (for percentages like "20%").
	if _, err := strconv.Atoi(trimmed); err == nil {
		return trimmed
	}
	return strconv.Quote(trimmed)
}
