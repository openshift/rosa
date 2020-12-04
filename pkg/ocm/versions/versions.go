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

package versions

import (
	"errors"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmerrors "github.com/openshift-online/ocm-sdk-go/errors"
)

const DefaultChannelGroup = "stable"

func GetVersions(client *cmv1.Client, channelGroup string) (versions []*cmv1.Version, err error) {
	collection := client.Versions()
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
			Order("default desc, id asc").
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
	return
}

func GetVersionID(cluster *cmv1.Cluster) string {
	if cluster.OpenshiftVersion() != "" {
		return createVersionID(cluster.OpenshiftVersion(), cluster.Version().ChannelGroup())
	}
	return cluster.Version().ID()
}

func GetAvailableUpgrades(client *cmv1.Client, versionID string) ([]string, error) {
	response, err := client.Versions().Version(versionID).Get().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	version := response.Body()
	availableUpgrades := []string{}

	for _, v := range version.AvailableUpgrades() {
		id := createVersionID(v, version.ChannelGroup())
		resp, err := client.Versions().Version(id).Get().Send()
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

func handleErr(res *ocmerrors.Error, err error) error {
	msg := res.Reason()
	if msg == "" {
		msg = err.Error()
	}
	return errors.New(msg)
}
