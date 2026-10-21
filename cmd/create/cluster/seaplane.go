package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	rosaaws "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/clusterautoscaler"
	"github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
	seaplanepathbind "github.com/openshift/rosa/pkg/seaplane/pathbind"
)

// maxSeaplaneClusterTags is the number of customer AWS tags the Platform API
// accepts on spec.tags. HyperShift caps platform.aws.resourceTags at 25 entries
// and the operator injects 2 system tags (red-hat-managed and
// kubernetes.io/cluster/<id>), leaving 23 for the customer.
const maxSeaplaneClusterTags = 23

// seaplaneClusterInput is the backing store for all seaplane-specific create cluster flags.
// RegisterClusterCreateFlags binds cobra flags to its fields; runSeaplane reads from it.
var seaplaneClusterInput seaplanepathbind.ClusterCreateInput

// seaplaneExitFn, seaplaneDescribeSubnets, and seaplaneCreateCluster are package-level
// vars so tests can stub the seaplane path without real AWS calls.
var (
	seaplaneExitFn = func(code int) { os.Exit(code) }

	seaplaneDescribeSubnets = func(
		ctx context.Context, cfg awssdk.Config, subnetID string,
	) (*ec2svc.DescribeSubnetsOutput, error) {
		return ec2svc.NewFromConfig(cfg).DescribeSubnets(ctx, &ec2svc.DescribeSubnetsInput{
			SubnetIds: []string{subnetID},
		})
	}

	seaplaneCreateCluster = func(cmd *cobra.Command) {
		r := rosa.NewRuntime().WithSeaplane()
		defer r.Cleanup()
		// Seaplane create bypasses the classic hosted-cp validation in run(); reject classic-only flags here.
		if err := rejectUnsupportedSeaplaneCreateFlags(cmd); err != nil {
			r.Reporter.Errorf("%v", err)
			seaplaneExitFn(1)
			return
		}
		if err := runSeaplaneCreate(context.Background(), r, cmd); err != nil {
			r.Reporter.Errorf("Failed to create cluster: %v", seaplane.WithAPIErrorDetails(err))
			seaplaneExitFn(1)
		}
	}
)

// rejectUnsupportedSeaplaneCreateFlags rejects flags that select classic-only
// behavior and must not reach Platform API create.
func rejectUnsupportedSeaplaneCreateFlags(cmd *cobra.Command) error {
	if cmd == nil {
		return nil
	}

	unsupportedFlags := classicOnlyFlagsChanged(cmd)
	if len(unsupportedFlags) == 0 {
		return nil
	}
	return fmt.Errorf("the following flags are not supported for Seaplane cluster creation: %s",
		strings.Join(unsupportedFlags, ", "))
}

// runSeaplane is a thin wrapper for direct test invocation without a real cobra.Command.
func runSeaplane(r *rosa.Runtime) {
	if err := runSeaplaneCreate(context.Background(), r, nil); err != nil {
		seaplaneExitFn(1)
	}
}

// runSeaplaneCreate retains the generated workflow and extends requests for
// proxy fields that the pinned SDK does not yet represent.
func runSeaplaneCreate(ctx context.Context, r *rosa.Runtime, cmd *cobra.Command) error {
	runtime := *r
	var proxy *seaplane.ClusterProxy
	if args.httpProxy != "" || args.httpsProxy != "" || len(args.noProxySlice) > 0 {
		proxy = &seaplane.ClusterProxy{
			HTTPProxy: args.httpProxy, HTTPSProxy: args.httpsProxy, NoProxy: strings.Join(args.noProxySlice, ","),
		}
	}
	if proxy != nil || args.additionalTrustBundleFile != "" {
		runtime.SeaplaneClient = seaplane.WithClusterProxy(r.SeaplaneClient, proxy, args.additionalTrustBundleFile)
	}
	return seaplanepathbind.RunCreateCluster(ctx, &runtime, cmd, &seaplaneClusterInput,
		&seaplaneClusterCreate{cmd: cmd, describeSubnets: seaplaneDescribeSubnets})
}

// seaplaneClusterCreate implements seaplanepathbind.ClusterCreateHandler for rosa create cluster.
type seaplaneClusterCreate struct {
	// interactive prompting for required fields
	seaplanepathbind.GeneratedClusterCreatePrompt
	cmd             *cobra.Command
	describeSubnets func(context.Context, awssdk.Config, string) (*ec2svc.DescribeSubnetsOutput, error)
}

