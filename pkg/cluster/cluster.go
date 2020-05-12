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

package cluster

import (
	"errors"
	"fmt"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/moactl/pkg/ocm/properties"
)

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)

func IsValidClusterKey(clusterKey string) bool {
	return clusterKeyRE.MatchString(clusterKey)
}

func HasClusters(client *cmv1.ClustersClient, creatorARN string) (bool, error) {
	query := fmt.Sprintf("properties.%s = '%s'", properties.CreatorARN, creatorARN)
	response, err := client.List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return false, fmt.Errorf("Failed to list clusters: %v", err)
	}

	return response.Total() > 0, nil
}

func CreateCluster(client *cmv1.ClustersClient, spec *cmv1.Cluster) (*cmv1.Cluster, error) {
	cluster, err := client.Add().Body(spec).Send()
	if err != nil {
		return nil, err
	}
	return cluster.Body(), nil
}

func GetClusters(client *cmv1.ClustersClient, creatorARN string, count int) (clusters []*cmv1.Cluster, err error) {
	if count < 1 {
		err = errors.New("Cannot fetch fewer than 1 cluster")
		return
	}
	query := fmt.Sprintf("properties.%s = '%s'", properties.CreatorARN, creatorARN)
	request := client.List().Search(query)
	page := 1
	for {
		response, err := request.Page(page).Size(count).Send()
		if err != nil {
			return clusters, err
		}
		response.Items().Each(func(cluster *cmv1.Cluster) bool {
			clusters = append(clusters, cluster)
			return true
		})
		if response.Size() != count {
			break
		}
		page++
	}
	return clusters, nil
}

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
		return nil, fmt.Errorf("Failed to locate cluster '%s': %v", clusterKey, err)
	}

	switch response.Total() {
	case 0:
		return nil, fmt.Errorf("There is no cluster with identifier or name '%s'", clusterKey)
	case 1:
		return response.Items().Slice()[0], nil
	default:
		return nil, fmt.Errorf("There are %d clusters with identifier or name '%s'", response.Total(), clusterKey)
	}
}

func DeleteCluster(client *cmv1.ClustersClient, clusterKey string, creatorARN string) error {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return err
	}

	_, err = client.Cluster(cluster.ID()).Delete().Send()
	if err != nil {
		return err
	}

	return nil
}
