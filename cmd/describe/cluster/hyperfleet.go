package cluster

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled         = hyperfleet.Enabled
	exitFn            = func(code int) { os.Exit(code) }
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
		exitFn(1)
	}

	clusterID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	clusters := r.HyperFleetClient.HyperfleetV1alpha1().Clusters()
	cluster, err := clusters.Get(ctx, clusterID, platform.GetOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		exitFn(1)
	}

	if output.HasFlag() {
		instanceType := ""
		list, err := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterID).List(ctx, platform.ListOptions{})
		if err == nil {
			instanceType = hfDefaultNodePoolInstanceTypeFromList(list)
		}
		m := hfClusterToMap(cluster, instanceType)
		if err := output.Print(m); err != nil {
			r.Reporter.Errorf("%s", err)
			exitFn(1)
		}
		return
	}

	fmt.Print(hfClusterToString(cluster))
}

func hfDefaultNodePoolInstanceTypeFromList(list *v1alpha1.NodePoolList) string {
	for i := range list.Items {
		np := &list.Items[i]
		if np.Spec.NodePool.Platform.AWS == nil {
			continue
		}
		if np.Name == "workers" || strings.HasPrefix(np.Name, "workers-") {
			return np.Spec.NodePool.Platform.AWS.InstanceType
		}
	}
	return ""
}

// hfClusterToMap converts a hyperfleet Cluster to a generic map suitable for
// JSON/YAML structured output, mirroring the shape of formatClusterHypershift.
func hfClusterToMap(c *v1alpha1.Cluster, defaultNodePoolInstanceType string) map[string]interface{} {
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

	apiURL := hfAPIURL(c)
	apiListening := hfAPIListening(aws)

	m := map[string]interface{}{
		"id":            string(c.UID),
		"name":          c.Name,
		"control_plane": "ROSA Service Hosted",
		"state":         string(c.Status.Phase),
		"created_at":    c.CreationTimestamp.UTC().Format(time.RFC3339),
		"hypershift": map[string]interface{}{
			"enabled": true,
		},
		"spec": map[string]interface{}{
			"oidc_issuer": c.Spec.HostedCluster.IssuerURL,
			"roles_ref":   rolesRef,
		},
		"private": apiListening == "internal",
	}
	if apiURL != "" || apiListening != "" {
		m["api"] = map[string]interface{}{
			"url":       apiURL,
			"listening": apiListening,
		}
	}

	if aws != nil {
		m["region"] = aws.Region
		if aws.CloudProviderConfig != nil {
			m["vpc"] = aws.CloudProviderConfig.VPC
			if aws.CloudProviderConfig.Subnet != nil && aws.CloudProviderConfig.Subnet.ID != nil {
				m["subnet"] = *aws.CloudProviderConfig.Subnet.ID
			}
		}
		stsMap := map[string]interface{}{
			"enabled":           true,
			"oidc_endpoint_url": c.Spec.HostedCluster.IssuerURL,
		}
		if c.Spec.OidcConfigID != "" {
			stsMap["oidc_config"] = map[string]interface{}{
				"id":         c.Spec.OidcConfigID,
				"issuer_url": c.Spec.HostedCluster.IssuerURL,
				"reusable":   false,
				"managed":    true,
			}
		}
		awsMap := map[string]interface{}{"sts": stsMap}
		if c.Spec.Properties != nil {
			if tokens := c.Spec.Properties["ec2_metadata_http_tokens"]; tokens != "" {
				awsMap["ec2_metadata_http_tokens"] = tokens
			}
		}
		m["aws"] = awsMap
	}

	if c.Status.Version != "" {
		channel := ""
		if c.Spec.Properties != nil {
			channel = c.Spec.Properties["channel_group"]
		}
		m["version"] = map[string]interface{}{
			"raw_id":        c.Status.Version,
			"channel_group": channel,
		}
	}
	if apiURL != "" {
		m["api_url"] = apiURL
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

	if defaultNodePoolInstanceType != "" {
		m["nodes"] = map[string]interface{}{
			"compute_machine_type": map[string]interface{}{
				"id": defaultNodePoolInstanceType,
			},
		}
	}

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

	apiURL := hfAPIURL(c)
	oidcLine := c.Spec.HostedCluster.IssuerURL
	if c.Spec.OidcConfigID != "" && oidcLine != "" {
		oidcLine += " (Managed)"
	}
	fips := "Disabled"
	if c.Spec.HostedCluster.FIPS {
		fips = "Enabled"
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
		"Private:                    %s\n"+
		"FIPS mode:                  %s\n",
		c.Name,
		string(c.UID),
		"ROSA Service Hosted",
		c.Status.Version,
		apiURL,
		region,
		vpc,
		subnet,
		oidcLine,
		string(c.Status.Phase),
		output.PrintBool(hfAPIListening(aws) == "internal"),
		fips,
	)
	if c.Spec.Properties != nil {
		if tokens := c.Spec.Properties["ec2_metadata_http_tokens"]; tokens != "" {
			s += fmt.Sprintf("EC2 Metadata Http Tokens:   %s\n", tokens)
		}
	}
	s += fmt.Sprintf("Created:                    %s\n",
		c.CreationTimestamp.UTC().Format("2006-01-02 15:04:05 UTC"))

	if c.Status.PlacementRef != nil {
		s += fmt.Sprintf("Management Cluster:         %s\n", c.Status.PlacementRef.ManagementCluster)
	}

	if c.Spec.ExpirationTimestamp != nil {
		s += fmt.Sprintf("Expiration:                 %s\n",
			c.Spec.ExpirationTimestamp.UTC().Format("2006-01-02 15:04:05 UTC"))
	}

	if aws != nil {
		var roles []string
		for _, arn := range []string{
			aws.RolesRef.IngressARN,
			aws.RolesRef.ImageRegistryARN,
			aws.RolesRef.StorageARN,
			aws.RolesRef.NetworkARN,
			aws.RolesRef.KubeCloudControllerARN,
			aws.RolesRef.ControlPlaneOperatorARN,
			aws.RolesRef.NodePoolManagementARN,
		} {
			if arn != "" {
				roles = append(roles, arn)
			}
		}
		if len(roles) > 0 {
			s += "Operator IAM Roles:\n"
			for _, arn := range roles {
				s += fmt.Sprintf(" - %s\n", arn)
			}
		}
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

func hfAPIURL(c *v1alpha1.Cluster) string {
	if c.Status.ControlPlaneEndpoint.Host == "" {
		return ""
	}
	return fmt.Sprintf("https://%s:%d",
		c.Status.ControlPlaneEndpoint.Host,
		c.Status.ControlPlaneEndpoint.Port)
}

func hfAPIListening(aws *hypershiftv1beta1.AWSPlatformSpec) string {
	if aws != nil && aws.EndpointAccess == hypershiftv1beta1.Private {
		return "internal"
	}
	return "external"
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
