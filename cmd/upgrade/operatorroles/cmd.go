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

package operatorroles

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	upgradeVersion string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operator-role", "operatorroles"},
	Short:   "Upgrade operator IAM roles for a cluster.",
	Long:    "Upgrade cluster-specific operator IAM roles to latest version.",
	Example: `  # Upgrade cluster-specific operator IAM roles
  rosa upgrade operators-roles`,
	Args: cobra.NoArgs,
	Run:  run,
}

func init() {
	flags := Cmd.Flags()

	interactive.AddModeFlag(Cmd)
	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.upgradeVersion,
		"version",
		"",
		"Version of OpenShift that the cluster will be upgraded to",
	)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	latestPolicyVersion, err := r.OCMClient.GetLatestVersion(cluster.Version().ChannelGroup())
	if err != nil {
		r.Reporter.Errorf("Error getting latest version: %s", err)
		os.Exit(1)
	}

	/**
	we dont want to give this option to the end-user. Adding this as a support for srep if needed.
	*/
	if args.upgradeVersion != "" {
		version := args.upgradeVersion
		availableUpgrades := ocm.GetAvailableUpgradesByCluster(cluster)
		if len(availableUpgrades) == 0 {
			r.Reporter.Warnf("There are no available upgrades")
			os.Exit(0)
		}
		// Check that the version is valid
		validVersion := false
		for _, v := range availableUpgrades {
			if v == version {
				validVersion = true
				break
			}
		}
		if !validVersion {
			r.Reporter.Errorf("Expected a valid version to upgrade the cluster")
			os.Exit(1)
		}
	}

	operatorRoles, hasOperatorRoles := cluster.AWS().STS().GetOperatorIAMRoles()
	if !hasOperatorRoles || len(operatorRoles) == 0 {
		r.Reporter.Errorf("Cluster '%s' doesn't have any operator roles associated with it",
			clusterKey)
		os.Exit(1)
	}

	prefix, err := aws.GetPrefixFromInstallerAccountRole(cluster)
	if err != nil {
		r.Reporter.Errorf("Error getting account role prefix for the cluster '%s'",
			clusterKey)
		os.Exit(1)
	}
	unifiedPath, err := aws.GetPathFromAccountRole(cluster, aws.AccountRoles[aws.InstallerAccountRole].Name)
	if err != nil {
		r.Reporter.Errorf("Expected a valid path for '%s': %v", cluster.AWS().STS().RoleARN(), err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		r.Reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	managedPolicies := cluster.AWS().STS().ManagedPolicies()

	credRequests, err := r.OCMClient.GetCredRequests(cluster.Hypershift().Enabled())
	if err != nil {
		r.Reporter.Errorf("Error getting operator credential request from OCM %s", err)
		os.Exit(1)
	}

	policies, err := r.OCMClient.GetPolicies("OperatorRole")
	if err != nil {
		r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	if managedPolicies {
		mode, err = handleModeFlag(cmd, mode)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}

		hostedCPPolicies := aws.IsHostedCPManagedPolicies(cluster)

		err = roles.ValidateOperatorRolesManagedPolicies(r, cluster, credRequests, policies, mode, prefix, unifiedPath,
			args.upgradeVersion, hostedCPPolicies)
		if err != nil {
			r.Reporter.Errorf("Failed while validating managed policies: %v", err)
			os.Exit(1)
		}

		r.Reporter.Infof("Cluster '%s' operator roles have attached managed policies. "+
			"An upgrade isn't needed", cluster.Name())
		os.Exit(0)
	}

	isAccountRoleUpgradeNeed := false
	// If this is invoked from upgrade cluster then we already performed upgrade account roles

	//Check if account roles are up-to-date
	isAccountRoleUpgradeNeed, err = r.AWSClient.IsUpgradedNeededForAccountRolePolicies(
		prefix, latestPolicyVersion)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if isAccountRoleUpgradeNeed {
		r.Reporter.Infof("Account roles with prefix '%s' need to be upgraded before operator roles. "+
			"Roles can be upgraded with the following command :"+
			"\n\n\trosa upgrade account-roles --prefix %s\n", prefix, prefix)
		os.Exit(1)
	}

	isOperatorPolicyUpgradeNeeded, err := r.AWSClient.IsUpgradedNeededForOperatorRolePoliciesUsingPrefix(prefix,
		r.Creator.Partition, r.Creator.AccountID, latestPolicyVersion, credRequests, unifiedPath)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	version := args.upgradeVersion
	if version == "" {
		version = cluster.Version().RawID()
	}

	//Check if the upgrade is needed for the operators
	missingRolesInCS, err := r.OCMClient.FindMissingOperatorRolesForUpgrade(cluster, version, credRequests)
	if err != nil {
		r.Reporter.Errorf("Error finding operator roles for upgrade '%s'", err)
		os.Exit(1)
	}

	if len(missingRolesInCS) <= 0 && !isOperatorPolicyUpgradeNeeded {
		r.Reporter.Infof("Operator roles associated with the cluster '%s' are already up-to-date.", cluster.ID())
		os.Exit(0)
	}

	if len(missingRolesInCS) > 0 || isOperatorPolicyUpgradeNeeded {
		r.Reporter.Infof("Starting to upgrade the operator IAM roles and policies")
	}

	mode, err = handleModeFlag(cmd, mode)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if isOperatorPolicyUpgradeNeeded {
		err = upgradeOperatorPolicies(mode, r, prefix, isAccountRoleUpgradeNeed,
			policies, env, latestPolicyVersion, credRequests, cluster, unifiedPath)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	if len(missingRolesInCS) > 0 {
		err = roles.CreateMissingRoles(r, missingRolesInCS, cluster, mode, prefix, policies, unifiedPath, false)
		if err != nil {
			r.Reporter.Errorf("Error creating operator roles: %s", err)
			os.Exit(1)
		}
	}
}

func upgradeOperatorPolicies(mode string, r *rosa.Runtime,
	prefix string, isAccountRoleUpgradeNeed bool, policies map[string]*cmv1.AWSSTSPolicy, env string,
	defaultPolicyVersion string, credRequests map[string]*cmv1.STSOperator, cluster *cmv1.Cluster,
	policyPath string) error {
	switch mode {
	case interactive.ModeAuto:
		if !confirm.Prompt(true, "Upgrade the operator role policy to version %s?", defaultPolicyVersion) {
			return nil
		}
		err := aws.UpgradeOperatorRolePolicies(r.Reporter, r.AWSClient, r.Creator.Partition,
			r.Creator.AccountID, prefix, policies, defaultPolicyVersion, credRequests, policyPath, cluster)
		if err != nil {
			if strings.Contains(err.Error(), "Throttling") {
				r.OCMClient.LogEvent("ROSAUpgradeOperatorRolesModeAuto", map[string]string{
					ocm.Response:   ocm.Failure,
					ocm.Version:    defaultPolicyVersion,
					ocm.IsThrottle: "true",
				})
			}
			return r.Reporter.Errorf("Error upgrading the operator policies: %s", err)
		}
		return nil
	case interactive.ModeManual:
		err := aws.GenerateOperatorRolePolicyFiles(r.Reporter, policies, credRequests,
			cluster.AWS().PrivateHostedZoneRoleARN(), r.Creator.Partition)
		if err != nil {
			r.Reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}

		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("All policy files saved to the current directory")
			r.Reporter.Infof("Run the following commands to upgrade the operator IAM policies:\n")
			if isAccountRoleUpgradeNeed {
				r.Reporter.Warnf("Operator role policies MUST only be upgraded after " +
					"Account Role policies upgrade has completed.\n")
			}
		}
		commands := aws.BuildOperatorRoleCommands(prefix, r.Creator.Partition, r.Creator.AccountID, r.AWSClient,
			defaultPolicyVersion, credRequests, policyPath, cluster)
		fmt.Println(awscb.JoinCommands(commands))
	default:
		return r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
	}
	return nil
}

func handleModeFlag(cmd *cobra.Command, mode string) (string, error) {
	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	if interactive.Enabled() {
		var err error
		mode, err = interactive.GetOptionMode(cmd, mode, "Operator IAM role/policy upgrade mode")
		if err != nil {
			return "", fmt.Errorf("expected a valid operator IAM role upgrade mode: %s", err)
		}
	}
	return mode, nil
}
