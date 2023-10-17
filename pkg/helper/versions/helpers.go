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
	MinorVersionsSupported = 2
)

func GetVersionList(r *rosa.Runtime, channelGroup string, isSTS bool, isHostedCP bool, filterHostedCP bool,
	defaultFirst bool) (versionList []string, err error) {
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

	for _, v := range vs {
		if isSTS && !ocm.HasSTSSupport(v.RawID(), v.ChannelGroup()) {
			continue
		}
		if filterHostedCP {
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