func (h *seaplaneClusterCreate) PreRequest(
	ctx context.Context,
	r *rosa.Runtime,
	input *seaplanepathbind.ClusterCreateInput,
) error {
	// Bridge flags that conflict with OCM v1 registrations (registerIfNew skips them,
	// so they remain backed by args.* rather than seaplaneClusterInput.*).
	input.Name = args.clusterName
	if args.version != "" {
		input.Version = args.version
	}
	input.OperatorRolesPrefix = args.operatorRolesPrefix

	// --subnet-id is a new Seaplane-only flag (no OCM equivalent) so it IS registered on
	// seaplaneClusterInput and arrives directly from cobra. Fall back to --subnet-ids[0]
	// for backward compatibility with existing scripts that use the OCM flag.
	if input.SubnetID == "" && len(args.subnetIDs) > 0 {
		input.SubnetID = args.subnetIDs[0]
	}

	// --expiration-time / --expiration are OCM-registered (hidden) flags; bridge via
	// validateExpiration() which reads args.*. Safe to call unconditionally — it
	// returns a zero time when neither flag was passed so nothing changes.
	expiry, err := validateExpiration()
	if err != nil {
		return err
	}
	if !expiry.IsZero() {
		input.ExpirationTimestamp = expiry.UTC().Format(time.RFC3339)
	}

	// input.DeleteProtection arrives directly from cobra via the Platform API flag
	// --delete-protection — no bridge needed.

	if input.Name == "" {
		return fmt.Errorf("--cluster-name is required")
	}
	if input.OperatorRolesPrefix == "" {
		return fmt.Errorf("--operator-roles-prefix is required")
	}
	if input.SubnetID == "" {
		return fmt.Errorf("--subnet-id (or --subnet-ids) is required")
	}
	if err := validateSeaplaneNetworkType(args.networkType); err != nil {
		return err
	}
	if args.noCni && args.networkType != "" {
		return fmt.Errorf("--no-cni and --network-type are mutually exclusive parameters")
	}
	if err := validateSeaplaneClusterProxyFlags(); err != nil {
		return err
	}
	if err := setAdditionalAllowedPrincipals(input); err != nil {
		return err
	}

	// OIDC config ID is optional but recommended
	if args.oidcConfigId == "" {
		r.Reporter.Warnf("--oidc-config-id not provided, cluster will use auto-generated OIDC config")
	} else {
		oidcConfig, err := seaplane.ResolveOidcConfig(ctx, r.SeaplaneClient, args.oidcConfigId)
		if err != nil {
			return fmt.Errorf("failed to resolve OIDC config %q: %w", args.oidcConfigId, err)
		}
		input.OidcConfigId = string(oidcConfig.UID)
		r.Reporter.Infof("Using OIDC config ID: %s", input.OidcConfigId)
	}

	// --tags is OCM-registered as a string slice, so registerIfNew skips the Seaplane
	// flag and input.Tags stays empty. Bridge args.tags into spec.tags, which the
	// Platform API applies to the AWS resources created for this cluster.
	if input.Tags == "" {
		tags, err := parseSeaplaneClusterTags(args.tags)
		if err != nil {
			return err
		}
		if len(tags) > 0 {
			encoded, err := json.Marshal(tags)
			if err != nil {
				return fmt.Errorf("failed to encode --tags: %w", err)
			}
			input.Tags = string(encoded)
		}
	}

	// Derive VPC ID and availability zone from the subnet.
	subnetOut, err := h.describeSubnets(ctx, r.AWSConfig, input.SubnetID)
	if err != nil {
		return fmt.Errorf("failed to describe subnet %q: %w", input.SubnetID, err)
	}
	if len(subnetOut.Subnets) == 0 {
		return fmt.Errorf("subnet %q not found", input.SubnetID)
	}
	input.VPC = awssdk.ToString(subnetOut.Subnets[0].VpcId)
	if input.VPC == "" {
		return fmt.Errorf("subnet %q has no VPC ID", input.SubnetID)
	}
	input.Zone = awssdk.ToString(subnetOut.Subnets[0].AvailabilityZone)
	if input.Zone == "" {
		return fmt.Errorf("subnet %q has no availability zone", input.SubnetID)
	}

	input.Region = r.Region

	// Convert individual networking flags to JSON format if provided
	if err := convertNetworkingFlags(input); err != nil {
		return err
	}
	if err := convertAutoscalerFlags(h.cmd, input); err != nil {
		return err
	}

	return nil
}

