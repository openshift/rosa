package versions

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
	ver "github.com/hashicorp/go-version"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	MinorVersionsSupported              = 2
	MajorMinorPatchFormattedErrorOutput = "An error occurred formatting the version for output: %v"
)

func GetVersionList(r *rosa.Runtime, channelGroup string, isSTS bool, isHostedCP bool, filterHostedCP bool,
	defaultFirst bool) (defaultVersion string, versionList []string, err error) {
	var vs []*v1.Version
	var product string
	if isHostedCP {
		product = ocm.HcpProduct
	}
	// Product can be empty. In this case, no filter will be applied
	vs, err = r.OCMClient.GetVersionsWithProduct(product, channelGroup, defaultFirst)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve versions: %s", err)
		return
	}

	defaultVersion, versionList, err = computeVersionListAndDefault(vs, isHostedCP, isSTS, filterHostedCP)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve versions: %s", err)
		return
	}

	if len(versionList) == 0 {
		err = fmt.Errorf("Could not find versions for the provided channel-group: '%s'", channelGroup)
		return
	}

	if defaultVersion == "" {
		// Normally this should not happen, as there is always a default version.
		// In case the default is not specified, we choose the most recent version.
		// Not having a default will break later.
		r.Reporter.Debugf("No default version found. Fallback to latest")
		defaultVersion = versionList[0]
	}

	return
}

func computeVersionListAndDefault(vs []*v1.Version, isHostedCP bool, isSTS bool,
	filterHostedCP bool) (string, []string, error) {
	var defaultVersion string
	var versionList []string
	for _, v := range vs {
		if defaultVersion == "" && isDefaultVersion(v, isHostedCP) {
			defaultVersion = v.RawID()
		}
		if isSTS && !ocm.HasSTSSupport(v.RawID(), v.ChannelGroup()) {
			continue
		}
		if filterHostedCP {
			valid, err := ocm.HasHostedCPSupport(v)
			if err != nil {
				return defaultVersion, versionList, fmt.Errorf("failed to check HostedCP support: %v", err)
			}
			if !valid {
				continue
			}
		}
		versionList = append(versionList, v.RawID())
	}
	return defaultVersion, versionList, nil
}

func isDefaultVersion(version *v1.Version, isHostedCP bool) bool {
	if (isHostedCP && version.HostedControlPlaneDefault()) || (!isHostedCP && version.Default()) {
		return true
	}
	return false
}

func GetFilteredVersionList(versionList []string, minVersion string, maxVersion string) []string {
	var filteredVersionList []string

	// Parse the versions for comparison
	min, errmin := ver.NewVersion(minVersion)
	max, errmax := ver.NewVersion(maxVersion)

	if errmin != nil || errmax != nil {
		return versionList
	}

	for _, version := range versionList {
		ver, errver := ver.NewVersion(version)
		if errver != nil {
			continue
		}
		if ver.GreaterThanOrEqual(min) && ver.LessThanOrEqual(max) {
			filteredVersionList = append(filteredVersionList, version)
		}
	}
	return filteredVersionList
}

// Used for hosted machinepool minimal version
func GetMinimalHostedMachinePoolVersion(controlPlaneVersion string) (string, error) {
	cpVersion, errcp := ver.NewVersion(controlPlaneVersion)
	if errcp != nil {
		return "", errcp
	}
	segments := cpVersion.Segments()
	// Hosted machinepools can be created with a minimal of two minor versions from the control plane
	minor := segments[1] - MinorVersionsSupported
	version := fmt.Sprintf("%d.%d.%d", segments[0], minor, 0)
	minimalVersion, errminver := ver.NewVersion(version)
	if errminver != nil {
		return "", errminver
	}

	lowestHostedCPSupport, errlow := ver.NewVersion(ocm.LowestHostedCpSupport)
	if errlow != nil {
		return "", errlow
	}

	if minimalVersion.LessThanOrEqual(lowestHostedCPSupport) {
		return ocm.LowestHostedCpSupport, nil
	}

	return version, nil
}

func IsGreaterThanOrEqual(version1, version2 string) (bool, error) {
	v1, err := version.NewVersion(strings.TrimPrefix(version1, ocm.VersionPrefix))
	if err != nil {
		return false, err
	}
	v2, err := version.NewVersion(strings.TrimPrefix(version2, ocm.VersionPrefix))
	if err != nil {
		return false, err
	}
	return v1.GreaterThanOrEqual(v2), nil
}

func FormatMajorMinorPatch(version string) (string, error) {
	major, minor, patch, err := getVersionSegments(version)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func getVersionSegments(rawVersionID string) (major, minor, patch int, err error) {
	version, err := ver.NewVersion(rawVersionID)
	if err != nil {
		return 0, 0, 0, err
	}
	segments := version.Segments()
	major = segments[0]
	minor = segments[1]
	patch = segments[2]
	return major, minor, patch, nil
}
