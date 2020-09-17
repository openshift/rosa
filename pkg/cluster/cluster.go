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
	"net"
	"regexp"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/info"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm/properties"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)

// Spec is the configuration for a cluster spec.
type Spec struct {
	// Basic configs
	Name         string
	Region       string
	MultiAZ      bool
	Version      string
	ChannelGroup string
	Expiration   time.Time

	// Scaling config
	ComputeMachineType string
	ComputeNodes       int

	// Network config
	MachineCIDR net.IPNet
	ServiceCIDR net.IPNet
	PodCIDR     net.IPNet
	HostPrefix  int
	Private     *bool

	// Properties
	CustomProperties map[string]string

	// Access control config
	ClusterAdmins *bool
}

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

func CreateCluster(client *cmv1.ClustersClient, config Spec) (*cmv1.Cluster, error) {
	reporter, err := rprtr.New().
		Build()

	if err != nil {
		return nil, fmt.Errorf("Unable to create reporter: %v", err)
	}

	logger, err := logging.NewLogger().
		Build()
	if err != nil {
		return nil, fmt.Errorf("Unable to create AWS logger: %v", err)
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Region(aws.DefaultRegion).
		Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create AWS client: %v", err)
	}

	spec, err := createClusterSpec(config, awsClient)
	if err != nil {
		return nil, fmt.Errorf("Unable to create cluster spec: %v", err)
	}

	cluster, err := client.Add().Body(spec).Send()
	if err != nil {
		return nil, fmt.Errorf("Error creating cluster in OCM: %v", err)
	}

	clusterObject := cluster.Body()

	// Add tags to the AWS administrator user containing the identifier and name of the cluster:
	err = awsClient.TagUser(aws.AdminUserName, clusterObject.ID(), clusterObject.Name())
	if err != nil {
		reporter.Warnf("Failed to add cluster tags to user '%s'", aws.AdminUserName)
	}
	return clusterObject, nil
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

func UpdateCluster(client *cmv1.ClustersClient, clusterKey string, creatorARN string, config Spec) error {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return err
	}

	clusterBuilder := cmv1.NewCluster()

	// Update expiration timestamp
	if !config.Expiration.IsZero() {
		clusterBuilder = clusterBuilder.ExpirationTimestamp(config.Expiration)
	}

	// Scale cluster
	if config.ComputeNodes != 0 {
		clusterBuilder = clusterBuilder.Nodes(
			cmv1.NewClusterNodes().
				Compute(config.ComputeNodes),
		)
	}

	// Toggle private mode
	if config.Private != nil {
		if *config.Private {
			clusterBuilder = clusterBuilder.API(
				cmv1.NewClusterAPI().
					Listening(cmv1.ListeningMethodInternal),
			)
		} else {
			clusterBuilder = clusterBuilder.API(
				cmv1.NewClusterAPI().
					Listening(cmv1.ListeningMethodExternal),
			)
		}
	}

	// Toggle cluster-admins group
	if config.ClusterAdmins != nil {
		clusterBuilder = clusterBuilder.ClusterAdminEnabled(*config.ClusterAdmins)
	}

	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		return err
	}

	_, err = client.Cluster(cluster.ID()).Update().Body(clusterSpec).Send()
	if err != nil {
		return err
	}

	return nil
}

func DeleteCluster(client *cmv1.ClustersClient, clusterKey string, creatorARN string) (*cmv1.Cluster, error) {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return nil, err
	}

	_, err = client.Cluster(cluster.ID()).Delete().Send()
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func InstallAddOn(client *cmv1.ClustersClient, clusterKey string, creatorARN string, addOnID string) error {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return err
	}

	addOnInstallation, err := cmv1.NewAddOnInstallation().
		Addon(cmv1.NewAddOn().ID(addOnID)).
		Build()
	if err != nil {
		return err
	}

	_, err = client.Cluster(cluster.ID()).Addons().Add().Body(addOnInstallation).Send()
	if err != nil {
		return err
	}

	return nil
}