func convertAutoscalerFlags(cmd *cobra.Command, input *seaplanepathbind.ClusterCreateInput) error {
	if cmd == nil || input == nil {
		return nil
	}
	// Depending on registration order, the generated Seaplane flags may bind
	// directly to input instead of the legacy autoscalerArgs structure. Merge
	// both sources before using the shared validation/conversion helper.
	effectiveArgs := clusterautoscaler.AutoscalerArgs{}
	if autoscalerArgs != nil {
		effectiveArgs = *autoscalerArgs
	}
	if input.MaxNodeProvisionTime != "" {
		effectiveArgs.MaxNodeProvisionTime = input.MaxNodeProvisionTime
	}
	if input.MaxNodesTotal != nil {
		effectiveArgs.ResourceLimits.MaxNodesTotal = int(*input.MaxNodesTotal)
	}
	if input.MaxPodGracePeriod != nil {
		effectiveArgs.MaxPodGracePeriod = int(*input.MaxPodGracePeriod)
	}
	if input.PodPriorityThreshold != nil {
		effectiveArgs.PodPriorityThreshold = int(*input.PodPriorityThreshold)
	}
	if !clusterautoscaler.IsAutoscalerSetViaCLI(cmd.Flags(), clusterAutoscalerFlagsPrefix) {
		return nil
	}
	supported, err := clusterautoscaler.ValidateAutoscalerFlagsForHostedCp(
		clusterAutoscalerFlagsPrefix, cmd)
	if !supported || err != nil {
		return err
	}
	changes := clusterautoscaler.SeaplaneAutoscalingChanges{
		MaxNodeProvisionTime: cmd.Flags().Changed(clusterAutoscalerFlagsPrefix + "max-node-provision-time"),
		MaxNodesTotal:        cmd.Flags().Changed(clusterAutoscalerFlagsPrefix + "max-nodes-total"),
		MaxPodGracePeriod:    cmd.Flags().Changed(clusterAutoscalerFlagsPrefix + "max-pod-grace-period"),
		PodPriorityThreshold: cmd.Flags().Changed(clusterAutoscalerFlagsPrefix + "pod-priority-threshold"),
	}
	config, err := clusterautoscaler.BuildSeaplaneAutoscalingConfig(
		&effectiveArgs, clusterAutoscalerFlagsPrefix, changes)
	if err != nil || config == nil {
		return err
	}
	if config.MaxNodeProvisionTime != nil {
		input.MaxNodeProvisionTime = *config.MaxNodeProvisionTime
	}
	input.MaxNodesTotal = config.MaxNodesTotal
	input.MaxPodGracePeriod = config.MaxPodGracePeriod
	input.PodPriorityThreshold = config.PodPriorityThreshold

	return nil
}

func validateSeaplaneClusterProxyFlags() error {
	if err := ocm.ValidateHTTPProxy(args.httpProxy); err != nil {
		return err
	}
	if err := ocm.ValidateHTTPSProxy(args.httpsProxy); err != nil {
		return err
	}
	if len(args.noProxySlice) > 0 {
		duplicate, found := rosaaws.HasDuplicates(args.noProxySlice)
		if found {
			//nolint:staticcheck // Preserve the v1 validation message.
			return fmt.Errorf("Invalid no-proxy list, duplicate key '%s' found", duplicate)
		}
		for _, domain := range args.noProxySlice {
			if err := rosaaws.UserNoProxyValidator(domain); err != nil {
				return err
			}
		}
		if args.httpProxy == "" && args.httpsProxy == "" {
			//nolint:staticcheck // Preserve the v1 validation message.
			return fmt.Errorf("Expected at least one of the following: http-proxy, https-proxy")
		}
	}
	return ocm.ValidateAdditionalTrustBundle(args.additionalTrustBundleFile)
}

