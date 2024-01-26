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
	"strings"

	"github.com/openshift/rosa/cmd/create/oidcprovider"
	"github.com/openshift/rosa/cmd/create/operatorroles"
	clusterdescribe "github.com/openshift/rosa/cmd/describe/cluster"
	installLogs "github.com/openshift/rosa/cmd/logs/install"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create cluster",
	Long:  "Create cluster.",
	Example: `  # Create a cluster named "mycluster"
  rosa create cluster --cluster-name=mycluster

  # Create a cluster in the us-east-2 region
  rosa create cluster --cluster-name=mycluster --region=us-east-2`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	Cmd.RegisterFlagCompletionFunc("network-type", networkTypeCompletion)
	aws.AddModeFlag(Cmd)
	interactive.AddFlag(flags)
	output.AddFlag(Cmd)
	confirm.AddFlag(flags)
}

func networkTypeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return ocm.NetworkTypes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, _ []string) {
	rawOpts := NewOptions()
	rawOpts.AddFlags(cmd.Flags())

	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	opts, err := rawOpts.Complete(cmd.Flags(), r)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if opts == nil {
		os.Exit(0)
	}

	if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof("Creating cluster '%s'", opts.ClusterName)
		if interactive.Enabled() {
			command := buildCommand(opts.Spec, opts.OperatorRolesPrefix, opts.ExpectedOperatorRolePath,
				opts.IsAvailabilityZonesSet || opts.SelectAvailabilityZones, opts.Labels, opts.Properties,
				opts.ClusterAdminPassword, opts.ClassicOidcConfig, opts.ExpirationDuration)
			r.Reporter.Infof("To create this cluster again in the future, you can run:\n   %s", command)
		}
		r.Reporter.Infof("To view a list of clusters and their status, run 'rosa list clusters'")
	}

	cluster, err := r.OCMClient.CreateCluster(opts.Spec)
	if err != nil {
		if opts.DryRun {
			r.Reporter.Errorf("Creating cluster '%s' should fail: %s", opts.ClusterName, err)
		} else {
			r.Reporter.Errorf("Failed to create cluster: %s", err)
		}
		os.Exit(1)
	}

	if opts.DryRun {
		r.Reporter.Infof(
			"Creating cluster '%s' should succeed. Run without the '--dry-run' flag to create the cluster.",
			opts.ClusterName)
		os.Exit(0)
	}

	if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof("Cluster '%s' has been created.", opts.ClusterName)
		r.Reporter.Infof(
			"Once the cluster is installed you will need to add an Identity Provider " +
				"before you can login into the cluster. See 'rosa create idp --help' " +
				"for more information.")
	}

	clusterdescribe.Cmd.Run(clusterdescribe.Cmd, []string{cluster.ID()})

	if opts.IsSTS {
		if opts.AWSMode != "" {
			if !output.HasFlag() || r.Reporter.IsTerminal() {
				r.Reporter.Infof("Preparing to create operator roles.")
			}
			err := operatorroles.Cmd.RunE(operatorroles.Cmd, []string{opts.ClusterName, opts.AWSMode, opts.PermissionsBoundary})
			if err != nil {
				r.Reporter.Errorf("There was a problem creating operator roles: %v", err)
				os.Exit(1)
			}
			if !output.HasFlag() || r.Reporter.IsTerminal() {
				r.Reporter.Infof("Preparing to create OIDC Provider.")
			}
			oidcprovider.Cmd.Run(oidcprovider.Cmd, []string{opts.ClusterName, opts.AWSMode, ""})
		} else {
			output := ""
			if len(opts.OperatorRoles) == 0 {
				rolesCMD := fmt.Sprintf("rosa create operator-roles --cluster %s", opts.ClusterName)
				if opts.PermissionsBoundary != "" {
					rolesCMD = fmt.Sprintf("%s --permissions-boundary %s", rolesCMD, opts.PermissionsBoundary)
				}
				output = fmt.Sprintf("%s\t%s\n", output, rolesCMD)
			}
			oidcEndpointURL := cluster.AWS().STS().OIDCEndpointURL()
			oidcProviderExists, err := r.AWSClient.HasOpenIDConnectProvider(oidcEndpointURL, r.Creator.AccountID)
			if err != nil {
				if strings.Contains(err.Error(), "AccessDenied") {
					r.Reporter.Debugf("Failed to verify if OIDC provider exists: %s", err)
				} else {
					r.Reporter.Errorf("Failed to verify if OIDC provider exists: %s", err)
					os.Exit(1)
				}
			}
			if !oidcProviderExists {
				oidcCMD := "rosa create oidc-provider"
				oidcCMD = fmt.Sprintf("%s --cluster %s", oidcCMD, opts.ClusterName)
				output = fmt.Sprintf("%s\t%s\n", output, oidcCMD)
			}
			if output != "" {
				output = fmt.Sprintf("Run the following commands to continue the cluster creation:\n\n%s",
					output)
				r.Reporter.Infof(output)
			}
		}
	}

	if opts.Watch {
		installLogs.Cmd.Run(installLogs.Cmd, []string{opts.ClusterName})
	} else if !output.HasFlag() || r.Reporter.IsTerminal() {
		r.Reporter.Infof(
			"To determine when your cluster is Ready, run 'rosa describe cluster -c %s'.",
			opts.ClusterName,
		)
		r.Reporter.Infof(
			"To watch your cluster installation logs, run 'rosa logs install -c %s --watch'.",
			opts.ClusterName,
		)
	}
}
