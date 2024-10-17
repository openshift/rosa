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
	"net"
	"os"
	"regexp"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ocmConsts "github.com/openshift-online/ocm-common/pkg/ocm/consts"
	asv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
)

var args ocm.CreateManagedServiceArgs
var knownFlags map[string]struct{}

var Cmd = &cobra.Command{
	Use:     "managed-service",
	Aliases: []string{"appliance", "service"},
	Short:   "Creates a managed service.",
	Long: `  Managed Services are OpenShift clusters that provide a specific function.
  Use this command to create managed services.`,
	Example: `  # Create a Managed Service of type service1.
  rosa create managed-service --type=service1 --name=clusterName`,
	Run:                run,
	Hidden:             true,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, argv []string) error {
		flags := cmd.Flags()
		knownFlags = make(map[string]struct{})
		flags.VisitAll(func(flag *pflag.Flag) {
			knownFlags[flag.Name] = struct{}{}
		})

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

	flags.StringSliceVar(
		&args.SubnetIDs,
		"subnet-ids",
		nil,
		"The Subnet IDs to use when installing the cluster. "+
			"Format should be a comma-separated list. "+
			"Leave empty for installer provisioned subnet IDs.",
	)

	flags.BoolVar(
		&args.Privatelink,
		"private-link",
		false,
		"Managed service will use a cluster that won't expose traffic to the public internet.",
	)

	flags.IPNetVar(
		&args.MachineCIDR,
		"machine-cidr",
		net.IPNet{},
		"Block of IP addresses used by OpenShift while installing the cluster, for example \"10.0.0.0/16\".",
	)
	flags.IPNetVar(
		&args.ServiceCIDR,
		"service-cidr",
		net.IPNet{},
		"Block of IP addresses for services, for example \"172.30.0.0/16\".",
	)
	flags.IPNetVar(
		&args.PodCIDR,
		"pod-cidr",
		net.IPNet{},
		"Block of IP addresses from which Pod IP addresses are allocated, for example \"10.128.0.0/14\".",
	)
	flags.IntVar(
		&args.HostPrefix,
		"host-prefix",
		0,
		"Subnet prefix length to assign to each individual node. For example, if host prefix is set "+
			"to \"23\", then each node is assigned a /23 subnet out of the given CIDR.",
	)

	flags.BoolVar(
		&args.FakeCluster,
		"fake-cluster",
		false,
		"Create a fake cluster that uses no AWS resources.",
	)
	flags.MarkHidden("fake-cluster")

	arguments.AddRegionFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	if args.ServiceType == "" {
		r.Reporter.Errorf("Service type not specified.")
		cmd.Help()
		os.Exit(1)
	}

	if args.ClusterName == "" {
		r.Reporter.Errorf("Cluster name not specified.")
		cmd.Help()
		os.Exit(1)
	}

	// Get AWS region
	var err error
	args.AwsRegion, err = aws.GetRegion(arguments.GetRegion())
	if err != nil {
		r.Reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}
	r.Reporter.Debugf("Using AWS region: %q", args.AwsRegion)

	args.AwsAccountID = r.Creator.AccountID
	args.Properties = map[string]string{
		ocmConsts.CreatorArn:  r.Creator.ARN,
		properties.CLIVersion: info.DefaultVersion,
	}

	if args.FakeCluster {
		args.Properties[properties.FakeCluster] = "true"
	}

	// Openshift version to use.
	version, err := r.OCMClient.ManagedServiceVersionInquiry(args.ServiceType)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	versionMajorMinor := ocm.GetVersionMinor(version)

	// Add-on parameter logic
	addOn, err := r.OCMClient.GetAddOn(args.ServiceType)
	if err != nil {
		r.Reporter.Errorf("Failed to get add-on %q: %s", args.ServiceType, err)
		os.Exit(1)
	}
	parameters := addOn.Parameters()

	visitedFlags := map[string]struct{}{}

	if parameters.Len() > 0 {
		args.Parameters = map[string]string{}
		// Determine if all required parameters have already been set as flags.
		parameters.Each(func(param *asv1.AddonParameter) bool {
			flag := cmd.Flags().Lookup(param.ID())
			if param.Required() && (flag == nil || flag.Value.String() == "") {
				r.Reporter.Errorf("Required parameter --%s missing", param.ID())
				os.Exit(1)
			}
			if flag != nil {

				visitedFlags[flag.Name] = struct{}{}

				val := strings.Trim(flag.Value.String(), " ")
				if val != "" && param.Validation() != "" {
					isValid, err := regexp.MatchString(param.Validation(), val)
					if err != nil || !isValid {
						valErrMsg := param.ValidationErrMsg()
						if valErrMsg != "" {
							r.Reporter.Errorf("Failed to process parameter --%s: %s", param.ID(), valErrMsg)
						} else {
							r.Reporter.Errorf("Failed to process parameter --%s: Expected %v to match /%s/",
								param.ID(), val, param.Validation())
						}
						os.Exit(1)
					}
				}
				args.Parameters[param.ID()] = flag.Value.String()
			}
			return true
		})
	}

	// Ensure all flags were used.
	unusedFlags := []string{}
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if _, found := knownFlags[flag.Name]; found {
			return
		}
		if _, found := visitedFlags[flag.Name]; !found {
			unusedFlags = append(unusedFlags, flag.Name)
		}
	})
	if len(unusedFlags) > 0 {
		var flagList string
		for i, flag := range unusedFlags {
			flagList += fmt.Sprintf("%q", flag)
			if i < len(unusedFlags)-1 {
				flagList += ", "
			}
		}
		r.Reporter.Errorf("Cannot create managed service with the following unknown flags: (%s)",
			flagList)
		os.Exit(1)
	}

	// BYO-VPC Logic
	subnetIDs := args.SubnetIDs
	subnetsProvided := len(subnetIDs) > 0
	r.Reporter.Debugf("Received the following subnetIDs: %v", args.SubnetIDs)

	var availabilityZones []string
	if subnetsProvided {
		subnets, err := r.AWSClient.ListSubnets()
		if err != nil {
			r.Reporter.Errorf("Failed to get the list of subnets: %s", err)
			os.Exit(1)
		}

		mapSubnetToAZ := make(map[string]string)
		mapAZCreated := make(map[string]bool)

		// Verify subnets provided exist.
		for _, subnetArg := range subnetIDs {
			verifiedSubnet := false
			for _, subnet := range subnets {
				if awssdk.ToString(subnet.SubnetId) == subnetArg {
					verifiedSubnet = true
				}
			}
			if !verifiedSubnet {
				r.Reporter.Errorf("Could not find the following subnet provided: %s", subnetArg)
				os.Exit(1)
			}
		}

		for _, subnet := range subnets {
			subnetID := awssdk.ToString(subnet.SubnetId)
			availabilityZone := awssdk.ToString(subnet.AvailabilityZone)

			mapSubnetToAZ[subnetID] = availabilityZone
			mapAZCreated[availabilityZone] = false
		}

		for _, subnet := range subnetIDs {
			az := mapSubnetToAZ[subnet]
			if !mapAZCreated[az] {
				availabilityZones = append(availabilityZones, az)
				mapAZCreated[az] = true
			}
		}
	}

	if len(availabilityZones) > 1 {
		args.MultiAZ = true
	}
	args.AvailabilityZones = availabilityZones
	r.Reporter.Debugf("Found the following availability zones for the subnets provided: %v", availabilityZones)
	// End BYO-VPC Logic

	// Find all installer roles in the current account using AWS resource tags
	var roleARN string
	var supportRoleARN string
	var controlPlaneRoleARN string
	var workerRoleARN string

	role := aws.AccountRoles[aws.InstallerAccountRole]

	roleARNs, err := r.AWSClient.FindRoleARNs(aws.InstallerAccountRole, versionMajorMinor)
	if err != nil {
		r.Reporter.Errorf("Failed to find %s role: %s", role.Name, err)
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
		r.Reporter.Warnf("More than one %s role found, using %q", role.Name, defaultRoleARN)
		roleARN = defaultRoleARN
	} else if len(roleARNs) == 1 {
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Using %q for the %s role", roleARNs[0], role.Name)
		}
		roleARN = roleARNs[0]
	} else {
		r.Reporter.Errorf("No account roles found. " +
			"You will need to run 'rosa create account-roles' to create them first.")
		os.Exit(1)
	}

	if roleARN != "" {
		// Get role prefix
		rolePrefix, err := getAccountRolePrefix(roleARN, role)
		if err != nil {
			r.Reporter.Errorf("Failed to find prefix from %q account role", role.Name)
			os.Exit(1)
		}
		r.Reporter.Debugf("Using %q as the role prefix", rolePrefix)

		for roleType, role := range aws.AccountRoles {
			if roleType == aws.InstallerAccountRole {
				// Already dealt with
				continue
			}
			roleARNs, err := r.AWSClient.FindRoleARNs(roleType, versionMajorMinor)
			if err != nil {
				r.Reporter.Errorf("Failed to find %s role: %s", role.Name, err)
				os.Exit(1)
			}
			selectedARN := ""
			for _, rARN := range roleARNs {
				if strings.Contains(rARN, fmt.Sprintf("%s-%s-Role", rolePrefix, role.Name)) {
					selectedARN = rARN
				}
			}
			if selectedARN == "" {
				r.Reporter.Errorf("No %s account roles found. "+
					"You will need to run 'rosa create account-roles' to create them first.",
					role.Name)
				os.Exit(1)
			}
			if !output.HasFlag() || r.Reporter.IsTerminal() {
				r.Reporter.Infof("Using %q for the %s role", selectedARN, role.Name)
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

	path, err := aws.GetPathFromARN(roleARN)
	if err != nil {
		r.Reporter.Errorf("Expected a valid path for  '%s': %v", roleARN, err)
		os.Exit(1)
	}

	// operator role logic.
	operatorRolesPrefix := roles.GeOperatorRolePrefixFromClusterName(args.ClusterName)
	operatorIAMRoleList := []ocm.OperatorIAMRole{}

	// Managed Services does not support Hypershift at this time.
	credRequests, err := r.OCMClient.GetCredRequests(false)
	if err != nil {
		r.Reporter.Errorf("Error getting operator credential request from OCM %s", err)
		os.Exit(1)
	}

	for _, operator := range credRequests {
		//If the cluster version is less than the supported operator version
		if operator.MinVersion() != "" {
			isSupported, err := ocm.CheckSupportedVersion(ocm.GetVersionMinor(version), operator.MinVersion())
			if err != nil {
				r.Reporter.Errorf("Error validating operator role %q version %s", operator.Name(), err)
				os.Exit(1)
			}
			if !isSupported {
				continue
			}
		}
		operatorIAMRoleList = append(operatorIAMRoleList, ocm.OperatorIAMRole{
			Name:      operator.Name(),
			Namespace: operator.Namespace(),
			RoleARN: aws.ComputeOperatorRoleArn(operatorRolesPrefix, operator,
				r.Creator, path),
		})
	}

	// Validate the role names are available on AWS
	for _, role := range operatorIAMRoleList {
		name, err := aws.GetResourceIdFromARN(role.RoleARN)
		if err != nil {
			r.Reporter.Errorf("Error validating role: %v", err)
			os.Exit(1)
		}
		err = r.AWSClient.ValidateRoleNameAvailable(name)
		if err != nil {
			r.Reporter.Errorf("Error validating role: %v", err)
			os.Exit(1)
		}
	}

	args.AwsOperatorIamRoleList = operatorIAMRoleList
	// end operator role logic.

	// Creating the service
	service, err := r.OCMClient.CreateManagedService(args)
	if err != nil {
		r.Reporter.Errorf("Failed to create managed service: %s", err)
		os.Exit(1)
	}

	r.Reporter.Infof("Service created!\n\n\tService ID: %s\n", service.ID())

	// The client must run these rosa commands after this for the cluster to properly install.
	rolesCMD := fmt.Sprintf("rosa create operator-roles --cluster %s", args.ClusterName)
	oidcCMD := fmt.Sprintf("rosa create oidc-provider --cluster %s", args.ClusterName)

	r.Reporter.Infof("Run the following commands to continue the cluster creation:\n\n"+
		"\t%s\n"+
		"\t%s\n",
		rolesCMD, oidcCMD)
}

func getAccountRolePrefix(roleARN string, role aws.AccountRole) (string, error) {
	roleName, err := aws.GetResourceIdFromARN(roleARN)
	if err != nil {
		return "", err
	}
	rolePrefix := aws.TrimRoleSuffix(roleName, fmt.Sprintf("-%s-Role", role.Name))
	return rolePrefix, nil
}
