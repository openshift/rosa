package versions

import (
	"fmt"

	ver "github.com/hashicorp/go-version"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	MinorVersionsSupported = 2
)

func GetVersionList(r *rosa.Runtime, channelGroup string, isSTS bool, isHostedCP bool,
	defaultFirst bool) (versionList []string, err error) {
	vs, err := r.OCMClient.GetVersions(channelGroup, defaultFirst)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve versions: %s", err)
		return
	}

	for _, v := range vs {
		if isSTS && !ocm.HasSTSSupport(v.RawID(), v.ChannelGroup()) {
			continue
		}
		if isHostedCP {
			valid, err := ocm.HasHostedCPSupport(v)
			if err != nil {
				return versionList, fmt.Errorf("failed to check HostedCP support: %v", err)
			}
			if !valid {
				continue
			}
		}
		versionList = append(versionList, v.RawID())
	}

	if len(versionList) == 0 {
		err = fmt.Errorf("Could not find versions for the provided channel-group: '%s'", channelGroup)
		return
	}

	return
}

func GetFilteredVersionListForCreation(versionList []string, minVersion string, maxVersion string) []string {
	return getFilteredVersionList(versionList, minVersion, maxVersion, false)
}

func GetFilteredVersionListForUpdate(versionList []string, minVersion string, maxVersion string) []string {
	return getFilteredVersionList(versionList, minVersion, maxVersion, true)
}

func getFilteredVersionList(versionList []string, minVersion string, maxVersion string, excludeCurrent bool) []string {
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
			// For upgrades, we don't want to show the current version. For creation, it should be shown
			if excludeCurrent && ver.Equal(min) {
				continue
			}
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

	lowestHostedCPSupport, errlow := ver.NewVersion(ocm.LowestHostedCPSupport)
	if errlow != nil {
		return "", errlow
	}

	if minimalVersion.LessThanOrEqual(lowestHostedCPSupport) {
		return ocm.LowestHostedCPSupport, nil
	}

	return version, nil
}
