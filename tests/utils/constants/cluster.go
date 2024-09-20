package constants

import (
	"regexp"
	"time"
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
	DeleteProtectionEnabled  = "Enabled"
	DeleteProtectionDisabled = "Disabled"
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

// cluster upgrade status
const (
	Scheduled = "scheduled"
	Started   = "started"
	Delayed   = "delayed"
)

// version pattern supported for the CI
var (
	VersionLatestPattern     = regexp.MustCompile("latest")
	VersionMajorMinorPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	VersionRawPattern        = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+-?[0-9a-z\.-]*`)
	VersionFlexyPattern      = regexp.MustCompile(`[zy]{1}-[1-3]{1}`)
)

// instance type
const (
	DefaultInstanceType = "m5.xlarge"
	CommonAWSRegion     = "us-west-2"

	M5XLarge  = "m5.xlarge"
	M52XLarge = "m5.2xlarge"
	M6gXLarge = "m6g.xlarge"
)

// cpu architecture
const (
	AMD = "amd64"
	ARM = "arm64"
)

// timeout for hcp node pool
const (
	// NodePoolCheckTimeout The timeout may depend on the resource
	NodePoolCheckTimeout = 30 * time.Minute
	NodePoolCheckPoll    = 10 * time.Second
)
