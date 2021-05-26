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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/ocm/machines"
	rprtr "github.com/openshift/rosa/pkg/reporter"
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

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
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

	// Get the client for the OCM collection of clusters:
	ocmClient := ocmConnection.ClustersMgmt().V1()

	reporter.Debugf("Fetching instance types")
	machineTypes, err := machines.GetMachineTypes(ocmClient)
	if err != nil {
		reporter.Errorf("Failed to fetch instance types: %v", err)
		os.Exit(1)
	}

	if len(machineTypes) == 0 {
		reporter.Warnf("There are no machine types supported for your account. Contact Red Hat support.")
		os.Exit(1)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\tCATEGORY\tCPU_CORES\tMEMORY\t\n")

	for _, machine := range machineTypes {
		fmt.Fprintf(writer,
			"%s\t%s\t%d\t%s\n",
			machine.ID(), machine.Category(), int(machine.CPU().Value()), ByteCountIEC(int(machine.Memory().Value()),
				machine.Memory().Unit()),
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
