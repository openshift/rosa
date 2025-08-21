/*
Copyright (c) 2021 Red Hat, Inc.

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

package instancetypes

import (
	"fmt"
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	interactiveRoles "github.com/openshift/rosa/pkg/interactive/roles"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = makeCmd()

func makeCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "instance-types",
		Aliases: []string{"instancetypes"},
		Short:   "List Instance types",
		Long:    "List Instance types that are available for use with ROSA.",
		Example: `  # List all instance types
	rosa list instance-types`,
		Run:  run,
		Args: cobra.NoArgs,
	}

	output.AddHideEmptyColumnsFlag(cmd)
	return cmd
}

func init() {
	initFlags(Cmd)
}

var args struct {
	region               string
	installerRoleArn     string
	externalId           string
	hostedClusterEnabled bool
}

const (
	InstallerRoleArnFlag = "role-arn"
)

func initFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(
		&args.hostedClusterEnabled,
		"hosted-cp",
		false,
		"Enable the use of Hosted Control Planes",
	)

	flags.StringVar(
		&args.externalId,
		"external-id",
		"",
		"An optional unique identifier that might be required when you assume a role in another account.",
	)

	// normalizing installer role argument to support deprecated flag
	flags.SetNormalizeFunc(arguments.NormalizeFlags)
	flags.StringVar(
		&args.installerRoleArn,
		InstallerRoleArnFlag,
		"",
		"STS Role ARN with get secrets permission.",
	)

	arguments.AddRegionFlag(flags)
	output.AddFlag(cmd)
	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func checkInteractiveModeNeeded(cmd *cobra.Command) {
	installerRoleArnNotSet := (!cmd.Flags().Changed(InstallerRoleArnFlag) || args.installerRoleArn == "") &&
		!confirm.Yes()
	if installerRoleArnNotSet {
		interactive.Enable()
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command) error {
	checkInteractiveModeNeeded(cmd)
	r.Reporter.Debugf("Fetching instance types")
	var machineTypes ocm.MachineTypeList
	if cmd.Flags().Changed("region") {
		if interactive.Enabled() || (confirm.Yes() && args.installerRoleArn == "") {
			args.installerRoleArn = interactiveRoles.
				GetInstallerRoleArn(
					r,
					cmd,
					args.installerRoleArn,
					"",
					r.AWSClient.FindRoleARNs,
				)
		}
		var availabilityZones []string
		roleArn := ""
		regionList, _, err := r.OCMClient.GetRegionList(false, args.installerRoleArn, args.externalId, "",
			r.AWSClient, args.hostedClusterEnabled, false)
		if err != nil {
			return err
		}
		if found := helper.Contains(regionList, arguments.GetRegion()); !found {
			return fmt.Errorf("Region '%s' not found.", arguments.GetRegion())
		}

		availableMachineTypes, err := r.OCMClient.GetAvailableMachineTypesInRegion(arguments.GetRegion(),
			availabilityZones, roleArn, r.AWSClient, args.externalId)
		if err != nil {
			return fmt.Errorf("Failed to fetch instance types: %v", err)
		}
		machineTypes = availableMachineTypes
	} else {
		availableMachineTypes, err := r.OCMClient.GetAvailableMachineTypes()
		if err != nil {
			return fmt.Errorf("Failed to fetch instance types: %v", err)
		}
		machineTypes = availableMachineTypes
	}

	if output.HasFlag() {
		var instanceTypes []*cmv1.MachineType
		for _, machine := range machineTypes.Items {
			instanceTypes = append(instanceTypes, machine.MachineType)
		}
		err := output.Print(instanceTypes)
		if err != nil {
			return err
		}

		return nil
	}

	if len(machineTypes.Items) == 0 {
		return fmt.Errorf("There are no machine types supported for your account. Contact Red Hat support.")
	}

	headers := []string{"ID", "CATEGORY", "CPU_CORES", "MEMORY"}
	var tableData [][]string
	for _, machine := range machineTypes.Items {
		if !machine.Available {
			continue
		}
		availableMachine := machine.MachineType
		row := []string{
			availableMachine.ID(),
			string(availableMachine.Category()),
			fmt.Sprintf("%d", int(availableMachine.CPU().Value())),
			ByteCountIEC(int(availableMachine.Memory().Value()), availableMachine.Memory().Unit()),
		}
		tableData = append(tableData, row)
	}

	if output.ShouldHideEmptyColumns() {
		tableData = output.RemoveEmptyColumns(headers, tableData)
	} else {
		tableData = append([][]string{headers}, tableData...)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	output.BuildTable(writer, "\t", tableData)

	if err := writer.Flush(); err != nil {
		return err
	}

	return nil
}

func ByteCountIEC(b int, uValue string) string {
	var unit int
	if uValue == "B" {
		unit = 1024
	}
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= int64(unit)
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
