package cluster

import (
	"context"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/hyperfleet-operator/api/v1alpha1"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled and hfCreateCluster are package-level vars so tests can stub the
// hyperfleet dispatch path without making real AWS or Platform API calls.
var (
	hfEnabled       = hyperfleet.Enabled
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
		os.Exit(1)
	}

	if args.operatorRolesPrefix == "" {
		r.Reporter.Errorf("--operator-roles-prefix is required")
		os.Exit(1)
	}

	if len(args.subnetIDs) == 0 {
		r.Reporter.Errorf("--subnet-ids is required")
		os.Exit(1)
	}
	subnetID := args.subnetIDs[0]

	// Derive VPC ID and availability zone from the subnet.
	ec2Client := ec2svc.NewFromConfig(r.AWSConfig)
	subnetOut, err := ec2Client.DescribeSubnets(ctx, &ec2svc.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	})
	if err != nil {
		r.Reporter.Errorf("Failed to describe subnet '%s': %v", subnetID, err)
		os.Exit(1)
	}
	if len(subnetOut.Subnets) == 0 {
		r.Reporter.Errorf("Subnet '%s' not found", subnetID)
		os.Exit(1)
	}
	vpcID := *subnetOut.Subnets[0].VpcId
	zone := *subnetOut.Subnets[0].AvailabilityZone

	rolesRef := hyperfleet.ComputeRolesRef(args.operatorRolesPrefix, r.Creator.AccountID)

	subnetRef := subnetID
	cluster, err := r.HyperFleetClient.HyperfleetV1alpha1().Clusters(r.Creator.AccountID).Create(
		ctx,
		&v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: clusterName},
			Spec: v1alpha1.ClusterSpec{
				CreatorARN: r.Creator.ARN,
				HostedCluster: hypershiftv1beta1.HostedClusterSpec{
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
		wrappers.CreateOptions{},
	)
	if err != nil {
		r.Reporter.Errorf("Failed to create cluster '%s': %v", clusterName, err)
		os.Exit(1)
	}

	r.Reporter.Infof("Cluster '%s' created with ID '%s'", clusterName, string(cluster.UID))
}
