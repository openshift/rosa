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

	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "instance-types",
	Aliases: []string{"instancetypes"},
	Short:   "List Instance types",
	Long:    "List Instance types that are available for use with ROSA.",
	Example: `  # List all instance types
  rosa list instance-types`,
	Run: run,
}

func init() {
	output.AddFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	r.Reporter.Debugf("Fetching instance types")

	machineTypes, err := r.OCMClient.GetAvailableMachineTypes()
	if err != nil {
		r.Reporter.Errorf("Failed to fetch instance types: %v", err)
		os.Exit(1)
	}

	if len(machineTypes) == 0 {
		r.Reporter.Warnf("There are no machine types supported for your account. Contact Red Hat support.")
		os.Exit(1)
	}

	if output.HasFlag() {
		var instanceTypes []*cmv1.MachineType
		for _, machine := range machineTypes {
			instanceTypes = append(instanceTypes, machine.MachineType)
		}
		err = output.Print(instanceTypes)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tCATEGORY\tCPU_CORES\tMEMORY\t\n")

	for _, machine := range machineTypes {
		if !machine.Available {
			continue
		}
		availableMachine := machine.MachineType
		fmt.Fprintf(writer,
			"%s\t%s\t%d\t%s\n",
			availableMachine.ID(), availableMachine.Category(), int(availableMachine.CPU().Value()),
			ByteCountIEC(int(availableMachine.Memory().Value()),
				availableMachine.Memory().Unit()),
		)
	}
	writer.Flush()
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
