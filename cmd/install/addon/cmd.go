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
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

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
			return fmt.Errorf("Expected exactly one command line parameter containing the id of the add-on")
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()
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
			roleArn := aws.GetRoleARN(r.Creator.AccountID, roleName)
			_, err = r.AWSClient.GetRoleByARN(roleArn)
			if err != nil {
				aerr, ok := err.(awserr.Error)
				if ok && aerr.Code() == iam.ErrCodeNoSuchEntityException {
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

	parameters, err := r.OCMClient.GetAddOnParameters(cluster.ID(), addOnID)
	if err != nil {
		r.Reporter.Errorf("Failed to get add-on '%s' parameters: %v", addOnID, err)
		os.Exit(1)
	}

	var params []ocm.AddOnParam
	if parameters.Len() > 0 {
		// Determine if all required parameters have already been set as flags and ensure
		// that interactive mode is enabled if they have not. If there are no parameters
		// set as flags, then we also ensure that interactive mode is enabled so that the
		// user gets prompted.
		if arguments.HasUnknownFlags() {
			parameters.Each(func(param *cmv1.AddOnParameter) bool {
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

		parameters.Each(func(param *cmv1.AddOnParameter) bool {
			var val string
			var hasVal bool
			// If value is already set in the CLI, ignore interactive prompt
			flag := cmd.Flags().Lookup(param.ID())
			if flag != nil {
				val = flag.Value.String()
				hasVal = true
			} else if interactive.Enabled() {
				input := interactive.Input{
					Question: param.Name(),
					Help:     fmt.Sprintf("%s: %s", param.ID(), param.Description()),
					Required: param.Required(),
				}
				// add a prompt to question name to indicate if the boolean param is required and check validation
				if param.ValueType() == "boolean" && param.Validation() == "^true$" && param.Required() {
					input.Question = fmt.Sprintf("%s (required)", param.Name())
					input.Validators = []interactive.Validator{
						interactive.RegExpBoolean(param.Validation()),
					}
				}
				switch param.ValueType() {
				case "boolean":
					var boolVal bool
					input.Default, _ = strconv.ParseBool(param.DefaultValue())
					boolVal, err = interactive.GetBool(input)
					if boolVal {
						val = "true"
					} else {
						val = "false"
					}
				case "cidr":
					var cidrVal net.IPNet
					if param.DefaultValue() != "" {
						_, defaultIDR, _ := net.ParseCIDR(param.DefaultValue())
						input.Default = *defaultIDR
					}
					cidrVal, err = interactive.GetIPNet(input)
					val = cidrVal.String()
					if val == "<nil>" {
						val = ""
					}
				case "number", "resource":
					var numVal int
					input.Default, _ = strconv.Atoi(param.DefaultValue())
					numVal, err = interactive.GetInt(input)
					val = fmt.Sprintf("%d", numVal)
				case "string":
					input.Default = param.DefaultValue()
					val, err = interactive.GetString(input)
				}
				if err != nil {
					r.Reporter.Errorf("Expected a valid value for '%s': %v", param.Name(), err)
					os.Exit(1)
				}
				hasVal = true
			}

			if hasVal {
				val = strings.Trim(val, " ")
				if val != "" && param.Validation() != "" {
					isValid, err := regexp.MatchString(param.Validation(), val)
					if err != nil || !isValid {
						r.Reporter.Errorf("Expected %v to match /%s/", val, param.Validation())
						os.Exit(1)
					}
				}
				params = append(params, ocm.AddOnParam{Key: param.ID(), Val: val})
			}

			return true
		})
	}

	r.Reporter.Debugf("Installing add-on '%s' on cluster '%s'", addOnID, clusterKey)
	err = r.OCMClient.InstallAddOn(cluster.ID(), addOnID, params)
	if err != nil {
		r.Reporter.Errorf("Failed to add add-on installation '%s' for cluster '%s': %v", addOnID, clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Add-on '%s' is now installing. To check the status run 'rosa list addons -c %s'",
		addOnID, clusterKey)
	if interactive.Enabled() {
		r.Reporter.Infof("To install this addOn again in the future, you can run:\n   %s",
			buildCommand(cluster.Name(), addOnID, params))
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

func createAddonRole(r *rosa.Runtime, roleName string, cr *cmv1.CredentialRequest, cmd *cobra.Command,
	cluster *cmv1.Cluster) error {
	policy := aws.NewPolicyDocument()
	policy.AllowActions(cr.PolicyPermissions()...)

	policies, err := r.OCMClient.GetPolicies("OperatorRole")
	if err != nil {
		return err
	}
	policyDetails := policies["operator_iam_role_policy"]
	assumePolicy, err := aws.GenerateAddonPolicyDoc(cluster, r.Creator.AccountID, cr, policyDetails)
	if err != nil {
		return err
	}

	r.Reporter.Debugf("Creating role '%s'", roleName)

	roleARN, err := r.AWSClient.EnsureRole(roleName, assumePolicy, "", "",
		map[string]string{
			tags.ClusterID:    cluster.ID(),
			"addon_namespace": cr.Namespace(),
			"addon_name":      cr.Name(),
		}, "")
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

func buildCommand(clusterName string, addonName string, params []ocm.AddOnParam) string {
	command := fmt.Sprintf("rosa install addon --cluster %s %s -y", clusterName, addonName)

	for _, param := range params {
		if param.Val != "" {
			command += fmt.Sprintf(" --%s %s", param.Key, param.Val)
		}
	}

	return command
}

func generateRoleName(cr *cmv1.CredentialRequest, prefix string) string {
	roleName := fmt.Sprintf("%s-%s-%s", prefix, cr.Namespace(), cr.Name())
	if len(roleName) > 64 {
		roleName = roleName[0:64]
	}
	return roleName
}
