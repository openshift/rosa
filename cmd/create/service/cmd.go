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

package service

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args ocm.CreateManagedServiceArgs

var Cmd = &cobra.Command{
	Use:     "service",
	Aliases: []string{"appliance"},
	Short:   "Creates a managed service.",
	Long: `  Managed Services are OpenShift clusters that provide a specific function.
  Use this command to create managed services.`,
	Example: `  # Create a Managed Service of type service1.
  rosa create service --type=service1 --name=clusterName`,
	Run:                run,
	Hidden:             true,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, argv []string) error {
		err := arguments.ParseUnknownFlags(cmd, argv)
		if err != nil {
			return err
		}

		if len(cmd.Flags().Args()) > 0 {
			return fmt.Errorf("Unrecognized command line parameter")
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	// Basic options
	flags.StringVar(
		&args.ServiceType,
		"type",
		"",
		"Type of service.",
	)

	flags.StringVar(
		&args.ClusterName,
		"name",
		"",
		"Name of the service instance.",
	)

	arguments.AddRegionFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Parse out CLI flags, then override positional arguments
	// This allows for arbitrary flags used for addon parameters
	_ = cmd.Flags().Parse(argv)

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	awsClient := aws.GetAWSClientForUserRegion(reporter, logger)

	// Get AWS region
	args.AwsRegion, err = aws.GetRegion(arguments.GetRegion())
	if err != nil {
		reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}
	reporter.Debugf("Using AWS region: %s", args.AwsRegion)

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Unable to get IAM credentials: %v", err)
		os.Exit(1)
	}
	reporter.Debugf("Using AWS creator: %s", awsCreator.ARN)

	args.AwsAccountID = awsCreator.AccountID
	args.Properties = map[string]string{
		properties.CreatorARN: awsCreator.ARN,
		properties.CLIVersion: info.Version,
	}

	// Openshift version to use.
	versionList, err := getVersionList(ocmClient)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}
	version := versionList[0]
	minor := ocm.GetVersionMinor(version)

	// Add-on parameter logic
	addOn, err := ocmClient.GetAddOn(args.ServiceType)
	if err != nil {
		reporter.Errorf("Failed to get add-on '%s': %s", args.ServiceType, err)
	}
	parameters := addOn.Parameters()

	if parameters.Len() > 0 {
		args.Parameters = map[string]string{}
		// Determine if all required parameters have already been set as flags.
		if arguments.HasUnknownFlags() {
			parameters.Each(func(param *cmv1.AddOnParameter) bool {
				flag := cmd.Flags().Lookup(param.ID())
				if param.Required() && (flag == nil || flag.Value.String() == "") {
					reporter.Errorf("Required parameter --%s missing", param.ID())
					os.Exit(1)
				}
				if flag != nil {
					val := strings.Trim(flag.Value.String(), " ")
					if val != "" && param.Validation() != "" {
						isValid, err := regexp.MatchString(param.Validation(), val)
						if err != nil || !isValid {
							reporter.Errorf("Expected %v to match /%s/", val, param.Validation())
							os.Exit(1)
						}
					}
					args.Parameters[param.ID()] = flag.Value.String()
				}
				return true
			})
		}
	}

	// Find all installer roles in the current account using AWS resource tags
	var roleARN string
	var supportRoleARN string
	var controlPlaneRoleARN string
	var workerRoleARN string

	role := aws.AccountRoles[aws.InstallerAccountRole]

	roleARNs, err := awsClient.FindRoleARNs(aws.InstallerAccountRole, minor)
	if err != nil {
		reporter.Errorf("Failed to find %s role: %s", role.Name, err)
		os.Exit(1)
	}

	if len(roleARNs) > 1 {
		defaultRoleARN := roleARNs[0]
		// Prioritize roles with the default prefix
		for _, rARN := range roleARNs {
			if strings.Contains(rARN, fmt.Sprintf("%s-%s-Role", aws.DefaultPrefix, role.Name)) {
				defaultRoleARN = rARN
			}
		}
		reporter.Warnf("More than one %s role found, using %s", role.Name, defaultRoleARN)
		roleARN = defaultRoleARN
	} else if len(roleARNs) == 1 {
		if !output.HasFlag() || reporter.IsTerminal() {
			reporter.Infof("Using %s for the %s role", roleARNs[0], role.Name)
		}
		roleARN = roleARNs[0]
	} else {
		reporter.Errorf("No account roles found. " +
			"You will need to run 'rosa create account-roles' to create them first.")
		os.Exit(1)
	}

	if roleARN != "" {
		// Get role prefix
		rolePrefix, err := getAccountRolePrefix(roleARN, role)
		if err != nil {
			reporter.Errorf("Failed to find prefix from %s account role", role.Name)
			os.Exit(1)
		}
		reporter.Debugf("Using '%s' as the role prefix", rolePrefix)

		for roleType, role := range aws.AccountRoles {
			if roleType == aws.InstallerAccountRole {
				// Already dealt with
				continue
			}
			roleARNs, err := awsClient.FindRoleARNs(roleType, minor)
			if err != nil {
				reporter.Errorf("Failed to find %s role: %s", role.Name, err)
				os.Exit(1)
			}
			selectedARN := ""
			for _, rARN := range roleARNs {
				if strings.Contains(rARN, fmt.Sprintf("%s-%s-Role", rolePrefix, role.Name)) {
					selectedARN = rARN
				}
			}
			if selectedARN == "" {
				reporter.Errorf("No %s account roles found. "+
					"You will need to run 'rosa create account-roles' to create them first.",
					role.Name)
				os.Exit(1)
			}
			if !output.HasFlag() || reporter.IsTerminal() {
				reporter.Infof("Using %s for the %s role", selectedARN, role.Name)
			}
			switch roleType {
			case aws.InstallerAccountRole:
				roleARN = selectedARN
			case aws.SupportAccountRole:
				supportRoleARN = selectedARN
			case aws.ControlPlaneAccountRole:
				controlPlaneRoleARN = selectedARN
			case aws.WorkerAccountRole:
				workerRoleARN = selectedARN
			}
		}
	}

	args.AwsRoleARN = roleARN
	args.AwsSupportRoleARN = supportRoleARN
	args.AwsControlPlaneRoleARN = controlPlaneRoleARN
	args.AwsWorkerRoleARN = workerRoleARN

	// operator role logic.
	operatorRolesPrefix := getRolePrefix(args.ClusterName)
	operatorIAMRoleList := []ocm.OperatorIAMRole{}

	for _, operator := range aws.CredentialRequests {
		//If the cluster version is less than the supported operator version
		if operator.MinVersion != "" {
			isSupported, err := ocm.CheckSupportedVersion(ocm.GetVersionMinor(version), operator.MinVersion)
			if err != nil {
				reporter.Errorf("Error validating operator role '%s' version %s", operator.Name, err)
				os.Exit(1)
			}
			if !isSupported {
				continue
			}
		}
		operatorIAMRoleList = append(operatorIAMRoleList, ocm.OperatorIAMRole{
			Name:      operator.Name,
			Namespace: operator.Namespace,
			RoleARN:   getOperatorRoleArn(operatorRolesPrefix, operator, awsCreator),
		})
	}

	// Validate the role names are available on AWS
	for _, role := range operatorIAMRoleList {
		name := strings.SplitN(role.RoleARN, "/", 2)[1]
		err := awsClient.ValidateRoleNameAvailable(name)
		if err != nil {
			reporter.Errorf("Error validating role: %v", err)
			os.Exit(1)
		}
	}

	args.AwsOperatorIamRoleList = operatorIAMRoleList
	// end operator role logic.

	// Creating the service
	service, err := ocmClient.CreateManagedService(args)
	if err != nil {
		reporter.Errorf("Failed to create managed service: %s", err)
		os.Exit(1)
	}

	reporter.Infof("%v", service)

	// The client must run these rosa commands after this for the cluster to properly install.
	rolesCMD := fmt.Sprintf("rosa create operator-roles --cluster %s", args.ClusterName)
	oidcCMD := fmt.Sprintf("rosa create oidc-provider --cluster %s", args.ClusterName)

	reporter.Infof("Run the following commands to continue the cluster creation:\n\n"+
		"\t%s\n"+
		"\t%s\n",
		rolesCMD, oidcCMD)
}

