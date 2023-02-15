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
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	StageURL      = "https://qaprodauth.console.redhat.com/openshift/details/s/"
	ProductionURL = "https://console.redhat.com/openshift/details/s/"
	StageEnv      = "https://api.stage.openshift.com"
	ProductionEnv = "https://api.openshift.com"
)

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Show details of a cluster",
	Long:  "Show details of a cluster",
	Example: `  # Describe a cluster named "mycluster"
  rosa describe cluster --cluster=mycluster`,
	Run: run,
}

func init() {
	output.AddFlag(Cmd)
	ocm.AddClusterFlag(Cmd)
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

	scheduledUpgrade, upgradeState, err := r.OCMClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		f, err := formatCluster(cluster, scheduledUpgrade, upgradeState)
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

	var str string
	creatorARN, err := arn.Parse(cluster.Properties()[properties.CreatorARN])
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

	clusterDNS := "Not ready"
	if cluster.Status() != nil && cluster.Status().DNSReady() {
		clusterDNS = strings.Join([]string{cluster.Name(), cluster.DNS().BaseDomain()}, ".")
	}

	clusterName := cluster.Name()

	isPrivate := "No"
	if cluster.API().Listening() == cmv1.ListeningMethodInternal {
		isPrivate = "Yes"
	}

	detailsPage := getDetailsLink(r.OCMClient.GetConnectionURL())

	networkType := ""
	if cluster.Network().Type() != ocm.NetworkTypes[0] {
		networkType = fmt.Sprintf(
			" - Type:                    %s\n",
			cluster.Network().Type(),
		)
	}

	// Print short cluster description:
	str = fmt.Sprintf("\n"+
		"Name:                       %s\n"+
		"ID:                         %s\n"+
		"External ID:                %s\n"+
		"Control Plane:              %s\n"+
		"OpenShift Version:          %s\n"+
		"Channel Group:              %s\n"+
		"DNS:                        %s\n"+
		"AWS Account:                %s\n"+
		"API URL:                    %s\n"+
		"Console URL:                %s\n"+
		"Region:                     %s\n"+
		"Multi-AZ:                   %t\n"+
		"%s"+
		"Network:\n"+
		"%s"+
		" - Service CIDR:            %s\n"+
		" - Machine CIDR:            %s\n"+
		" - Pod CIDR:                %s\n"+
		" - Host Prefix:             /%d\n",
		clusterName,
		cluster.ID(),
		cluster.ExternalID(),
		controlPlaneConfig(cluster),
		cluster.OpenshiftVersion(),
		cluster.Version().ChannelGroup(),
		clusterDNS,
		creatorARN.AccountID,
		cluster.API().URL(),
		cluster.Console().URL(),
		cluster.Region().ID(),
		cluster.MultiAZ(),
		clusterInfraConfig(cluster, clusterKey, r),
		networkType,
		cluster.Network().ServiceCIDR(),
		cluster.Network().MachineCIDR(),
		cluster.Network().PodCIDR(),
		cluster.Network().HostPrefix(),
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

	if cluster.AWS().STS().RoleARN() != "" {
		str = fmt.Sprintf("%s"+
			"STS Role ARN:               %s\n", str,
			cluster.AWS().STS().RoleARN())
		if cluster.AWS().STS().ExternalID() != "" {
			str = fmt.Sprintf("%s"+
				"STS External ID:            %s\n", str,
				cluster.AWS().STS().ExternalID())
		}
		if cluster.AWS().STS().SupportRoleARN() != "" {
			str = fmt.Sprintf("%s"+
				"Support Role ARN:           %s\n", str,
				cluster.AWS().STS().SupportRoleARN())
		}
		if cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN() != "" ||
			cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN() != "" {
			str = fmt.Sprintf("%sInstance IAM Roles:\n", str)
			if cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN() != "" {
				str = fmt.Sprintf("%s"+
					" - Control plane:           %s\n", str,
					cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN())
			}
			if cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN() != "" {
				str = fmt.Sprintf("%s"+
					" - Worker:                  %s\n", str,
					cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN())
			}
		}
		if len(cluster.AWS().STS().OperatorIAMRoles()) > 0 {
			str = fmt.Sprintf("%sOperator IAM Roles:\n", str)
			for _, operatorIAMRole := range cluster.AWS().STS().OperatorIAMRoles() {
				str = fmt.Sprintf("%s"+
					" - %s\n", str,
					operatorIAMRole.RoleARN())
			}
		}
	}

	str = fmt.Sprintf("%s"+
		"State:                      %s %s\n"+
		"Private:                    %s\n"+
		"Created:                    %s\n", str,
		cluster.State(), phase,
		isPrivate,
		cluster.CreationTimestamp().Format("Jan _2 2006 15:04:05 MST"))

	if cluster.DisableUserWorkloadMonitoring() {
		str = fmt.Sprintf("%s"+
			"User Workload Monitoring:   %s\n",
			str,
			getUseworkloadMonitoring(cluster.DisableUserWorkloadMonitoring()))
	}
	if cluster.FIPS() {
		str = fmt.Sprintf("%s"+
			"FIPS mode:                  %s\n",
			str,
			"enabled")
	}
	if detailsPage != "" {
		str = fmt.Sprintf("%s"+
			"Details Page:               %s%s\n", str,
			detailsPage, cluster.Subscription().ID())
	}
	if cluster.AWS().STS().OIDCEndpointURL() != "" {
		str = fmt.Sprintf("%s"+
			"OIDC Endpoint URL:          %s\n", str,
			cluster.AWS().STS().OIDCEndpointURL())
	}
	if scheduledUpgrade != nil {
		str = fmt.Sprintf("%s"+
			"Scheduled Upgrade:          %s %s on %s\n",
			str,
			upgradeState.Value(),
			scheduledUpgrade.Version(),
			scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"),
		)
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
	str = fmt.Sprintf("%s\n", str)

	// Print short cluster description:
	fmt.Print(str)
}

func controlPlaneConfig(cluster *cmv1.Cluster) string {
	if cluster.Hypershift().Enabled() {
		return "Red Hat hosted"
	}
	return "Customer hosted"
}

func clusterInfraConfig(cluster *cmv1.Cluster, clusterKey string, r *rosa.Runtime) string {
	var infraConfig string
	if cluster.Hypershift().Enabled() {
		minNodes := 0
		maxNodes := 0
		currentNodes := 0
		// Reusing design as classic machinePools, in the future those both APIs will converge
		nodePools, err := r.OCMClient.GetNodePools(cluster.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
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
			infraConfig = fmt.Sprintf(""+
				"Nodes:\n"+
				" - Compute (Autoscaled):        %d-%d\n"+
				" - Compute (current):           %d\n",
				minNodes,
				maxNodes,
				currentNodes,
			)
		} else {
			infraConfig = fmt.Sprintf(""+
				"Nodes:\n"+
				" - Compute (desired):           %d\n"+
				" - Compute (current):           %d\n",
				maxNodes,
				currentNodes,
			)
		}
	} else {
		// Display number of all worker nodes across the cluster
		minNodes := 0
		maxNodes := 0
		machinePools, err := r.OCMClient.GetMachinePools(cluster.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to get machine pools for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
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
		// Add compute nodes as well
		if cluster.Nodes().AutoscaleCompute() != nil {
			minNodes += cluster.Nodes().AutoscaleCompute().MinReplicas()
			maxNodes += cluster.Nodes().AutoscaleCompute().MaxReplicas()
		} else {
			minNodes += cluster.Nodes().Compute()
			maxNodes += cluster.Nodes().Compute()
		}
		// Determine whether there is any auto-scaling in the cluster
		if minNodes == maxNodes {
			infraConfig = fmt.Sprintf(""+
				"Nodes:\n"+
				" - Control plane:           %d\n"+
				" - Infra:                   %d\n"+
				" - Compute:                 %d\n",
				cluster.Nodes().Master(),
				cluster.Nodes().Infra(),
				minNodes,
			)
		} else {
			infraConfig = fmt.Sprintf(""+
				"Nodes:\n"+
				" - Control plane:           %d\n"+
				" - Infra:                   %d\n"+
				" - Compute (Autoscaled):    %d-%d\n",
				cluster.Nodes().Master(),
				cluster.Nodes().Infra(),
				minNodes, maxNodes,
			)
		}
	}
	return infraConfig
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
		return "disabled"
	}
	return "enabled"
}

func formatCluster(cluster *cmv1.Cluster, scheduledUpgrade *cmv1.UpgradePolicy,
	upgradeState *cmv1.UpgradePolicyState) (map[string]interface{}, error) {

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

	return ret, nil
}
