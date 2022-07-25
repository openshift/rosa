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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args ocm.DescribeManagedServiceArgs

var Cmd = &cobra.Command{
	Use:     "managed-service",
	Aliases: []string{"appliance", "service"},
	Short:   "Show details of a managed-service",
	Long:    "Show details of a managed-service",
	Example: `  # Describe a managed-service with id aaabbbccc"
  rosa describe managed-service --id=aaabbbccc`,
	Run:    run,
	Hidden: true,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.ID,
		"id",
		"",
		"The id of the service to describe",
	)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	if args.ID == "" {
		r.Reporter.Errorf("id not specified.")
		cmd.Help()
		os.Exit(1)
	}

	// Try to find the cluster:
	r.Reporter.Debugf("Loading service with id %q", args.ID)
	service, err := r.OCMClient.GetManagedService(args)
	if err != nil {
		r.Reporter.Errorf("Failed to get service with id %q: %s", args.ID, output.ErrorToString(err))
		os.Exit(1)
	}

	fmt.Printf(`%-28s%s
%-28s%s
%-28s%s
%-28s%s
%-28s%s
%-28s%s
%-28s%s
`,
		"Id:", service.ID(),
		"Href:", service.HREF(),
		"Service type:", service.Service(),
		"Service State:", service.ServiceState(),
		"Cluster Name:", service.Cluster().Name(),
		"Created At:", service.CreatedAt(),
		"Updated At:", service.UpdatedAt())

	parameters := service.Parameters()
	if len(parameters) > 0 {
		fmt.Printf("%-28s\n", "Parameters:")
	}
	for _, param := range parameters {
		fmt.Printf("\t%-28q: %q\n",
			param.ID(),
			param.Value())
	}
}
