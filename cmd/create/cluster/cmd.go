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
	"os"
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
	"gitlab.cee.redhat.com/service/moactl/pkg/interactive"
	"gitlab.cee.redhat.com/service/moactl/pkg/logging"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm/machines"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm/properties"
	"gitlab.cee.redhat.com/service/moactl/pkg/ocm/versions"
	rprtr "gitlab.cee.redhat.com/service/moactl/pkg/reporter"
)

var args struct {
	// Basic options
	name               string
	region             string
	multiAZ            bool
	version            string
	expirationTime     string
	expirationDuration time.Duration

	// Scaling options
	computeMachineType string
	computeNodes       int

	// Networking options
	machineCIDR net.IPNet
	serviceCIDR net.IPNet
	podCIDR     net.IPNet
	hostPrefix  int
	private     bool
}

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create cluster",
	Long:  "Create cluster.",
	Example: `  # Create a cluster named "mycluster"
  moactl create cluster --name=mycluster

  # Create a cluster in the us-east-2 region
  moactl create cluster --name=mycluster --region=us-east-2`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	// Basic options
	flags.StringVarP(
		&args.name,
		"name",
		"n",
		"",
		"Name of the cluster. This will be used when generating a sub-domain for your cluster on openshiftapps.com.",
	)
	flags.StringVarP(
		&args.region,
		"region",
		"r",
		"",
		"AWS region where your worker pool will be located. (overrides the AWS_REGION environment variable)",
	)
	flags.StringVar(
		&args.version,
		"version",
		"",
		"Version of OpenShift that will be used to install the cluster, for example \"4.3.10\"",
	)
	flags.BoolVar(
		&args.multiAZ,
		"multi-az",
		false,
		"Deploy to multiple data centers.",
	)
	flags.StringVar(
		&args.expirationTime,
		"expiration-time",
		"",
		"Specific time when cluster should expire (RFC3339). Only one of expiration-time / expiration may be used.",
	)
	flags.DurationVar(
		&args.expirationDuration,
		"expiration",
		0,
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.\n",
	)

	// Scaling options
	flags.StringVar(
		&args.computeMachineType,
		"compute-machine-type",
		"",
		"Instance type for the compute nodes. Determines the amount of memory and vCPU allocated to each compute node.",
	)
	flags.IntVar(
		&args.computeNodes,
		"compute-nodes",
		0,
		"Number of worker nodes to provision per zone. Single zone clusters need at least 4 nodes, while multizone clusters need at least 9 nodes (3 per zone) for resiliency.\n",
	)

	flags.IPNetVar(
		&args.machineCIDR,
		"machine-cidr",
		net.IPNet{},
		"Block of IP addresses used by OpenShift while installing the cluster, for example \"10.0.0.0/16\".",
	)
	flags.IPNetVar(
		&args.serviceCIDR,
		"service-cidr",
		net.IPNet{},
		"Block of IP addresses for services, for example \"172.30.0.0/16\".",
	)
	flags.IPNetVar(
		&args.podCIDR,
		"pod-cidr",
		net.IPNet{},
		"Block of IP addresses from which Pod IP addresses are allocated, for example \"10.128.0.0/14\".",
	)
	flags.IntVar(
		&args.hostPrefix,
		"host-prefix",
		0,
		"Subnet prefix length to assign to each individual node. For example, if host prefix is set to \"23\", then each node is assigned a /23 subnet out of the given CIDR.",
	)
	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict master API endpoint and application routes to direct, private connectivity.",
	)
}

