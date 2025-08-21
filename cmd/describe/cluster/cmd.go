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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	ocmConsts "github.com/openshift-online/ocm-common/pkg/ocm/consts"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper/rolepolicybindings"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	StageURL      = "https://console.dev.redhat.com/openshift/details/s/"
	ProductionURL = "https://console.redhat.com/openshift/details/s/"
	StageEnv      = "https://api.stage.openshift.com"
	ProductionEnv = "https://api.openshift.com"

	EnabledOutput  = "Enabled"
	DisabledOutput = "Disabled"
)

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Show details of a cluster",
	Long:  "Show details of a cluster",
	Example: `  # Describe a cluster named "mycluster"
  rosa describe cluster --cluster=mycluster`,
	Run:  run,
	Args: cobra.MaximumNArgs(1),
}

var args struct {
	getRolePolicyBindings bool
}

func init() {
	output.AddFlag(Cmd)
	ocm.AddClusterFlag(Cmd)

	Cmd.Flags().BoolVar(
		&args.getRolePolicyBindings,
		"get-role-policy-bindings",
		false,
		"List the attached policies for the sts roles",
	)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM().WithAWS()
	defer r.Cleanup()

	var err error

	// Allow the command to be called programmatically
	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
	}
	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	isHypershift := cluster.Hypershift().Enabled()

	displayName := ""
	subscription, subscriptionExists, err := r.OCMClient.GetSubscriptionBySubscriptionID(cluster.Subscription().ID())
	if err != nil {
		r.Reporter.Debugf("Failed to get subscription by ID: %s", err)
	}
	if subscriptionExists {
		displayName = subscription.DisplayName()
	}

	var scheduledUpgrade *cmv1.UpgradePolicy
	var upgradeState *cmv1.UpgradePolicyState
	var controlPlaneScheduledUpgrade *cmv1.ControlPlaneUpgradePolicy

	// SDN -> OVN Migration Status
	migrationsStr := ""
	collection, err := r.OCMClient.FetchClusterMigrations(cluster.ID())
	migrations := map[string]*cmv1.ClusterMigration{}
	if err != nil {
		if !strings.Contains(err.Error(), "is not enabled for this organization") {
			r.Reporter.Warnf("Failed to get cluster migrations: %v", err)
		}
	} else if len(collection.Items().Items()) > 0 {

		for _, migration := range collection.Items().Items() {
			state, ok := migration.State().GetValue()
			if !ok {
				r.Reporter.Warnf("Failed to get cluster migration state")
			}
			if state != cmv1.ClusterMigrationStateValueCompleted {
				migrations[migration.ID()] = migration
			}
		}
	}

	if !isHypershift {
		scheduledUpgrade, upgradeState, err = r.OCMClient.GetScheduledUpgrade(cluster.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		if output.HasFlag() {
			f, err := formatCluster(cluster, scheduledUpgrade, upgradeState, displayName, migrations)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
			err = output.Print(f)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
			return
		}
	} else {
		controlPlaneScheduledUpgrade, err = r.OCMClient.GetControlPlaneScheduledUpgrade(cluster.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		if output.HasFlag() {
			f, err := formatClusterHypershift(cluster, controlPlaneScheduledUpgrade, displayName, migrations)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
			err = output.Print(f)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
			return
		}
	}

	var str string
	creatorARN, err := arn.Parse(cluster.Properties()[ocmConsts.CreatorArn])
	if err != nil {
		r.Reporter.Errorf("Failed to parse creator ARN for cluster '%s'", clusterKey)
		os.Exit(1)
	}
	phase := ""

	switch cluster.State() {
	case cmv1.ClusterStateWaiting:
		phase = "(Waiting for user action)"
	case cmv1.ClusterStatePending:
		phase = "(Preparing account)"
	case cmv1.ClusterStateInstalling:
		if !cluster.Status().DNSReady() {
			phase = "(DNS setup in progress)"
		}
		if cluster.Status().ProvisionErrorMessage() != "" {
			errorCode := ""
			if cluster.Status().ProvisionErrorCode() != "" {
				errorCode = cluster.Status().ProvisionErrorCode() + " - "
			}
			phase = "(" + errorCode + "Install is taking longer than expected)"
		}
	}
	if cluster.Status().Description() != "" {
		phase = fmt.Sprintf("(%s)", cluster.Status().Description())
	}

	domainPrefix := cluster.DomainPrefix()

	clusterDNS := "Not ready"
	if cluster.Status() != nil && cluster.Status().DNSReady() {
		clusterDNS = strings.Join([]string{domainPrefix, cluster.DNS().BaseDomain()}, ".")
	}

	clusterName := cluster.Name()

	isPrivate := output.No
	if cluster.API().Listening() == cmv1.ListeningMethodInternal {
		isPrivate = output.Yes
	}

	detailsPage := getDetailsLink(r.OCMClient.GetConnectionURL())

	networkType := ""
	if cluster.Network().Type() != ocm.NetworkTypes[0] {
		networkType = fmt.Sprintf(
			" - Type:                    %s\n",
			cluster.Network().Type(),
		)
	}

	// SDN -> OVN Migration Status
	if len(collection.Items().Items()) > 0 {

		if len(migrations) > 0 {
			migrationsStr = "Migrations:\n"
			for migrationID, migration := range migrations {
				migrationsStr += fmt.Sprintf(" - %s \n"+
					"    - Type:                 %s\n"+
					"    - State:                %s\n"+
					"    - Description:          %s\n", migrationID, migration.Type(), migration.State().Value(),
					migration.State().Description(),
				)
			}
		}
	}

	subnetsStr := ""
	if len(cluster.AWS().SubnetIDs()) > 0 {
		subnetsStr = fmt.Sprintf(" - Subnets:                 %s\n",
			output.PrintStringSlice(cluster.AWS().SubnetIDs()))
	}

	var machinePools []*cmv1.MachinePool
	var nodePools []*cmv1.NodePool

	if isHypershift {
		nodePools, err = r.OCMClient.GetNodePools(cluster.ID())
	} else {
		machinePools, err = r.OCMClient.GetMachinePools(cluster.ID())
	}
	if err != nil {
		r.Reporter.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Print short cluster description:
	str = fmt.Sprintf("\n"+
		"Name:                       %s\n"+
		"Domain Prefix:              %s\n"+
		"Display Name:               %s\n"+
		"ID:                         %s\n"+
		"External ID:                %s\n"+
		"Control Plane:              %s\n"+
		"OpenShift Version:          %s\n"+
		"Channel Group:              %s\n"+
		"DNS:                        %s\n"+
		"AWS Account:                %s\n"+
		"%s"+
		"API URL:                    %s\n"+
		"Console URL:                %s\n"+
		"Region:                     %s\n"+
		"%s"+
		"%s"+
		"Network:\n"+
		"%s"+
		" - Service CIDR:            %s\n"+
		" - Machine CIDR:            %s\n"+
		" - Pod CIDR:                %s\n"+
		" - Host Prefix:             /%d\n"+
		"%s"+
		"%s"+
		"%s",
		clusterName,
		domainPrefix,
		displayName,
		cluster.ID(),
		cluster.ExternalID(),
		controlPlaneConfig(cluster),
		cluster.OpenshiftVersion(),
		cluster.Version().ChannelGroup(),
		clusterDNS,
		creatorARN.AccountID,
		BillingAccount(cluster),
		cluster.API().URL(),
		cluster.Console().URL(),
		cluster.Region().ID(),
		clusterMultiAZ(cluster, nodePools),
		clusterInfraConfig(cluster, clusterKey, r, machinePools, nodePools),
		networkType,
		cluster.Network().ServiceCIDR(),
		cluster.Network().MachineCIDR(),
		cluster.Network().PodCIDR(),
		cluster.Network().HostPrefix(),
		migrationsStr,
		subnetsStr,
		str,
	)

	if cluster.InfraID() != "" {
		str = fmt.Sprintf("%s"+"Infra ID:                   %s\n", str, cluster.InfraID())
	}

	if cluster.Proxy() != nil && (cluster.Proxy().HTTPProxy() != "" || cluster.Proxy().HTTPSProxy() != "") {
		str = fmt.Sprintf("%s"+"Proxy:\n", str)
		if cluster.Proxy().HTTPProxy() != "" {
			str = fmt.Sprintf("%s"+
				" - HTTPProxy:               %s\n", str,
				cluster.Proxy().HTTPProxy())
		}
		if cluster.Proxy().HTTPSProxy() != "" {
			str = fmt.Sprintf("%s"+
				" - HTTPSProxy:              %s\n", str,
				cluster.Proxy().HTTPSProxy())
		}
		if cluster.Proxy().NoProxy() != "" {
			str = fmt.Sprintf("%s"+
				" - NoProxy:                 %s\n", str,
				cluster.Proxy().NoProxy())
		}
	}

	if cluster.AdditionalTrustBundle() != "" {
		str = fmt.Sprintf("%s"+"Additional trust bundle:    REDACTED\n", str)
	}

	if cluster.AWS().Ec2MetadataHttpTokens() != "" {
		str = fmt.Sprintf("%s"+"EC2 Metadata Http Tokens:   %s\n", str, cluster.AWS().Ec2MetadataHttpTokens())
	} else {
		// show default value for clusters that didn't set it.
		str = fmt.Sprintf("%s"+"EC2 Metadata Http Tokens:   %s\n", str, cmv1.Ec2MetadataHttpTokensOptional)
	}

	if cluster.AWS().STS().RoleARN() != "" {
		rolePolicyDetails := map[string][]aws.PolicyDetail{}
		if args.getRolePolicyBindings {
			rolePolicyBindings, err := r.OCMClient.ListRolePolicyBindings(cluster.ID(), true)
			if err != nil {
				r.Reporter.Errorf("Failed to get rolePolicyBinding: %s", err)
				os.Exit(1)
			}
			rolePolicyDetails = rolepolicybindings.TransformToRolePolicyDetails(rolePolicyBindings)
		}
		str = fmt.Sprintf("%s"+
			"Role (STS) ARN:             %s\n", str,
			cluster.AWS().STS().RoleARN())
		if args.getRolePolicyBindings {
			policyStr, err := getRolePolicyBindings(cluster.AWS().STS().RoleARN(), rolePolicyDetails,
				"                            -")
			if err != nil {
				r.Reporter.Errorf(err.Error())
				os.Exit(1)
			}
			str = str + policyStr
		}
		if cluster.AWS().STS().ExternalID() != "" {
			str = fmt.Sprintf("%s"+
				"STS External ID:            %s\n", str,
				cluster.AWS().STS().ExternalID())
		}
		if cluster.AWS().STS().SupportRoleARN() != "" {
			str = fmt.Sprintf("%s"+
				"Support Role ARN:           %s\n", str,
				cluster.AWS().STS().SupportRoleARN())
			if args.getRolePolicyBindings {
				policyStr, err := getRolePolicyBindings(cluster.AWS().STS().SupportRoleARN(), rolePolicyDetails,
					"                            -")
				if err != nil {
					r.Reporter.Errorf(err.Error())
					os.Exit(1)
				}
				str = str + policyStr
			}
		}
		if cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN() != "" ||
			cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN() != "" {
			str = fmt.Sprintf("%sInstance IAM Roles:\n", str)
			if cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN() != "" {
				str = fmt.Sprintf("%s"+
					" - Control plane:           %s\n", str,
					cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN())
				if args.getRolePolicyBindings {
					policyStr, err := getRolePolicyBindings(cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN(),
						rolePolicyDetails,
						"                            -")
					if err != nil {
						r.Reporter.Errorf(err.Error())
						os.Exit(1)
					}
					str = str + policyStr
				}
			}
			if cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN() != "" {
				str = fmt.Sprintf("%s"+
					" - Worker:                  %s\n", str,
					cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN())
				if args.getRolePolicyBindings {
					policyStr, err := getRolePolicyBindings(cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN(),
						rolePolicyDetails,
						"                            -")
					if err != nil {
						r.Reporter.Errorf(err.Error())
						os.Exit(1)
					}
					str = str + policyStr
				}
			}
		}
		if len(cluster.AWS().STS().OperatorIAMRoles()) > 0 {
			str = fmt.Sprintf("%sOperator IAM Roles:\n", str)
			for _, operatorIAMRole := range cluster.AWS().STS().OperatorIAMRoles() {
				str = fmt.Sprintf("%s"+
					" - %s\n", str,
					operatorIAMRole.RoleARN())
				if args.getRolePolicyBindings {
					policyStr, err := getRolePolicyBindings(operatorIAMRole.RoleARN(),
						rolePolicyDetails,
						"   -")
					if err != nil {
						r.Reporter.Errorf(err.Error())
						os.Exit(1)
					}
					str = str + policyStr
				}
			}
		}

		awsManaged := output.No
		if cluster.AWS().STS().ManagedPolicies() {
			awsManaged = output.Yes
		}
		str = fmt.Sprintf("%sManaged Policies:           %s\n", str, awsManaged)
	}

	deleteProtection := DisabledOutput
	if cluster.DeleteProtection().Enabled() {
		deleteProtection = EnabledOutput
	}

	str = fmt.Sprintf("%s"+
		"State:                      %s %s\n"+
		"Private:                    %s\n"+
		"Delete Protection:          %s\n"+
		"Created:                    %s\n",
		str,
		cluster.State(), phase,
		isPrivate,
		deleteProtection,
		cluster.CreationTimestamp().Format("Jan _2 2006 15:04:05 MST"))

	uwmString := "%s" +
		"User Workload Monitoring:   %s\n"
	if isHypershift {
		uwmString = "%s" +
			"[DEPRECATED] User Workload Monitoring:   %s\n"
	}

	str = fmt.Sprintf(uwmString,
		str,
		getUseworkloadMonitoring(cluster.DisableUserWorkloadMonitoring()))

	if cluster.FIPS() {
		str = fmt.Sprintf("%s"+
			"FIPS mode:                  %s\n",
			str,
			EnabledOutput)
	}
	if detailsPage != "" {
		str = fmt.Sprintf("%s"+
			"Details Page:               %s%s\n", str,
			detailsPage, cluster.Subscription().ID())
	}
	managementType := "Classic"
	if cluster.AWS().STS().OidcConfig() != nil {
		managementType = "Unmanaged"
		if cluster.AWS().STS().OidcConfig().Managed() {
			managementType = "Managed"
		}
	}
	if cluster.AWS().STS().OIDCEndpointURL() != "" {
		str = fmt.Sprintf("%s"+
			"OIDC Endpoint URL:          %s (%s)\n", str,
			cluster.AWS().STS().OIDCEndpointURL(), managementType)
	}
	if cluster.AWS().PrivateHostedZoneID() != "" {
		if cluster.Hypershift().Enabled() {
			str = fmt.Sprintf("%s"+"Shared VPC Config:\n", str)
			str = fmt.Sprintf("%s"+
				" - Ingress Private HZ ID:   %s\n", str,
				cluster.AWS().PrivateHostedZoneID())
			str = fmt.Sprintf("%s"+
				" - Internal Comms. HZ ID:   %s\n", str,
				cluster.AWS().HcpInternalCommunicationHostedZoneId())
			str = fmt.Sprintf("%s"+
				" - Route53 Role ARN:        %s\n", str,
				cluster.AWS().PrivateHostedZoneRoleARN())
			str = fmt.Sprintf("%s"+
				" - VPC Endpoint Role ARN:   %s\n", str,
				cluster.AWS().VpcEndpointRoleArn())
		} else {
			str = fmt.Sprintf("%s"+"Private Hosted Zone:\n", str)
			str = fmt.Sprintf("%s"+
				" - ID:                      %s\n", str,
				cluster.AWS().PrivateHostedZoneID())
			str = fmt.Sprintf("%s"+
				" - Role ARN:                %s\n", str,
				cluster.AWS().PrivateHostedZoneRoleARN())
		}
	}
	if !isHypershift {
		if scheduledUpgrade != nil {
			str = fmt.Sprintf("%s"+
				"Scheduled Upgrade:          %s %s on %s\n",
				str,
				upgradeState.Value(),
				scheduledUpgrade.Version(),
				scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
			)
		}
	} else {
		if controlPlaneScheduledUpgrade != nil {
			str = fmt.Sprintf("%s"+
				"Scheduled Upgrade:          %s %s on %s\n",
				str,
				controlPlaneScheduledUpgrade.State().Value(),
				controlPlaneScheduledUpgrade.Version(),
				controlPlaneScheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
			)
		}
	}

	// etcd encryption is shared between classic and hcp
	str = fmt.Sprintf("%s"+
		"Etcd Encryption:            %s\n", str, getEtcdStatus(cluster))

	if isHypershift {
		// kms key should be next to etcd encryption
		if cluster.AWS().EtcdEncryption().KMSKeyARN() != "" {
			str = fmt.Sprintf("%s"+
				"Etcd KMS key ARN:           %s\n", str, cluster.AWS().EtcdEncryption().KMSKeyARN())
		}
		str = fmt.Sprintf("%s"+
			"Audit Log Forwarding:       %s\n", str, getAuditLogForwardingStatus(cluster))
		if cluster.AWS().AuditLog().RoleArn() != "" {
			str = fmt.Sprintf("%s"+
				"Audit Log Role ARN:         %s\n", str, cluster.AWS().AuditLog().RoleArn())
		}
		str = fmt.Sprintf("%s"+
			"External Authentication:    %s\n", str, getExternalAuthConfigStatus(cluster))
		if len(cluster.AWS().AdditionalAllowedPrincipals()) > 0 {
			// Omitted the 'Allowed' due to formatting
			str = fmt.Sprintf("%s"+
				"Additional Principals:      %s\n", str,
				strings.Join(cluster.AWS().AdditionalAllowedPrincipals(), ","))
		}
		if cluster.RegistryConfig() != nil {
			var allowlist *cmv1.RegistryAllowlist
			if cluster.RegistryConfig().PlatformAllowlist().ID() != "" {
				allowlist, err = r.OCMClient.GetAllowlist(cluster.RegistryConfig().PlatformAllowlist().ID())
				if err != nil {
					r.Reporter.Errorf("Failed to get allowlist with id '%s': %v",
						cluster.RegistryConfig().PlatformAllowlist().ID(), err)
					os.Exit(1)
				}
			}
			registryConfigOutput := getClusterRegistryConfig(cluster, allowlist)
			str = fmt.Sprintf("%s"+
				"Registry Configuration:\n"+
				"%s\n", str, registryConfigOutput)
		}

		zeroEgressStatus, err := getZeroEgressStatus(r, cluster)
		if err != nil {
			r.Reporter.Errorf("Failed to get zero egress info: %v", err)
			os.Exit(1)
		}
		if zeroEgressStatus != "" {
			str = fmt.Sprintf("%s"+"%s\n", str, zeroEgressStatus)
		}
	}

	if cluster.Status().State() == cmv1.ClusterStateError {
		str = fmt.Sprintf("%s"+
			"Provisioning Error Code:    %s\n"+
			"Provisioning Error Message: %s\n",
			str,
			cluster.Status().ProvisionErrorCode(),
			cluster.Status().ProvisionErrorMessage(),
		)
	}

	limitedSupportReasons, err := r.OCMClient.GetLimitedSupportReasons(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get limited support reasons for cluster '%s': %v", cluster.ID(), err)
		os.Exit(1)
	}
	if len(limitedSupportReasons) > 0 {
		str = fmt.Sprintf("%s"+"Limited Support:\n", str)
	}
	for _, reason := range limitedSupportReasons {
		str = fmt.Sprintf("%s"+
			" - Summary:                 %s\n"+
			" - Details:                 %s\n",
			str, reason.Summary(), reason.Details())
	}

	inflightChecks, err := r.OCMClient.GetInflightChecks(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get inflight checks for cluster '%s': %v", cluster.ID(), err)
		os.Exit(1)
	}
	if len(inflightChecks) > 0 {
		summaries := []string{}
		for _, inflight := range inflightChecks {
			if inflight.State() != "failed" {
				continue
			}
			if inflight.Name() != "egress" {
				continue
			}
			summary := fmt.Sprintf("\t"+
				"ID:                 %s\n"+
				"\tLast run:           %s\n",
				inflight.ID(), inflight.EndedAt().Format("Jan _2 2006 15:04:05 MST"))
			details, err := parseInflightCheckDetails(inflight)
			if err != nil {
				r.Logger.Errorf("Unexpected error parsing inflight details '%s: %v", inflight.ID(), err)
				continue
			}
			summary += details
			summaries = append(summaries, summary)
		}
		if len(summaries) > 0 {
			str += fmt.Sprintf("Failed Inflight Checks:\n%s\n", strings.Join(summaries, "\n"))
			str += fmt.Sprintf("\tPlease run `rosa verify network -c %s` after adjusting"+
				" the cluster's network configuration to remove the warning", cluster.Name())
		}
	}

	str = fmt.Sprintf("%s\n", str)

	// Print short cluster description:
	fmt.Print(str)
}

var mapInflightErrorTypeToTitle = map[string]string{
	"egress_url_errors": "Egress URL access issues",
	"tag_violation":     "Tag violation",
}

func parseInflightCheckDetails(inflight *cmv1.InflightCheck) (string, error) {
	var inflightDetails map[string]interface{}
	out, err := json.Marshal(inflight.Details())
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(out, &inflightDetails)
	if err != nil {
		return "", err
	}
	details := ""
	for inflightKey, inflightValue := range inflightDetails {
		// send non egress error details to service log
		if inflightKey == "error" {
			details += fmt.Sprintf("\tAn inflight error '%s' has occurred: %s", inflightKey, inflightValue.(string))
			continue
		}
		if !strings.Contains(inflightKey, "subnet") {
			continue
		}
		subnetDetails := inflightValue.(map[string]interface{})
		mapTypeToErrors := map[string][]string{}
		for key := range mapInflightErrorTypeToTitle {
			mapTypeToErrors[key] = []string{}
		}
		for subnetKey, subnetValue := range subnetDetails {
			// Remove index from error type key
			lastIndex := strings.LastIndex(subnetKey, "-")
			adjustedKey := subnetKey
			if lastIndex != -1 {
				adjustedKey = subnetKey[:lastIndex]
			}
			if adjustedKey != "egress_url_errors" {
				continue
			}
			// Keep only the url in the reason
			firstIndex := strings.Index(subnetValue.(string), ":")
			errorDescription := subnetValue.(string)[firstIndex+1:]

			mapTypeToErrors[adjustedKey] = append(mapTypeToErrors[subnetKey], strings.TrimSpace(errorDescription))
		}
		for key, errors := range mapTypeToErrors {
			if len(errors) > 0 {
				details += fmt.Sprintf("\tInvalid configurations on subnet '%s' have been identified: \n", inflightKey)
				errorDetails := fmt.Sprintf("\t\tDetails for '%s':\n", mapInflightErrorTypeToTitle[key])
				for _, errReason := range errors {
					errorDetails += fmt.Sprintf("\t\t\t- %s\n", errReason)
				}
				details += errorDetails
			}
		}
	}
	return details, nil
}

func controlPlaneConfig(cluster *cmv1.Cluster) string {
	if cluster.Hypershift().Enabled() {
		return "ROSA Service Hosted"
	}
	return "Customer Hosted"
}

func clusterMultiAZ(cluster *cmv1.Cluster, nodePools []*cmv1.NodePool) string {
	var multiaz string
	if cluster.Hypershift().Enabled() {
		dataPlaneAvailability := "SingleAZ"
		if cluster.NodePools() != nil {
			multiAzMap := make(map[string]struct{})
			for _, nodePool := range nodePools {
				multiAzMap[nodePool.AvailabilityZone()] = struct{}{}
			}
			if len(multiAzMap) > 1 {
				dataPlaneAvailability = "MultiAZ"
			}
		}
		multiaz = fmt.Sprintf("Availability:\n"+
			" - Control Plane:           MultiAZ\n"+
			" - Data Plane:              %s\n",
			dataPlaneAvailability)
	} else {
		multiaz = fmt.Sprintf("Multi-AZ:                   %t\n", cluster.MultiAZ())
	}
	return multiaz
}

func clusterInfraConfig(cluster *cmv1.Cluster, clusterKey string, r *rosa.Runtime,
	machinePools []*cmv1.MachinePool, nodePools []*cmv1.NodePool) string {
	var nodeConfig string
	if cluster.Hypershift().Enabled() {
		minNodes := 0
		maxNodes := 0
		currentNodes := 0
		// Accumulate all replicas across machine pools
		for _, nodePool := range nodePools {
			if nodePool.Autoscaling() != nil {
				minNodes += nodePool.Autoscaling().MinReplica()
				maxNodes += nodePool.Autoscaling().MaxReplica()
			} else {
				minNodes += nodePool.Replicas()
				maxNodes += nodePool.Replicas()
			}
			if nodePool.Status() != nil {
				currentNodes += nodePool.Status().CurrentReplicas()
			}
		}
		if minNodes != maxNodes {
			nodeConfig = fmt.Sprintf(`
Nodes:
 - Compute (Autoscaled):    %d-%d
 - Compute (current):       %d
`,
				minNodes,
				maxNodes,
				currentNodes,
			)
		} else {
			nodeConfig = fmt.Sprintf(`
Nodes:
 - Compute (desired):       %d
 - Compute (current):       %d
`,
				maxNodes,
				currentNodes,
			)
		}
	} else {
		// Display number of all worker nodes across the cluster
		minNodes := 0
		maxNodes := 0
		// Accumulate all replicas across machine pools
		for _, machinePool := range machinePools {
			if machinePool.Autoscaling() != nil {
				minNodes += machinePool.Autoscaling().MinReplicas()
				maxNodes += machinePool.Autoscaling().MaxReplicas()
			} else {
				minNodes += machinePool.Replicas()
				maxNodes += machinePool.Replicas()
			}
		}

		nodeConfig = fmt.Sprintf(`
Nodes:
 - Control plane:           %d
 - Infra:                   %d
`,
			cluster.Nodes().Master(),
			cluster.Nodes().Infra())

		// Determine whether there is any auto-scaling in the cluster
		if minNodes == maxNodes {
			nodeConfig += fmt.Sprintf(
				" - Compute:                 %d\n",
				minNodes,
			)
		} else {
			nodeConfig += fmt.Sprintf(
				" - Compute (Autoscaled):    %d-%d\n",
				minNodes, maxNodes,
			)
		}
	}
	hasSgsControlPlane := len(cluster.AWS().AdditionalControlPlaneSecurityGroupIds()) > 0
	hasSgsInfra := len(cluster.AWS().AdditionalInfraSecurityGroupIds()) > 0
	if hasSgsControlPlane || hasSgsInfra {
		nodeConfig += " - Additional Security Group IDs:\n"
		if hasSgsControlPlane {
			nodeConfig += fmt.Sprintf(
				"   - Control Plane:	%s\n",
				output.PrintStringSlice(
					cluster.AWS().AdditionalControlPlaneSecurityGroupIds()))
		}
		if hasSgsInfra {
			nodeConfig += fmt.Sprintf(
				"   - Infra:		%s\n",
				output.PrintStringSlice(
					cluster.AWS().AdditionalInfraSecurityGroupIds()))
		}
	}
	return nodeConfig
}

func getDetailsLink(environment string) string {
	switch environment {
	case StageEnv:
		return StageURL
	case ProductionEnv:
		return ProductionURL
	default:
		return ""
	}
}

func getUseworkloadMonitoring(disabled bool) string {
	if disabled {
		return DisabledOutput
	}
	return EnabledOutput
}

func formatCluster(cluster *cmv1.Cluster, scheduledUpgrade *cmv1.UpgradePolicy,
	upgradeState *cmv1.UpgradePolicyState, displayName string,
	migrations map[string]*cmv1.ClusterMigration) (map[string]interface{}, error) {

	var b bytes.Buffer
	err := cmv1.MarshalCluster(cluster, &b)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]interface{})
	err = json.Unmarshal(b.Bytes(), &ret)
	if err != nil {
		return nil, err
	}
	if scheduledUpgrade != nil &&
		upgradeState != nil &&
		len(scheduledUpgrade.Version()) > 0 &&
		len(upgradeState.Value()) > 0 {
		upgrade := make(map[string]interface{})
		upgrade["version"] = scheduledUpgrade.Version()
		upgrade["state"] = upgradeState.Value()
		upgrade["nextRun"] = scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST")
		ret["scheduledUpgrade"] = upgrade
	}
	ret["displayName"] = displayName

	allMigrations := make([]map[string]interface{}, 0)
	for k, v := range migrations {
		migration := make(map[string]interface{})
		migration["type"] = v.Type()
		migration["state"] = v.State().Value()
		migration["id"] = k
		allMigrations = append(allMigrations, migration)
	}

	ret["migrations"] = allMigrations

	return ret, nil
}

