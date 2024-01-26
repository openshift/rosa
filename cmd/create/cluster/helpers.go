package cluster

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	clustervalidations "github.com/openshift-online/ocm-common/pkg/cluster/validations"
	accountsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/cmd/create/admin"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/helper/versions"
	"github.com/openshift/rosa/pkg/ingress"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/consts"
	interactiveOidc "github.com/openshift/rosa/pkg/interactive/oidc"
	"github.com/openshift/rosa/pkg/interactive/securitygroups"
	interactiveSgs "github.com/openshift/rosa/pkg/interactive/securitygroups"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/pflag"
)

// validateNetworkType ensure user passes a valid network type parameter at creation
func validateNetworkType(r *rosa.Runtime, input string) (string, error) {
	var networkType string
	if input == "" {
		// Parameter not specified, nothing to do
		return networkType, nil
	}
	if helper.Contains(ocm.NetworkTypes, input) {
		networkType = input
	}
	if networkType == "" {
		return "", fmt.Errorf("expected a valid network type; valid values: %v, got %s", ocm.NetworkTypes, input)
	}
	return networkType, nil
}

func GetBillingAccountContracts(cloudAccounts []*accountsv1.CloudAccount,
	billingAccount string) ([]*accountsv1.Contract, bool) {
	var contracts []*accountsv1.Contract
	for _, account := range cloudAccounts {
		if account.CloudAccountID() == billingAccount {
			contracts = account.Contracts()
			if ocm.HasValidContracts(account) {
				return contracts, true
			}
		}
	}
	return contracts, false
}

func GenerateContractDisplay(contract *accountsv1.Contract) string {
	format := "Jan 02, 2006"
	dimensions := contract.Dimensions()

	numberOfVCPUs, numberOfClusters := ocm.GetNumsOfVCPUsAndClusters(dimensions)
	prePurchaseInfo := fmt.Sprintf("   | Number of vCPUs:    |'%s'             | \n"+
		"   | Number of clusters: |'%s'             | \n",
		strconv.Itoa(numberOfVCPUs), strconv.Itoa(numberOfClusters))

	contractDisplay := "\n" +
		"   +---------------------+----------------+ \n" +
		"   | Start Date          |" + contract.StartDate().Format(format) + "    | \n" +
		"   | End Date            |" + contract.EndDate().Format(format) + "    | \n" +
		prePurchaseInfo +
		"   +---------------------+----------------+ \n"

	return contractDisplay

}

