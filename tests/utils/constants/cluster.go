// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

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

// cluster topology
const (
	HostedCP = "Hosted CP"
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
	DefaultInstanceType    = "m5.xlarge"
	DefaultHCPInstanceType = "m7i.xlarge"
	CommonAWSRegion        = "us-west-2"

	R5XLarge  = "r5.xlarge"
	M5XLarge  = "m5.xlarge"
	M52XLarge = "m5.2xlarge"
	M6gXLarge = "m6g.xlarge"
)

// cpu architecture
const (
	AMD = "amd64"
	ARM = "arm64"
)

// IAM/OIDC propagation
const (
	// IAMOIDCPropagationDelay is the time to wait after creating IAM roles
	// and OIDC providers in manual mode, allowing AWS to propagate the
	// trust chain before AssumeRoleWithWebIdentity calls.
	IAMOIDCPropagationDelay = 30 * time.Second
)

// timeout for hcp node pool
const (
	// NodePoolCheckTimeout The timeout may depend on the resource
	NodePoolCheckTimeout = 30 * time.Minute
	NodePoolCheckPoll    = 10 * time.Second

	// Day2TagPropagationTimeout is the maximum time to wait for day-2 AWS
	// tag edits to propagate through the HCP control plane to EC2 workers.
	Day2TagPropagationTimeout = 15 * time.Minute
	// Day2TagPropagationPoll is the interval between EC2 tag checks during
	// day-2 tag propagation polling.
	Day2TagPropagationPoll = 15 * time.Second
)

// disk size
const (
	MinClassicDiskSize = 128
	MinHCPDiskSize     = 75
	MaxDiskSize        = 16384
)

// disk size invalid message
const (
	// DiskSizeErrRangeMsg in order to solve lint issue
	DiskSizeErrRangeMsg = "Invalid root disk size: %d GiB. Must be between %d GiB and %d GiB"
)
