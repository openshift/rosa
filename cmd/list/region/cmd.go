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

package region

import (
	"fmt"
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	roleArnFlag = "role-arn"
)

var args struct {
	multiAZ       bool
	roleARN       string
	externalID    string
	hostedCluster bool
}

var Cmd = &cobra.Command{
	Use:     "regions",
	Aliases: []string{"region"},
	Short:   "List available regions",
	Long:    "List regions that are available for the current AWS account.",
	Example: `  # List all available regions
  rosa list regions`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.multiAZ,
		"multi-az",
		false,
		"List only regions with support for multiple availability zones",
	)
	flags.StringVar(
		&args.roleARN,
		"role-arn",
		"",
		"The Amazon Resource Name of the role that the API will assume to fetch available regions.",
	)
	flags.StringVar(
		&args.externalID,
		"external-id",
		"",
		"A unique identifier that might be required when you assume a role in another account",
	)
	flags.BoolVar(
		&args.hostedCluster,
		"hosted-cp",
		false,
		"List only regions with support for Hosted Control Planes",
	)

	output.AddFlag(Cmd)
	output.AddHideEmptyColumnsFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	callerIdentity, err := r.AWSClient.GetCallerIdentity()
	if err != nil {
		r.Reporter.Errorf("Failed to get caller identity: %v", err)
		os.Exit(1)
	}

	isUsingAssumedRole, err := aws.IsArnAssumedRole(*callerIdentity.Arn)
	if err != nil {
		r.Reporter.Errorf("Failed to check if role is an assumed role: %v", err)
		os.Exit(1)
	}

	if isUsingAssumedRole && r.Creator.IsSTS && !cmd.Flag(roleArnFlag).Changed {
		args.roleARN, err = interactive.GetString(interactive.Input{
			Question: "ARN of role the API will use to fetch regions",
			Help: "The AWS profile you are using is using an assumed role, please provide the ARN\n " +
				cmd.Flag(roleArnFlag).Usage,
			Default:  args.roleARN,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid role arn value: %v", err)
			os.Exit(1)
		}
	}

	// Try to find the cluster:
	r.Reporter.Debugf("Fetching regions")
	regions, err := r.OCMClient.GetRegions(args.roleARN, args.externalID)
	if err != nil {
		r.Reporter.Errorf("Failed to fetch regions: %v", err)
		os.Exit(1)
	}

	// Filter out unwanted regions
	var availableRegions []*cmv1.CloudRegion
	for _, region := range regions {
		if !region.Enabled() {
			continue
		}
		if cmd.Flags().Changed("multi-az") {
			if args.multiAZ != region.SupportsMultiAZ() {
				continue
			}
		}
		if cmd.Flags().Changed("hosted-cp") {
			if args.hostedCluster != region.SupportsHypershift() {
				continue
			}
		}
		availableRegions = append(availableRegions, region)
	}

	if output.HasFlag() {
		err = output.Print(availableRegions)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(availableRegions) == 0 {
		r.Reporter.Warnf("There are no regions available for this AWS account")
		os.Exit(1)
	}

	headers := []string{"ID", "NAME", "MULTI-AZ SUPPORT", "HOSTED-CP SUPPORT"}
	var tableData [][]string
	for _, region := range availableRegions {
		row := []string{
			region.ID(),
			region.DisplayName(),
			fmt.Sprintf("%t", region.SupportsMultiAZ()),
			fmt.Sprintf("%t", region.SupportsHypershift()),
		}
		tableData = append(tableData, row)
	}

	if output.ShouldHideEmptyColumns() {
		tableData = output.RemoveEmptyColumns(headers, tableData)
	} else {
		tableData = append([][]string{headers}, tableData...)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	output.BuildTable(writer, "\t\t", tableData)

	if err := writer.Flush(); err != nil {
		_ = r.Reporter.Errorf("Failed to flush output: %v", err)
		os.Exit(1)
	}
}
