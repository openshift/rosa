/*
Copyright (c) 2022 Red Hat, Inc.

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

package roles

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	semver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/zgalor/weberr"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	awscbRoles "github.com/openshift/rosa/pkg/aws/commandbuilder/helper/roles"
	"github.com/openshift/rosa/pkg/aws/tags"
	missingOperatorRolesHelper "github.com/openshift/rosa/pkg/helper/missingoperatorroles"
	"github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	isInvokedFromClusterUpgrade bool
	clusterID                   string
	clusterUpgradeVersion       string
	policyUpgradeversion        string
	channelGroup                string
}

var Cmd = &cobra.Command{
	Use:     "roles",
	Aliases: []string{},
	Short:   "Upgrade account-wide IAM roles to the latest version.",
	Long:    "Upgrade account-wide IAM roles to the latest version before upgrading your cluster.",
	Example: `  # Upgrade account/operator roles for ROSA STS clusters 
		rosa upgrade roles -c <cluster_key>`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

	aws.AddModeFlag(Cmd)

	flags.StringVar(
		&args.clusterUpgradeVersion,
		"cluster-version",
		"",
		"Version of OpenShift that the cluster will be upgraded to",
	)

	Cmd.MarkFlagRequired("cluster-version")

	flags.StringVar(
		&args.policyUpgradeversion,
		"policy-version",
		"",
		"Version of OpenShift that will be used to setup policy tag, for example \"4.11\"",
	)
	flags.MarkHidden("version")

	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		ocm.DefaultChannelGroup,
		"Channel group is the name of the channel where this image belongs, for example \"stable\" or \"fast\".",
	)
	flags.MarkHidden("channel-group")

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) error {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	reporter := r.Reporter
	awsClient := r.AWSClient
	ocmClient := r.OCMClient

	isInvokedFromClusterUpgrade := false
	skipInteractive := false
	var cluster *v1.Cluster
	if len(argv) >= 2 && !cmd.Flag("cluster").Changed {
		aws.SetModeKey(argv[0])
		ocm.SetClusterKey(argv[1])
		skipInteractive = true
		args.clusterUpgradeVersion = argv[2]
		args.channelGroup = argv[3]
		isInvokedFromClusterUpgrade = true
	}
	args.isInvokedFromClusterUpgrade = isInvokedFromClusterUpgrade

	r.GetClusterKey()
	cluster = r.FetchCluster()

	mode, err := aws.GetMode()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	policyVersion := args.policyUpgradeversion
	isPolicyVersionChosen := policyVersion != ""
	channelGroup := args.channelGroup
	policyVersion, err = ocmClient.GetPolicyVersion(policyVersion, channelGroup)
	if err != nil {
		reporter.Errorf("Error getting version: %s", err)
		os.Exit(1)
	}

	clusterUpgradeVersion := args.clusterUpgradeVersion

	availableUpgrades, err := r.OCMClient.GetAvailableUpgrades(ocm.GetVersionID(cluster))
	if err != nil {
		r.Reporter.Errorf("Failed to find available upgrades: %v", err)
		os.Exit(1)
	}
	if len(availableUpgrades) == 0 {
		r.Reporter.Warnf("There are no available upgrades")
		os.Exit(0)
	}
	err = ocmClient.CheckUpgradeClusterVersion(availableUpgrades, clusterUpgradeVersion, cluster)
	if err != nil {
		reporter.Errorf("%v", err)
		os.Exit(1)
	}

	err = checkPolicyAndClusterVersionCompatibility(policyVersion, clusterUpgradeVersion)
	if err != nil {
		reporter.Errorf("%v", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get IAM credentials: %s", err)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") && !skipInteractive {
		interactive.Enable()
	}

	if interactive.Enabled() && !skipInteractive {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Roles upgrade mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid Account role upgrade mode: %s", err)
			os.Exit(1)
		}
		aws.SetModeKey(mode)
	}

	var spin *spinner.Spinner
	if reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if !args.isInvokedFromClusterUpgrade {
		reporter.Infof("Ensuring account role/policies compatibility for upgrade")
	}

	if spin != nil {
		spin.Start()
	}

	//ACCOUNT ROLES

	isUpgradeNeedForAccountRolePolicies, err := awsClient.IsUpgradedNeededForAccountRolePoliciesForCluster(
		cluster,
		policyVersion,
	)
	if err != nil {
		reporter.Errorf("%s", err)
		LogError(roles.RosaUpgradeAccRolesModeAuto, ocmClient, policyVersion, err, reporter)
		os.Exit(1)
	}

	if spin != nil {
		spin.Stop()
	}

	if !isUpgradeNeedForAccountRolePolicies {
		reporter.Infof("Account roles/policies for cluster '%s' are already up-to-date.", r.ClusterKey)
	} else {
		accountRolePolicies, err := ocmClient.GetPolicies("")
		if err != nil {
			reporter.Errorf("Expected a valid role creation mode: %s", err)
			os.Exit(1)
		}

		switch mode {
		case aws.ModeAuto:
			if isUpgradeNeedForAccountRolePolicies {
				reporter.Infof("Starting to upgrade the policies")
				err = upgradeAccountRolePoliciesFromCluster(
					mode,
					reporter,
					awsClient,
					cluster,
					creator.AccountID,
					accountRolePolicies,
					policyVersion,
					isPolicyVersionChosen,
				)
				if err != nil {
					LogError(roles.RosaUpgradeAccRolesModeAuto, ocmClient, policyVersion, err, reporter)
					if args.isInvokedFromClusterUpgrade {
						return err
					}
					reporter.Errorf("Error upgrading the role polices: %s", err)
					os.Exit(1)
				}
			}
		case aws.ModeManual:
			err = aws.GeneratePolicyFiles(reporter, env, isUpgradeNeedForAccountRolePolicies,
				false, accountRolePolicies, nil)
			if err != nil {
				reporter.Errorf("There was an error generating the policy files: %s", err)
				os.Exit(1)
			}
			if reporter.IsTerminal() {
				reporter.Infof("All policy files saved to the current directory")
				reporter.Infof("Run the following commands to upgrade the account role policies:\n")
			}

			commands, err := buildAccountRoleCommandsFromCluster(
				mode,
				cluster,
				creator.AccountID,
				isUpgradeNeedForAccountRolePolicies,
				awsClient,
				policyVersion,
			)
			if err != nil {
				return err
			}

			fmt.Println(commands)
			if args.isInvokedFromClusterUpgrade {
				reporter.Infof("Run the following command to continue scheduling cluster upgrade"+
					" once account and operator roles have been upgraded : \n\n"+
					"\trosa upgrade cluster --cluster %s\n", r.ClusterKey)
				return nil
			}

		default:
			reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
			os.Exit(1)
		}
	}

	//OPERATOR ROLES
	if !args.isInvokedFromClusterUpgrade {
		reporter.Infof("Ensuring operator role/policies compatibility for upgrade")
	}

	if spin != nil {
		spin.Start()
	}

	operatorRoles, hasOperatorRoles := cluster.AWS().STS().GetOperatorIAMRoles()
	if !hasOperatorRoles || len(operatorRoles) == 0 {
		r.Reporter.Errorf("Cluster '%s' doesnt have any operator roles associated with it",
			r.ClusterKey)
		os.Exit(1)
	}

	unifiedPath, err := aws.GetInstallerRolePathFromCluster(cluster)
	if err != nil {
		r.Reporter.Errorf("Expected a valid path for '%s': %v", cluster.AWS().STS().RoleARN(), err)
		os.Exit(1)
	}

	credRequests, err := ocmClient.GetCredRequests(cluster.Hypershift().Enabled())
	if err != nil {
		r.Reporter.Errorf("Error getting operator credential request from OCM %s", err)
		os.Exit(1)
	}

	operatorRolePolicyPrefix, err := aws.GetOperatorRolePolicyPrefixFromCluster(cluster, r.AWSClient)
	if err != nil {
		return err
	}

	isOperatorPolicyUpgradeNeeded := false
	isOperatorPolicyUpgradeNeeded, err = r.AWSClient.IsUpgradedNeededForOperatorRolePoliciesUsingCluster(
		cluster, r.Creator.AccountID, policyVersion, credRequests, operatorRolePolicyPrefix)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	missingRolesInCS, err := ocmClient.FindMissingOperatorRolesForUpgrade(cluster, clusterUpgradeVersion)
	if err != nil {
		return err
	}

	if spin != nil {
		spin.Stop()
	}

	if len(missingRolesInCS) <= 0 && !isOperatorPolicyUpgradeNeeded {
		r.Reporter.Infof(
			"Operator roles/policies associated with the cluster '%s' are already up-to-date.",
			cluster.ID(),
		)
		if args.isInvokedFromClusterUpgrade {
			return nil
		}
		os.Exit(0)
	}

	operatorRolePolicies, err := ocmClient.GetPolicies("OperatorRole")
	if err != nil {
		r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	if isOperatorPolicyUpgradeNeeded {
		err = upgradeOperatorPolicies(
			mode, r, isUpgradeNeedForAccountRolePolicies, operatorRolePolicies, env,
			policyVersion, credRequests, cluster, operatorRolePolicyPrefix)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	err = missingOperatorRolesHelper.HandleMissingOperatorRoles(
		mode, r, cluster, missingRolesInCS, operatorRolePolicies,
		unifiedPath, operatorRolePolicyPrefix, args.isInvokedFromClusterUpgrade)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	return nil
}

func LogError(key string, ocmClient *ocm.Client, defaultPolicyVersion string, err error, reporter *rprtr.Object) {
	reporter.Debugf("Logging throttle error")
	if strings.Contains(err.Error(), "Throttling") {
		ocmClient.LogEvent(key, map[string]string{
			ocm.Response:   ocm.Failure,
			ocm.Version:    defaultPolicyVersion,
			ocm.IsThrottle: "true",
		})
	}
}

func handleAccountRolePolicyARN(
	mode string,
	awsClient aws.Client,
	roleName string,
	rolePath string,
	accountID string,
) (string, []string, error) {
	commands := make([]string, 0)
	policiesDetails, err := awsClient.GetAttachedPolicy(&roleName)
	if err != nil {
		return "", commands, err
	}

	hasMoreThanOneAttachedPolicy := aws.HasMoreThanOneAttachedPolicy(policiesDetails)
	if hasMoreThanOneAttachedPolicy {
		promptString := fmt.Sprintf("More than one policy attached to account role '%s'.\n"+
			"\tWould you like to dettach current policies and setup a new one ?", roleName)
		if !confirm.Prompt(true, promptString) {
			attachedPoliciesDetail := aws.FindAllAttachedPolicyDetails(policiesDetails)
			attachedPoliciesArns := make([]string, len(attachedPoliciesDetail))
			for _, attachedPolicyDetail := range attachedPoliciesDetail {
				attachedPoliciesArns = append(attachedPoliciesArns, attachedPolicyDetail.PolicyArn)
			}
			chosenPolicyARN, err := interactive.GetOption(interactive.Input{
				Question: "Choose Policy ARN to upgrade",
				Options:  attachedPoliciesArns,
				Default:  attachedPoliciesArns[0],
				Required: true,
			})
			if err != nil {
				return "", commands, err
			}
			return chosenPolicyARN, commands, nil
		}

		switch mode {
		case aws.ModeAuto:
			err := awsClient.DetachRolePolicies(roleName)
			if err != nil {
				return "", commands, err
			}
		case aws.ModeManual:
			for _, policyDetail := range policiesDetails {
				detachManagedPoliciesCommand := awscbRoles.ManualCommandsForDetachRolePolicy(
					awscbRoles.ManualCommandsForDetachRolePolicyInput{
						RoleName:  roleName,
						PolicyARN: policyDetail.PolicyArn,
					},
				)
				commands = append(commands, detachManagedPoliciesCommand)
			}
		default:
			return "", commands, weberr.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		}

		generatedPolicyARN := aws.GetPolicyARN(accountID, roleName, rolePath)
		return generatedPolicyARN, commands, nil
	}
	policyDetail := policiesDetails[0]
	return policyDetail.PolicyArn, commands, nil
}

func upgradeAccountRolePoliciesFromCluster(
	mode string, reporter *rprtr.Object, awsClient aws.Client,
	cluster *v1.Cluster, accountID string, policies map[string]string,
	policyVersion string, isVersionChosen bool) error {
	for file, role := range aws.AccountRoles {
		roleName, err := aws.GetAccountRoleNameFromCluster(cluster, role)
		if err != nil {
			return err
		}
		if roleName == "" {
			reporter.Debugf("Cluster '%s' does not include expected role '%s'", cluster.ID(), role.Name)
			continue
		}
		prefix, err := aws.GetAccountRolePrefixFromCluster(cluster, role)
		if err != nil {
			return err
		}
		rolePath, err := aws.GetAccountRolePathFromCluster(cluster, role.Name)
		if err != nil {
			return err
		}
		promptString := fmt.Sprintf("Upgrade the '%s' role policy latest version (%s) ?", roleName, policyVersion)
		if isVersionChosen {
			promptString = fmt.Sprintf("Upgrade the '%s' role policy to version '%s' ?", roleName, policyVersion)
		}
		if !confirm.Prompt(true, promptString) {
			if args.isInvokedFromClusterUpgrade {
				return reporter.Errorf("Account roles need to be upgraded to proceed")
			}
			continue
		}
		filename := fmt.Sprintf("sts_%s_permission_policy", file)

		policyARN, _, err := handleAccountRolePolicyARN(mode, awsClient, roleName, rolePath, accountID)
		if err != nil {
			return err
		}

		accountPolicyPath, err := aws.GetPathFromARN(policyARN)
		if err != nil {
			return err
		}

		policyDetails := policies[filename]
		policyARN, err = awsClient.EnsurePolicy(policyARN, policyDetails,
			policyVersion, map[string]string{
				tags.OpenShiftVersion: policyVersion,
				tags.RolePrefix:       prefix,
				tags.RoleType:         file,
				tags.RedHatManaged:    "true",
			}, accountPolicyPath)
		if err != nil {
			return err
		}

		err = awsClient.AttachRolePolicy(roleName, policyARN)
		if err != nil {
			return err
		}
		//Delete if present else continue
		err = awsClient.DeleteInlineRolePolicies(roleName)
		if err != nil {
			reporter.Debugf("Error deleting inline role policy %s : %s", policyARN, err)
		}
		reporterString := fmt.Sprintf("Upgraded policy with ARN '%s' to version '%s'", policyARN, policyVersion)
		reporter.Infof(reporterString)
		err = awsClient.UpdateTag(roleName, policyVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildAccountRoleCommandsFromCluster(
	mode string, cluster *v1.Cluster, accountID string,
	isUpgradeNeedForAccountRolePolicies bool, awsClient aws.Client, defaultPolicyVersion string) (string, error) {
	commands := []string{}
	if isUpgradeNeedForAccountRolePolicies {
		for file, role := range aws.AccountRoles {
			accRoleName, err := aws.GetAccountRoleNameFromCluster(cluster, role)
			if err != nil {
				return "", err
			}
			prefix, err := aws.GetAccountRolePrefixFromCluster(cluster, role)
			if err != nil {
				return "", err
			}
			rolePath, err := aws.GetAccountRolePathFromCluster(cluster, role.Name)
			if err != nil {
				return "", err
			}

			policyARN, detachPoliciesCommands, err := handleAccountRolePolicyARN(
				mode, awsClient, accRoleName, rolePath, accountID)
			if err != nil {
				return "", err
			}

			commands = append(commands, detachPoliciesCommands...)

			accountPolicyPath, err := aws.GetPathFromARN(policyARN)
			if err != nil {
				return "", err
			}
			_, err = awsClient.IsPolicyExists(policyARN)
			hasPolicy := err == nil
			policyName := aws.GetPolicyName(accRoleName)
			_, err = awsClient.IsRolePolicyExists(accRoleName, policyName)
			hasInlinePolicy := err == nil
			hasDetachPolicyCommandsForExpectedPolicy := checkHasDetachPolicyCommandsForExpectedPolicy(
				detachPoliciesCommands,
				policyARN,
			)
			upgradeAccountPolicyCommands := awscbRoles.ManualCommandsForUpgradeAccountRolePolicy(
				awscbRoles.ManualCommandsForUpgradeAccountRolePolicyInput{
					DefaultPolicyVersion:                     defaultPolicyVersion,
					RoleName:                                 accRoleName,
					HasPolicy:                                hasPolicy,
					Prefix:                                   prefix,
					File:                                     file,
					PolicyName:                               policyName,
					AccountPolicyPath:                        accountPolicyPath,
					PolicyARN:                                policyARN,
					HasInlinePolicy:                          hasInlinePolicy,
					HasDetachPolicyCommandsForExpectedPolicy: hasDetachPolicyCommandsForExpectedPolicy,
				},
			)
			commands = append(commands, upgradeAccountPolicyCommands...)
		}
	}
	return awscb.JoinCommands(commands), nil
}

func checkHasDetachPolicyCommandsForExpectedPolicy(detachedPoliciesCommands []string, policyARN string) bool {
	for _, command := range detachedPoliciesCommands {
		if strings.Contains(command, policyARN) {
			return true
		}
	}
	return false
}

func upgradeOperatorPolicies(
	mode string, r *rosa.Runtime, isAccountRoleUpgradeNeed bool,
	policies map[string]string, env string, defaultPolicyVersion string,
	credRequests map[string]*v1.STSOperator, cluster *v1.Cluster, operatorRolePolicyPrefix string) error {
	switch mode {
	case aws.ModeAuto:
		if !confirm.Prompt(true, "Upgrade each operator role policy to latest version (%s)?", defaultPolicyVersion) {
			if args.isInvokedFromClusterUpgrade {
				return r.Reporter.Errorf("Operator roles need to be upgraded to proceed")
			}
			return nil
		}
		err := upgradeOperatorRolePoliciesFromCluster(
			mode, r.Reporter,
			r.AWSClient, r.Creator.AccountID,
			policies, defaultPolicyVersion,
			credRequests,
			cluster.AWS().STS().OperatorIAMRoles(),
			operatorRolePolicyPrefix,
		)
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
	case aws.ModeManual:
		err := aws.GeneratePolicyFiles(r.Reporter, env, false,
			true, policies, credRequests)
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
		commands, err := buildOperatorRoleCommandsFromCluster(
			mode, operatorRolePolicyPrefix,
			r.Creator.AccountID, r.AWSClient,
			defaultPolicyVersion, credRequests,
			cluster.AWS().STS().OperatorIAMRoles())
		if err != nil {
			r.Reporter.Errorf("There was an error generating the commands: %s", err)
			os.Exit(1)
		}
		fmt.Println(commands)
	default:
		return r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
	}
	return nil
}

func upgradeOperatorRolePoliciesFromCluster(
	mode string, reporter *rprtr.Object, awsClient aws.Client,
	accountID string, policies map[string]string,
	defaultPolicyVersion string, credRequests map[string]*v1.STSOperator,
	operatorRoles []*v1.OperatorIAMRole, operatorRolePolicyPrefix string) error {
	generalPath, err := aws.GetPathFromARN(operatorRoles[0].RoleARN())
	if err != nil {
		return err
	}
	for credrequest, operator := range credRequests {
		policyARN := ""
		operatorPolicyPath := generalPath
		operatorRoleARN := aws.FindOperatorRoleARNBySTSOperator(operatorRoles, operator)
		if operatorRoleARN == "" {
			policyARN = aws.GetOperatorPolicyARN(
				accountID, operatorRolePolicyPrefix,
				operator.Namespace(), operator.Name(), operatorPolicyPath)
		} else {
			operatorRoleName, err := aws.GetResourceIdFromARN(operatorRoleARN)
			if err != nil {
				return err
			}

			policyARN, _, err = handleOperatorRolePolicyARN(
				mode, awsClient, operatorRoleName,
				operatorRolePolicyPrefix, operatorPolicyPath,
				operator, accountID)
			if err != nil {
				return err
			}
			operatorPolicyPath, err = aws.GetPathFromARN(policyARN)
			if err != nil {
				return err
			}
		}
		filename := fmt.Sprintf("openshift_%s_policy", credrequest)
		policyDetails := policies[filename]
		policyARN, err = awsClient.EnsurePolicy(policyARN, policyDetails,
			defaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion:  defaultPolicyVersion,
				tags.RolePrefix:        operatorRolePolicyPrefix,
				tags.OperatorNamespace: operator.Namespace(),
				tags.OperatorName:      operator.Name(),
			}, operatorPolicyPath)
		if err != nil {
			return err
		}
		reporter.Infof("Upgraded policy with ARN '%s' to version '%s'", policyARN, defaultPolicyVersion)
	}
	return nil
}

func buildOperatorRoleCommandsFromCluster(
	mode string, operatorRolePolicyPrefix string,
	accountID string, awsClient aws.Client, defaultPolicyVersion string,
	credRequests map[string]*v1.STSOperator, operatorRoles []*v1.OperatorIAMRole) (string, error) {
	commands := []string{}
	generalPath, err := aws.GetPathFromARN(operatorRoles[0].RoleARN())
	if err != nil {
		return "", err
	}
	for credrequest, operator := range credRequests {
		policyARN := ""
		operatorPolicyPath := generalPath
		hasDetachPolicyCommandsForExpectedPolicy := false
		operatorRoleARN := aws.FindOperatorRoleARNBySTSOperator(operatorRoles, operator)
		operatorRoleName := ""
		if operatorRoleARN == "" {
			policyARN = aws.GetOperatorPolicyARN(
				accountID, operatorRolePolicyPrefix,
				operator.Namespace(), operator.Name(),
				operatorPolicyPath)
		} else {
			operatorRoleName, err = aws.GetResourceIdFromARN(operatorRoleARN)
			if err != nil {
				return "", err
			}
			foundPolicyARN, detachPoliciesCommands, err := handleOperatorRolePolicyARN(
				mode, awsClient,
				operatorRoleName, operatorRolePolicyPrefix,
				operatorPolicyPath, operator, accountID)
			if err != nil {
				return "", err
			}
			hasDetachPolicyCommandsForExpectedPolicy = checkHasDetachPolicyCommandsForExpectedPolicy(
				detachPoliciesCommands,
				foundPolicyARN,
			)

			commands = append(commands, detachPoliciesCommands...)
			operatorPolicyPath, err = aws.GetPathFromARN(foundPolicyARN)
			if err != nil {
				return "", err
			}
			policyARN = foundPolicyARN
		}
		policyName := aws.GetOperatorPolicyName(
			operatorRolePolicyPrefix,
			operator.Namespace(),
			operator.Name(),
		)
		_, err = awsClient.IsPolicyExists(policyARN)
		hasPolicy := err == nil

		upgradePoliciesCommands := awscbRoles.ManualCommandsForUpgradeOperatorRolePolicy(
			awscbRoles.ManualCommandsForUpgradeOperatorRolePolicyInput{
				HasPolicy:                                hasPolicy,
				OperatorRolePolicyPrefix:                 operatorRolePolicyPrefix,
				Operator:                                 operator,
				CredRequest:                              credrequest,
				OperatorPolicyPath:                       operatorPolicyPath,
				PolicyARN:                                policyARN,
				DefaultPolicyVersion:                     defaultPolicyVersion,
				PolicyName:                               policyName,
				HasDetachPolicyCommandsForExpectedPolicy: hasDetachPolicyCommandsForExpectedPolicy,
				OperatorRoleName:                         operatorRoleName,
			},
		)
		commands = append(commands, upgradePoliciesCommands...)
	}
	return awscb.JoinCommands(commands), nil
}

func handleOperatorRolePolicyARN(
	mode string, awsClient aws.Client, operatorRoleName string,
	operatorRolePolicyPrefix string, operatorPolicyPath string,
	operator *v1.STSOperator, accountID string) (string, []string, error) {
	commands := make([]string, 0)
	policiesDetails, err := awsClient.GetAttachedPolicy(&operatorRoleName)
	if err != nil {
		return "", commands, err
	}

	hasMoreThanOneAttachedPolicy := aws.HasMoreThanOneAttachedPolicy(policiesDetails)
	if hasMoreThanOneAttachedPolicy {
		promptString := fmt.Sprintf("More than one policy attached to operator role '%s'.\n"+
			"\tWould you like to dettach current policies and setup a new one ?", operatorRoleName)
		if !confirm.Prompt(true, promptString) {
			attachedPoliciesDetails := aws.FindAllAttachedPolicyDetails(policiesDetails)
			attachedPoliciesArns := make([]string, len(attachedPoliciesDetails))
			for _, attachedPolicyDetail := range attachedPoliciesDetails {
				attachedPoliciesArns = append(attachedPoliciesArns, attachedPolicyDetail.PolicyArn)
			}
			chosenPolicyARN, err := interactive.GetOption(interactive.Input{
				Question: "Choose Policy ARN to upgrade",
				Options:  attachedPoliciesArns,
				Default:  attachedPoliciesArns[0],
				Required: true,
			})
			if err != nil {
				return "", commands, err
			}
			return chosenPolicyARN, commands, nil
		}
		switch mode {
		case aws.ModeAuto:
			err := awsClient.DetachRolePolicies(operatorRoleName)
			if err != nil {
				return "", commands, err
			}
		case aws.ModeManual:
			for _, policyDetail := range policiesDetails {
				detachManagedPoliciesCommand := awscbRoles.ManualCommandsForDetachRolePolicy(
					awscbRoles.ManualCommandsForDetachRolePolicyInput{
						RoleName:  operatorRoleName,
						PolicyARN: policyDetail.PolicyArn,
					},
				)
				commands = append(commands, detachManagedPoliciesCommand)
			}
		default:
			return "", commands, weberr.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		}
		generatedPolicyARN := aws.GetOperatorPolicyARN(
			accountID,
			operatorRolePolicyPrefix,
			operator.Namespace(),
			operator.Name(),
			operatorPolicyPath,
		)
		return generatedPolicyARN, commands, nil
	}
	policyDetail := policiesDetails[0]
	return policyDetail.PolicyArn, commands, nil
}

func checkPolicyAndClusterVersionCompatibility(policyVersion, clusterVersion string) (err error) {
	parsedPolicyVersion, err := ocm.ParseVersion(policyVersion)
	if err != nil {
		return
	}
	parsedClusterVersion, err := ocm.ParseVersion(clusterVersion)
	if err != nil {
		return
	}

	semPolicyVersion, _ := semver.NewVersion(parsedPolicyVersion)
	semClusterVersion, _ := semver.NewVersion(parsedClusterVersion)

	if semPolicyVersion.LessThan(semClusterVersion) {
		return weberr.Errorf(
			"Desired major.minor policy version (%s) should be greater or equal to desired cluster version major.minor (%s)",
			policyVersion,
			clusterVersion,
		)
	}
	return nil
}
