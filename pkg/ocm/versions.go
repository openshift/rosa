/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

import (
	"fmt"
	"sort"
	"strings"

	ver "github.com/hashicorp/go-version"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	DefaultChannelGroup   = "stable"
	NightlyChannelGroup   = "nightly"
	LowestSTSSupport      = "4.7.11"
	LowestSTSMinor        = "4.7"
	LowestHostedCPSupport = "4.12.0-0.a" //TODO: Remove the 0.a once stable 4.12 builds are available
)

func (c *Client) GetVersions(channelGroup string) (versions []*cmv1.Version, err error) {
	collection := c.ocm.ClustersMgmt().V1().Versions()
	page := 1
	size := 100
	filter := "enabled = 'true' AND rosa_enabled = 'true'"
	if channelGroup != "" {
		filter = fmt.Sprintf("%s AND channel_group = '%s'", filter, channelGroup)
	}
	for {
		var response *cmv1.VersionsListResponse
		response, err = collection.List().
			Search(filter).
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return nil, handleErr(response.Error(), err)
		}
		versions = append(versions, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}

	// Sort list in descending order
	sort.Slice(versions, func(i, j int) bool {
		a, erra := ver.NewVersion(versions[i].RawID())
		b, errb := ver.NewVersion(versions[j].RawID())
		if erra != nil || errb != nil {
			return false
		}
		return a.GreaterThan(b)
	})

	return
}

func HasSTSSupport(rawID string, channelGroup string) bool {
	if channelGroup == NightlyChannelGroup {
		return true
	}

	a, erra := ver.NewVersion(rawID)
	b, errb := ver.NewVersion(LowestSTSSupport)
	if erra != nil || errb != nil {
		return rawID >= LowestSTSSupport
	}

	return a.GreaterThanOrEqual(b)
}

func HasSTSSupportMinor(minor string) bool {
	a, erra := ver.NewVersion(minor)
	b, errb := ver.NewVersion(LowestSTSMinor)
	if erra != nil || errb != nil {
		return minor >= LowestSTSMinor
	}

	return a.GreaterThanOrEqual(b)
}

func HasHostedCPSupport(version *cmv1.Version) (bool, error) {
	if !version.HostedControlPlaneEnabled() {
		return false, nil
	}
	v, err := ver.NewVersion(version.RawID())
	if err != nil {
		return false, fmt.Errorf("error while parsing OCP version '%s': %v", version.RawID(), err)
	}
	b, err := ver.NewVersion(LowestHostedCPSupport)
	if err != nil {
		return false, fmt.Errorf("error while parsing OCP version '%s': %v", version.RawID(), err)
	}
	// Check minimum OCP supported version
	return v.GreaterThanOrEqual(b), nil
}

func GetVersionID(cluster *cmv1.Cluster) string {
	if cluster.OpenshiftVersion() != "" {
		return CreateVersionID(cluster.OpenshiftVersion(), cluster.Version().ChannelGroup())
	}
	return cluster.Version().ID()
}

func (c *Client) GetAvailableUpgrades(versionID string) ([]string, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Versions().
		Version(versionID).
		Get().
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	version := response.Body()
	availableUpgrades := []string{}

	for _, v := range version.AvailableUpgrades() {
		id := CreateVersionID(v, version.ChannelGroup())
		resp, err := c.ocm.ClustersMgmt().V1().
			Versions().
			Version(id).
			Get().
			Send()
		if err != nil {
			return nil, handleErr(response.Error(), err)
		}
		if resp.Body().ROSAEnabled() {
			// Prepend versions so that the latest one shows up first
			availableUpgrades = append([]string{v}, availableUpgrades...)
		}
	}

	return availableUpgrades, nil
}

func CreateVersionID(version string, channelGroup string) string {
	versionID := fmt.Sprintf("openshift-v%s", version)
	if channelGroup != DefaultChannelGroup {
		versionID = fmt.Sprintf("%s-%s", versionID, channelGroup)
	}
	return versionID
}

func GetRawVersionId(versionId string) string {
	trimmedPrefix := strings.TrimPrefix(versionId, "openshift-v")
	channelSeparator := strings.LastIndex(trimmedPrefix, "-")
	if channelSeparator > 0 {
		return trimmedPrefix[:channelSeparator]
	}
	return trimmedPrefix
}

// Get a list of all STS-supported minor versions
func GetVersionMinorList(ocmClient *Client) (versionList []string, err error) {
	vs, err := ocmClient.GetVersions("")
	if err != nil {
		err = fmt.Errorf("Failed to retrieve versions: %s", err)
		return
	}

	// Make a set-map of all minors
	minorSet := make(map[string]struct{})
	for _, v := range vs {
		if !HasSTSSupport(v.RawID(), v.ChannelGroup()) {
			continue
		}
		version, errv := ver.NewVersion(v.RawID())
		if errv != nil {
			return versionList, errv
		}
		segments := version.Segments64()
		minor := fmt.Sprintf("%d.%d", segments[0], segments[1])
		minorSet[minor] = struct{}{}
	}

	// Extract minor keys into a slice
	for m := range minorSet {
		versionList = append(versionList, m)
	}

	return
}

