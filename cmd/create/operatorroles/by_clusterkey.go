package operatorroles

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

func handleOperatorRoleCreationByClusterKey(r *rosa.Runtime, env string,
	permissionsBoundary string, mode string,
	policies map[string]*cmv1.AWSSTSPolicy,
	defaultPolicyVersion string) error {
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()
	if cluster.AWS().STS().RoleARN() == "" {
		r.Reporter.Errorf("Cluster '%s' is not an STS cluster.", clusterKey)
		os.Exit(1)
	}
	// Check to see if IAM operator roles have already created
	missingRoles, err := validateOperatorRoles(r, cluster)
	if err != nil {
		if strings.Contains(err.Error(), "AccessDenied") {
			r.Reporter.Debugf("Failed to verify if operator roles exist: %s", err)
		} else {
			r.Reporter.Errorf("Failed to verify if operator roles exist: %s", err)
			os.Exit(1)
		}
	}

	if ocm.IsOidcConfigReusable(cluster) && len(missingRoles) == 0 {
		err := validateOperatorRolesMatchOidcProvider(r, cluster)
		if err != nil {
			return err
		}
		if !args.forcePolicyCreation {
			r.Reporter.Infof("Cluster '%s' is using reusable OIDC Config and operator roles already exist.", clusterKey)
			return nil
		}
	}

	if len(missingRoles) == 0 &&
		cluster.State() != cmv1.ClusterStateWaiting && cluster.State() != cmv1.ClusterStatePending &&
		!args.forcePolicyCreation {
		r.Reporter.Infof("Cluster '%s' is %s and does not need additional configuration.",
			clusterKey, cluster.State())
		os.Exit(0)
	}
	roleName, err := aws.GetInstallerAccountRoleName(cluster)
	if err != nil {
		r.Reporter.Errorf("Expected parsing role account role '%s': %v", cluster.AWS().STS().RoleARN(), err)
		os.Exit(1)
	}

	path, err := aws.GetPathFromAccountRole(cluster, aws.AccountRoles[aws.InstallerAccountRole].Name)
	if err != nil {
		r.Reporter.Errorf("Expected a valid path for '%s': %v", cluster.AWS().STS().RoleARN(), err)
		os.Exit(1)
	}
	if path != "" && !output.HasFlag() && r.Reporter.IsTerminal() {
		r.Reporter.Infof("ARN path '%s' detected in installer role '%s'. "+
			"This ARN path will be used for subsequent created operator roles and policies.",
			path, cluster.AWS().STS().RoleARN())
	}
	accountRoleVersion, err := r.AWSClient.GetAccountRoleVersion(roleName)
	if err != nil {
		r.Reporter.Errorf("Error getting account role version %s", err)
		os.Exit(1)
	}
	managedPolicies := cluster.AWS().STS().ManagedPolicies()
	if args.forcePolicyCreation && managedPolicies {
		r.Reporter.Warnf("Forcing creation of policies only works for unmanaged policies")
		os.Exit(1)
	}
	credRequests, err := r.OCMClient.GetCredRequests(cluster.Hypershift().Enabled())
	if err != nil {
		r.Reporter.Errorf("Error getting operator credential request from OCM %s", err)
		os.Exit(1)
	}

	operatorRolePolicyPrefix, err := aws.GetOperatorRolePolicyPrefixFromCluster(cluster, r.AWSClient)
	if err != nil {
		r.Reporter.Errorf("%s", err)
	}

	hostedCPPolicies := aws.IsHostedCPManagedPolicies(cluster)

	switch mode {
	case aws.ModeAuto:
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Creating roles using '%s'", r.Creator.ARN)
		}
		err = createRoles(r, operatorRolePolicyPrefix, permissionsBoundary, cluster,
			accountRoleVersion, policies, defaultPolicyVersion, credRequests, managedPolicies, hostedCPPolicies)
		if err != nil {
			r.Reporter.Errorf("There was an error creating the operator roles: %s", err)
			isThrottle := "false"
			if strings.Contains(err.Error(), "Throttling") {
				isThrottle = helper.True
			}
			r.OCMClient.LogEvent("ROSACreateOperatorRolesModeAuto", map[string]string{
				ocm.ClusterID:  clusterKey,
				ocm.Response:   ocm.Failure,
				ocm.IsThrottle: isThrottle,
			})
			os.Exit(1)
		}
		r.OCMClient.LogEvent("ROSACreateOperatorRolesModeAuto", map[string]string{
			ocm.ClusterID: clusterKey,
			ocm.Response:  ocm.Success,
		})
	case aws.ModeManual:
		commands, err := buildCommands(r, env, operatorRolePolicyPrefix, permissionsBoundary, defaultPolicyVersion,
			cluster, policies, credRequests, managedPolicies, hostedCPPolicies)
		if err != nil {
			r.Reporter.Errorf("There was an error building the list of resources: %s", err)
			os.Exit(1)
			r.OCMClient.LogEvent("ROSACreateOperatorRolesModeManual", map[string]string{
				ocm.ClusterID: clusterKey,
				ocm.Response:  ocm.Failure,
			})
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("All policy files saved to the current directory")
			r.Reporter.Infof("Run the following commands to create the operator roles:\n")
		}
		r.OCMClient.LogEvent("ROSACreateOperatorRolesModeManual", map[string]string{
			ocm.ClusterID: clusterKey,
		})
		fmt.Println(commands)

	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
	return nil
}

