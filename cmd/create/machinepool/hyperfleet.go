package machinepool

import (
	"context"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled           = hyperfleet.Enabled
	exitFn              = func(code int) { os.Exit(code) }
	hfCreateMachinePool = func(userOptions *mpOpts.CreateMachinepoolUserOptions, argv []string) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetCreate(r, userOptions, argv)
	}
)

// runHyperfleetCreate creates a node pool via the Platform API v2.
// It reads --name (or positional arg), --replicas, --instance-type, and --subnet
// from the existing create machinepool flags. The release image is managed by the
// Platform API and does not need to be specified by the caller.
func runHyperfleetCreate(r *rosa.Runtime, userOptions *mpOpts.CreateMachinepoolUserOptions, argv []string) {
	ctx := context.Background()

	nodePoolName := userOptions.Name
	if nodePoolName == "" && len(argv) > 0 {
		nodePoolName = argv[0]
	}
	if nodePoolName == "" {
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

	// Resolve cluster name → UID, and fetch the cluster to default release image.
	clusterUID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
		return
	}

	cluster, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().
		Get(ctx, clusterUID, platform.GetOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		exitFn(1)
		return
	}

	instanceType := userOptions.InstanceType
	if instanceType == "" {
		instanceType = mpOpts.DefaultInstanceType
	}

	subnetID := userOptions.Subnet
	if subnetID == "" {
		r.Reporter.Errorf("--subnet is required for Platform API node pool creation")
		exitFn(1)
		return
	}

	var rolesRef hypershiftv1beta1.AWSRolesRef
	if cluster.Spec.HostedCluster.Platform.AWS != nil {
		rolesRef = cluster.Spec.HostedCluster.Platform.AWS.RolesRef
	}
	instanceProfile := hyperfleet.InstanceProfileFromRolesRef(rolesRef)
	if instanceProfile == "" {
		r.Reporter.Errorf("Cannot derive worker instance profile from cluster roles ref")
		exitFn(1)
		return
	}

	replicas := int32(userOptions.Replicas)

	np := &v1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodePoolName,
		},
		Spec: v1alpha1.NodePoolSpec{
			NodePool: v1alpha1.NodePoolSpecPassthrough{
				ClusterName: clusterKey,
				Replicas:    &replicas,
				Platform: hypershiftv1beta1.NodePoolPlatform{
					Type: hypershiftv1beta1.AWSPlatform,
					AWS: &hypershiftv1beta1.AWSNodePoolPlatform{
						InstanceType:    instanceType,
						InstanceProfile: instanceProfile,
						Subnet: hypershiftv1beta1.AWSResourceReference{
							ID: &subnetID,
						},
					},
				},
			},
		},
	}

	created, err := r.HyperFleetClient.HyperfleetV1alpha1().NodePools(clusterUID).Create(ctx, np, platform.CreateOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to create node pool '%s': %v", nodePoolName, err)
		exitFn(1)
		return
	}

	r.Reporter.Infof("Node pool '%s' created in cluster '%s' (ID: %s)", created.Name, clusterKey, string(created.UID))
}
