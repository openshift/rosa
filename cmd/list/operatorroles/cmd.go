/*
Copyright (c) 2023 Red Hat, Inc.

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

package operatorroles

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	version string
	prefix  string
	cluster string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operatorrole", "operator-role", "operatorroles"},
	Short:   "List operator roles and policies",
	Long:    "List operator roles and policies for the current AWS account.",
	Example: `  # List all operator roles
  rosa list operator-roles`,
	Run:  run,
	Args: cobra.NoArgs,
}

const (
	versionFlag = "version"
	prefixFlag  = "prefix"
)

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false
	flags.StringVar(
		&args.version,
		versionFlag,
		"",
		"List only operator-roles that are associated with the given version.",
	)
	flags.StringVar(
		&args.prefix,
		prefixFlag,
		"",
		"List only operator-roles that are associated with the given prefix."+
			" The prefix must match up to openshift|kube-system",
	)
	interactive.AddFlag(flags)
	ocm.AddOptionalClusterFlag(Cmd)
	output.AddFlag(Cmd)
	output.AddHideEmptyColumnsFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	versionList, err := ocm.GetVersionMinorList(r.OCMClient)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	_, err = r.OCMClient.ValidateVersion(args.version, versionList,
		r.Cluster.Version().ChannelGroup(), r.Cluster.AWS().STS().RoleARN() == "", r.Cluster.Hypershift().Enabled())
	if err != nil {
		r.Reporter.Errorf("Version '%s' is invalid", args.version)
		os.Exit(1)
	}

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Fetching operator roles")
		spin.Start()
	}

	clusterId := ""
	if cmd.Flags().Changed("cluster") {
		clusterKey := r.GetClusterKey()

		cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
		if err != nil {
			r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
		clusterId = cluster.ID()
		args.prefix = cluster.AWS().STS().OperatorRolePrefix()
	}

	operatorsMap, err := r.AWSClient.ListOperatorRoles(args.version, clusterId, args.prefix)
	prefixes := helper.MapKeys(operatorsMap)
	helper.SortStringRespectLength(prefixes)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		r.Reporter.Errorf("Failed to get operator roles: %v", err)
		os.Exit(1)
	}

	if len(operatorsMap) == 0 {
		noOperatorRolesOutput := "No operator roles available"
		if args.version != "" {
			noOperatorRolesOutput = fmt.Sprintf("%s in version '%s'", noOperatorRolesOutput, args.version)
		}
		if args.prefix != "" {
			if _, ok := operatorsMap[args.prefix]; !ok {
				r.Reporter.Infof("No operator roles available for prefix '%s'", args.prefix)
				os.Exit(0)
			}
		}
		r.Reporter.Infof(noOperatorRolesOutput)
		os.Exit(0)
	}
	if output.HasFlag() {
		var resource interface{} = operatorsMap
		if args.prefix != "" {
			resource = operatorsMap[args.prefix]
		}
		err = output.Print(resource)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if clusterId != "" {
		for key, value := range operatorsMap {
			if value[0].ClusterID == clusterId {
				args.prefix = key
			}
		}
	}
	if args.prefix == "" {
		// Define headers once
		headers := []string{"ROLE PREFIX", "AMOUNT IN BUNDLE"}

		// Prepare table data
		var tableData [][]string
		for _, key := range prefixes {
			row := []string{
				key,
				fmt.Sprintf("%d", len(operatorsMap[key])),
			}
			tableData = append(tableData, row)
		}

		// Process headers and data if hiding empty columns
		if output.ShouldHideEmptyColumns() {
			tableData = output.RemoveEmptyColumns(headers, tableData)
		} else {
			tableData = append([][]string{headers}, tableData...)
		}

		// Print the table
		output.BuildTable(writer, "\t", tableData)

		// Check for flush errors
		if err := writer.Flush(); err != nil {
			_ = r.Reporter.Errorf("Failed to flush output: %v", err)
			os.Exit(1)
		}
		if !interactive.Enabled() {
			os.Exit(0)
		}
		if !confirm.Prompt(true, "Would you like to detail a specific prefix") {
			os.Exit(0)
		}
		args.prefix, err = interactive.GetOption(interactive.Input{
			Question: "Operator Role Prefix",
			Help:     cmd.Flags().Lookup("prefix").Usage,
			Options:  prefixes,
			Default:  prefixes[0],
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid OIDC Config ID: %s", err)
			os.Exit(1)
		}
	}
	if args.prefix != "" {
		if _, ok := operatorsMap[args.prefix]; !ok {
			noOperatorRolesPrefixOutput := fmt.Sprintf("No operator roles available for prefix '%s'", args.prefix)
			if args.version != "" {
				noOperatorRolesPrefixOutput =
					fmt.Sprintf("%s in version '%s'", noOperatorRolesPrefixOutput, args.version)
			}
			r.Reporter.Infof(noOperatorRolesPrefixOutput)
			os.Exit(0)
		}
		hasClusterUsingOperatorRolesPrefix, err := r.OCMClient.HasAClusterUsingOperatorRolesPrefix(args.prefix)
		if err != nil {
			r.Reporter.Errorf("There was a problem checking if any clusters"+
				" are using Operator Roles Prefix '%s' : %v", args.prefix, err)
			os.Exit(1)
		}

		headers := []string{"OPERATOR NAME", "OPERATOR NAMESPACE", "ROLE NAME", "ROLE ARN",
			"CLUSTER ID", "VERSION", "POLICIES", "AWS Managed", "IN USE"}
		var tableData [][]string
		for _, operatorRole := range operatorsMap[args.prefix] {
			awsManaged := "No"
			inUse := "No"
			if operatorRole.ManagedPolicy {
				awsManaged = "Yes"
			}
			if hasClusterUsingOperatorRolesPrefix {
				inUse = "Yes"
			}
			row := []string{
				operatorRole.OperatorName,
				operatorRole.OperatorNamespace,
				operatorRole.RoleName,
				operatorRole.RoleARN,
				operatorRole.ClusterID,
				operatorRole.Version,
				output.PrintStringSlice(operatorRole.AttachedPolicies),
				awsManaged,
				inUse,
			}
			tableData = append(tableData, row)
		}

		if output.ShouldHideEmptyColumns() {
			tableData = output.RemoveEmptyColumns(headers, tableData)
		} else {
			tableData = append([][]string{headers}, tableData...)
		}

		output.BuildTable(writer, "\t", tableData)

		if err := writer.Flush(); err != nil {
			_ = r.Reporter.Errorf("Failed to flush output: %v", err)
			os.Exit(1)
		}
	}
}