func createRoles(r *rosa.Runtime,
	prefix string, permissionsBoundary string,
	cluster *cmv1.Cluster, accountRoleVersion string, policies map[string]*cmv1.AWSSTSPolicy,
	defaultVersion string, credRequests map[string]*cmv1.STSOperator, managedPolicies bool, hostedCPPolicies bool) error {
	sharedVpcRoleArn := cluster.AWS().PrivateHostedZoneRoleARN()
	isSharedVpc := sharedVpcRoleArn != ""

	for credrequest, operator := range credRequests {
		ver := cluster.Version()
		if ver != nil && operator.MinVersion() != "" {
			isSupported, err := ocm.CheckSupportedVersion(ocm.GetVersionMinor(ver.ID()), operator.MinVersion())
			if err != nil {
				r.Reporter.Errorf("Error validating operator role '%s' version %s", operator.Name(), err)
				os.Exit(1)
			}
			if !isSupported {
				continue
			}
		}
		roleName, _ := aws.FindOperatorRoleNameBySTSOperator(cluster, operator)
		if roleName == "" {
			return fmt.Errorf("Failed to find operator IAM role")
		}

		path, err := aws.GetPathFromAccountRole(cluster, aws.AccountRoles[aws.InstallerAccountRole].Name)
		if err != nil {
			return err
		}

		var policyArn string
		filename := aws.GetOperatorPolicyKey(credrequest, hostedCPPolicies, isSharedVpc)
		if managedPolicies {
			policyArn, err = aws.GetManagedPolicyARN(policies, filename)
			if err != nil {
				return err
			}
		} else {
			policyArn = aws.GetOperatorPolicyARN(r.Creator.AccountID, prefix, operator.Namespace(),
				operator.Name(), path)
			policyDetails := aws.GetPolicyDetails(policies, filename)

			if isSharedVpc && credrequest == aws.IngressOperatorCloudCredentialsRoleType {
				err = validateIngressOperatorPolicyOverride(r, policyArn, sharedVpcRoleArn, prefix)
				if err != nil {
					return err
				}

				policyDetails = aws.InterpolatePolicyDocument(policyDetails, map[string]string{
					"shared_vpc_role_arn": sharedVpcRoleArn,
				})
			}

			operatorPolicyTags := map[string]string{
				tags.OpenShiftVersion:  accountRoleVersion,
				tags.RolePrefix:        prefix,
				tags.RedHatManaged:     helper.True,
				tags.OperatorNamespace: operator.Namespace(),
				tags.OperatorName:      operator.Name(),
			}

			if args.forcePolicyCreation || (isSharedVpc && credrequest == aws.IngressOperatorCloudCredentialsRoleType) {
				policyArn, err = r.AWSClient.ForceEnsurePolicy(policyArn, policyDetails,
					defaultVersion, operatorPolicyTags, path)
				if err != nil {
					return err
				}
			} else {
				policyArn, err = r.AWSClient.EnsurePolicy(policyArn, policyDetails,
					defaultVersion, operatorPolicyTags, path)
				if err != nil {
					return err
				}
			}
		}

		policyDetails := aws.GetPolicyDetails(policies, "operator_iam_role_policy")
		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, r.Creator.AccountID, operator, policyDetails)
		if err != nil {
			return err
		}

		r.Reporter.Debugf("Creating role '%s'", roleName)
		tagsList := map[string]string{
			tags.OperatorNamespace: operator.Namespace(),
			tags.OperatorName:      operator.Name(),
			tags.RedHatManaged:     helper.True,
		}
		if !ocm.IsOidcConfigReusable(cluster) {
			tagsList[tags.ClusterID] = cluster.ID()
		}
		if managedPolicies {
			tagsList[tags.ManagedPolicies] = helper.True
		}
		if hostedCPPolicies {
			tagsList[tags.HypershiftPolicies] = helper.True
		}

		roleARN, err := r.AWSClient.EnsureRole(roleName, policy, permissionsBoundary, accountRoleVersion,
			tagsList, path, managedPolicies)
		if err != nil {
			return err
		}
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)
		}

		r.Reporter.Debugf("Attaching permission policy '%s' to role '%s'", policyArn, roleName)
		err = r.AWSClient.AttachRolePolicy(roleName, policyArn)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildCommands(r *rosa.Runtime, env string,
	prefix string, permissionsBoundary string, defaultPolicyVersion string, cluster *cmv1.Cluster,
	policies map[string]*cmv1.AWSSTSPolicy, credRequests map[string]*cmv1.STSOperator,
	managedPolicies bool, hostedCPPolicies bool) (string, error) {
	sharedVpcRoleArn := cluster.AWS().PrivateHostedZoneRoleARN()
	isSharedVpc := sharedVpcRoleArn != ""

	err := aws.GeneratePolicyFiles(r.Reporter, env, false,
		true, policies, credRequests, managedPolicies, sharedVpcRoleArn)
	if err != nil {
		r.Reporter.Errorf("There was an error generating the policy files: %s", err)
		os.Exit(1)
	}

	commands := []string{}

	for credrequest, operator := range credRequests {
		ver := cluster.Version()
		if ver != nil && operator.MinVersion() != "" {
			isSupported, err := ocm.CheckSupportedVersion(ocm.GetVersionMinor(ver.ID()), operator.MinVersion())
			if err != nil {
				r.Reporter.Errorf("Error validating operator role '%s' version %s", operator.Name(), err)
				os.Exit(1)
			}
			if !isSupported {
				continue
			}
		}
		roleName, _ := aws.FindOperatorRoleNameBySTSOperator(cluster, operator)
		path, err := aws.GetPathFromAccountRole(cluster, aws.AccountRoles[aws.InstallerAccountRole].Name)
		if err != nil {
			return "", err
		}

		var policyARN string
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, aws.GetOperatorPolicyKey(
				credrequest, hostedCPPolicies, isSharedVpc))
			if err != nil {
				return "", err
			}
		} else {
			policyARN = computePolicyARN(r.Creator.AccountID, prefix, operator.Namespace(), operator.Name(), path)
			name := aws.GetOperatorPolicyName(prefix, operator.Namespace(), operator.Name())
			iamTags := map[string]string{
				tags.OpenShiftVersion:  defaultPolicyVersion,
				tags.RolePrefix:        prefix,
				tags.OperatorNamespace: operator.Namespace(),
				tags.OperatorName:      operator.Name(),
				tags.RedHatManaged:     helper.True,
			}
			operatorPolicyKey := aws.GetOperatorPolicyKey(credrequest, hostedCPPolicies, isSharedVpc)
			fileName := fmt.Sprintf("file://%s.json", operatorPolicyKey)
			_, err = r.AWSClient.IsPolicyExists(policyARN)
			if err != nil {
				createPolicy := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.CreatePolicy).
					AddParam(awscb.PolicyName, name).
					AddParam(awscb.PolicyDocument, fileName).
					AddTags(iamTags).
					AddParam(awscb.Path, path).
					Build()
				commands = append(commands, createPolicy)
			} else if isSharedVpc && credrequest == aws.IngressOperatorCloudCredentialsRoleType {
				err := validateIngressOperatorPolicyOverride(r, policyARN, sharedVpcRoleArn, prefix)
				if err != nil {
					return "", err
				}

				createPolicyVersion := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.CreatePolicyVersion).
					AddParam(awscb.PolicyArn, policyARN).
					AddParam(awscb.PolicyDocument, fileName).
					AddParamNoValue(awscb.SetAsDefault).
					Build()
				commands = append(commands, createPolicyVersion)
			}
		}

		policyDetail := aws.GetPolicyDetails(policies, "operator_iam_role_policy")
		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, r.Creator.AccountID, operator, policyDetail)
		if err != nil {
			return "", err
		}

		filename := fmt.Sprintf("operator_%s_policy", credrequest)
		filename = aws.GetFormattedFileName(filename)
		r.Reporter.Debugf("Saving '%s' to the current directory", filename)
		err = helper.SaveDocument(policy, filename)
		if err != nil {
			return "", err
		}
		iamTags := map[string]string{
			tags.OperatorNamespace: operator.Namespace(),
			tags.OperatorName:      operator.Name(),
			tags.RedHatManaged:     helper.True,
		}
		if !ocm.IsOidcConfigReusable(cluster) {
			iamTags[tags.ClusterID] = cluster.ID()
		}
		if managedPolicies {
			iamTags[tags.ManagedPolicies] = helper.True
		}
		if hostedCPPolicies {
			iamTags[tags.HypershiftPolicies] = helper.True
		}
		createRole := awscb.NewIAMCommandBuilder().
			SetCommand(awscb.CreateRole).
			AddParam(awscb.RoleName, roleName).
			AddParam(awscb.AssumeRolePolicyDocument, fmt.Sprintf("file://%s", filename)).
			AddParam(awscb.PermissionsBoundary, permissionsBoundary).
			AddTags(iamTags).
			AddParam(awscb.Path, path).
			Build()

		attachRolePolicy := awscb.NewIAMCommandBuilder().
			SetCommand(awscb.AttachRolePolicy).
			AddParam(awscb.RoleName, roleName).
			AddParam(awscb.PolicyArn, policyARN).
			Build()
		commands = append(commands, createRole, attachRolePolicy)
	}
	return awscb.JoinCommands(commands), nil
}

