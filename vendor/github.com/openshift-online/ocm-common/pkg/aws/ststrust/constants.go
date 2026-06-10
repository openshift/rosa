package ststrust

import "regexp"

// Length and format limits for aws.sts.external_id (OCM Clusters Service parity).
const (
	// MinSTSExternalIDLength matches OCM Clusters Service validation for aws.sts.external_id.
	MinSTSExternalIDLength = 2
	// MaxSTSExternalIDLength matches OCM Clusters Service validation for aws.sts.external_id.
	MaxSTSExternalIDLength = 1224
)

// STSExternalIDRegex matches OCM Clusters Service validation for aws.sts.external_id.
var STSExternalIDRegex = regexp.MustCompile(`^[\w+=,.@:\/-]*$`)
