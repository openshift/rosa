package constants

import (
	"regexp"
)

const (
	ClusterDescriptionComputeDesired    = "Compute (desired)"
	ClusterDescriptionComputeAutoscaled = "Compute (autoscaled)"
)

// role and OIDC config
const (
	MaxRolePrefixLength       = 32
	MaxOIDCConfigPrefixLength = 15
)

// profile
const (
	DefaultNamePrefix = "rosacli-ci"
	DefaultNameLength = 15
)

// cluster configuration
const (
	DefaultVPCCIDRValue      = "10.0.0.0/16"
	DeleteProtectionDisabled = "Disabled"
	DeleteProtectionEnabled  = "Enabled"
)

// cluster status
const (
	Ready        = "ready"
	Installing   = "installing"
	Waiting      = "waiting"
	Pending      = "pending"
	Error        = "error"
	Uninstalling = "uninstalling"
	Validating   = "validating"
)

// version pattern supported for the CI
var (
	VersionLatestPattern     = regexp.MustCompile("latest")
	VersionMajorMinorPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	VersionRawPattern        = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+-?[0-9a-z\.-]*`)
	VersionFlexyPattern      = regexp.MustCompile(`[zy]{1}-[1-3]{1}`)
)