func formatClusterHypershift(cluster *cmv1.Cluster,
	scheduledUpgrade *cmv1.ControlPlaneUpgradePolicy,
	displayName string, migrations map[string]*cmv1.ClusterMigration) (map[string]interface{}, error) {

	var b bytes.Buffer
	err := cmv1.MarshalCluster(cluster, &b)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]interface{})
	err = json.Unmarshal(b.Bytes(), &ret)
	if err != nil {
		return nil, err
	}
	if scheduledUpgrade != nil &&
		scheduledUpgrade.State() != nil &&
		len(scheduledUpgrade.Version()) > 0 &&
		len(scheduledUpgrade.State().Value()) > 0 {
		upgrade := make(map[string]interface{})
		upgrade["version"] = scheduledUpgrade.Version()
		upgrade["state"] = scheduledUpgrade.State().Value()
		upgrade["nextRun"] = scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST")
		ret["scheduledUpgrade"] = upgrade
	}
	ret["display_name"] = displayName

	allMigrations := make([]map[string]interface{}, 0)
	for k, v := range migrations {
		migration := make(map[string]interface{})
		migration["type"] = v.Type()
		migration["state"] = v.State().Value()
		migration["id"] = k
		allMigrations = append(allMigrations, migration)
	}

	ret["migrations"] = allMigrations

	return ret, nil
}

