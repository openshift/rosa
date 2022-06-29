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
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	StageURL      = "https://qaprodauth.cloud.redhat.com/openshift/details/s/"
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

	var clusterKey string
	var err error

	// Allow the command to be called programmatically
	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		clusterKey = argv[0]
	} else {
		clusterKey = r.GetClusterKey()
	}

	cluster := r.FetchCluster()
	var str string
	if output.HasFlag() {
		err = output.Print(cluster)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		return
	}

	creatorARN, err := arn.Parse(cluster.Properties()[properties.CreatorARN])
	if err != nil {
		r.Reporter.Errorf("Failed to parse creator ARN for cluster '%s'", clusterKey)
		os.Exit(1)
	}
	phase := ""

	switch cluster.State() {
	case cmv1.ClusterStateWaiting:
		phase = "(Waiting for user action)"
		status, _ := r.OCMClient.GetClusterStatus(cluster.ID())
		if status.Description() != "" {
			phase = fmt.Sprintf("(%s)", status.Description())
		}
	case cmv1.ClusterStatePending:
		phase = "(Preparing account)"
		status, _ := r.OCMClient.GetClusterStatus(cluster.ID())
		if status.Description() != "" {
			phase = fmt.Sprintf("(%s)", status.Description())
		}
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

	clusterName := cluster.Name()
	if clusterName == "" {
		clusterName = cluster.DisplayName()
	}

	isPrivate := "No"
	if cluster.API().Listening() == cmv1.ListeningMethodInternal {
		isPrivate = "Yes"
	}

	scheduledUpgrade, upgradeState, err := r.OCMClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	detailsPage := getDetailsLink(r.OCMClient.GetConnectionURL())

	// Display number of all worker nodes across the cluster
	minNodes := 0
	maxNodes := 0
	var nodesStr string
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
		nodesStr = fmt.Sprintf(""+
			"Nodes:\n"+
			" - Control plane:           %d\n"+
			" - Infra:                   %d\n"+
			" - Compute:                 %d\n",
			cluster.Nodes().Master(),
			cluster.Nodes().Infra(),
			minNodes,
		)
	} else {
		nodesStr = fmt.Sprintf(""+
			"Nodes:\n"+
			" - Control plane:           %d\n"+
			" - Infra:                   %d\n"+
			" - Compute (Autoscaled):    %d-%d\n",
			cluster.Nodes().Master(),
			cluster.Nodes().Infra(),
			minNodes, maxNodes,
		)
	}

	networkType := ""
	if cluster.Network().Type() != ocm.NetworkTypes[0] {
		networkType = fmt.Sprintf(
			" - Type:                    %s\n",
			cluster.Network().Type(),
		)
	}

	// Print short cluster description:
	str = fmt.Sprintf(""+
		"Name:                       %s\n"+
		"ID:                         %s\n"+
		"External ID:                %s\n"+
		"OpenShift Version:          %s\n"+
		"Channel Group:              %s\n"+
		"DNS:                        %s.%s\n"+
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
		cluster.OpenshiftVersion(),
		cluster.Version().ChannelGroup(),
		cluster.Name(), cluster.DNS().BaseDomain(),
		creatorARN.AccountID,
		cluster.API().URL(),
		cluster.Console().URL(),
		cluster.Region().ID(),
		cluster.MultiAZ(),

		nodesStr,
		networkType,
		cluster.Network().ServiceCIDR(),
		cluster.Network().MachineCIDR(),
		cluster.Network().PodCIDR(),
		cluster.Network().HostPrefix(),
	)

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
	// Print short cluster description:
	fmt.Print(str)
	fmt.Println()
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
