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
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/hyperfleet"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/rosa"
)

// maxHyperfleetClusterTags is the number of customer AWS tags the Platform API
// accepts on spec.tags. HyperShift caps platform.aws.resourceTags at 25 entries
// and the operator injects 2 system tags (red-hat-managed and
// kubernetes.io/cluster/<id>), leaving 23 for the customer.
const maxHyperfleetClusterTags = 23

// hfClusterInput is the backing store for all hyperfleet-specific create cluster flags.
// RegisterClusterCreateFlags binds cobra flags to its fields; runHyperfleet reads from it.
var hfClusterInput hfpathbind.ClusterCreateInput

// hfExitFn, hfDescribeSubnets, and hfCreateCluster are package-level
// vars so tests can stub the hyperfleet path without real AWS calls.
var (
	hfExitFn = func(code int) { os.Exit(code) }

	hfDescribeSubnets = func(
		ctx context.Context, cfg awssdk.Config, subnetID string,
	) (*ec2svc.DescribeSubnetsOutput, error) {
		return ec2svc.NewFromConfig(cfg).DescribeSubnets(ctx, &ec2svc.DescribeSubnetsInput{
			SubnetIds: []string{subnetID},
		})
	}

	hfCreateCluster = func(cmd *cobra.Command) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		// HF create bypasses the classic hosted-cp validation in run(); reject classic-only flags here.
		if err := rejectUnsupportedHyperfleetCreateFlags(cmd); err != nil {
			r.Reporter.Errorf("%v", err)
			hfExitFn(1)
			return
		}
		if err := hfpathbind.RunCreateCluster(context.Background(), r, cmd, &hfClusterInput,
			&hyperfleetClusterCreate{describeSubnets: hfDescribeSubnets},
		); err != nil {
			r.Reporter.Errorf("Failed to create cluster: %v", err)
			hfExitFn(1)
		}
	}
)

// rejectUnsupportedHyperfleetCreateFlags rejects flags that select classic-only
// behavior and must not reach Platform API create.
func rejectUnsupportedHyperfleetCreateFlags(cmd *cobra.Command) error {
	if cmd == nil {
		return nil
	}

	unsupportedFlags := classicOnlyFlagsChanged(cmd)
	if len(unsupportedFlags) == 0 {
		return nil
	}
	return fmt.Errorf("the following flags are not supported for HyperFleet cluster creation: %s",
		strings.Join(unsupportedFlags, ", "))
}

// classicOnlyFlagsChanged identifies explicit flags that require classic architecture.
func classicOnlyFlagsChanged(cmd *cobra.Command) []string {
	var changed []string
	for _, name := range []string{"non-sts", "mint-mode", privateLinkFlagName, "disable-workload-monitoring"} {
		flag := cmd.Flags().Lookup(name)
		if flag != nil && flag.Changed && flag.Value.String() == "true" {
			changed = append(changed, "--"+name)
		}
	}
	if flag := cmd.Flags().Lookup("sts"); flag != nil && flag.Changed && flag.Value.String() == "false" {
		changed = append(changed, "--sts=false")
	}
	for _, name := range []string{
		"additional-infra-security-group-ids", "additional-control-plane-security-group-ids",
		"controlplane-iam-role-arn", "master-iam-role", "worker-mp-labels",
		"default-ingress-route-selector", "default-ingress-excluded-namespaces",
		"default-ingress-wildcard-policy", "default-ingress-namespace-ownership-policy", "availability-zones",
	} {
		flag := cmd.Flags().Lookup(name)
		if flag != nil && flag.Changed && flag.Value.String() != flag.DefValue {
			changed = append(changed, "--"+name)
		}
	}
	return changed
}

// runHyperfleet is a thin wrapper for direct test invocation without a real cobra.Command.
func runHyperfleet(r *rosa.Runtime) {
	if err := hfpathbind.RunCreateCluster(context.Background(), r, nil, &hfClusterInput,
		&hyperfleetClusterCreate{describeSubnets: hfDescribeSubnets},
	); err != nil {
		hfExitFn(1)
	}
}

// hyperfleetClusterCreate implements hfpathbind.ClusterCreateHandler for rosa create cluster.
type hyperfleetClusterCreate struct {
	// interactive prompting for required fields
	hfpathbind.GeneratedClusterCreatePrompt
	describeSubnets func(context.Context, awssdk.Config, string) (*ec2svc.DescribeSubnetsOutput, error)
}

