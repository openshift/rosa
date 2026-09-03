package cluster

import (
	"context"
	"fmt"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/spf13/cobra"

	pkgaws "github.com/openshift/rosa/pkg/aws"
	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled    func() bool
		origRunCluster func(*cobra.Command)
	)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origRunCluster = hfCreateCluster
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfCreateCluster = origRunCluster
	})

	It("routes to hfCreateCluster when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfCreateCluster = func(*cobra.Command) { called = true }

		run(nil, nil)

		Expect(called).To(BeTrue())
	})
})

var _ = Describe("runHyperfleet", func() {
	var (
		origExitFn          func(int)
		origDescribeSubnets func(context.Context, awssdk.Config, string) (*ec2svc.DescribeSubnetsOutput, error)
		origArgs            struct {
			clusterName         string
			operatorRolesPrefix string
			subnetIDs           []string
		}
		exited bool
		t      *test.TestingRuntime
	)

	BeforeEach(func() {
		origExitFn = hfExitFn
		origDescribeSubnets = hfDescribeSubnets
		origArgs.clusterName = args.clusterName
		origArgs.operatorRolesPrefix = args.operatorRolesPrefix
		origArgs.subnetIDs = args.subnetIDs

		exited = false
		hfExitFn = func(int) { exited = true }

		args.clusterName = "test-cluster"
		args.operatorRolesPrefix = "test-cluster"
		args.subnetIDs = []string{"subnet-abc123"}

		t = test.NewTestRuntime()
		t.RosaRuntime.Creator = &pkgaws.Creator{
			AccountID: "123456789012",
			ARN:       "arn:aws-us-gov:iam::123456789012:user/test",
			Partition: "aws-us-gov",
		}
	})

	AfterEach(func() {
		hfExitFn = origExitFn
		hfDescribeSubnets = origDescribeSubnets
		args.clusterName = origArgs.clusterName
		args.operatorRolesPrefix = origArgs.operatorRolesPrefix
		args.subnetIDs = origArgs.subnetIDs
	})

	stubSubnets := func(vpcID, az string) {
		hfDescribeSubnets = func(_ context.Context, _ awssdk.Config, _ string) (*ec2svc.DescribeSubnetsOutput, error) {
			return &ec2svc.DescribeSubnetsOutput{
				Subnets: []ec2types.Subnet{
					{VpcId: awssdk.String(vpcID), AvailabilityZone: awssdk.String(az)},
				},
			}, nil
		}
	}

	It("creates a cluster and uses the partition in role ARNs", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf := hfmocks.NewMockInterface(ctrl)
		v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
		clusters := hfmocks.NewMockClusterInterface(ctrl)
		hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
		v1.EXPECT().Clusters().Return(clusters).AnyTimes()

		var capturedCluster *v1alpha1.Cluster
		clusters.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, c *v1alpha1.Cluster, _ interface{}) (*v1alpha1.Cluster, error) {
				capturedCluster = c
				return &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{UID: types.UID("cluster-uid")}}, nil
			})

		stubSubnets("vpc-123", "us-gov-east-1a")
		t.RosaRuntime.HyperFleetClient = hf

		runHyperfleet(t.RosaRuntime)

		Expect(exited).To(BeFalse())
		Expect(capturedCluster).NotTo(BeNil())
		rolesRef := capturedCluster.Spec.HostedCluster.Platform.AWS.RolesRef
		Expect(rolesRef.IngressARN).To(HavePrefix("arn:aws-us-gov:iam::"),
			"role ARNs must use the GovCloud partition")
		Expect(rolesRef.NodePoolManagementARN).To(HavePrefix("arn:aws-us-gov:iam::"),
			"role ARNs must use the GovCloud partition")
	})

	It("exits when cluster name is missing", func() {
		args.clusterName = ""
		runHyperfleet(&rosa.Runtime{Reporter: t.RosaRuntime.Reporter})
		Expect(exited).To(BeTrue())
	})

	It("exits when operator roles prefix is missing", func() {
		args.operatorRolesPrefix = ""
		runHyperfleet(&rosa.Runtime{Reporter: t.RosaRuntime.Reporter})
		Expect(exited).To(BeTrue())
	})

	It("exits when subnet IDs are missing", func() {
		args.subnetIDs = nil
		runHyperfleet(&rosa.Runtime{Reporter: t.RosaRuntime.Reporter})
		Expect(exited).To(BeTrue())
	})

	It("exits when DescribeSubnets fails", func() {
		hfDescribeSubnets = func(_ context.Context, _ awssdk.Config, _ string) (*ec2svc.DescribeSubnetsOutput, error) {
			return nil, fmt.Errorf("describe error")
		}
		runHyperfleet(t.RosaRuntime)
		Expect(exited).To(BeTrue())
	})

	It("exits when subnet is not found", func() {
		hfDescribeSubnets = func(_ context.Context, _ awssdk.Config, _ string) (*ec2svc.DescribeSubnetsOutput, error) {
			return &ec2svc.DescribeSubnetsOutput{Subnets: []ec2types.Subnet{}}, nil
		}
		runHyperfleet(t.RosaRuntime)
		Expect(exited).To(BeTrue())
	})

	It("exits when VPC ID is missing from subnet", func() {
		hfDescribeSubnets = func(_ context.Context, _ awssdk.Config, _ string) (*ec2svc.DescribeSubnetsOutput, error) {
			return &ec2svc.DescribeSubnetsOutput{
				Subnets: []ec2types.Subnet{{AvailabilityZone: awssdk.String("us-east-1a")}},
			}, nil
		}
		runHyperfleet(t.RosaRuntime)
		Expect(exited).To(BeTrue())
	})

	It("exits when availability zone is missing from subnet", func() {
		hfDescribeSubnets = func(_ context.Context, _ awssdk.Config, _ string) (*ec2svc.DescribeSubnetsOutput, error) {
			return &ec2svc.DescribeSubnetsOutput{
				Subnets: []ec2types.Subnet{{VpcId: awssdk.String("vpc-123")}},
			}, nil
		}
		runHyperfleet(t.RosaRuntime)
		Expect(exited).To(BeTrue())
	})

	It("exits when cluster create fails", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf := hfmocks.NewMockInterface(ctrl)
		v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
		clusters := hfmocks.NewMockClusterInterface(ctrl)
		hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
		v1.EXPECT().Clusters().Return(clusters).AnyTimes()
		clusters.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("create error"))

		stubSubnets("vpc-123", "us-gov-east-1a")
		t.RosaRuntime.HyperFleetClient = hf

		runHyperfleet(t.RosaRuntime)
		Expect(exited).To(BeTrue())
	})
})
