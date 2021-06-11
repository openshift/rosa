/*
Copyright (c) 2021 Red Hat, Inc.

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
	"net/http"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func GetUser(client *cmv1.ClustersClient, clusterID string, group string, username string) (*cmv1.User, error) {
	response, err := client.Cluster(clusterID).
		Groups().Group(group).
		Users().User(username).
		Get().Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil, nil
		}
		return nil, handleErr(response.Error(), err)
	}

	return response.Body(), nil
}

func GetUsers(client *cmv1.ClustersClient, clusterID string, group string) ([]*cmv1.User, error) {
	usersClient := client.Cluster(clusterID).Groups().Group(group).Users()
	response, err := usersClient.List().
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Items().Slice(), nil
}
