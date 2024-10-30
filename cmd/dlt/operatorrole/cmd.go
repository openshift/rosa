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

package operatorrole

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	PrefixFlag = "prefix"
)

var args struct {
	prefix string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operatorrole"},
	Short:   "Delete Operator Roles",
	Long:    "Cleans up operator roles of deleted STS cluster.",
	Example: `  # Delete Operator roles for cluster named "mycluster"
  rosa delete operator-roles --cluster=mycluster`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.prefix,
		PrefixFlag,
		"",
		"Operator role prefix, this flag needs to be used in case of reusable OIDC Config",
	)

	ocm.AddOptionalClusterFlag(Cmd)
	interactive.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
}

const (
	hypershiftSubscriptionPlanId = "MOA-HostedControlPlane"
)

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	if !cmd.Flag("cluster").Changed && !cmd.Flag(PrefixFlag).Changed {
		r.Reporter.Errorf("Either a cluster key or a prefix must be specified.")
		os.Exit(1)
	}

	if interactive.Enabled() {
		mode, err = interactive.GetOptionMode(cmd, mode, "Operator roles deletion mode")
		if err != nil {
			r.Reporter.Errorf("Expected a valid operator role deletion mode: %s", err)
			os.Exit(1)
		}
	}

	clusterKey := ""
	var foundOperatorRoles []string
	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	fetchingReporterOutput := "Fetching operator roles for the"
	if args.prefix == "" {
		clusterKey = r.GetClusterKey()
		r.Reporter.Debugf("Loading cluster '%s'", clusterKey)
		sub, err := r.OCMClient.GetClusterUsingSubscription(clusterKey, r.Creator)
		if err != nil {
			if errors.GetType(err) == errors.Conflict {
				r.Reporter.Errorf("More than one cluster found with the same name '%s'. Please "+
					"use cluster ID instead", clusterKey)
				os.Exit(1)
			}
			r.Reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
		if sub != nil {
			clusterKey = sub.ClusterID()
		}
		cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
		if err != nil {
			if errors.GetType(err) != errors.NotFound {
				r.Reporter.Errorf("Error validating cluster '%s': %v", clusterKey, err)
				os.Exit(1)
			} else if sub == nil {
				r.Reporter.Errorf("Failed to get cluster '%s': %v", r.ClusterKey, err)
				os.Exit(1)
			}
		}

		if cluster != nil && cluster.ID() != "" {
			r.Reporter.Errorf("Cluster '%s' is in '%s' state. Operator roles can be deleted only for the "+
				"uninstalled clusters", cluster.ID(), cluster.State())
			os.Exit(1)
		}
		isHypershift := false
		if cluster != nil {
			isHypershift = cluster.Hypershift().Enabled()
		} else {
			subPlanId := sub.Plan().ID()
			isHypershift = subPlanId == hypershiftSubscriptionPlanId
		}
		if spin != nil {
			fetchingReporterOutput = fmt.Sprintf("%s cluster: %s", fetchingReporterOutput, clusterKey)
			r.Reporter.Infof("%s", fetchingReporterOutput)
			spin.Start()
		}
		credRequests, err := r.OCMClient.GetCredRequests(isHypershift)
		if err != nil {
			r.Reporter.Errorf("Error getting operator credential request from OCM %s", err)
			os.Exit(1)
		}
		foundOperatorRoles, _ = r.AWSClient.GetOperatorRolesFromAccountByClusterID(sub.ClusterID(), credRequests)
	} else {
		if spin != nil {
			fetchingReporterOutput = fmt.Sprintf("%s prefix: %s", fetchingReporterOutput, args.prefix)
			r.Reporter.Infof("%s", fetchingReporterOutput)
			spin.Start()
		}
		hasClusterUsingOperatorRolesPrefix, err := r.OCMClient.HasAClusterUsingOperatorRolesPrefix(args.prefix)
		if err != nil {
			r.Reporter.Errorf("There was a problem checking if any clusters"+
				" are using Operator Roles Prefix '%s' : %v", args.prefix, err)
			os.Exit(1)
		}
		if hasClusterUsingOperatorRolesPrefix {
			if spin != nil {
				spin.Stop()
			}
			r.Reporter.Errorf("There are clusters using Operator Roles Prefix '%s', can't delete the IAM roles", args.prefix)
			os.Exit(1)
		}
		credRequests, err := r.OCMClient.GetAllCredRequests()
		if err != nil {
			r.Reporter.Errorf("Error getting operator credential request from OCM %v", err)
			os.Exit(1)
		}
		foundOperatorRoles, err = r.AWSClient.GetOperatorRolesFromAccountByPrefix(args.prefix, credRequests)
		if err != nil {
			r.Reporter.Errorf("There was a problem retrieving the Operator Roles from AWS: %v", err)
			os.Exit(1)
		}
	}

	if len(foundOperatorRoles) == 0 {
		if spin != nil {
			spin.Stop()
		}
		noRoleOutput := "There are no operator roles to delete"
		if args.prefix != "" {
			noRoleOutput = fmt.Sprintf("%s for prefix '%s'", noRoleOutput, args.prefix)
		} else {
			noRoleOutput = fmt.Sprintf("%s for cluster '%s'", noRoleOutput, clusterKey)
		}
		r.Reporter.Infof("%s", noRoleOutput)
		return
	}
	if spin != nil {
		spin.Stop()
	}

	_, roleARN, err := r.AWSClient.CheckRoleExists(foundOperatorRoles[0])
	if err != nil {
		r.Reporter.Errorf("Failed to get '%s' role ARN", foundOperatorRoles[0])
		os.Exit(1)
	}
	managedPolicies, err := r.AWSClient.HasManagedPolicies(roleARN)
	if err != nil {
		r.Reporter.Errorf("Failed to determine if cluster has managed policies: %v", err)
		os.Exit(1)
	}

	errOccured := false
	switch mode {
	case interactive.ModeAuto:
		r.OCMClient.LogEvent("ROSADeleteOperatorroleModeAuto", nil)
		for _, role := range foundOperatorRoles {
			if !confirm.Prompt(true, "Delete the operator role '%s'?", role) {
				continue
			}
			r.Reporter.Infof("Deleting operator role '%s'", role)
			if spin != nil {
				spin.Start()
			}
			err := r.AWSClient.DeleteOperatorRole(role, managedPolicies)

			if err != nil {
				if spin != nil {
					spin.Stop()
				}
				r.Reporter.Warnf("There was an error deleting the Operator Role or Policies: %s", err)
				errOccured = true
				continue
			}
			if spin != nil {
				spin.Stop()
			}
		}
		if !errOccured {
			r.Reporter.Infof("Successfully deleted the operator roles")
		}
	case interactive.ModeManual:
		r.OCMClient.LogEvent("ROSADeleteOperatorroleModeManual", nil)
		policyMap, arbitraryPolicyMap, err := r.AWSClient.GetOperatorRolePolicies(foundOperatorRoles)
		if err != nil {
			r.Reporter.Errorf("There was an error getting the policy: %v", err)
			os.Exit(1)
		}
		commands := buildCommand(r, foundOperatorRoles, policyMap, arbitraryPolicyMap, managedPolicies)
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to delete the Operator roles and policies:\n")
		}
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		os.Exit(1)
	}
}

