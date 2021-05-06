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
	ocmerrors "github.com/openshift-online/ocm-sdk-go/errors"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm/properties"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)

// Cluster names must be valid DNS-1035 labels, so they must consist of lower case alphanumeric
// characters or '-', start with an alphabetic character, and end with an alphanumeric character
var clusterNameRE = regexp.MustCompile(`^[a-z]([-a-z0-9]{0,13}[a-z0-9])?$`)

// Spec is the configuration for a cluster spec.
type Spec struct {
	// Basic configs
	Name         string
	Region       string
	MultiAZ      bool
	Version      string
	ChannelGroup string
	Expiration   time.Time
	Flavour      string

	// Scaling config
	ComputeMachineType string
	ComputeNodes       int
	Autoscaling        bool
	MinReplicas        int
	MaxReplicas        int

	// SubnetIDs
	SubnetIds []string

	// AvailabilityZones
	AvailabilityZones []string

	// Network config
	MachineCIDR net.IPNet
	ServiceCIDR net.IPNet
	PodCIDR     net.IPNet
	HostPrefix  int
	Private     *bool
	PrivateLink *bool

	// Properties
	CustomProperties map[string]string

	// Simulate creating a cluster but don't actually create it
	DryRun *bool

	// Disable SCP checks in the installer by setting credentials mode as mint
	DisableSCPChecks *bool
}

func IsValidClusterKey(clusterKey string) bool {
	return clusterKeyRE.MatchString(clusterKey)
}

func IsValidClusterName(clusterName string) bool {
	return clusterNameRE.MatchString(clusterName)
}

func HasClusters(client *cmv1.ClustersClient, creatorARN string) (bool, error) {
	query := fmt.Sprintf("properties.%s = '%s'", properties.CreatorARN, creatorARN)
	response, err := client.List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return false, handleErr(response.Error(), err)
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

	cluster, err := client.Add().Parameter("dryRun", *config.DryRun).Body(spec).Send()
	if config.DryRun != nil && *config.DryRun {
		if cluster.Error() != nil {
			return nil, handleErr(cluster.Error(), err)
		}
		return nil, nil
	}
	if err != nil {
		return nil, handleErr(cluster.Error(), err)
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
		return nil, handleErr(response.Error(), err)
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
	clusterNodesBuilder := cmv1.NewClusterNodes()
	if config.Autoscaling {
		autoscalingBuilder := cmv1.NewMachinePoolAutoscaling()
		if config.MinReplicas != 0 {
			autoscalingBuilder = autoscalingBuilder.MinReplicas(config.MinReplicas)
		}
		if config.MaxReplicas != 0 {
			autoscalingBuilder = autoscalingBuilder.MaxReplicas(config.MaxReplicas)
		}
		clusterNodesBuilder = clusterNodesBuilder.AutoscaleCompute(autoscalingBuilder)
		clusterBuilder = clusterBuilder.Nodes(clusterNodesBuilder)
	} else if config.ComputeNodes != 0 {
		clusterNodesBuilder = clusterNodesBuilder.Compute(config.ComputeNodes)
		clusterBuilder = clusterBuilder.Nodes(clusterNodesBuilder)
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

	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		return err
	}

	response, err := client.Cluster(cluster.ID()).Update().Body(clusterSpec).Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}

	return nil
}

func DeleteCluster(client *cmv1.ClustersClient, clusterKey string, creatorARN string) (*cmv1.Cluster, error) {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return nil, err
	}

	response, err := client.Cluster(cluster.ID()).Delete().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return cluster, nil
}

func GetAddOnParameters(client *cmv1.AddOnsClient, addOnID string) (*cmv1.AddOnParameterList, error) {
	response, err := client.Addon(addOnID).Get().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body().Parameters(), nil
}

type AddOnParam struct {
	Key string
	Val string
}

func InstallAddOn(client *cmv1.ClustersClient, clusterKey string, creatorARN string, addOnID string,
	params []AddOnParam) error {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return err
	}

	addOnInstallationBuilder := cmv1.NewAddOnInstallation().
		Addon(cmv1.NewAddOn().ID(addOnID))

	if len(params) > 0 {
		addOnParamList := make([]*cmv1.AddOnInstallationParameterBuilder, len(params))
		for i, param := range params {
			addOnParamList[i] = cmv1.NewAddOnInstallationParameter().ID(param.Key).Value(param.Val)
		}
		addOnInstallationBuilder = addOnInstallationBuilder.
			Parameters(cmv1.NewAddOnInstallationParameterList().Items(addOnParamList...))
	}

	addOnInstallation, err := addOnInstallationBuilder.Build()
	if err != nil {
		return err
	}

	response, err := client.Cluster(cluster.ID()).Addons().Add().Body(addOnInstallation).Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}

	return nil
}

func UninstallAddOn(client *cmv1.ClustersClient, clusterKey string, creatorARN string, addOnID string) error {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return err
	}

	response, err := client.Cluster(cluster.ID()).Addons().Addoninstallation(addOnID).Delete().Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}

	return nil
}

func GetAddOnInstallation(client *cmv1.ClustersClient, clusterKey string, creatorARN string,
	addOnID string) (*cmv1.AddOnInstallation, error) {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return nil, err
	}

	response, err := client.Cluster(cluster.ID()).Addons().Addoninstallation(addOnID).Get().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Body(), nil
}

