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

package machinepool

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/aws"
	c "github.com/openshift/moactl/pkg/cluster"
	"github.com/openshift/moactl/pkg/interactive"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	"github.com/openshift/moactl/pkg/ocm/machines"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var machinePoolKeyRE = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

var args struct {
	clusterKey   string
	name         string
	instanceType string
	replicas     int
	labels       string
}

var Cmd = &cobra.Command{
	Use:     "machinepool",
	Aliases: []string{"machinepools", "machine-pool", "machine-pools"},
	Short:   "Add machine pool to cluster",
	Long:    "Add a machine pool to the cluster.",
	Example: `  # Interactively add a machine pool to a cluster named "mycluster"
  rosa create machinepool --cluster=mycluster --interactive

  # Add a machine pool mp-1 with 3 replicas of m5.xlarge to a cluster
  rosa create machinepool --cluster=mycluster --replicas=3 --instance-type=m5.xlarge mp-1

  # Add a machine pool with labels to a cluster
  rosa create machinepool --cluster mycluster --replicas=2 --instance-type=r5.2xlarge --labels =foo=bar,bar=baz" mp-1`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to add the machine pool to (required).",
	)
	Cmd.MarkFlagRequired("cluster")

	flags.StringVar(
		&args.name,
		"name",
		"",
		"Name for the machine pool (required).",
	)

	flags.IntVar(
		&args.replicas,
		"replicas",
		0,
		"Count of machines for this machine pool (required).",
	)

	flags.StringVar(
		&args.instanceType,
		"instance-type",
		"",
		"Instance type that should be used.",
	)

	flags.StringVar(
		&args.labels,
		"labels",
		"",
		"Labels for machine pool. Format should be a comma-separated list of 'key=value'. "+
			"This list will overwrite any modifications made to Node labels on an ongoing basis.",
	)
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !c.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	// Create the AWS client:
	var err error
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

	// Get the client for the OCM collection of clusters:
	ocmClient := ocmConnection.ClustersMgmt().V1()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocm.GetCluster(ocmClient.Clusters(), clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Machine pool name:
	name := args.name
	if name == "" || interactive.Enabled() {
		name, err = interactive.GetString(interactive.Input{
			Question: "Machine pool name",
			Default:  name,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid name for the machine pool: %s", err)
			os.Exit(1)
		}
	}
	if !machinePoolKeyRE.MatchString(name) {
		reporter.Errorf("Expected a valid name for the machine pool")
		os.Exit(1)
	}

	// Number of replicas:
	replicas := args.replicas
	if interactive.Enabled() {
		replicas, err = interactive.GetInt(interactive.Input{
			Question: "Replicas",
			Help:     cmd.Flags().Lookup("replicas").Usage,
			Default:  replicas,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid number of replicas: %s", err)
			os.Exit(1)
		}
	}

	// Machine pool instance type:
	instanceType := args.instanceType
	instanceTypeList, err := machines.GetMachineTypeList(ocmClient)
	if err != nil {
		reporter.Errorf(fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	if interactive.Enabled() {
		if instanceType == "" {
			instanceType = instanceTypeList[0]
		}
		instanceType, err = interactive.GetOption(interactive.Input{
			Question: "Instance type",
			Help:     cmd.Flags().Lookup("instance-type").Usage,
			Options:  instanceTypeList,
			Default:  instanceType,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid machine type: %s", err)
			os.Exit(1)
		}
	}
	instanceType, err = machines.ValidateMachineType(instanceType, instanceTypeList)
	if err != nil {
		reporter.Errorf("Expected a valid machine type: %s", err)
		os.Exit(1)
	}

	labels := args.labels
	labelMap := make(map[string]string)
	if interactive.Enabled() {
		labels, err = interactive.GetString(interactive.Input{
			Question: "Labels",
			Help:     cmd.Flags().Lookup("labels").Usage,
			Default:  labels,
		})
		if err != nil {
			reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	if labels != "" {
		for _, label := range strings.Split(labels, ",") {
			if !strings.Contains(label, "=") {
				reporter.Errorf("Expected key=value format for labels")
				os.Exit(1)
			}
			tokens := strings.Split(label, "=")
			labelMap[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
		}
	}

	machinePool, err := cmv1.NewMachinePool().
		ID(name).
		Replicas(replicas).
		InstanceType(instanceType).
		Labels(labelMap).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create machine pool for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	_, err = ocmClient.Clusters().
		Cluster(cluster.ID()).
		MachinePools().
		Add().
		Body(machinePool).
		Send()
	if err != nil {
		reporter.Errorf("Failed to add machine pool to cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	reporter.Infof("Machine pool '%s' created successfully on cluster '%s'", name, clusterKey)
}
