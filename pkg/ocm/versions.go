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

const DefaultChannelGroup = "stable"

const LowestSTSSupport = "4.7.11"
const LowestSTSMinor = "4.7"

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
			Order("default desc, id desc").
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

	// Sort list in descending order, ensuring the 'default' version at the top
	sort.Slice(versions, func(i, j int) bool {
		if versions[i].Default() {
			return true
		}
		if versions[j].Default() {
			return false
		}
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
	if channelGroup == "nightly" {
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

func GetVersionID(cluster *cmv1.Cluster) string {
	if cluster.OpenshiftVersion() != "" {
		return createVersionID(cluster.OpenshiftVersion(), cluster.Version().ChannelGroup())
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
		id := createVersionID(v, version.ChannelGroup())
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

func createVersionID(version string, channelGroup string) string {
	versionID := fmt.Sprintf("openshift-v%s", version)
	if channelGroup != "stable" {
		versionID = fmt.Sprintf("%s-%s", versionID, channelGroup)
	}
	return versionID
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

// Validate OpenShift versions
func ValidateVersion(version string, versionList []string) (string, error) {
	if version != "" {
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

		if !HasSTSSupportMinor(version) {
			err := fmt.Errorf("Version '%s' is not supported for STS clusters", version)
			return version, err
		}
	}

	return version, nil
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