func BillingAccount(cluster *cmv1.Cluster) string {
	if cluster.AWS().BillingAccountID() == "" {
		return ""
	}
	return fmt.Sprintf("AWS Billing Account:        %s\n", cluster.AWS().BillingAccountID())
}

func getAuditLogForwardingStatus(cluster *cmv1.Cluster) string {
	auditLogForwardingStatus := DisabledOutput
	if cluster.AWS().AuditLog().RoleArn() != "" {
		auditLogForwardingStatus = EnabledOutput
	}
	return auditLogForwardingStatus
}

func getClusterRegistryConfig(cluster *cmv1.Cluster, allowlist *cmv1.RegistryAllowlist) string {
	var output string
	if cluster.RegistryConfig().RegistrySources() != nil {
		registryResources := cluster.RegistryConfig().RegistrySources()
		if registryResources.AllowedRegistries() != nil {
			output = fmt.Sprintf("%s"+
				" - Allowed Registries:      %s\n", output,
				strings.Join(registryResources.AllowedRegistries(), ","))
		}
		if registryResources.BlockedRegistries() != nil {
			output = fmt.Sprintf("%s"+
				" - Blocked Registries:      %s\n", output,
				strings.Join(registryResources.BlockedRegistries(), ","))

		}
		if registryResources.InsecureRegistries() != nil {
			output = fmt.Sprintf("%s"+
				" - Insecure Registries:     %s\n", output,
				strings.Join(registryResources.InsecureRegistries(), ","))
		}
	}
	if cluster.RegistryConfig().AllowedRegistriesForImport() != nil {
		output = fmt.Sprintf("%s"+
			" - Allowed Registries for Import:         \n", output,
		)
		allowedRegistriesForImport := cluster.RegistryConfig().AllowedRegistriesForImport()
		for _, registry := range allowedRegistriesForImport {
			output = fmt.Sprintf("%s"+
				"    - Domain Name:          %s\n    - Insecure:             %s\n", output,
				registry.DomainName(), strconv.FormatBool(registry.Insecure()))
		}
	}
	if cluster.RegistryConfig().PlatformAllowlist().ID() != "" && allowlist != nil {
		output = fmt.Sprintf("%s"+
			" - Platform Allowlist:      %s\n"+
			"    - Registries:           %s\n", output,
			cluster.RegistryConfig().PlatformAllowlist().ID(), strings.Join(allowlist.Registries(), ","))
	}
	if cluster.RegistryConfig().AdditionalTrustedCa() != nil {
		additionalTrustedCa := cluster.RegistryConfig().AdditionalTrustedCa()
		output = fmt.Sprintf("%s"+
			" - Additional Trusted CA:         \n", output,
		)
		keys := make([]string, 0, len(additionalTrustedCa))
		for k := range additionalTrustedCa {
			keys = append(keys, k)
		}

		sort.Strings(keys)
		for _, registry := range keys {
			output = fmt.Sprintf("%s"+
				"    - %s: REDACTED\n", output,
				registry)
		}
	}
	return output
}