func (h *hyperfleetClusterCreate) PreRequest(
	ctx context.Context,
	r *rosa.Runtime,
	input *hfpathbind.ClusterCreateInput,
) error {
	// Bridge flags that conflict with OCM v1 registrations (registerIfNew skips them,
	// so they remain backed by args.* rather than hfClusterInput.*).
	input.Name = args.clusterName
	if args.version != "" {
		input.Version = args.version
	}
	input.OperatorRolesPrefix = args.operatorRolesPrefix

	// --subnet-id is a new HF-only flag (no OCM equivalent) so it IS registered on
	// hfClusterInput and arrives directly from cobra. Fall back to --subnet-ids[0]
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

	// input.DisplayName and input.DeleteProtection arrive directly from cobra via
	// the new HF-only flags (--display-name, --delete-protection) — no bridge needed.

	if input.Name == "" {
		return fmt.Errorf("--cluster-name is required")
	}
	if input.OperatorRolesPrefix == "" {
		return fmt.Errorf("--operator-roles-prefix is required")
	}
	if input.SubnetID == "" {
		return fmt.Errorf("--subnet-id (or --subnet-ids) is required")
	}
	if err := validateNetworkType(args.networkType); err != nil {
		return err
	}
	if args.noCni && args.networkType != "" {
		return fmt.Errorf("--no-cni and --network-type are mutually exclusive parameters")
	}

	// OIDC config ID is optional but recommended
	if args.oidcConfigId == "" {
		r.Reporter.Warnf("--oidc-config-id not provided, cluster will use auto-generated OIDC config")
	} else {
		r.Reporter.Infof("Using OIDC config ID: %s", args.oidcConfigId)
		input.OidcConfigId = args.oidcConfigId
	}

	// --tags is OCM-registered as a string slice, so registerIfNew skips the HF
	// flag and input.Tags stays empty. Bridge args.tags into spec.tags, which the
	// Platform API applies to the AWS resources created for this cluster.
	if input.Tags == "" {
		tags, err := parseHyperfleetClusterTags(args.tags)
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

	return nil
}

func (h *hyperfleetClusterCreate) PostExpand(
	_ context.Context,
	r *rosa.Runtime,
	input *hfpathbind.ClusterCreateInput,
	obj *v1alpha1.Cluster,
) error {
	obj.Spec.HostedCluster.Platform.Type = hypershiftv1beta1.AWSPlatform
	obj.Spec.HostedCluster.Platform.AWS.RolesRef =
		hyperfleet.ComputeRolesRef(input.OperatorRolesPrefix, r.Creator.AccountID, r.Creator.Partition)
	return nil
}

func (h *hyperfleetClusterCreate) PostResponse(_ context.Context, r *rosa.Runtime, cluster *v1alpha1.Cluster) error {
	r.Reporter.Infof("Cluster %q created with ID %q", cluster.Name, string(cluster.UID))
	return nil
}

// parseHyperfleetClusterTags converts the --tags values into the map the Platform
// API expects on spec.tags. Tags are applied at provisioning time only: AWS
// resources created by OpenShift cannot be retagged, so spec.tags is immutable
// once the cluster exists.
func parseHyperfleetClusterTags(input []string) (map[string]string, error) {
	if len(input) == 0 {
		return nil, nil
	}
	if len(input) > maxHyperfleetClusterTags {
		return nil, fmt.Errorf("invalid cluster AWS tags: a maximum of %d tags is supported, got %d",
			maxHyperfleetClusterTags, len(input))
	}
	if err := aws.UserTagValidator(input); err != nil {
		return nil, err
	}

	delimiter := aws.GetTagsDelimiter(input)
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
func convertNetworkingFlags(input *hfpathbind.ClusterCreateInput) error {
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

	// Set network type if not already set
	if input.NetworkType == "" && args.networkType != "" {
		input.NetworkType = args.networkType
	}

	// Set allowed CIDR blocks (default to open if not specified)
	if input.AllowedCIDRBlocks == "" {
		input.AllowedCIDRBlocks = `["0.0.0.0/0"]`
	}

	return nil
}

// isEmptyCIDR checks if a CIDR is empty (all zeros)
func isEmptyCIDR(cidr interface{ String() string }) bool {
	s := cidr.String()
	return s == "" || s == "<nil>"
}
