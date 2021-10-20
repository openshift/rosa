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
	"net"
	"os"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/properties"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	errors "github.com/zgalor/weberr"
)

// Spec is the configuration for a cluster spec.
type Spec struct {
	// Basic configs
	Name                      string
	Region                    string
	MultiAZ                   bool
	Version                   string
	ChannelGroup              string
	Expiration                time.Time
	Flavour                   string
	DisableWorkloadMonitoring bool
	//Encryption
	EtcdEncryption bool
	KMSKeyArn      string
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

	// User-defined tags for AWS resources
	Tags map[string]string

	// Simulate creating a cluster but don't actually create it
	DryRun *bool

	// Disable SCP checks in the installer by setting credentials mode as mint
	DisableSCPChecks *bool

	// STS
	IsSTS               bool
	RoleARN             string
	ExternalID          string
	SupportRoleARN      string
	OperatorIAMRoles    []OperatorIAMRole
	ControlPlaneRoleARN string
	WorkerRoleARN       string

	NodeDrainGracePeriodInMinutes float64
}

type OperatorIAMRole struct {
	Name      string
	Namespace string
	RoleARN   string
}

// Generate a query that filters clusters running on the current AWS session account
func getClusterFilter(creator *aws.Creator) string {
	return fmt.Sprintf(
		"product.id = 'rosa' AND (properties.%s LIKE '%%:%s:%%' OR aws.sts.role_arn LIKE '%%:%s:%%')",
		properties.CreatorARN,
		creator.AccountID,
		creator.AccountID,
	)
}

func (c *Client) HasClusters(creator *aws.Creator) (bool, error) {
	query := getClusterFilter(creator)
	response, err := c.ocm.ClustersMgmt().V1().Clusters().
		List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return false, handleErr(response.Error(), err)
	}

	return response.Total() > 0, nil
}

func (c *Client) CreateCluster(config Spec) (*cmv1.Cluster, error) {
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

	spec, err := c.createClusterSpec(config, awsClient)
	if err != nil {
		return nil, fmt.Errorf("Unable to create cluster spec: %v", err)
	}

	cluster, err := c.ocm.ClustersMgmt().V1().Clusters().
		Add().
		Parameter("dryRun", *config.DryRun).
		Body(spec).
		Send()
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

	return clusterObject, nil
}