func getExternalAuthConfigStatus(cluster *cmv1.Cluster) string {
	externalAuthConfigStatus := DisabledOutput
	if cluster.ExternalAuthConfig().Enabled() {
		externalAuthConfigStatus = EnabledOutput
	}
	return externalAuthConfigStatus
}

func getEtcdStatus(cluster *cmv1.Cluster) string {
	etcdStatus := DisabledOutput
	if cluster.EtcdEncryption() {
		etcdStatus = EnabledOutput
	}
	return etcdStatus
}

func getRolePolicyBindings(roleARN string, rolePolicyDetails map[string][]aws.PolicyDetail,
	prefix string) (string, error) {
	roleName, err := aws.GetResourceIdFromARN(roleARN)
	if err != nil {
		return "", fmt.Errorf("Failed to get role name from arn %s: %v", roleARN, err)
	}
	str := ""
	if rolePolicyDetails[roleName] != nil {
		for _, policy := range rolePolicyDetails[roleName] {
			str = fmt.Sprintf("%s"+
				"%s %s\n", str, prefix, policy.PolicyArn)
		}
	}
	return str, nil
}

func getZeroEgressStatus(r *rosa.Runtime, cluster *cmv1.Cluster) (string, error) {
	techPreviewMsg, err := r.OCMClient.GetTechnologyPreviewMessage("hcp-zero-egress", time.Now())
	if err != nil {
		return "", fmt.Errorf("Failed to get technology preview message for zero egress: %v", err)
	}
	if techPreviewMsg != "" {
		zeroEgressOutput := DisabledOutput
		zeroEgressPropVal, ok := cluster.Properties()["zero_egress"]
		if ok && zeroEgressPropVal == "true" {
			zeroEgressOutput = EnabledOutput
		}
		str := fmt.Sprintf("%s:                %s\n", techPreviewMsg, zeroEgressOutput)
		return str, nil
	}
	return "", nil
}
