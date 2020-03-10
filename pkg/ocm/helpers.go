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
	"errors"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"gitlab.cee.redhat.com/service/moactl/pkg/ocm/properties"
)

func GetCluster(client *cmv1.ClustersClient, clusterKey string, creatorARN string) (*cmv1.Cluster, error) {
	query := fmt.Sprintf(
		"(id = '%s' or name = '%s') and properties.%s = '%s'",
		clusterKey, clusterKey, properties.CreatorARN, creatorARN,
	)
	response, err := client.List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to locate cluster '%s': %v", clusterKey, err))
	}

	switch response.Total() {
	case 0:
		return nil, errors.New(fmt.Sprintf("There is no cluster with identifier or name '%s'", clusterKey))
	case 1:
		return response.Items().Slice()[0], nil
	default:
		return nil, errors.New(fmt.Sprintf("There are %d clusters with identifier or name '%s'", response.Total(), clusterKey))
	}
}