/**
pass 0 to get all clusters
*/
func (c *Client) GetClusters(creator *aws.Creator, count int) (clusters []*cmv1.Cluster, err error) {
	if count < 0 {
		err = errors.Errorf("Invalid Cluster count")
		return
	}
	query := getClusterFilter(creator)
	request := c.ocm.ClustersMgmt().V1().Clusters().List().Search(query)
	page := 1
	for {
		clusterRequestList := request.Page(page)
		if count > 0 {
			clusterRequestList = clusterRequestList.Size(count)
		}
		response, err := clusterRequestList.Send()
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

func (c *Client) GetAllClusters(creator *aws.Creator) (clusters []*cmv1.Cluster, err error) {
	return c.GetClusters(creator, 0)
}

func (c *Client) GetCluster(clusterKey string, creator *aws.Creator) (*cmv1.Cluster, error) {
	query := fmt.Sprintf("%s AND (id = '%s' OR name = '%s' OR external_id = '%s')",
		getClusterFilter(creator),
		clusterKey, clusterKey, clusterKey,
	)
	response, err := c.ocm.ClustersMgmt().V1().Clusters().List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	switch response.Total() {
	case 0:
		return nil, errors.NotFound.Errorf("There is no cluster with identifier or name '%s'", clusterKey)
	case 1:
		return response.Items().Slice()[0], nil
	default:
		return nil, fmt.Errorf("There are %d clusters with identifier or name '%s'", response.Total(), clusterKey)
	}
}

func (c *Client) GetClusterByID(clusterKey string, creator *aws.Creator) (*cmv1.Cluster, error) {
	query := fmt.Sprintf("%s AND id = '%s'",
		getClusterFilter(creator),
		clusterKey,
	)
	response, err := c.ocm.ClustersMgmt().V1().Clusters().List().
		Search(query).
		Page(1).
		Size(1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	switch response.Total() {
	case 0:
		return nil, errors.NotFound.Errorf("There is no cluster with identifier '%s'", clusterKey)
	case 1:
		return response.Items().Slice()[0], nil
	default:
		return nil, fmt.Errorf("There are %d clusters with identifier '%s'", response.Total(), clusterKey)
	}
}

// Gets only pending non-STS clusters that are installed in the same AWS account
func (c *Client) GetPendingClusterForARN(creator *aws.Creator) (cluster *cmv1.Cluster, err error) {
	query := fmt.Sprintf(
		"state = 'pending' AND product.id = 'rosa' AND aws.sts.role_arn = '' AND properties.%s LIKE '%%:%s:%%'",
		properties.CreatorARN,
		creator.AccountID,
	)
	request := c.ocm.ClustersMgmt().V1().Clusters().List().Search(query)

	response, err := request.Send()
	if err != nil {
		return cluster, err
	}
	return response.Items().Get(0), nil
}

func (c *Client) IsSTSClusterExists(creator *aws.Creator, count int, roleARN string) (exists bool, err error) {
	if count < 1 {
		err = errors.Errorf("Cannot fetch fewer than 1 cluster")
		return
	}
	query := fmt.Sprintf(
		"product.id = 'rosa' AND ("+
			"properties.%s LIKE '%%:%s:%%' OR "+
			"aws.sts.role_arn = '%s' OR "+
			"aws.sts.support_role_arn = '%s' OR "+
			"aws.sts.instance_iam_roles.master_role_arn = '%s' OR "+
			"aws.sts.instance_iam_roles.worker_role_arn = '%s')",
		properties.CreatorARN,
		creator.AccountID,
		roleARN,
		roleARN,
		roleARN,
		roleARN,
	)
	request := c.ocm.ClustersMgmt().V1().Clusters().List().Search(query)
	page := 1
	response, err := request.Page(page).Size(count).Send()
	if err != nil {
		return false, err
	}
	if response.Total() > 0 {
		return true, nil
	}
	return false, nil
}

func (c *Client) GetClusterStatus(clusterID string) (*cmv1.ClusterStatus, error) {
	response, err := c.ocm.ClustersMgmt().V1().Clusters().
		Cluster(clusterID).
		Status().
		Get().
		Send()
	if err != nil || response.Body() == nil {
		return nil, err
	}
	return response.Body(), nil
}

func (c *Client) GetClusterState(clusterID string) (cmv1.ClusterState, error) {
	response, err := c.ocm.ClustersMgmt().V1().Clusters().
		Cluster(clusterID).
		Status().
		Get().
		Send()
	if err != nil || response.Body() == nil {
		return cmv1.ClusterState(""), err
	}
	return response.Body().State(), nil
}

func (c *Client) UpdateCluster(clusterKey string, creator *aws.Creator, config Spec) error {
	cluster, err := c.GetCluster(clusterKey, creator)
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

	if config.NodeDrainGracePeriodInMinutes != 0 {
		clusterBuilder = clusterBuilder.NodeDrainGracePeriod(
			cmv1.NewValue().
				Value(config.NodeDrainGracePeriodInMinutes).
				Unit("minutes"),
		)
	}

	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		return err
	}

	response, err := c.ocm.ClustersMgmt().V1().Clusters().
		Cluster(cluster.ID()).
		Update().
		Body(clusterSpec).
		Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}

	return nil
}

func (c *Client) DeleteCluster(clusterKey string, creator *aws.Creator) (*cmv1.Cluster, error) {
	cluster, err := c.GetCluster(clusterKey, creator)
	if err != nil {
		return nil, err
	}

	response, err := c.ocm.ClustersMgmt().V1().Clusters().
		Cluster(cluster.ID()).
		Delete().
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return cluster, nil
}

func (c *Client) createClusterSpec(config Spec, awsClient aws.Client) (*cmv1.Cluster, error) {
	reporter, err := rprtr.New().
		Build()

	if err != nil {
		return nil, fmt.Errorf("Error creating cluster reporter: %v", err)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		return nil, fmt.Errorf("Failed to get AWS creator: %v", err)
	}

	var awsAccessKey *aws.AccessKey
	if config.RoleARN == "" {
		/**
		1) Poll the cluster with same arn from ocm
		2) Check the status and if pending enter to a loop until it becomes installing
		3) Do it only for ROSA clusters and before UpsertAccessKey
		*/
		deadline := time.Now().Add(5 * time.Minute)
		for {
			pendingCluster, err := c.GetPendingClusterForARN(awsCreator)
			if err != nil {
				reporter.Errorf("Error getting cluster using ARN '%s'", awsCreator.ARN)
				os.Exit(1)
			}
			if time.Now().After(deadline) {
				reporter.Errorf("Timeout waiting for the cluster '%s' installation. Try again in a few minutes",
					pendingCluster.ID())
				os.Exit(1)
			}
			if pendingCluster == nil {
				break
			} else {
				reporter.Infof("Waiting for cluster '%s' with the same creator ARN to start installing",
					pendingCluster.ID())
				time.Sleep(30 * time.Second)
			}
		}

		// Create the access key for the AWS user:
		awsAccessKey, err = awsClient.GetAWSAccessKeys()
		if err != nil {
			return nil, fmt.Errorf("Failed to get access keys for user '%s': %v",
				aws.AdminUserName, err)
		}
		reporter.Debugf("Access key identifier is '%s'", awsAccessKey.AccessKeyID)
		reporter.Debugf("Secret access key is '%s'", awsAccessKey.SecretAccessKey)
	}

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
		EtcdEncryption(config.EtcdEncryption).
		Properties(clusterProperties).
		DisableUserWorkloadMonitoring(config.DisableWorkloadMonitoring)

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
		AccountID(awsCreator.AccountID)

	if config.SubnetIds != nil {
		awsBuilder = awsBuilder.SubnetIDs(config.SubnetIds...)
	}

	if config.PrivateLink != nil {
		awsBuilder = awsBuilder.PrivateLink(*config.PrivateLink)
		if *config.PrivateLink {
			*config.Private = true
		}
	}

	if config.RoleARN != "" {
		stsBuilder := cmv1.NewSTS().RoleARN(config.RoleARN)
		if config.ExternalID != "" {
			stsBuilder = stsBuilder.ExternalID(config.ExternalID)
		}
		if config.SupportRoleARN != "" {
			stsBuilder = stsBuilder.SupportRoleARN(config.SupportRoleARN)
		}
		if len(config.OperatorIAMRoles) > 0 {
			roles := []*cmv1.OperatorIAMRoleBuilder{}
			for _, role := range config.OperatorIAMRoles {
				roles = append(roles, cmv1.NewOperatorIAMRole().
					Name(role.Name).
					Namespace(role.Namespace).
					RoleARN(role.RoleARN),
				)
			}
			stsBuilder = stsBuilder.OperatorIAMRoles(roles...)
		}
		instanceIAMRolesBuilder := cmv1.NewInstanceIAMRoles()
		if config.ControlPlaneRoleARN != "" {
			instanceIAMRolesBuilder.MasterRoleARN(config.ControlPlaneRoleARN)
		}
		if config.WorkerRoleARN != "" {
			instanceIAMRolesBuilder.WorkerRoleARN(config.WorkerRoleARN)
		}
		stsBuilder = stsBuilder.InstanceIAMRoles(instanceIAMRolesBuilder)
		awsBuilder = awsBuilder.STS(stsBuilder)
	} else {
		awsBuilder = awsBuilder.
			AccessKeyID(awsAccessKey.AccessKeyID).
			SecretAccessKey(awsAccessKey.SecretAccessKey)
	}
	if config.KMSKeyArn != "" {
		awsBuilder = awsBuilder.KMSKeyArn(config.KMSKeyArn)
	}
	if len(config.Tags) > 0 {
		awsBuilder = awsBuilder.Tags(config.Tags)
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

func (c *Client) HibernateCluster(clusterID string) error {
	err := c.IsHibernateCapabilityEnabled()
	if err != nil {
		return err
	}
	_, err = c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).Hibernate().Send()
	if err != nil {
		return fmt.Errorf("Failed to hibernate the cluster: %v", err)
	}

	return nil
}

func (c *Client) ResumeCluster(clusterID string) error {
	err := c.IsHibernateCapabilityEnabled()
	if err != nil {
		return err
	}
	_, err = c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).Resume().Send()
	if err != nil {
		return fmt.Errorf("Failed to resume the cluster: %v", err)
	}

	return nil
}
