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
	"os"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/aws"
	clusterprovider "github.com/openshift/moactl/pkg/cluster"
	"github.com/openshift/moactl/pkg/interactive"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var args struct {
	clusterKey string

	// Basic options
	expirationTime     string
	expirationDuration time.Duration

	// Scaling options
	computeNodes int

	// Networking options
	private bool

	// Access control options
	clusterAdmins bool
}

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Edit cluster",
	Long:  "Edit cluster.",
	Example: `  # Edit a cluster named "mycluster" to make it private
  rosa edit cluster mycluster --private

  # Enable the cluster-admins group using the --cluster flag
  rosa edit cluster --cluster=mycluster --enable-cluster-admins

  # Edit all options interactively
  rosa edit cluster -c mycluster --interactive`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to edit.",
	)

	// Basic options
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
		"Expire cluster after a relative duration like 2h, 8h, 72h. Only one of expiration-time / expiration may be used.",
	)
	// Cluster expiration is not supported in production
	flags.MarkHidden("expiration-time")
	flags.MarkHidden("expiration")

	// Scaling options
	flags.IntVar(
		&args.computeNodes,
		"compute-nodes",
		0,
		"Number of worker nodes to provision per zone. Single zone clusters need at least 2 nodes, "+
			"while multizone clusters need at least 3 nodes (1 per zone) for resiliency.",
	)

	// Networking options
	flags.BoolVar(
		&args.private,
		"private",
		false,
		"Restrict master API endpoint to direct, private connectivity.",
	)

	// Access control options
	flags.BoolVar(
		&args.clusterAdmins,
		"enable-cluster-admins",
		false,
		"Enable the cluster-admins role for your cluster.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()

	// Check command line arguments:
	clusterKey := args.clusterKey
	if clusterKey == "" {
		if len(argv) != 1 {
			reporter.Errorf(
				"Expected exactly one command line argument or flag containing the name " +
					"or identifier of the cluster",
			)
			os.Exit(1)
		}
		clusterKey = argv[0]
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	if !clusterprovider.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	isInteractive := interactive.Enabled()
	if !isInteractive {
		changedFlags := false
		for _, flag := range []string{"compute-nodes", "private", "enable-cluster-admins"} {
			if cmd.Flags().Changed(flag) {
				changedFlags = true
			}
		}
		if !changedFlags {
			isInteractive = true
		}
	}

	logger := logging.CreateLoggerOrExit(reporter)

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

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := clusterprovider.GetCluster(ocmClient.Clusters(), clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Validate flags:
	expiration, err := validateExpiration()
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}

	if interactive.Enabled() {
		reporter.Infof("Interactive mode enabled.\n" +
			"Any optional fields can be ignored and will not be updated.")
	}

	var private *bool
	var privateValue bool
	if cmd.Flags().Changed("private") {
		privateValue = args.private
		private = &privateValue
	} else if isInteractive {
		privateValue = cluster.API().Listening() == cmv1.ListeningMethodInternal
	}

	if isInteractive {
		privateValue, err = interactive.GetBool(interactive.Input{
			Question: "Private cluster",
			Help:     cmd.Flags().Lookup("private").Usage,
			Default:  privateValue,
		})
		if err != nil {
			reporter.Errorf("Expected a valid private value: %s", err)
			os.Exit(1)
		}
		private = &privateValue
	}

	var clusterAdmins *bool
	var clusterAdminsValue bool
	if cmd.Flags().Changed("enable-cluster-admins") {
		clusterAdminsValue = args.clusterAdmins
		clusterAdmins = &clusterAdminsValue
	} else if isInteractive {
		clusterAdminsValue = cluster.ClusterAdminEnabled()
	}

	if isInteractive {
		clusterAdminsValue, err = interactive.GetBool(interactive.Input{
			Question: "Enable cluster admins",
			Help:     cmd.Flags().Lookup("enable-cluster-admins").Usage,
			Default:  clusterAdminsValue,
		})
		if err != nil {
			reporter.Errorf("Expected a valid cluster-admins value: %s", err)
			os.Exit(1)
		}
		clusterAdmins = &clusterAdminsValue
	}

	var computeNodes int
	if cmd.Flags().Changed("compute-nodes") {
		computeNodes = args.computeNodes
	} else if isInteractive {
		computeNodes = cluster.Nodes().Compute()
	}

	if isInteractive {
		computeNodes, err = interactive.GetInt(interactive.Input{
			Question: "Compute nodes",
			Help:     cmd.Flags().Lookup("compute-nodes").Usage,
			Default:  computeNodes,
		})
		if err != nil {
			reporter.Errorf("Expected a valid number of compute nodes: %s", err)
			os.Exit(1)
		}
	}

	clusterConfig := clusterprovider.Spec{
		Expiration:    expiration,
		ComputeNodes:  computeNodes,
		Private:       private,
		ClusterAdmins: clusterAdmins,
	}

	reporter.Debugf("Updating cluster '%s'", clusterKey)
	err = clusterprovider.UpdateCluster(ocmClient.Clusters(), clusterKey, awsCreator.ARN, clusterConfig)
	if err != nil {
		reporter.Errorf("Failed to update cluster: %v", err)
		os.Exit(1)
	}
}

func validateExpiration() (expiration time.Time, err error) {
	// Validate options
	if len(args.expirationTime) > 0 && args.expirationDuration != 0 {
		err = errors.New("At most one of 'expiration-time' or 'expiration' may be specified")
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

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}
