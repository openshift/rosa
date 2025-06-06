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
	"fmt"
	"os"
	"text/tabwriter"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "clusters",
	Aliases: []string{"cluster"},
	Short:   "List clusters",
	Long:    "List clusters.",
	Example: `  # List all clusters
  rosa list clusters`,
	Args: cobra.NoArgs,
	Run:  run,
}

const clusterCount = 1000

var args struct {
	listAll        bool
	accountRoleArn string
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	output.AddFlag(Cmd)
	flags.BoolVarP(&args.listAll, "all", "a", false, "List all clusters across different AWS "+
		"accounts under the same Red Hat organization")
	flags.StringVar(&args.accountRoleArn, "account-role-arn", "", "List all clusters "+
		"using the account role identified by the ARN")
}

func listClustersUsingAccountRole(creator *aws.Creator, runtime *rosa.Runtime) ([]*v1.Cluster, error) {
	role, err := runtime.AWSClient.GetAccountRoleByArn(args.accountRoleArn)
	if err != nil {
		return []*v1.Cluster{}, err
	}

	return runtime.OCMClient.GetClustersUsingAccountRole(creator, role, clusterCount)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWSWarnInsteadOfExit().WithOCM()
	defer r.Cleanup()

	// Retrieve the list of clusters:
	var creator *aws.Creator
	if args.listAll {
		creator = nil
	} else {
		creator = r.Creator
	}

	var clusters []*v1.Cluster
	var err error

	if args.accountRoleArn != "" {
		clusters, err = listClustersUsingAccountRole(creator, r)
	} else {
		clusters, err = r.OCMClient.GetClusters(creator, clusterCount)
	}

	if err != nil {
		r.Reporter.Errorf("Failed to get clusters: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(clusters)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(clusters) == 0 {
		r.Reporter.Infof("No clusters available")
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tNAME\tSTATE\tTOPOLOGY\n")
	for _, cluster := range clusters {
		typeOutput := "Classic"
		if cluster.AWS() != nil && cluster.AWS().STS() != nil && cluster.AWS().STS().Enabled() {
			typeOutput = "Classic (STS)"
		}
		if cluster.Hypershift().Enabled() {
			typeOutput = "Hosted CP"
		}
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			cluster.ID(),
			cluster.Name(),
			cluster.State(),
			typeOutput,
		)
	}
	writer.Flush()
}