func buildCommand(r *rosa.Runtime, roleNames []string, policyMap map[string][]string,
	arbitraryPolicyMap map[string][]string, managedPolicies bool) string {
	commands := []string{}
	for _, roleName := range roleNames {
		policyARN := policyMap[roleName]
		arbitraryPolicyARN := arbitraryPolicyMap[roleName]
		detachPolicy := ""
		deletePolicy := ""
		deletePolicyVersion := ""
		if len(policyARN) > 0 {
			detachPolicy = awscb.NewIAMCommandBuilder().
				SetCommand(awscb.DetachRolePolicy).
				AddParam(awscb.RoleName, roleName).
				AddParam(awscb.PolicyArn, policyARN[0]).Build()
			commands = append(commands, detachPolicy)

			policyVersions, err := r.AWSClient.ListPolicyVersions(policyARN[0])
			if err != nil {
				fmt.Printf("Failed to list policy versions for %s: %v\n", policyARN[0], err)
				return ""
			}

			for _, version := range policyVersions {
				if !version.IsDefaultVersion {
					deletePolicyVersion = awscb.NewIAMCommandBuilder().
						SetCommand(awscb.DeletePolicyVersion).
						AddParam(awscb.PolicyArn, policyARN[0]).
						AddParam(awscb.VersionID, version.VersionID).Build()
					commands = append(commands, deletePolicyVersion)
				}
			}

			if !managedPolicies {
				deletePolicy = awscb.NewIAMCommandBuilder().
					SetCommand(awscb.DeletePolicy).
					AddParam(awscb.PolicyArn, policyARN[0]).
					Build()
				commands = append(commands, deletePolicy)
			}
		}
		for _, policy := range arbitraryPolicyARN {
			detachArbitraryPolicy := awscb.NewIAMCommandBuilder().
				SetCommand(awscb.DetachRolePolicy).
				AddParam(awscb.RoleName, roleName).
				AddParam(awscb.PolicyArn, policy).Build()
			commands = append(commands, detachArbitraryPolicy)
		}
		deleteRole := awscb.NewIAMCommandBuilder().
			SetCommand(awscb.DeleteRole).
			AddParam(awscb.RoleName, roleName).
			Build()
		commands = append(commands, deleteRole)
	}
	return strings.Join(commands, "\n")
}
