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

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	exitFn            = func(code int) { os.Exit(code) }
	hfDescribeCluster = func(cmd *cobra.Command, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet().WithAWSOnly()
		defer r.Cleanup()
		runHyperfleetDescribe(r, cmd, argv)
	}
)

// deriveNodePoolAvailabilityZones queries AWS to determine availability zones from nodepool subnet IDs.
// NOTE: V1 (OCM) stores AZ directly on nodepool.AvailabilityZone().
// V2 (Hyperfleet) stores only subnet ID, requiring AWS lookup.
// TODO: Consider adding AZ field to Hyperfleet NodePool status to avoid AWS API calls during describe.
func deriveNodePoolAvailabilityZones(ctx context.Context, r *rosa.Runtime, cluster *v1alpha1.Cluster, npList *v1alpha1.NodePoolList) map[string]struct{} {
	azMap := make(map[string]struct{})
	if npList == nil || len(npList.Items) == 0 {
		return azMap
	}

	// Get cluster region
	if cluster.Spec.HostedCluster.Platform.AWS == nil || cluster.Spec.HostedCluster.Platform.AWS.Region == "" {
		r.Reporter.Warnf("Cluster region not available, cannot determine data plane availability")
		return azMap
	}
	clusterRegion := cluster.Spec.HostedCluster.Platform.AWS.Region

	// Create a region-specific AWS client if the cluster region differs from the current client region
	// This ensures subnet queries work correctly
	awsClient := r.AWSClient
	if r.AWSClient != nil && r.AWSClient.GetRegion() != clusterRegion {
		// Create a new AWS client for the cluster's region
		regionClient, err := aws.NewClient().
			Logger(r.Logger).
			Region(clusterRegion).
			Build()
		if err != nil {
			r.Reporter.Warnf("Failed to create AWS client for region %s: %v", clusterRegion, err)
			return azMap
		}
		awsClient = regionClient
	}

	// Query AWS to get availability zone for each nodepool's subnet
	for _, np := range npList.Items {
		if np.Spec.NodePool.Platform.AWS != nil &&
			np.Spec.NodePool.Platform.AWS.Subnet.ID != nil &&
			*np.Spec.NodePool.Platform.AWS.Subnet.ID != "" {

			subnetID := *np.Spec.NodePool.Platform.AWS.Subnet.ID
			az, err := awsClient.GetSubnetAvailabilityZone(subnetID)
			if err != nil {
				r.Reporter.Warnf("Failed to get availability zone for subnet %s: %v", subnetID, err)
				continue
			}
			if az != "" {
				azMap[az] = struct{}{}
			}
		}
	}

	return azMap
}

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

	// Get nodepools to determine data plane availability
	nodePools := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(string(cluster.UID))
	npList, err := nodePools.List(ctx, platform.ListOptions{})
	if err != nil {
		r.Reporter.Warnf("Failed to list nodepools: %v", err)
		npList = nil
	}

	// Derive availability zones from subnet IDs
	// NOTE: V1 (OCM) stores AZ directly on nodepool; V2 (Hyperfleet) requires AWS subnet lookup
	// TODO: Consider adding AZ field to Hyperfleet NodePool status to avoid AWS API calls
	dataPlaneAZs := deriveNodePoolAvailabilityZones(ctx, r, cluster, npList)

	if output.HasFlag() {
		instanceType := ""
		if npList != nil {
			instanceType = hfDefaultNodePoolInstanceTypeFromList(npList)
		}
		m := hfClusterToMap(cluster, dataPlaneAZs, npList, instanceType)
		if err := output.Print(m); err != nil {
			r.Reporter.Errorf("%s", err)
			exitFn(1)
		}
		return
	}

	fmt.Print(hfClusterToString(cluster, dataPlaneAZs, npList))
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
func hfClusterToMap(c *v1alpha1.Cluster, dataPlaneAZs map[string]struct{}, npList *v1alpha1.NodePoolList, defaultNodePoolInstanceType string) map[string]interface{} {
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

	networking := map[string]interface{}{}
	net := c.Spec.HostedCluster.Networking
	if net.NetworkType != "" {
		networking["network_type"] = net.NetworkType
	}
	if len(net.ClusterNetwork) > 0 {
		networking["cluster_network"] = net.ClusterNetwork
	}
	if len(net.ServiceNetwork) > 0 {
		networking["service_network"] = net.ServiceNetwork
	}
	if len(net.MachineNetwork) > 0 {
		networking["machine_network"] = net.MachineNetwork
	}
	if net.APIServer != nil {
		apiServer := map[string]interface{}{}
		if net.APIServer.AdvertiseAddress != nil && *net.APIServer.AdvertiseAddress != "" {
			apiServer["advertise_address"] = *net.APIServer.AdvertiseAddress
		}
		if net.APIServer.Port != nil && *net.APIServer.Port != 0 {
			apiServer["port"] = *net.APIServer.Port
		}
		if len(net.APIServer.AllowedCIDRBlocks) > 0 {
			apiServer["allowed_cidr_blocks"] = net.APIServer.AllowedCIDRBlocks
		}
		if len(apiServer) > 0 {
			networking["api_server"] = apiServer
		}
	}

	// Format DNS as a single string to match OCM-based cluster describe output
	// The test expects DNS to be a string, not a nested object
	// Use Status.BaseDomain which is populated by the operator
	dnsString := c.Status.BaseDomain

	m := map[string]interface{}{
		"id":            string(c.UID),
		"name":          c.Name,
		"control_plane": "ROSA Service Hosted",
		"state":         strings.ToLower(string(c.Status.Phase)), // Normalize to lowercase to match V1 (OCM)
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

	if dnsString != "" {
		m["dns"] = dnsString
	}
	if len(networking) > 0 {
		m["networking"] = networking
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
func hfClusterToString(c *v1alpha1.Cluster, dataPlaneAZs map[string]struct{}, npList *v1alpha1.NodePoolList) string {
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

	// Format DNS string
	dnsStr := c.Status.BaseDomain
	if dnsStr == "" {
		dnsStr = "Not ready"
	}

	s := fmt.Sprintf("\n"+
		"Name:                       %s\n"+
		"ID:                         %s\n"+
		"Control Plane:              %s\n"+
		"OpenShift Version:          %s\n"+
		"DNS:                        %s\n"+
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
		dnsStr,
		apiURL,
		region,
		vpc,
		subnet,
		oidcLine,
		strings.ToLower(string(c.Status.Phase)), // Normalize to lowercase to match V1 (OCM)
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

	// Availability - HCP control plane is always MultiAZ
	// Data plane depends on nodepool subnet availability zones
	dataPlaneAvailability := "SingleAZ"
	if len(dataPlaneAZs) > 1 {
		dataPlaneAvailability = "MultiAZ"
	} else if len(dataPlaneAZs) == 0 {
		// No AZ information available (e.g., AWS query failed or no nodepools)
		dataPlaneAvailability = "Unknown"
	}
	s += fmt.Sprintf("Availability:\n"+
		" - Control Plane:           MultiAZ\n"+
		" - Data Plane:              %s\n",
		dataPlaneAvailability)

	// Nodes section - aggregate compute node information from nodepools
	// NOTE: V2 API doesn't yet expose:
	//   - Autoscaling configuration (min/max replicas) in NodePoolSpecPassthrough
	//   - Current replicas in NodePoolStatus
	// V1 (OCM) shows both desired and current replicas, plus autoscaling range
	// TODO: Update when V2 API exposes status.replicas and autoscaling fields to match V1
	if npList != nil && len(npList.Items) > 0 {
		desiredNodes := int32(0)
		for _, np := range npList.Items {
			if np.Spec.NodePool.Replicas != nil {
				desiredNodes += *np.Spec.NodePool.Replicas
			}
		}
		// Show desired only until V2 API exposes current replicas in status
		s += fmt.Sprintf("Nodes:\n"+
			" - Compute (desired):       %d\n",
			desiredNodes)
	}

	// Network section - format to match V1 (OCM) output for test compatibility
	// The test expects: Type, Service CIDR, Machine CIDR, Pod CIDR, Host Prefix, Subnets
	net := c.Spec.HostedCluster.Networking

	// Extract network values with defaults
	networkType := net.NetworkType
	serviceCIDR := ""
	if len(net.ServiceNetwork) > 0 {
		serviceCIDR = net.ServiceNetwork[0].CIDR.String()
	}
	machineCIDR := ""
	if len(net.MachineNetwork) > 0 {
		machineCIDR = net.MachineNetwork[0].CIDR.String()
	}
	podCIDR := ""
	hostPrefix := int32(0)
	if len(net.ClusterNetwork) > 0 {
		podCIDR = net.ClusterNetwork[0].CIDR.String()
		hostPrefix = net.ClusterNetwork[0].HostPrefix
	}

	// Get subnet from cluster AWS config
	subnetID := ""
	if aws != nil && aws.CloudProviderConfig != nil && aws.CloudProviderConfig.Subnet != nil && aws.CloudProviderConfig.Subnet.ID != nil {
		subnetID = *aws.CloudProviderConfig.Subnet.ID
	}

	// Build Network section in V1 format
	s += "Network:\n"
	// Always include Type - default to empty string if not set
	s += fmt.Sprintf(" - Type:                    %s\n", networkType)
	s += fmt.Sprintf(" - Service CIDR:            %s\n", serviceCIDR)
	s += fmt.Sprintf(" - Machine CIDR:            %s\n", machineCIDR)
	s += fmt.Sprintf(" - Pod CIDR:                %s\n", podCIDR)
	if hostPrefix != 0 {
		s += fmt.Sprintf(" - Host Prefix:             /%d\n", hostPrefix)
	}
	if subnetID != "" {
		s += fmt.Sprintf(" - Subnets:                 %s\n", subnetID)
	}

	// NOTE: Account-level STS roles (Installer, Support, Worker) are not used in V2 yet
	// V1 (OCM) shows: Role (STS) ARN, Support Role ARN, Instance IAM Roles
	// V2 (Hyperfleet) doesn't require these account-level roles
	// These sections are intentionally omitted for V2 clusters

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
