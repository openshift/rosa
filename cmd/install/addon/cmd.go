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

package addon

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	awserr "github.com/openshift-online/ocm-common/pkg/aws/errors"
	awsCommonUtils "github.com/openshift-online/ocm-common/pkg/aws/utils"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	asv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	billingModelFlag          = "billing-model"
	billingModelAccountIDFlag = "billing-model-account-id"
)

var args struct {
	billingModel          string
	billingModelAccountID string
}

var Cmd = &cobra.Command{
	Use:     "addon ID",
	Aliases: []string{"addons", "add-on", "add-ons"},
	Short:   "Install add-ons on cluster",
	Long:    "Install Red Hat managed add-ons on a cluster",
	Example: `  # Add the CodeReady Workspaces add-on installation to the cluster
  rosa install addon --cluster=mycluster codeready-workspaces`,
	Run:                run,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, argv []string) error {
		err := arguments.ParseUnknownFlags(cmd, argv)
		if err != nil {
			return err
		}

		if len(cmd.Flags().Args()) != 1 {
			return fmt.Errorf("expected exactly one command line parameter containing the id of the add-on")
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.billingModel,
		billingModelFlag,
		string(amv1.BillingModelStandard),
		"Set the billing model to be used for the addon installation resource",
	)

	flags.StringVar(
		&args.billingModelAccountID,
		billingModelAccountIDFlag,
		"",
		"Account ID of associated billing model for the addon installation resource",
	)

	confirm.AddFlag(flags)
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Parse out CLI flags, then override positional arguments
	_ = cmd.Flags().Parse(argv)
	argv = cmd.Flags().Args()
	addOnID := argv[0]

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	ensureAddonNotInstalled(r, cluster.ID(), addOnID)

	addOn, err := r.OCMClient.GetAddOn(addOnID)
	if err != nil {
		r.Reporter.Warnf("Failed to get add-on '%s'", addOnID)
		os.Exit(1)
	}

	// Verify if addon requires STS authentication
	isSTS := cluster.AWS().STS().RoleARN() != "" && len(addOn.CredentialsRequests()) > 0
	if isSTS {
		r.Reporter.Warnf("Addon '%s' needs access to resources in account '%s'", addOnID, r.Creator.AccountID)
	}

	if !confirm.Confirm("install add-on '%s' on cluster '%s'", addOnID, clusterKey) {
		os.Exit(0)
	}

	if isSTS {
		prefix := aws.GetPrefixFromOperatorRole(cluster)

		for _, cr := range addOn.CredentialsRequests() {
			roleName := generateRoleName(cr, prefix)
			roleArn := aws.GetRoleARN(r.Creator.AccountID, roleName, "", r.Creator.Partition)
			_, err = r.AWSClient.GetRoleByARN(roleArn)
			if err != nil {
				if awserr.IsNoSuchEntityException(err) {
					err = createAddonRole(r, roleName, cr, cmd, cluster)
					if err != nil {
						r.Reporter.Errorf("%s", err)
						os.Exit(1)
					}
				} else {
					r.Reporter.Errorf("%s", err)
					os.Exit(1)
				}
			}
			// TODO : verify the role has the right permissions

			operatorRole, err := cmv1.NewOperatorIAMRole().
				Name(cr.Name()).
				Namespace(cr.Namespace()).
				RoleARN(roleArn).
				ServiceAccount(cr.ServiceAccount()).
				Build()
			if err != nil {
				r.Reporter.Errorf("Failed to build operator role '%s': %s", roleName, err)
				os.Exit(1)
			}

			err = r.OCMClient.AddClusterOperatorRole(cluster, operatorRole)
			if err != nil {
				r.Reporter.Errorf("Failed to add operator role to cluster '%s': %s", clusterKey, err)
				os.Exit(1)
			}
		}
	}

	addonParameters, err := r.OCMClient.GetAddOnParameters(cluster.ID(), addOnID)
	if err != nil {
		r.Reporter.Errorf("Failed to get add-on '%s' parameters: %v", addOnID, err)
		os.Exit(1)
	}

	var addonArguments []ocm.AddOnParam
	if addonParameters.Len() > 0 {
		// Determine if all required parameters have already been set as flags and ensure
		// that interactive mode is enabled if they have not. If there are no parameters
		// set as flags, then we also ensure that interactive mode is enabled so that the
		// user gets prompted.
		if arguments.HasUnknownFlags() {
			addonParameters.Each(func(param *asv1.AddonParameter) bool {
				flag := cmd.Flags().Lookup(param.ID())
				if param.Required() && (flag == nil || flag.Value.String() == "") {
					interactive.Enable()
					return false
				}
				return true
			})
		} else {
			interactive.Enable()
		}

		addonParameters.Each(func(param *asv1.AddonParameter) bool {
			var val string
			var options []string
			var values []string

			parameterOptions, _ := param.GetOptions()
			for _, opt := range parameterOptions {
				options = append(options, opt.Name())
				values = append(values, opt.Value())
			}

			// If value is already set in the CLI, ignore interactive prompt
			flag := cmd.Flags().Lookup(param.ID())
			if flag != nil {
				val = flag.Value.String()
			}
			if interactive.Enabled() {
				val, err = interactive.GetAddonArgument(*param, param.DefaultValue())
				if err != nil {
					r.Reporter.Errorf("%s", err)
					os.Exit(1)
				}
			}

			val = strings.Trim(val, " ")
			if len(options) > 0 && !helper.Contains(values, val) {
				r.Reporter.Errorf("Expected %v to match one of the options /%v/", val, options)
				os.Exit(1)
			}
			if val != "" && param.Validation() != "" {
				isValid, err := regexp.MatchString(param.Validation(), val)
				if err != nil || !isValid {
					r.Reporter.Errorf("Expected %v to match /%s/", val, param.Validation())
					os.Exit(1)
				}
			}
			addonArguments = append(addonArguments, ocm.AddOnParam{Key: param.ID(), Val: val})

			return true
		})
	}

	billingModel := args.billingModel
	billingModelAccountID := args.billingModelAccountID

	if !cmd.Flags().Changed(billingModelFlag) {
		billingModel = string(amv1.BillingModelStandard)
		if interactive.Enabled() {
			billingModel, err = interactive.GetOption(interactive.Input{
				Question: "Billing Model",
				Help:     cmd.Flags().Lookup(billingModelFlag).Usage,
				Default:  string(amv1.BillingModelStandard),
				Options:  ocm.BillingOptions,
				Required: true,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid billing model: %s", err)
				os.Exit(1)
			}
		}
	}

	if billingModel != string(amv1.BillingModelStandard) && !cmd.Flags().Changed(billingModelAccountIDFlag) {
		billingModelAccountID, err = interactive.GetString(interactive.Input{
			Question: "Billing Account ID",
			Help:     cmd.Flags().Lookup(billingModelAccountIDFlag).Usage,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid account id: %s", err)
			os.Exit(1)
		}
	}

	billing := ocm.AddOnBilling{
		BillingModel:     billingModel,
		BillingAccountID: billingModelAccountID,
	}

	r.Reporter.Debugf("Installing add-on '%s' on cluster '%s'", addOnID, clusterKey)
	err = r.OCMClient.InstallAddOn(cluster.ID(), addOnID, addonArguments, billing)
	if err != nil {
		r.Reporter.Errorf("Failed to add add-on installation '%s' for cluster '%s': %v", addOnID, clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Add-on '%s' is now installing. To check the status run 'rosa list addons -c %s'",
		addOnID, clusterKey)
	if interactive.Enabled() {
		r.Reporter.Infof("To install this addOn again in the future, you can run:\n   %s",
			buildCommand(cluster.Name(), addOnID, addonArguments, billing))
	}
}

func ensureAddonNotInstalled(r *rosa.Runtime, clusterID, addOnID string) {
	installation, err := r.OCMClient.GetAddOnInstallation(clusterID, addOnID)
	if err != nil && errors.GetType(err) != errors.NotFound {
		r.Reporter.Errorf("An error occurred while trying to get addon installation : %v", err)
		os.Exit(1)
	}
	if installation != nil {
		r.Reporter.Warnf("Addon '%s' is already installed on cluster '%s'", addOnID, clusterID)
		os.Exit(0)
	}
}

func createAddonRole(r *rosa.Runtime, roleName string, cr *asv1.CredentialRequest, cmd *cobra.Command,
	cluster *cmv1.Cluster) error {
	policy := aws.NewPolicyDocument()
	policy.AllowActions(cr.PolicyPermissions()...)

	policies, err := r.OCMClient.GetPolicies("OperatorRole")
	if err != nil {
		return err
	}
	policyDetails := aws.GetPolicyDetails(policies, "operator_iam_role_policy")
	assumePolicy, err := aws.GenerateAddonPolicyDoc(r.Creator.Partition, cluster, r.Creator.AccountID, cr, policyDetails)
	if err != nil {
		return err
	}

	r.Reporter.Debugf("Creating role '%s'", roleName)

	roleARN, err := r.AWSClient.EnsureRole(r.Reporter, roleName, assumePolicy, "", "",
		map[string]string{
			tags.ClusterID:    cluster.ID(),
			"addon_namespace": cr.Namespace(),
			"addon_name":      cr.Name(),
		}, "", false)
	if err != nil {
		return err
	}
	r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

	err = r.AWSClient.PutRolePolicy(roleName, roleName, policy.String())
	if err != nil {
		return err
	}
	return nil
}

func buildCommand(
	clusterName string,
	addonName string,
	addonArguments []ocm.AddOnParam,
	billing ocm.AddOnBilling,
) string {
	command := fmt.Sprintf("rosa install addon --cluster %s %s -y", clusterName, addonName)

	for _, arg := range addonArguments {
		if arg.Val != "" {
			command += fmt.Sprintf(" --%s %s", arg.Key, arg.Val)
		}
	}

	command += fmt.Sprintf(" --%s %s", billingModelFlag, billing.BillingModel)
	if billing.BillingAccountID != "" {
		command += fmt.Sprintf(" --%s %s", billingModelAccountIDFlag, billing.BillingAccountID)
	}

	return command
}

func generateRoleName(cr *asv1.CredentialRequest, prefix string) string {
	roleName := fmt.Sprintf("%s-%s-%s", prefix, cr.Namespace(), cr.Name())
	return awsCommonUtils.TruncateRoleName(roleName)
}