func setAdditionalAllowedPrincipals(input *seaplanepathbind.ClusterCreateInput) error {
	if len(args.additionalAllowedPrincipals) == 0 {
		return nil
	}
	if err := roles.ValidateAdditionalAllowedPrincipals(args.additionalAllowedPrincipals); err != nil {
		return err
	}

	encodedPrincipals, err := json.Marshal(args.additionalAllowedPrincipals)
	if err != nil {
		return fmt.Errorf("failed to encode additional allowed principals: %w", err)
	}
	input.AdditionalAllowedPrincipals = string(encodedPrincipals)
	return nil
}

func (h *seaplaneClusterCreate) PostExpand(
	_ context.Context,
	r *rosa.Runtime,
	input *seaplanepathbind.ClusterCreateInput,
	obj *v1alpha1.Cluster,
) error {
	obj.Spec.HostedCluster.Platform.Type = hypershiftv1beta1.AWSPlatform
	obj.Spec.HostedCluster.Platform.AWS.RolesRef =
		seaplane.ComputeRolesRef(input.OperatorRolesPrefix, r.Creator.AccountID, r.Creator.Partition)

	return nil
}

func (h *seaplaneClusterCreate) PostResponse(_ context.Context, r *rosa.Runtime, cluster *v1alpha1.Cluster) error {
	r.Reporter.Infof("Cluster %q created with ID %q", cluster.Name, string(cluster.UID))
	return nil
}

// parseSeaplaneClusterTags converts the --tags values into the map the Platform
// API expects on spec.tags. Tags are applied at provisioning time only: AWS
// resources created by OpenShift cannot be retagged, so spec.tags is immutable
// once the cluster exists.
func parseSeaplaneClusterTags(input []string) (map[string]string, error) {
	if len(input) == 0 {
		return nil, nil
	}
	if len(input) > maxSeaplaneClusterTags {
		return nil, fmt.Errorf("invalid cluster AWS tags: a maximum of %d tags is supported, got %d",
			maxSeaplaneClusterTags, len(input))
	}
	if err := rosaaws.UserTagValidator(rosaaws.UserTagOptions{})(input); err != nil {
		return nil, err
	}

	delimiter := rosaaws.GetTagsDelimiter(input)
	tags := make(map[string]string, len(input))
	for _, tag := range input {
		parts := strings.SplitN(tag, delimiter, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag format for tag '%s'. Expected tag format: 'key value'", tag)
		}
		tags[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return tags, nil
}

// convertNetworkingFlags converts individual networking flags (--machine-cidr, --service-cidr,
// --pod-cidr, --host-prefix) into the JSON format expected by the V2 API.
func convertNetworkingFlags(input *seaplanepathbind.ClusterCreateInput) error {
	// Only convert if the JSON flags are not already set
	if input.MachineNetwork == "" && !isEmptyCIDR(&args.machineCIDR) {
		input.MachineNetwork = fmt.Sprintf(`[{"cidr":"%s"}]`, args.machineCIDR.String())
	}

	if input.ServiceNetwork == "" && !isEmptyCIDR(&args.serviceCIDR) {
		input.ServiceNetwork = fmt.Sprintf(`[{"cidr":"%s"}]`, args.serviceCIDR.String())
	}

	if input.ClusterNetwork == "" && !isEmptyCIDR(&args.podCIDR) {
		// ClusterNetwork includes both CIDR and hostPrefix
		hostPrefix := args.hostPrefix
		if hostPrefix == 0 {
			hostPrefix = 23 // default host prefix
		}
		input.ClusterNetwork = fmt.Sprintf(`[{"cidr":"%s","hostPrefix":%d}]`, args.podCIDR.String(), hostPrefix)
	}

	if args.noCni {
		input.NetworkType = "Other"
	} else if args.networkType != "" {
		input.NetworkType = args.networkType
	}

	// Set allowed CIDR blocks (default to open if not specified)
	if input.AllowedCIDRBlocks == "" {
		input.AllowedCIDRBlocks = `["0.0.0.0/0"]`
	}

	return nil
}

func validateSeaplaneNetworkType(networkType string) error {
	if networkType == "" || networkType == ocm.NetworkTypeOvn {
		return nil
	}
	return fmt.Errorf("expected a valid network type. Valid values: [%s]", ocm.NetworkTypeOvn)
}

// isEmptyCIDR checks if a CIDR is empty (all zeros)
func isEmptyCIDR(cidr interface{ String() string }) bool {
	s := cidr.String()
	return s == "" || s == "<nil>"
}