func run(_ *cobra.Command, _ []string) {
	// Create the reporter:
	reporter, err := rprtr.New().
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create reporter: %v\n", err)
		os.Exit(1)
	}

	// Create the logger:
	logger, err := logging.NewLogger().Build()
	if err != nil {
		reporter.Errorf("Failed to create logger: %v", err)
		os.Exit(1)
	}

	// Get cluster name
	name := args.name
	if name == "" {
		name, err = interactive.GetInput("Cluster name")
		if err != nil {
			reporter.Errorf("Expected a valid cluster name")
			os.Exit(1)
		}
	}

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	// Get AWS region
	region := args.region
	if region == "" {
		region = awsClient.GetRegion()
	}
	if region == "" {
		region, err = interactive.GetInput("AWS region")
		if err != nil {
			reporter.Errorf("Expected a valid AWS region")
			os.Exit(1)
		}
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Create the access key for the AWS user:
	awsAccessKey, err := awsClient.GetAccessKeyFromStack(aws.OsdCcsAdminStackName)
	if err != nil {
		reporter.Errorf("Failed to get access keys for user '%s'", aws.AdminUserName)
		os.Exit(1)
	}
	reporter.Debugf("Access key identifier is '%s'", awsAccessKey.AccessKeyID)
	reporter.Debugf("Secret access key is '%s'", awsAccessKey.SecretAccessKey)

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()
	ocmClient := ocmConnection.ClustersMgmt().V1()

	// Validate all remaining flags:
	version, err := validateVersion(ocmClient)
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	expiration, err := validateExpiration()
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	computeMachineType, err := validateMachineType(ocmClient)
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	// Create the cluster:
	clusterBuilder := cmv1.NewCluster().
		Name(name).
		DisplayName(name).
		MultiAZ(args.multiAZ).
		Product(
			cmv1.NewProduct().
				ID("moa"),
		).
		Region(
			cmv1.NewCloudRegion().
				ID(region),
		).
		AWS(
			cmv1.NewAWS().
				AccountID(awsCreator.AccountID).
				AccessKeyID(awsAccessKey.AccessKeyID).
				SecretAccessKey(awsAccessKey.SecretAccessKey),
		).
		Properties(map[string]string{
			properties.CreatorARN: awsCreator.ARN,
		})

	if version != "" {
		clusterBuilder = clusterBuilder.Version(
			cmv1.NewVersion().
				ID(version),
		)
		reporter.Debugf("Using OpenShift version '%s'", version)
	}

	if !expiration.IsZero() {
		clusterBuilder = clusterBuilder.ExpirationTimestamp(expiration)
	}

	if computeMachineType != "" || args.computeNodes != 0 {
		clusterNodesBuilder := cmv1.NewClusterNodes()
		if computeMachineType != "" {
			clusterNodesBuilder = clusterNodesBuilder.ComputeMachineType(
				cmv1.NewMachineType().ID(computeMachineType),
			)
			reporter.Debugf("Using machine type '%s'", computeMachineType)
		}
		if args.computeNodes != 0 {
			clusterNodesBuilder = clusterNodesBuilder.Compute(args.computeNodes)
		}
		clusterBuilder = clusterBuilder.Nodes(clusterNodesBuilder)
	}

	if !cidrIsEmpty(args.machineCIDR) || !cidrIsEmpty(args.serviceCIDR) || !cidrIsEmpty(args.podCIDR) || args.hostPrefix != 0 {
		networkBuilder := cmv1.NewNetwork()
		if !cidrIsEmpty(args.machineCIDR) {
			networkBuilder = networkBuilder.MachineCIDR(args.machineCIDR.String())
		}
		if !cidrIsEmpty(args.serviceCIDR) {
			networkBuilder = networkBuilder.ServiceCIDR(args.serviceCIDR.String())
		}
		if !cidrIsEmpty(args.podCIDR) {
			networkBuilder = networkBuilder.PodCIDR(args.podCIDR.String())
		}
		if args.hostPrefix != 0 {
			networkBuilder = networkBuilder.HostPrefix(args.hostPrefix)
		}
		clusterBuilder = clusterBuilder.Network(networkBuilder)
	}

	if args.private {
		clusterBuilder = clusterBuilder.API(
			cmv1.NewClusterAPI().
				Listening(cmv1.ListeningMethodInternal),
		)
	}

	cluster, err := clusterBuilder.Build()
	if err != nil {
		reporter.Errorf("Failed to create description of cluster: %v", err)
		os.Exit(1)
	}
	createClusterResponse, err := ocmClient.Clusters().Add().
		Body(cluster).
		Send()
	if err != nil {
		// Unwrap and clean up API errors:
		wrapped := strings.Split(err.Error(), ": ")
		errorMessage := wrapped[len(wrapped)-1]
		reporter.Errorf("Failed to create cluster: %v", errorMessage)
		os.Exit(1)
	}

	cluster = createClusterResponse.Body()
	clusterID := cluster.ID()
	clusterName := cluster.Name()
	reporter.Infof("Creating cluster with identifier '%s' and name '%s'", clusterID, clusterName)
	reporter.Infof("To view list of clusters and their status, run `moactl list clusters`")

	// Add tags to the AWS administrator user containing the identifier and name of the cluster:
	err = awsClient.TagUser(aws.AdminUserName, clusterID, clusterName)
	if err != nil {
		reporter.Warnf("Failed to add cluster tags to user '%s'", aws.AdminUserName)
	}

	reporter.Infof("Cluster '%s' has been created.", clusterName)
	reporter.Infof(
		"Once the cluster is 'Ready' you will need to add an Identity Provider " +
			"and define the list of cluster administrators. See `moactl create idp --help` " +
			"and `moactl create user --help` for more information.")
	reporter.Infof(
		"To determine when your cluster is Ready, run `moactl describe cluster %s`.",
		clusterName,
	)
}

func validateVersion(client *cmv1.Client) (version string, err error) {
	// Validate OpenShift versions
	version = args.version
	if version != "" {
		versionList := sets.NewString()
		versions, err := versions.GetVersions(client)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve versions: %s", err)
			return version, err
		}

		for _, v := range versions {
			versionList.Insert(v.ID())
		}

		// Check and set the cluster version
		if !versionList.Has("openshift-v" + version) {
			allVersions := strings.ReplaceAll(strings.Join(versionList.List(), " "), "openshift-v", "")
			err = fmt.Errorf("A valid version number must be specified\nValid versions: %s", allVersions)
			return version, err
		}

		version = "openshift-v" + version
	}

	return
}

func validateExpiration() (expiration time.Time, err error) {
	// Validate options
	if len(args.expirationTime) > 0 && args.expirationDuration != 0 {
		err = errors.New("At most one of `expiration-time` or `expiration` may be specified")
		return
	}

	// Parse the expiration options
	if len(args.expirationTime) > 0 {
		t, err := parseRFC3339(args.expirationTime)
		if err != nil {
			err = fmt.Errorf("Failed to parse expiration-time: %s", err)
			return expiration, err
		}

		expiration = t
	}
	if args.expirationDuration != 0 {
		// round up to the nearest second
		expiration = time.Now().Add(args.expirationDuration).Round(time.Second)
	}

	return
}

func validateMachineType(client *cmv1.Client) (machineType string, err error) {
	// Validate AWS machine types
	machineType = args.computeMachineType
	if machineType != "" {
		machineTypeList := sets.NewString()
		machineTypes, err := machines.GetMachineTypes(client)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve machine types: %s", err)
			return machineType, err
		}

		for _, v := range machineTypes {
			machineTypeList.Insert(v.ID())
		}

		// Check and set the cluster machineType
		if !machineTypeList.Has(machineType) {
			err = fmt.Errorf("A valid machine type must be specified\nValid types: %s", machineTypeList.List())
			return machineType, err
		}
	}

	return
}

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func cidrIsEmpty(cidr net.IPNet) bool {
	return cidr.String() == "<nil>"
}