func createClusterSpec(config Spec, awsClient aws.Client) (*cmv1.Cluster, error) {
	reporter, err := rprtr.New().
		Build()

	if err != nil {
		return nil, fmt.Errorf("Error creating cluster reporter: %v", err)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		return nil, fmt.Errorf("Failed to get AWS creator: %v", err)
	}

	// Create the access key for the AWS user:
	awsAccessKey, err := awsClient.GetAccessKeyFromStack(aws.OsdCcsAdminStackName)
	if err != nil {
		return nil, fmt.Errorf("Failed to get access keys for user '%s': %v", aws.AdminUserName, err)
	}
	reporter.Debugf("Access key identifier is '%s'", awsAccessKey.AccessKeyID)
	reporter.Debugf("Secret access key is '%s'", awsAccessKey.SecretAccessKey)

	clusterProperties := map[string]string{}

	if config.CustomProperties != nil {
		for key, value := range config.CustomProperties {
			clusterProperties[key] = value
		}
	}

	// Make sure we don't have a custom properties collision
	if _, present := clusterProperties[properties.CreatorARN]; present {
		return nil, fmt.Errorf("Custom properties key %s collides with a property needed by moactl", properties.CreatorARN)
	}

	if _, present := clusterProperties[properties.CLIVersion]; present {
		return nil, fmt.Errorf("Custom properties key %s collides with a property needed by moactl", properties.CLIVersion)
	}

	clusterProperties[properties.CreatorARN] = awsCreator.ARN
	clusterProperties[properties.CLIVersion] = info.Version

	// Create the cluster:
	clusterBuilder := cmv1.NewCluster().
		Name(config.Name).
		DisplayName(config.Name).
		MultiAZ(config.MultiAZ).
		Product(
			cmv1.NewProduct().
				ID("moa"),
		).
		Region(
			cmv1.NewCloudRegion().
				ID(config.Region),
		).
		AWS(
			cmv1.NewAWS().
				AccountID(awsCreator.AccountID).
				AccessKeyID(awsAccessKey.AccessKeyID).
				SecretAccessKey(awsAccessKey.SecretAccessKey),
		).
		Properties(clusterProperties)

	if config.Version != "" {
		clusterBuilder = clusterBuilder.Version(
			cmv1.NewVersion().
				ID(config.Version).
				ChannelGroup(config.ChannelGroup),
		)

		reporter.Debugf(
			"Using OpenShift version '%s' on channel group '%s'",
			config.Version, config.ChannelGroup)
	}

	if !config.Expiration.IsZero() {
		clusterBuilder = clusterBuilder.ExpirationTimestamp(config.Expiration)
	}

	if config.ComputeMachineType != "" || config.ComputeNodes != 0 {
		clusterNodesBuilder := cmv1.NewClusterNodes()
		if config.ComputeMachineType != "" {
			clusterNodesBuilder = clusterNodesBuilder.ComputeMachineType(
				cmv1.NewMachineType().ID(config.ComputeMachineType),
			)

			reporter.Debugf("Using machine type '%s'", config.ComputeMachineType)
		}
		if config.ComputeNodes != 0 {
			clusterNodesBuilder = clusterNodesBuilder.Compute(config.ComputeNodes)
		}
		clusterBuilder = clusterBuilder.Nodes(clusterNodesBuilder)
	}

	if !cidrIsEmpty(config.MachineCIDR) ||
		!cidrIsEmpty(config.ServiceCIDR) ||
		!cidrIsEmpty(config.PodCIDR) ||
		config.HostPrefix != 0 {
		networkBuilder := cmv1.NewNetwork()
		if !cidrIsEmpty(config.MachineCIDR) {
			networkBuilder = networkBuilder.MachineCIDR(config.MachineCIDR.String())
		}
		if !cidrIsEmpty(config.ServiceCIDR) {
			networkBuilder = networkBuilder.ServiceCIDR(config.ServiceCIDR.String())
		}
		if !cidrIsEmpty(config.PodCIDR) {
			networkBuilder = networkBuilder.PodCIDR(config.PodCIDR.String())
		}
		if config.HostPrefix != 0 {
			networkBuilder = networkBuilder.HostPrefix(config.HostPrefix)
		}
		clusterBuilder = clusterBuilder.Network(networkBuilder)
	}

	if config.Private != nil {
		if *config.Private {
			clusterBuilder = clusterBuilder.API(
				cmv1.NewClusterAPI().
					Listening(cmv1.ListeningMethodInternal),
			)
		} else {
			clusterBuilder = clusterBuilder.API(
				cmv1.NewClusterAPI().
					Listening(cmv1.ListeningMethodExternal),
			)
		}
	}

	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create description of cluster: %v", err)
	}

	return clusterSpec, nil
}

func cidrIsEmpty(cidr net.IPNet) bool {
	return cidr.String() == "<nil>"
}
