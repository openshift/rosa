package tags

import "regexp"

// AWS / OCM length limits for custom resource tags.
const (
	// MaxKeyLength is the maximum length of an AWS tag key.
	MaxKeyLength = 128
	// MaxValueLength is the maximum length of an AWS tag value.
	MaxValueLength = 256
)

// Reserved exact tag keys that callers must not set (ROSA / OCM product docs).
const (
	ReservedKeyAWS               = "aws"
	ReservedKeyRedHatManaged     = "red-hat-managed"
	ReservedKeyRedHatClusterType = "red-hat-clustertype"
	ReservedKeyName              = "Name"
)

// ReservedKeyPrefixKubernetesCluster is the forbidden key prefix for custom tags.
const ReservedKeyPrefixKubernetesCluster = "kubernetes.io/cluster/"

// ReservedKeyPrefixAWS is the AWS-reserved key prefix (case-insensitive).
const ReservedKeyPrefixAWS = "aws:"

// KeyRE and ValueRE match OCM Hosted Control Plane tag charset rules
// (ASCII alphanumerics, ordinary space, and _.:/=+-@; tabs and other
// whitespace are rejected).
var (
	KeyRE   = regexp.MustCompile(`^[0-9A-Za-z_.:/=+\-@ ]+$`)
	ValueRE = regexp.MustCompile(`^[0-9A-Za-z_.:/=+\-@ ]*$`)
)

// reservedExactKeys lists keys that are rejected on write regardless of value.
var reservedExactKeys = map[string]struct{}{
	ReservedKeyAWS:               {},
	ReservedKeyRedHatManaged:     {},
	ReservedKeyRedHatClusterType: {},
	ReservedKeyName:              {},
}
