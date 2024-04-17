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
	VersionFlexyPattern      = regexp.MustCompile(`[xy]{1}-[1-3]{1}`)
)
