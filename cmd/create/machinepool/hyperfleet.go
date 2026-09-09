package machinepool

import (
	"context"
	"fmt"
	"os"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/ocm"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
)

// clusterNamespacePrefix is the prefix the Platform API requires on the
// metadata.namespace field of NodePool resources ("cluster-<uuid>").
// TODO: this derivation ideally belongs in the SDK (clientset/pathbind or
// the bridge wrapper) so consumers don't need to know the namespace format.
const clusterNamespacePrefix = "cluster-"

// hfNodePoolInput is the backing store for hyperfleet-specific create machinepool flags.
var hfNodePoolInput hfpathbind.NodePoolCreateInput

var (
	hfEnabled = hyperfleet.Enabled
	exitFn    = func(code int) { os.Exit(code) }

	hfCreateMachinePool = func(userOptions *mpOpts.CreateMachinepoolUserOptions, argv []string, cmd *cobra.Command) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()

		clusterKey, err := ocm.GetClusterKey()
		if err != nil || clusterKey == "" {
			r.Reporter.Errorf("--cluster is required")
			exitFn(1)
			return
		}
		clusterUID, err := hyperfleet.ResolveClusterUID(context.Background(), r.HyperFleetClient, clusterKey)
		if err != nil {
			r.Reporter.Errorf("%v", err)
			exitFn(1)
			return
		}

		handler := &hyperfleetNodePoolCreate{
			userOptions: userOptions,
			argv:        argv,
			clusterKey:  clusterKey,
			clusterUID:  clusterUID,
		}
		if err := hfpathbind.RunCreateNodePool(
			context.Background(),
			r,
			cmd,
			&hfNodePoolInput,
			handler,
			clusterNamespacePrefix+clusterUID,
		); err != nil {
			r.Reporter.Errorf("Failed to create node pool: %v", err)
			exitFn(1)
		}
	}
)

// runHyperfleetCreate is a thin wrapper for direct test invocation without a real cobra.Command.
func runHyperfleetCreate(r *rosa.Runtime, userOptions *mpOpts.CreateMachinepoolUserOptions, argv []string) {
	// Validate name first so tests that don't mock List don't get unexpected calls.
	name := userOptions.Name
	if name == "" && len(argv) > 0 {
		name = argv[0]
	}
	if name == "" {
		r.Reporter.Errorf("--name is required")
		exitFn(1)
		return
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
		return
	}
	clusterUID, err := hyperfleet.ResolveClusterUID(context.Background(), r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}
	handler := &hyperfleetNodePoolCreate{
		userOptions: userOptions,
		argv:        argv,
		clusterKey:  clusterKey,
		clusterUID:  clusterUID,
	}
	if err := hfpathbind.RunCreateNodePool(
		context.Background(),
		r,
		nil,
		&hfNodePoolInput,
		handler,
		clusterUID,
	); err != nil {
		exitFn(1)
	}
}

// hyperfleetNodePoolCreate implements hfpathbind.NodePoolCreateHandler for rosa create machinepool.
type hyperfleetNodePoolCreate struct {
	hfpathbind.GeneratedNodePoolCreatePrompt
	userOptions *mpOpts.CreateMachinepoolUserOptions
	argv        []string
	clusterKey  string
	clusterUID  string
}

func (h *hyperfleetNodePoolCreate) PreRequest(
	_ context.Context,
	r *rosa.Runtime,
	input *hfpathbind.NodePoolCreateInput,
) error {
	// Bridge: all nodepool flags (--name, --replicas, --instance-type, --subnet) share
	// names with OCM registrations so registerIfNew skips them; read from userOptions.
	// When these OCM flag registrations are removed, the bridges below can be dropped.
	name := h.userOptions.Name
	if name == "" && len(h.argv) > 0 {
		name = h.argv[0]
	}
	input.Name = name
	input.ClusterName = h.clusterKey

	instanceType := h.userOptions.InstanceType
	if instanceType == "" {
		instanceType = mpOpts.DefaultInstanceType
	}
	input.InstanceType = instanceType

	input.SubnetID = h.userOptions.Subnet

	replicas := int32(h.userOptions.Replicas)
	input.Replicas = &replicas

	if input.Name == "" {
		return fmt.Errorf("--name is required")
	}
	if input.SubnetID == "" {
		return fmt.Errorf("--subnet is required for Platform API node pool creation")
	}
	return nil
}

func (h *hyperfleetNodePoolCreate) PostExpand(
	ctx context.Context,
	r *rosa.Runtime,
	_ *hfpathbind.NodePoolCreateInput,
	obj *v1alpha1.NodePool,
) error {
	cluster, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().
		Get(ctx, h.clusterUID, platform.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cluster %q: %w", h.clusterKey, err)
	}

	var rolesRef hypershiftv1beta1.AWSRolesRef
	if cluster.Spec.HostedCluster.Platform.AWS != nil {
		rolesRef = cluster.Spec.HostedCluster.Platform.AWS.RolesRef
	}
	instanceProfile := hyperfleet.InstanceProfileFromRolesRef(rolesRef)
	if instanceProfile == "" {
		return fmt.Errorf("cannot derive worker instance profile from cluster roles ref")
	}

	obj.Spec.NodePool.Platform.Type = hypershiftv1beta1.AWSPlatform
	obj.Spec.NodePool.Platform.AWS.InstanceProfile = instanceProfile
	return nil
}

func (h *hyperfleetNodePoolCreate) PostResponse(_ context.Context, r *rosa.Runtime, created *v1alpha1.NodePool) error {
	r.Reporter.Infof("Node pool %q created in cluster %q (ID: %s)", created.Name, h.clusterKey, string(created.UID))
	return nil
}
