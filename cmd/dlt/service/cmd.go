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
	"strings"

	msv1 "github.com/openshift-online/ocm-sdk-go/servicemgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	// ID of service
	ID string
}

var Cmd = &cobra.Command{
	Use:     "managed-service",
	Aliases: []string{"appliance", "service"},
	Short:   "Deletes a managed-service",
	Long:    "Deletes a managed-service.",
	Example: `  # Delete a managed-service with ID "aabbcc"
  rosa delete managed-service --id=aabbcc`,
	Run:    run,
	Hidden: true,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(&args.ID,
		"id",
		"",
		"The ID of the service to be deleted.")
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	if args.ID == "" {
		r.Reporter.Errorf("id not specified.")
		cmd.Help()
		os.Exit(1)
	}

	if !confirm.Confirm("delete service with id '%s'", args.ID) {
		os.Exit(0)
	}

	// First get the service to report additional resources
	// that must be manually deleted.
	service, err := r.OCMClient.GetManagedService(ocm.DescribeManagedServiceArgs{ID: args.ID})
	if err != nil {
		r.Reporter.Errorf("Failed to get Managed Service: %s", err)
		os.Exit(1)
	}

	r.Reporter.Debugf("Deleting service with id %q", args.ID)
	_, err = r.OCMClient.DeleteManagedService(args)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	r.Reporter.Infof("Service %q will start uninstalling now", args.ID)

	if service.Cluster().AWS().STS().RoleARN() != "" {
		r.Reporter.Infof(
			"Your service %q will be deleted by the following objects may remain",
			args.ID,
		)
		if len(service.Cluster().AWS().STS().OperatorIAMRoles()) > 0 {
			str := "Operator IAM Roles:"
			for _, operatorIAMRole := range service.Cluster().AWS().STS().OperatorIAMRoles() {
				str = fmt.Sprintf("%s"+
					" - %s\n", str,
					operatorIAMRole.RoleARN())
			}
			r.Reporter.Infof("%s", str)
		}
		r.Reporter.Infof("OIDC Provider : %q\n", service.Cluster().AWS().STS().OIDCEndpointURL())
		r.Reporter.Infof("Once the service is uninstalled use the following commands to remove the " +
			"above aws resources.\n")
		commands := buildCommands(service.Cluster())
		fmt.Print(commands, "\n")
	}
}

func buildCommands(cluster *msv1.Cluster) string {
	commands := []string{}
	deleteOperatorRole := fmt.Sprintf("\trosa delete operator-roles -c %s", cluster.Id())
	deleteOIDCProvider := fmt.Sprintf("\trosa delete oidc-provider -c %s", cluster.Id())
	commands = append(commands, deleteOperatorRole, deleteOIDCProvider)
	return strings.Join(commands, "\n")
}
