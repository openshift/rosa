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

package service

import (
	"os"
	"text/tabwriter"

	msv1 "github.com/openshift-online/ocm-sdk-go/servicemgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use: "managed-services",
	Aliases: []string{"service", "services",
		"appliance", "appliances",
		"managed-service"},
	Short: "List managed-services",
	Long:  "List managed-services.",
	Example: `  # List all managed-services
  rosa list managed-services`,
	Args:   cobra.NoArgs,
	Run:    run,
	Hidden: true,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	output.AddFlag(Cmd)
	output.AddHideEmptyColumnsFlag(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	// Parse out CLI flags, then override positional arguments
	// This allows for arbitrary flags used for addon parameters
	_ = cmd.Flags().Parse(argv)

	servicesList, err := r.OCMClient.ListManagedServices(1000)
	if err != nil {
		r.Reporter.Errorf("Failed to retrieve list of managed services: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		outList := []*msv1.ManagedService{}
		servicesList.Each(func(srv *msv1.ManagedService) bool {
			outList = append(outList, srv)
			return true
		})
		output.Print(outList)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	headers := []string{"SERVICE_ID", "SERVICE", "SERVICE_STATE", "CLUSTER_NAME"}
	var tableData [][]string
	servicesList.Each(func(srv *msv1.ManagedService) bool {
		row := []string{
			srv.ID(),
			srv.Service(),
			srv.ServiceState(),
			srv.Cluster().Name(),
		}
		tableData = append(tableData, row)
		return true
	})

	if output.ShouldHideEmptyColumns() {
		tableData = output.RemoveEmptyColumns(headers, tableData)
	} else {
		tableData = append([][]string{headers}, tableData...)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	output.BuildTable(writer, "\t", tableData)

	if err := writer.Flush(); err != nil {
		_ = r.Reporter.Errorf("Failed to flush output: %v", err)
		os.Exit(1)
	}
}