func validateOperatorRoles(r *rosa.Runtime, cluster *cmv1.Cluster) ([]string, error) {
	var missingRoles []string
	operatorIAMRoles := cluster.AWS().STS().OperatorIAMRoles()
	if len(operatorIAMRoles) == 0 {
		return missingRoles, fmt.Errorf("No Operator IAM roles found for cluster %s", cluster.Name())
	}
	for _, operatorIAMRole := range operatorIAMRoles {
		roleARN := operatorIAMRole.RoleARN()
		roleName, err := aws.GetResourceIdFromARN(roleARN)
		if err != nil {
			return missingRoles, err
		}
		exists, _, err := r.AWSClient.CheckRoleExists(roleName)
		if err != nil {
			return missingRoles, err
		}
		if !exists {
			missingRoles = append(missingRoles, roleName)
		}
	}
	return missingRoles, nil
}

func validateOperatorRolesMatchOidcProvider(r *rosa.Runtime, cluster *cmv1.Cluster) error {
	operatorRolesList, err := convertV1OperatorIAMRoleIntoOcmOperatorIamRole(
		cluster.AWS().STS().OperatorIAMRoles())
	if err != nil {
		return err
	}
	expectedPath, err := aws.GetPathFromARN(cluster.AWS().STS().RoleARN())
	if err != nil {
		return err
	}
	return ocm.ValidateOperatorRolesMatchOidcProvider(r.Reporter, r.AWSClient,
		operatorRolesList, cluster.AWS().STS().OidcConfig().IssuerUrl(),
		ocm.GetVersionMinor(cluster.Version().RawID()), expectedPath, cluster.AWS().STS().ManagedPolicies())
}