func (c *Client) GetDefaultVersion() (version string, err error) {
	response, err := c.GetVersions("")
	if err != nil {
		return "", err
	}
	if len(response) > 0 {
		if response[0] != nil {
			parsedVersion, err := ver.NewVersion(response[0].RawID())
			if err != nil {
				return "", err
			}
			versionSplit := parsedVersion.Segments64()
			return fmt.Sprintf("%d.%d", versionSplit[0], versionSplit[1]), nil
		}

	}
	return "", fmt.Errorf("There are no openShift versions available")
}

func IsValidVersion(userRequestedVersion string, supportedVersion string, clusterVersion string) (bool, error) {

	a, err := ver.NewVersion(userRequestedVersion)
	if err != nil {
		return false, err
	}

	b, err := ver.NewVersion(supportedVersion)
	if err != nil {
		return false, err
	}

	isPreRelease := a.Prerelease() != "" && b.Prerelease() != ""

	versionSplit := a.Segments64()
	// If user has specified patch or not specified patch but is a preRelease
	// Check directly
	if len(versionSplit) > 2 && versionSplit[2] > 0 || (versionSplit[2] == 0 && isPreRelease) {
		return a.Equal(b), err
	}

	//If the user has specified only major and minor we check
	//if the major minor is greater than cluster major minor if so return true
	//else return false.
	userRequestedMajorMinorVersion := fmt.Sprintf("%d.%d",
		versionSplit[0], versionSplit[1])

	userRequestedParsedVersion, err := ver.NewVersion(userRequestedMajorMinorVersion)
	if err != nil {
		return false, err
	}

	isValid, err := checkClusterVersion(clusterVersion, userRequestedParsedVersion)
	if err != nil {
		return false, err
	}
	if !isValid {
		return false, nil
	}

	v := b.Segments64()
	supportedMajorMinorVersion := fmt.Sprintf("%d.%d",
		v[0], v[1])

	parsedSupportedVersion, err := ver.NewVersion(supportedMajorMinorVersion)
	if err != nil {
		return false, err
	}

	//If major minor specified is not equal to cluster version and is not
	//in the supported version then return error
	if userRequestedParsedVersion.Equal(parsedSupportedVersion) {
		return true, nil
	}
	return false, nil
}

func checkClusterVersion(clusterVersion string, userRequestedParsedVersion *ver.Version) (bool, error) {
	c, err := ver.NewVersion(clusterVersion)
	if err != nil {
		return false, err
	}
	cSplit := c.Segments64()

	clusterMajorMinorVersion := fmt.Sprintf("%d.%d",
		cSplit[0], cSplit[1])

	clusterParsedVersion, err := ver.NewVersion(clusterMajorMinorVersion)
	if err != nil {
		return false, err
	}

	//If major minor is only specified and it is equal to cluster current version
	//return error
	if userRequestedParsedVersion.Equal(clusterParsedVersion) {
		return false, nil
	}
	return true, nil
}

func CheckAndParseVersion(availableUpgrades []string, version string) (string, error) {
	a, err := ver.NewVersion(version)
	if err != nil {
		return "", err
	}
	isPreRelease := a.Prerelease() != ""
	versionSplit := a.Segments64()
	if len(versionSplit) > 2 && versionSplit[2] > 0 || (versionSplit[2] == 0 && isPreRelease) {
		return version, nil
	}
	return availableUpgrades[0], nil
}

// Validate OpenShift versions
func (c *Client) ValidateVersion(version string, versionList []string, channelGroup string, isSTS,
	isHostedCP bool) (string, error) {
	if version == "" {
		return version, nil
	}
	// Check and set the cluster version
	hasVersion := false
	for _, v := range versionList {
		if v == version {
			hasVersion = true
		}
	}
	if !hasVersion {
		allVersions := strings.Join(versionList, " ")
		err := fmt.Errorf("A valid version number must be specified\nValid versions: %s", allVersions)
		return version, err
	}

	if isSTS && !HasSTSSupport(version, channelGroup) {
		err := fmt.Errorf("Version '%s' is not supported for STS clusters", version)
		return version, err
	}
	if !HasSTSSupportMinor(version) {
		err := fmt.Errorf("Version '%s' is not supported for STS clusters", version)
		return version, err
	}
	if isHostedCP {
		collection := c.ocm.ClustersMgmt().V1().Versions()
		filter := fmt.Sprintf("raw_id='%s'", version)
		response, err := collection.List().
			Search(filter).
			Page(1).
			Size(1).
			Send()
		if err != nil {
			return "", handleErr(response.Error(), err)
		}
		valid, err := HasHostedCPSupport(response.Items().Get(0))
		if err != nil || !valid {
			return "", err
		}
	}

	return CreateVersionID(version, channelGroup), nil
}