func getVersionList(ocmClient *ocm.Client) (versionList []string, err error) {
	vs, err := ocmClient.GetVersions("")
	if err != nil {
		err = fmt.Errorf("Failed to find available OpenShift versions: %s", err)
		return
	}

	for _, v := range vs {
		if !ocm.HasSTSSupport(v.RawID(), v.ChannelGroup()) {
			continue
		}
		versionList = append(versionList, v.ID())
	}

	if len(versionList) == 0 {
		err = fmt.Errorf("Failed to find available OpenShift versions")
		return
	}

	return
}

func getAccountRolePrefix(roleARN string, role aws.AccountRole) (string, error) {
	parsedARN, err := arn.Parse(roleARN)
	if err != nil {
		return "", err
	}
	roleName := strings.SplitN(parsedARN.Resource, "/", 2)[1]
	rolePrefix := strings.TrimSuffix(roleName, fmt.Sprintf("-%s-Role", role.Name))
	return rolePrefix, nil
}

func getRolePrefix(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, ocm.RandomLabel(4))
}

func getOperatorRoleArn(prefix string, operator aws.Operator, creator *aws.Creator) string {
	role := fmt.Sprintf("%s-%s-%s", prefix, operator.Namespace, operator.Name)
	if len(role) > 64 {
		role = role[0:64]
	}
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", creator.AccountID, role)
}
