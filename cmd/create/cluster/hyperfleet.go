package cluster

import (
	"context"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, hfDescribeSubnets, and hfCreateCluster are package-level
// vars so tests can stub the hyperfleet dispatch path without real AWS calls.
var (
	hfEnabled         = hyperfleet.Enabled
	hfExitFn          = func(code int) { os.Exit(code) }
	hfDescribeSubnets = func(
		ctx context.Context, cfg awssdk.Config, subnetID string,
	) (*ec2svc.DescribeSubnetsOutput, error) {
		return ec2svc.NewFromConfig(cfg).DescribeSubnets(ctx, &ec2svc.DescribeSubnetsInput{
			SubnetIds: []string{subnetID},
		})
	}
	hfCreateCluster = func() {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleet(r)
	}
)

// runHyperfleet creates an HCP cluster via the Platform API v2.
// It derives VPC ID and availability zone from the provided subnet.
func runHyperfleet(r *rosa.Runtime) {
	ctx := context.Background()

	clusterName := args.clusterName
	if clusterName == "" {
		r.Reporter.Errorf("--cluster-name is required")
		hfExitFn(1)
		return
	}

	if args.operatorRolesPrefix == "" {
		r.Reporter.Errorf("--operator-roles-prefix is required")
		hfExitFn(1)
		return
	}

	if len(args.subnetIDs) == 0 {
		r.Reporter.Errorf("--subnet-ids is required")
		hfExitFn(1)
		return
	}
	subnetID := args.subnetIDs[0]

	// Derive VPC ID and availability zone from the subnet.
	subnetOut, err := hfDescribeSubnets(ctx, r.AWSConfig, subnetID)
	if err != nil {
		r.Reporter.Errorf("Failed to describe subnet '%s': %v", subnetID, err)
		hfExitFn(1)
		return
	}
	if len(subnetOut.Subnets) == 0 {
		r.Reporter.Errorf("Subnet '%s' not found", subnetID)
		hfExitFn(1)
		return
	}
	vpcID := awssdk.ToString(subnetOut.Subnets[0].VpcId)
	if vpcID == "" {
		r.Reporter.Errorf("Subnet '%s' has no VPC ID", subnetID)
		hfExitFn(1)
		return
	}
	zone := awssdk.ToString(subnetOut.Subnets[0].AvailabilityZone)
	if zone == "" {
		r.Reporter.Errorf("Subnet '%s' has no availability zone", subnetID)
		hfExitFn(1)
		return
	}

	rolesRef := hyperfleet.ComputeRolesRef(args.operatorRolesPrefix, r.Creator.AccountID, r.Creator.Partition)

	subnetRef := subnetID
	cluster, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters().Create(
		ctx,
		&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: clusterName},
			Spec: v1alpha1.ClusterSpec{
				HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
					Release: hypershiftv1beta1.Release{Image: args.version},
					Platform: hypershiftv1beta1.PlatformSpec{
						Type: hypershiftv1beta1.AWSPlatform,
						AWS: &hypershiftv1beta1.AWSPlatformSpec{
							Region:   r.Region,
							RolesRef: rolesRef,
							CloudProviderConfig: &hypershiftv1beta1.AWSCloudProviderConfig{
								VPC:  vpcID,
								Zone: zone,
								Subnet: &hypershiftv1beta1.AWSResourceReference{
									ID: &subnetRef,
								},
							},
						},
					},
				},
			},
		},
		platform.CreateOptions{},
	)
	if err != nil {
		r.Reporter.Errorf("Failed to create cluster '%s': %v", clusterName, err)
		hfExitFn(1)
		return
	}

	r.Reporter.Infof("Cluster '%s' created with ID '%s'", clusterName, string(cluster.UID))
}
