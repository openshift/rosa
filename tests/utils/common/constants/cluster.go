package constants

import (
	"regexp"
)

const (
	ClusterDescriptionComputeDesired    = "Compute (desired)"
	ClusterDescriptionComputeAutoscaled = "Compute (autoscaled)"
)

var (
	VersionLatestPattern     = regexp.MustCompile("latest")
	VersionMajorMinorPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	VersionRawPattern        = regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+-?[0-9a-z\.-]*`)
	VersionFlexyPattern      = regexp.MustCompile(`[xy]{1}-[1-3]{1}`)
)
