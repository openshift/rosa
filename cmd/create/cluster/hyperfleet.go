package cluster

import (
	"context"
	"fmt"
	"os"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfClusterInput is the backing store for all hyperfleet-specific create cluster flags.
// RegisterClusterCreateFlags binds cobra flags to its fields; runHyperfleet reads from it.
var hfClusterInput hfpathbind.ClusterCreateInput

// hfEnabled, hfExitFn, hfDescribeSubnets, and hfCreateCluster are package-level
// vars so tests can stub the hyperfleet dispatch path without real AWS calls.
var (
	hfEnabled = hyperfleet.Enabled
	hfExitFn  = func(code int) { os.Exit(code) }

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
		if err := hfpathbind.RunCreateCluster(context.Background(), r, cmd, &hfClusterInput,
			&hyperfleetClusterCreate{describeSubnets: hfDescribeSubnets},
		); err != nil {
			r.Reporter.Errorf("Failed to create cluster: %v", err)
			hfExitFn(1)
		}
	}
)

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
	input.Version = args.version
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
