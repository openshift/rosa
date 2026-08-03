package cluster

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/hyperfleet-operator/api/v1alpha1"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled        = hyperfleet.Enabled
	hfDescribeCluster = func(cmd *cobra.Command, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetDescribe(r, cmd, argv)
	}
)

func runHyperfleetDescribe(r *rosa.Runtime, cmd *cobra.Command, argv []string) {
	ctx := context.Background()

	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		os.Exit(1)
	}

	clusters := r.HyperFleetClient.HyperfleetV1alpha1().Clusters(r.Creator.AccountID)

	list, err := clusters.List(ctx, wrappers.ListOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to list clusters: %v", err)
		os.Exit(1)
	}

	var clusterID string
	for _, c := range list.Items {
		if c.Name == clusterKey {
			clusterID = string(c.UID)
			break
		}
	}
	if clusterID == "" {
		r.Reporter.Errorf("Cluster '%s' not found", clusterKey)
		os.Exit(1)
	}

	cluster, err := clusters.Get(ctx, clusterID, wrappers.GetOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if output.HasFlag() {
		m := hfClusterToMap(cluster)
		if err := output.Print(m); err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		return
	}

	fmt.Print(hfClusterToString(cluster))
}

// hfClusterToMap converts a hyperfleet Cluster to a generic map suitable for
// JSON/YAML structured output, mirroring the shape of formatClusterHypershift.
func hfClusterToMap(c *v1alpha1.Cluster) map[string]interface{} {
	aws := c.Spec.HostedCluster.Platform.AWS

	rolesRef := map[string]string{}
	if aws != nil {
		ref := aws.RolesRef
		rolesRef["ingressARN"] = ref.IngressARN
		rolesRef["imageRegistryARN"] = ref.ImageRegistryARN
		rolesRef["storageARN"] = ref.StorageARN
		rolesRef["networkARN"] = ref.NetworkARN
		rolesRef["kubeCloudControllerARN"] = ref.KubeCloudControllerARN
		rolesRef["controlPlaneOperatorARN"] = ref.ControlPlaneOperatorARN
		rolesRef["nodePoolManagementARN"] = ref.NodePoolManagementARN
	}

	m := map[string]interface{}{
		"id":            string(c.UID),
		"name":          c.Name,
		"control_plane": "ROSA Service Hosted",
		"state":         string(c.Status.Phase),
		"created_at":    c.CreationTimestamp.UTC().Format(time.RFC3339),
		"spec": map[string]interface{}{
			"creator_arn":   c.Spec.CreatorARN,
			"oidc_issuer":   c.Spec.HostedCluster.IssuerURL,
			"roles_ref":     rolesRef,
		},
	}

	if aws != nil {
		m["region"] = aws.Region
		if aws.CloudProviderConfig != nil {
			m["vpc"] = aws.CloudProviderConfig.VPC
			if aws.CloudProviderConfig.Subnet != nil && aws.CloudProviderConfig.Subnet.ID != nil {
				m["subnet"] = *aws.CloudProviderConfig.Subnet.ID
			}
		}
	}

	if c.Status.Version != "" {
		m["version"] = c.Status.Version
	}
	if c.Status.ControlPlaneEndpoint.Host != "" {
		m["api_url"] = fmt.Sprintf("https://%s:%d",
			c.Status.ControlPlaneEndpoint.Host,
			c.Status.ControlPlaneEndpoint.Port)
	}
	if c.Status.PlacementRef != nil {
		m["management_cluster"] = c.Status.PlacementRef.ManagementCluster
	}
	if c.Spec.ExpirationTimestamp != nil {
		m["expiration"] = c.Spec.ExpirationTimestamp.UTC().Format(time.RFC3339)
	}

	conditions := make([]map[string]interface{}, 0, len(c.Status.Conditions))
	for _, cond := range c.Status.Conditions {
		conditions = append(conditions, map[string]interface{}{
			"type":    cond.Type,
			"status":  string(cond.Status),
			"reason":  cond.Reason,
			"message": cond.Message,
		})
	}
	m["conditions"] = conditions

	return m
}

// hfClusterToString formats a hyperfleet Cluster as a human-readable string,
// following the same label-alignment style as rosa describe cluster.
func hfClusterToString(c *v1alpha1.Cluster) string {
	aws := c.Spec.HostedCluster.Platform.AWS

	region := ""
	vpc := ""
	subnet := ""
	if aws != nil {
		region = aws.Region
		if aws.CloudProviderConfig != nil {
			vpc = aws.CloudProviderConfig.VPC
			if aws.CloudProviderConfig.Subnet != nil && aws.CloudProviderConfig.Subnet.ID != nil {
				subnet = *aws.CloudProviderConfig.Subnet.ID
			}
		}
	}

	apiURL := ""
	if c.Status.ControlPlaneEndpoint.Host != "" {
		apiURL = fmt.Sprintf("https://%s:%d",
			c.Status.ControlPlaneEndpoint.Host,
			c.Status.ControlPlaneEndpoint.Port)
	}

	s := fmt.Sprintf("\n"+
		"Name:                       %s\n"+
		"ID:                         %s\n"+
		"Control Plane:              %s\n"+
		"OpenShift Version:          %s\n"+
		"API URL:                    %s\n"+
		"Region:                     %s\n"+
		"VPC:                        %s\n"+
		"Subnet:                     %s\n"+
		"OIDC Endpoint URL:          %s\n"+
		"State:                      %s\n"+
		"Created:                    %s\n",
		c.Name,
		string(c.UID),
		"ROSA Service Hosted",
		c.Status.Version,
		apiURL,
		region,
		vpc,
		subnet,
		c.Spec.HostedCluster.IssuerURL,
		string(c.Status.Phase),
		c.CreationTimestamp.UTC().Format("2006-01-02 15:04:05 UTC"),
	)

	if c.Status.PlacementRef != nil {
		s += fmt.Sprintf("Management Cluster:         %s\n", c.Status.PlacementRef.ManagementCluster)
	}

	if c.Spec.ExpirationTimestamp != nil {
		s += fmt.Sprintf("Expiration:                 %s\n",
			c.Spec.ExpirationTimestamp.UTC().Format("2006-01-02 15:04:05 UTC"))
	}

	if aws != nil {
		ref := aws.RolesRef
		s += "Operator IAM Roles:\n" +
			fmt.Sprintf(" - Ingress:                 %s\n", ref.IngressARN) +
			fmt.Sprintf(" - Image Registry:          %s\n", ref.ImageRegistryARN) +
			fmt.Sprintf(" - EBS CSI:                 %s\n", ref.StorageARN) +
			fmt.Sprintf(" - Network Config:          %s\n", ref.NetworkARN) +
			fmt.Sprintf(" - Cloud Controller:        %s\n", ref.KubeCloudControllerARN) +
			fmt.Sprintf(" - Control Plane Operator:  %s\n", ref.ControlPlaneOperatorARN) +
			fmt.Sprintf(" - Node Pool Management:    %s\n", ref.NodePoolManagementARN)
	}

	if len(c.Status.Conditions) > 0 {
		s += "Conditions:\n"
		for _, cond := range c.Status.Conditions {
			s += fmt.Sprintf(" - %-10s %-5s  %s\n",
				cond.Type+":", string(cond.Status),
				conditionSummary(cond.Reason, cond.Message))
		}
	}

	return s
}

// conditionSummary returns a concise reason+message string for a condition row.
func conditionSummary(reason, message string) string {
	if reason == "" {
		return message
	}
	if message == "" || strings.HasPrefix(message, reason) {
		return reason
	}
	return reason + ": " + message
}