func UpdateAddOnInstallation(client *cmv1.ClustersClient, clusterKey string, creatorARN string, addOnID string,
	params []AddOnParam) error {
	cluster, err := GetCluster(client, clusterKey, creatorARN)
	if err != nil {
		return err
	}

	addOnInstallationBuilder := cmv1.NewAddOnInstallation().
		Addon(cmv1.NewAddOn().ID(addOnID))

	if len(params) > 0 {
		addOnParamList := make([]*cmv1.AddOnInstallationParameterBuilder, len(params))
		for i, param := range params {
			addOnParamList[i] = cmv1.NewAddOnInstallationParameter().ID(param.Key).Value(param.Val)
		}
		addOnInstallationBuilder = addOnInstallationBuilder.
			Parameters(cmv1.NewAddOnInstallationParameterList().Items(addOnParamList...))
	}

	addOnInstallation, err := addOnInstallationBuilder.Build()
	if err != nil {
		return err
	}

	response, err := client.Cluster(cluster.ID()).
		Addons().Addoninstallation(addOnID).
		Update().Body(addOnInstallation).Send()
	if err != nil {
		return handleErr(response.Error(), err)
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
	awsAccessKey, err := awsClient.GetAWSAccessKeys()
	if err != nil {
		return nil, fmt.Errorf("Failed to get access keys for user '%s': %v",
			aws.AdminUserName, err)
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
		return nil, fmt.Errorf("Custom properties key %s collides with a property needed by rosa", properties.CreatorARN)
	}

	if _, present := clusterProperties[properties.CLIVersion]; present {
		return nil, fmt.Errorf("Custom properties key %s collides with a property needed by rosa", properties.CLIVersion)
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
				ID("rosa"),
		).
		Region(
			cmv1.NewCloudRegion().
				ID(config.Region),
		).
		Properties(clusterProperties)

	if config.Flavour != "" {
		clusterBuilder = clusterBuilder.Flavour(
			cmv1.NewFlavour().
				ID(config.Flavour),
		)
		reporter.Debugf("Using cluster flavour '%s'", config.Flavour)
	}

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

	if config.ComputeMachineType != "" || config.ComputeNodes != 0 || len(config.AvailabilityZones) > 0 ||
		config.Autoscaling {
		clusterNodesBuilder := cmv1.NewClusterNodes()
		if config.ComputeMachineType != "" {
			clusterNodesBuilder = clusterNodesBuilder.ComputeMachineType(
				cmv1.NewMachineType().ID(config.ComputeMachineType),
			)

			reporter.Debugf("Using machine type '%s'", config.ComputeMachineType)
		}
		if config.Autoscaling {
			clusterNodesBuilder = clusterNodesBuilder.AutoscaleCompute(
				cmv1.NewMachinePoolAutoscaling().
					MinReplicas(config.MinReplicas).
					MaxReplicas(config.MaxReplicas))
		} else if config.ComputeNodes != 0 {
			clusterNodesBuilder = clusterNodesBuilder.Compute(config.ComputeNodes)
		}
		if len(config.AvailabilityZones) > 0 {
			clusterNodesBuilder = clusterNodesBuilder.AvailabilityZones(config.AvailabilityZones...)
		}
		clusterBuilder = clusterBuilder.Nodes(clusterNodesBuilder)
	}

	if !IsEmptyCIDR(config.MachineCIDR) ||
		!IsEmptyCIDR(config.ServiceCIDR) ||
		!IsEmptyCIDR(config.PodCIDR) ||
		config.HostPrefix != 0 {
		networkBuilder := cmv1.NewNetwork()
		if !IsEmptyCIDR(config.MachineCIDR) {
			networkBuilder = networkBuilder.MachineCIDR(config.MachineCIDR.String())
		}
		if !IsEmptyCIDR(config.ServiceCIDR) {
			networkBuilder = networkBuilder.ServiceCIDR(config.ServiceCIDR.String())
		}
		if !IsEmptyCIDR(config.PodCIDR) {
			networkBuilder = networkBuilder.PodCIDR(config.PodCIDR.String())
		}
		if config.HostPrefix != 0 {
			networkBuilder = networkBuilder.HostPrefix(config.HostPrefix)
		}
		clusterBuilder = clusterBuilder.Network(networkBuilder)
	}

	awsBuilder := cmv1.NewAWS().
		AccountID(awsCreator.AccountID).
		AccessKeyID(awsAccessKey.AccessKeyID).
		SecretAccessKey(awsAccessKey.SecretAccessKey)

	if config.SubnetIds != nil {
		awsBuilder = awsBuilder.SubnetIDs(config.SubnetIds...)
	}

	if config.PrivateLink != nil {
		awsBuilder = awsBuilder.PrivateLink(*config.PrivateLink)
		if *config.PrivateLink {
			*config.Private = true
		}
	}

	clusterBuilder = clusterBuilder.AWS(awsBuilder)

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

	if config.DisableSCPChecks != nil && *config.DisableSCPChecks {
		clusterBuilder = clusterBuilder.CCS(cmv1.NewCCS().
			Enabled(true).
			DisableSCPChecks(true),
		)
	}

	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to create description of cluster: %v", err)
	}

	return clusterSpec, nil
}

// nolint:interfacer
func IsEmptyCIDR(cidr net.IPNet) bool {
	return cidr.String() == "<nil>"
}

func handleErr(res *ocmerrors.Error, err error) error {
	msg := res.Reason()
	if msg == "" {
		msg = err.Error()
	}
	// Hack to always display the correct terms and conditions message
	if res.Code() == "CLUSTERS-MGMT-451" {
		msg = "You must accept the Terms and Conditions in order to continue.\n" +
			"Go to https://www.redhat.com/wapps/tnc/ackrequired?site=ocm&event=register\n" +
			"Once you accept the terms, you will need to retry the action that was blocked."
	}
	return errors.New(msg)
}
