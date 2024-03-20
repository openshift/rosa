package features

import (
	"fmt"

	"github.com/openshift/rosa/pkg/helper/versions"
)

type OCMFeature string

const (
	AdditionalDay2SecurityGroupsHcpFeature = "AdditionalDay2SecurityGroupsHcpFeature"
)

var ocmFeatureVersions = map[OCMFeature]string{
	AdditionalDay2SecurityGroupsHcpFeature: "4.15.0-0.a",
}

// IsFeatureSupported checks if a feature is supported for an OCP version.
// If the version string is empty, or the feature isn't defined, the function
// returns true.
func IsFeatureSupported(feature OCMFeature, version string) (bool, error) {
	// The version is not specified.
	if version == "" {
		return true, nil
	}

	supportedVersion := ocmFeatureVersions[feature]
	// OCM feature isn't defined.
	if supportedVersion == "" {
		return true, nil
	}

	isSupported, err := versions.IsGreaterThanOrEqual(version, supportedVersion)
	if err != nil {
		return false, fmt.Errorf("failed to check if feature '%s' is supported for version '%s': %v",
			feature, version, err)
	}

	return isSupported, nil
}