func validateOperatorRolesAvailabilityUnderUserAwsAccount(awsClient aws.Client,
	operatorIAMRoleList []ocm.OperatorIAMRole) error {
	for _, role := range operatorIAMRoleList {
		name, err := aws.GetResourceIdFromARN(role.RoleARN)
		if err != nil {
			return err
		}

		err = awsClient.ValidateRoleNameAvailable(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Options) handleOidcConfigOptions(r *rosa.Runtime, flags *pflag.FlagSet, isSTS bool, isHostedCP bool) (*v1.OidcConfig, error) {
	if !isSTS {
		return nil, nil
	}
	oidcConfigId := o.oidcConfigId
	isOidcConfig := false
	if isHostedCP && !o.classicOidcConfig {
		isOidcConfig = true
		if oidcConfigId == "" {
			interactive.Enable()
		}
	}
	if oidcConfigId == "" && interactive.Enabled() {
		if !isHostedCP {
			_isOidcConfig, err := interactive.GetBool(interactive.Input{
				Question: "Deploy cluster using pre registered OIDC Configuration ID",
				Default:  true,
				Required: true,
			})
			if err != nil {
				return nil, fmt.Errorf("expected a valid value: %w", err)
			}
			isOidcConfig = _isOidcConfig
		}
		if isOidcConfig {
			oidcConfigId = interactiveOidc.GetOidcConfigID(r, flags)
		}
	}
	if oidcConfigId == "" {
		if !isHostedCP {
			if isOidcConfig {
				r.Reporter.Warnf("No OIDC Configuration found; will continue with the classic flow.")
			}
			return nil, nil
		}
		if o.classicOidcConfig {
			return nil, nil
		}
		return nil, fmt.Errorf("hosted Control Plane requires an OIDC Configuration ID\n" +
			"Please run `rosa create oidc-config -h` and create one.")
	}
	oidcConfig, err := r.OCMClient.GetOidcConfig(oidcConfigId)
	if err != nil {
		return nil, fmt.Errorf("there was a problem retrieving OIDC Config '%s': %w", oidcConfigId, err)
	}
	return oidcConfig, nil
}

func filterPrivateSubnets(initialSubnets []*ec2.Subnet, r *rosa.Runtime) ([]*ec2.Subnet, error) {
	excludedSubnetsDueToPublic := []string{}
	filteredSubnets := []*ec2.Subnet{}
	publicSubnetMap, err := r.AWSClient.FetchPublicSubnetMap(initialSubnets)
	if err != nil {
		return nil, fmt.Errorf("unable to check if subnet have an IGW: %w", err)
	}
	for _, subnet := range initialSubnets {
		skip := false
		if isPublic, ok := publicSubnetMap[awssdk.StringValue(subnet.SubnetId)]; ok {
			if isPublic {
				excludedSubnetsDueToPublic = append(
					excludedSubnetsDueToPublic,
					awssdk.StringValue(subnet.SubnetId),
				)
				skip = true
			}
		}
		if !skip {
			filteredSubnets = append(filteredSubnets, subnet)
		}
	}
	if len(excludedSubnetsDueToPublic) > 0 {
		r.Reporter.Warnf("The following subnets have been excluded"+
			" because they have an Internet Gateway Targetded Route and the Cluster choice is private: %s",
			helper.SliceToSortedString(excludedSubnetsDueToPublic))
	}
	return filteredSubnets, nil
}

func filterCidrRangeSubnets(
	initialSubnets []*ec2.Subnet,
	machineNetwork *net.IPNet,
	serviceNetwork *net.IPNet,
	r *rosa.Runtime,
) ([]*ec2.Subnet, error) {
	excludedSubnetsDueToCidr := []string{}
	filteredSubnets := []*ec2.Subnet{}
	for _, subnet := range initialSubnets {
		skip := false
		subnetIP, subnetNetwork, err := net.ParseCIDR(*subnet.CidrBlock)
		if err != nil {
			return nil, fmt.Errorf("unable to parse subnet CIDR")
		}

		if !isValidCidrRange(subnetIP, subnetNetwork, machineNetwork, serviceNetwork) {
			excludedSubnetsDueToCidr = append(excludedSubnetsDueToCidr, awssdk.StringValue(subnet.SubnetId))
			skip = true
		}

		if !skip {
			filteredSubnets = append(filteredSubnets, subnet)
		}
	}
	if len(excludedSubnetsDueToCidr) > 0 {
		r.Reporter.Warnf("The following subnets have been excluded"+
			" because they do not fit into chosen CIDR ranges: %s", helper.SliceToSortedString(excludedSubnetsDueToCidr))
	}
	return filteredSubnets, nil
}

func isValidCidrRange(
	subnetIP net.IP,
	subnetNetwork *net.IPNet,
	machineNetwork *net.IPNet,
	serviceNetwork *net.IPNet,
) bool {
	return machineNetwork.Contains(subnetIP) &&
		!subnetNetwork.Contains(serviceNetwork.IP) &&
		!serviceNetwork.Contains(subnetIP)
}

func minReplicaValidator(multiAZ bool, isHostedCP bool, privateSubnetsCount int) interactive.Validator {
	return func(val interface{}) error {
		minReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}

		if isHostedCP && minReplicas < 2 {
			return fmt.Errorf("hosted Control Plane clusters require a minimum of 2 nodes, "+
				"but %d was requested", minReplicas)
		}

		return clustervalidations.MinReplicasValidator(minReplicas, multiAZ, isHostedCP, privateSubnetsCount)
	}
}

func maxReplicaValidator(multiAZ bool, minReplicas int, isHostedCP bool,
	privateSubnetsCount int) interactive.Validator {
	return func(val interface{}) error {
		maxReplicas, err := strconv.Atoi(fmt.Sprintf("%v", val))
		if err != nil {
			return err
		}
		return clustervalidations.MaxReplicasValidator(
			minReplicas,
			maxReplicas,
			multiAZ,
			isHostedCP,
			privateSubnetsCount,
		)
	}
}

const (
	HostPrefixMin = 23
	HostPrefixMax = 26
)

func hostPrefixValidator(val interface{}) error {
	hostPrefix, err := strconv.Atoi(fmt.Sprintf("%v", val))
	if err != nil {
		return err
	}
	if hostPrefix == 0 {
		return nil
	}
	if hostPrefix < HostPrefixMin || hostPrefix > HostPrefixMax {
		return fmt.Errorf(
			"Invalid Network Host Prefix /%d: Subnet length should be between %d and %d",
			hostPrefix, HostPrefixMin, HostPrefixMax)
	}
	return nil
}

func getAccountRolePrefix(hostedCPPolicies bool, roleARN string, roleType string) (string, error) {

	accountRoles := aws.AccountRoles

	if hostedCPPolicies {
		accountRoles = aws.HCPAccountRoles
	}

	roleName, err := aws.GetResourceIdFromARN(roleARN)
	if err != nil {
		return "", err
	}
	rolePrefix := aws.TrimRoleSuffix(roleName, fmt.Sprintf("-%s-Role", accountRoles[roleType].Name))
	return rolePrefix, nil
}

func evaluateDuration(duration time.Duration) time.Time {
	// round up to the nearest second
	return time.Now().Add(duration).Round(time.Second)
}

func validateExpiration(expirationTime string, expirationDuration time.Duration) (expiration time.Time, err error) {
	// Validate options
	if len(expirationTime) > 0 && expirationDuration != 0 {
		err = errors.New("At most one of 'expiration-time' or 'expiration' may be specified")
		return
	}

	// Parse the expiration options
	if len(expirationTime) > 0 {
		t, err := parseRFC3339(expirationTime)
		if err != nil {
			err = fmt.Errorf("failed to parse expiration-time: %w", err)
			return expiration, err
		}

		expiration = t
	}
	if expirationDuration != 0 {
		expiration = evaluateDuration(expirationDuration)
	}

	return
}

func selectAvailabilityZonesInteractively(flags *pflag.FlagSet, optionsAvailabilityZones []string,
	multiAZ bool) ([]string, error) {
	var availabilityZones []string
	var err error

	if multiAZ {
		availabilityZones, err = interactive.GetMultipleOptions(interactive.Input{
			Question: "Availability zones",
			Help:     flags.Lookup("availability-zones").Usage,
			Required: true,
			Options:  optionsAvailabilityZones,
			Validators: []interactive.Validator{
				interactive.AvailabilityZonesCountValidator(multiAZ),
			},
		})
		if err != nil {
			return availabilityZones, fmt.Errorf("expected valid availability zones: %w", err)
		}
	} else {
		var availabilityZone string
		availabilityZone, err = interactive.GetOption(interactive.Input{
			Question: "Availability zone",
			Help:     flags.Lookup("availability-zones").Usage,
			Required: true,
			Options:  optionsAvailabilityZones,
		})
		if err != nil {
			return availabilityZones, fmt.Errorf("expected valid availability zone: %w", err)
		}
		availabilityZones = append(availabilityZones, availabilityZone)
	}

	return availabilityZones, nil
}

func validateAvailabilityZones(multiAZ bool, availabilityZones []string, awsClient aws.Client) error {
	err := clustervalidations.ValidateAvailabilityZonesCount(multiAZ, len(availabilityZones))
	if err != nil {
		return err
	}

	regionAvailabilityZones, err := awsClient.DescribeAvailabilityZones()
	if err != nil {
		return fmt.Errorf("failed to get the list of the availability zone: %w", err)
	}
	for _, az := range availabilityZones {
		if !helper.Contains(regionAvailabilityZones, az) {
			return fmt.Errorf("expected a valid availability zone, "+
				"'%s' doesn't belong to region '%s' availability zones", az, awsClient.GetRegion())
		}
	}

	return nil
}

// parseRFC3339 parses an RFC3339 date in either RFC3339Nano or RFC3339 format.
func parseRFC3339(s string) (time.Time, error) {
	if t, timeErr := time.Parse(time.RFC3339Nano, s); timeErr == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func buildCommand(spec ocm.Spec, operatorRolesPrefix string,
	operatorRolePath string, userSelectedAvailabilityZones bool, labels string,
	properties []string, clusterAdminPassword string, classicOidcConfig bool, expirationDuration time.Duration) string {
	command := "rosa create cluster"
	command += fmt.Sprintf(" --cluster-name %s", spec.Name)
	if spec.IsSTS {
		command += " --sts"
		if spec.Mode != "" {
			command += fmt.Sprintf(" --mode %s", spec.Mode)
		}
	}
	if spec.ClusterAdminUser != "" {
		// Checks if admin password is from user (both flag and interactive)
		if clusterAdminPassword != "" && spec.ClusterAdminPassword != "" {
			command += fmt.Sprintf(" --cluster-admin-password %s", spec.ClusterAdminPassword)
		} else {
			command += " --create-admin-user"
		}
	}
	if spec.RoleARN != "" {
		command += fmt.Sprintf(" --role-arn %s", spec.RoleARN)
		command += fmt.Sprintf(" --support-role-arn %s", spec.SupportRoleARN)
		if !spec.Hypershift.Enabled {
			command += fmt.Sprintf(" --controlplane-iam-role %s", spec.ControlPlaneRoleARN)
		}
		command += fmt.Sprintf(" --worker-iam-role %s", spec.WorkerRoleARN)
	}
	if spec.ExternalID != "" {
		command += fmt.Sprintf(" --external-id %s", spec.ExternalID)
	}
	if operatorRolesPrefix != "" {
		command += fmt.Sprintf(" --operator-roles-prefix %s", operatorRolesPrefix)
	}
	if spec.OidcConfigId != "" {
		command += fmt.Sprintf(" --%s %s", OidcConfigIdFlag, spec.OidcConfigId)
	}
	if classicOidcConfig {
		command += fmt.Sprintf(" --%s", ClassicOidcConfigFlag)
	}
	if len(spec.Tags) > 0 {
		command += fmt.Sprintf(" --tags \"%s\"", strings.Join(buildTagsCommand(spec.Tags), ","))
	}
	if spec.MultiAZ && !spec.Hypershift.Enabled {
		command += " --multi-az"
	}
	if spec.Region != "" {
		command += fmt.Sprintf(" --region %s", spec.Region)
	}
	if spec.DisableSCPChecks != nil && *spec.DisableSCPChecks {
		command += " --disable-scp-checks"
	}
	if spec.Version != "" {
		commandVersion := ocm.GetRawVersionId(spec.Version)
		if spec.ChannelGroup != ocm.DefaultChannelGroup {
			command += fmt.Sprintf(" --channel-group %s", spec.ChannelGroup)
		}
		command += fmt.Sprintf(" --version %s", commandVersion)
	}

	if spec.Ec2MetadataHttpTokens != "" {
		command += fmt.Sprintf(" --ec2-metadata-http-tokens %s", spec.Ec2MetadataHttpTokens)
	}

	// Only account for expiration duration, as a fixed date may be obsolete if command is re-run later
	if expirationDuration != 0 {
		command += fmt.Sprintf(" --expiration %s", expirationDuration)
	}

	if spec.Autoscaling {
		command += " --enable-autoscaling"
		if spec.MinReplicas > 0 {
			command += fmt.Sprintf(" --min-replicas %d", spec.MinReplicas)
		}
		if spec.MaxReplicas > 0 {
			command += fmt.Sprintf(" --max-replicas %d", spec.MaxReplicas)
		}
	} else {
		if spec.ComputeNodes != 0 {
			command += fmt.Sprintf(" --replicas %d", spec.ComputeNodes)
		}
	}
	if spec.ComputeMachineType != "" {
		command += fmt.Sprintf(" --compute-machine-type %s", spec.ComputeMachineType)
	}

	if len(spec.ComputeLabels) != 0 {
		command += fmt.Sprintf(" --%s \"%s\"", arguments.NewDefaultMPLabelsFlag, labels)
	}

	if spec.NetworkType != "" {
		command += fmt.Sprintf(" --network-type %s", spec.NetworkType)
	}
	if !ocm.IsEmptyCIDR(spec.MachineCIDR) {
		command += fmt.Sprintf(" --machine-cidr %s", spec.MachineCIDR.String())
	}
	if !ocm.IsEmptyCIDR(spec.ServiceCIDR) {
		command += fmt.Sprintf(" --service-cidr %s", spec.ServiceCIDR.String())
	}
	if !ocm.IsEmptyCIDR(spec.PodCIDR) {
		command += fmt.Sprintf(" --pod-cidr %s", spec.PodCIDR.String())
	}
	if spec.HostPrefix != 0 {
		command += fmt.Sprintf(" --host-prefix %d", spec.HostPrefix)
	}
	if spec.PrivateLink != nil && *spec.PrivateLink {
		command += " --private-link"
	} else if spec.Private != nil && *spec.Private {
		command += " --private"
	}
	if len(spec.SubnetIds) > 0 {
		command += fmt.Sprintf(" --subnet-ids %s", strings.Join(spec.SubnetIds, ","))
	}
	if spec.PrivateHostedZoneID != "" {
		command += fmt.Sprintf(" --private-hosted-zone-id %s", spec.PrivateHostedZoneID)
		command += fmt.Sprintf(" --shared-vpc-role-arn %s", spec.SharedVPCRoleArn)
		command += fmt.Sprintf(" --base-domain %s", spec.BaseDomain)
	}
	if spec.FIPS {
		command += " --fips"
	} else if spec.EtcdEncryption {
		command += " --etcd-encryption"
		if spec.EtcdEncryptionKMSArn != "" {
			command += fmt.Sprintf(" --etcd-encryption-kms-arn %s", spec.EtcdEncryptionKMSArn)
		}
	}

	if spec.EnableProxy {
		if spec.HTTPProxy != nil && *spec.HTTPProxy != "" {
			command += fmt.Sprintf(" --http-proxy %s", *spec.HTTPProxy)
		}
		if spec.HTTPSProxy != nil && *spec.HTTPSProxy != "" {
			command += fmt.Sprintf(" --https-proxy %s", *spec.HTTPSProxy)
		}
		if spec.NoProxy != nil && *spec.NoProxy != "" {
			command += fmt.Sprintf(" --no-proxy \"%s\"", *spec.NoProxy)
		}
	}
	if spec.AdditionalTrustBundleFile != nil && *spec.AdditionalTrustBundleFile != "" {
		command += fmt.Sprintf(" --additional-trust-bundle-file %s", *spec.AdditionalTrustBundleFile)
	}
	if spec.KMSKeyArn != "" {
		command += fmt.Sprintf(" --kms-key-arn %s", spec.KMSKeyArn)
	}
	if spec.DisableWorkloadMonitoring != nil && *spec.DisableWorkloadMonitoring {
		command += " --disable-workload-monitoring"
	}
	if userSelectedAvailabilityZones {
		command += fmt.Sprintf(" --availability-zones %s", strings.Join(spec.AvailabilityZones, ","))
	}
	if spec.Hypershift.Enabled {
		command += " --hosted-cp"
	}

	if spec.AuditLogRoleARN != nil && *spec.AuditLogRoleARN != "" {
		command += fmt.Sprintf(" --audit-log-arn %s", *spec.AuditLogRoleARN)
	}
	if spec.MachinePoolRootDisk != nil {
		machinePoolRootDiskSize := spec.MachinePoolRootDisk.Size
		if machinePoolRootDiskSize != 0 {
			command += fmt.Sprintf(" --%s %dGiB", workerDiskSizeFlag, machinePoolRootDiskSize)
		}
	}

	if !reflect.DeepEqual(spec.DefaultIngress, ocm.NewDefaultIngressSpec()) {
		if len(spec.DefaultIngress.RouteSelectors) != 0 {
			selectors := []string{}
			for k, v := range spec.DefaultIngress.RouteSelectors {
				selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
			}
			command += fmt.Sprintf(" --%s %s", ingress.DefaultIngressRouteSelectorFlag, strings.Join(selectors, ","))
		}
		if len(spec.DefaultIngress.ExcludedNamespaces) != 0 {
			command += fmt.Sprintf(" --%s %s", ingress.DefaultIngressExcludedNamespacesFlag,
				strings.Join(spec.DefaultIngress.ExcludedNamespaces, ","))
		}
		if !helper.Contains([]string{"", consts.SkipSelectionOption}, spec.DefaultIngress.WildcardPolicy) {
			command += fmt.Sprintf(
				" --%s %s",
				ingress.DefaultIngressWildcardPolicyFlag,
				spec.DefaultIngress.WildcardPolicy,
			)
		}
		if !helper.Contains([]string{"", consts.SkipSelectionOption}, spec.DefaultIngress.NamespaceOwnershipPolicy) {
			command += fmt.Sprintf(" --%s %s", ingress.DefaultIngressNamespaceOwnershipPolicyFlag,
				spec.DefaultIngress.NamespaceOwnershipPolicy)
		}
	}

	command += clusterautoscaler.BuildAutoscalerOptions(spec.AutoscalerConfig, clusterAutoscalerFlagsPrefix)

	if len(spec.AdditionalComputeSecurityGroupIds) > 0 {
		command += fmt.Sprintf(" --%s %s",
			securitygroups.ComputeSecurityGroupFlag,
			strings.Join(spec.AdditionalComputeSecurityGroupIds, ","))
	}

	if len(spec.AdditionalInfraSecurityGroupIds) > 0 {
		command += fmt.Sprintf(" --%s %s",
			securitygroups.InfraSecurityGroupFlag,
			strings.Join(spec.AdditionalInfraSecurityGroupIds, ","))
	}

	if len(spec.AdditionalControlPlaneSecurityGroupIds) > 0 {
		command += fmt.Sprintf(" --%s %s",
			securitygroups.ControlPlaneSecurityGroupFlag,
			strings.Join(spec.AdditionalControlPlaneSecurityGroupIds, ","))
	}

	if spec.BillingAccount != "" {
		command += fmt.Sprintf(" --billing-account %s", spec.BillingAccount)
	}

	if spec.NoCni {
		command += " --no-cni"
	}

	for _, p := range properties {
		command += fmt.Sprintf(" --properties \"%s\"", p)
	}
	return command
}

func buildTagsCommand(tags map[string]string) []string {
	// set correct delim, if a key or value contains `:` the delim should be " "
	delim := ":"
	for k, v := range tags {
		if strings.Contains(k, ":") || strings.Contains(v, ":") {
			delim = " "
			break
		}
	}

	// build list of formatted tags to return in command
	var formattedTags []string
	for k, v := range tags {
		formattedTags = append(formattedTags, fmt.Sprintf("%s%s%s", k, delim, v))
	}
	return formattedTags
}

func getRolePrefix(clusterName string) string {
	return fmt.Sprintf("%s-%s", clusterName, helper.RandomLabel(4))
}

func calculateReplicas(
	isMinReplicasSet bool,
	isMaxReplicasSet bool,
	minReplicas int,
	maxReplicas int,
	privateSubnetsCount int,
	isHostedCP bool,
	multiAZ bool) (int, int) {

	newMinReplicas := minReplicas
	newMaxReplicas := maxReplicas

	if !isMinReplicasSet {
		if isHostedCP {
			newMinReplicas = privateSubnetsCount
			if privateSubnetsCount == 1 {
				newMinReplicas = MinReplicasSingleAZ
			}
		} else {
			if multiAZ {
				newMinReplicas = MinReplicaMultiAZ
			}
		}
	}
	if !isMaxReplicasSet && multiAZ {
		newMaxReplicas = newMinReplicas
	}

	return newMinReplicas, newMaxReplicas
}

func getExpectedResourceIDForAccRole(hostedCPPolicies bool, roleARN string, roleType string) (string, string, error) {

	accountRoles := aws.AccountRoles

	rolePrefix, err := getAccountRolePrefix(hostedCPPolicies, roleARN, aws.InstallerAccountRole)
	if err != nil {
		return "", "", err
	}

	if hostedCPPolicies {
		accountRoles = aws.HCPAccountRoles
	}

	return strings.ToLower(fmt.Sprintf("%s-%s-Role", rolePrefix, accountRoles[roleType].Name)), rolePrefix, nil
}

func getInitialValidSubnets(aws aws.Client, ids []string, r *reporter.Object) ([]*ec2.Subnet, error) {
	initialValidSubnets := []*ec2.Subnet{}
	excludedSubnets := []string{}

	validSubnets, err := aws.ListSubnets(ids...)

	if err != nil {
		return initialValidSubnets, err
	}
	for _, subnet := range validSubnets {
		hasRHManaged := tags.Ec2ResourceHasTag(subnet.Tags, tags.RedHatManaged, strconv.FormatBool(true))
		if !hasRHManaged {
			initialValidSubnets = append(initialValidSubnets, subnet)
		} else {
			excludedSubnets = append(excludedSubnets, awssdk.StringValue(subnet.SubnetId))
		}
	}
	if len(validSubnets) != len(initialValidSubnets) {
		r.Warnf("The following subnets were excluded because they belong"+
			" to a VPC that is managed by Red Hat: %s", helper.SliceToSortedString(excludedSubnets))
	}
	return initialValidSubnets, nil
}

func outputClusterAdminDetails(r *rosa.Runtime, isClusterAdmin bool, createAdminPassword string) {
	if isClusterAdmin {
		r.Reporter.Infof("cluster admin user is %s", admin.ClusterAdminUsername)
		r.Reporter.Infof("cluster admin password is %s", createAdminPassword)
	}
}

func getSecurityGroups(r *rosa.Runtime, flags *pflag.FlagSet, isVersionCompatibleComputeSgIds bool,
	kind string, useExistingVpc bool, isHostedCp bool, currentSubnets []*ec2.Subnet, subnetIds []string,
	additionalSgIds *[]string) error {
	hasChangedSgIdsFlag := flags.Changed(securitygroups.SgKindFlagMap[kind])
	if hasChangedSgIdsFlag {
		if !useExistingVpc {
			return fmt.Errorf("setting the `%s` flag is only allowed for BYO VPC clusters",
				securitygroups.SgKindFlagMap[kind])
		}
		// HCP is still unsupported
		if isHostedCp {
			return fmt.Errorf("parameter '%s' is not supported for Hosted Control Plane clusters",
				securitygroups.SgKindFlagMap[kind])
		}
		if !isVersionCompatibleComputeSgIds {
			formattedVersion, err := versions.FormatMajorMinorPatch(
				ocm.MinVersionForAdditionalComputeSecurityGroupIdsDay1,
			)
			if err != nil {
				return fmt.Errorf(versions.MajorMinorPatchFormattedErrorOutput, err)
			}
			return fmt.Errorf("parameter '%s' is not supported prior to version '%s'",
				securitygroups.SgKindFlagMap[kind], formattedVersion)
		}
	} else if interactive.Enabled() && isVersionCompatibleComputeSgIds && useExistingVpc && !isHostedCp {
		vpcId := ""
		for _, subnet := range currentSubnets {
			if awssdk.StringValue(subnet.SubnetId) == subnetIds[0] {
				vpcId = awssdk.StringValue(subnet.VpcId)
			}
		}
		if vpcId == "" {
			r.Reporter.Warnf("Unexpected situation a VPC ID should have been selected based on chosen subnets")
		}
		*additionalSgIds = interactiveSgs.
			GetSecurityGroupIds(r, flags, vpcId, kind)
	}
	for i, sg := range *additionalSgIds {
		(*additionalSgIds)[i] = strings.TrimSpace(sg)
	}
	return nil
}
