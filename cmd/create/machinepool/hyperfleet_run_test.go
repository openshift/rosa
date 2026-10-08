package machinepool

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	awsapi "github.com/openshift/rosa/pkg/aws/api_interface"
	awsmocks "github.com/openshift/rosa/pkg/aws/mocks"
	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/ocm"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/test"
)

func newCreateMPMocks(ctrl *gomock.Controller) (
	*hfmocks.MockInterface,
	*hfmocks.MockClusterInterface,
	*hfmocks.MockNodePoolInterface,
) {
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	nodePools := hfmocks.NewMockNodePoolInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	v1.EXPECT().NodePools(gomock.Any()).Return(nodePools).AnyTimes()
	return hf, clusters, nodePools
}

func makeCluster(name, uid string) *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(uid)},
		Spec: v1alpha1.ClusterSpec{
			HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
				Platform: v1alpha1.PlatformSpec{
					AWS: &hypershiftv1beta1.AWSPlatformSpec{
						CloudProviderConfig: &hypershiftv1beta1.AWSCloudProviderConfig{VPC: "vpc-cluster"},
						RolesRef: hypershiftv1beta1.AWSRolesRef{
							NodePoolManagementARN: "arn:aws:iam::123456789:role/cluster1-node-pool-management",
						},
					},
				},
			},
		},
	}
}

var _ = Describe("runHyperfleetCreate (machinepool)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		mockMachinePoolAWSValidation()
	})

	It("creates a node pool on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newCreateMPMocks(ctrl)

		cluster := makeCluster("cluster1", "cluster-uid")
		cluster.Status.Version = "4.20.3"
		cluster.Spec.HostedCluster.Release.Image = "quay.io/release@sha256:abc"
		created := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: "np-uid-new"}}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		nodePools.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, np *v1alpha1.NodePool, _ platform.CreateOptions) (*v1alpha1.NodePool, error) {
				Expect(np.Name).To(Equal("my-np"))
				Expect(np.Spec.NodePool.Release.Image).To(Equal("quay.io/release@sha256:abc"))
				Expect(np.Spec.NodePool.ClusterName).To(Equal("cluster1"))
				Expect(*np.Spec.NodePool.Replicas).To(Equal(int32(2)))
				Expect(np.Spec.NodePool.Platform.AWS.InstanceProfile).To(Equal("cluster1-ROSA-Worker-Role"))
				Expect(*np.Spec.NodePool.Platform.AWS.Subnet.ID).To(Equal("subnet-abc123"))
				Expect(np.Spec.NodePool.Platform.AWS.RootVolume.Size).To(Equal(int64(75)))
				Expect(np.Spec.NodePool.Platform.AWS.ResourceTags).To(ConsistOf(
					hypershiftv1beta1.AWSResourceTag{Key: "test", Value: "testvalue"},
					hypershiftv1beta1.AWSResourceTag{Key: "test2", Value: "testValue/openshift"},
				))
				return created, nil
			})

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
			Name:         "my-np",
			Version:      "4.20.3",
			Replicas:     2,
			InstanceType: "m5.xlarge",
			Subnet:       "subnet-abc123",
			RootDiskSize: "75GiB",
			Tags:         []string{"test:testvalue", "test2:testValue/openshift"},
		}, nil)
	})

	It("resolves name from argv when Name option is empty", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newCreateMPMocks(ctrl)

		cluster := makeCluster("cluster1", "cluster-uid")
		created := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-np", UID: "np-uid-new"}}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		nodePools.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, np *v1alpha1.NodePool, _ platform.CreateOptions) (*v1alpha1.NodePool, error) {
				Expect(np.Name).To(Equal("my-np"))
				Expect(np.Spec.NodePool.ClusterName).To(Equal("cluster1"))
				Expect(*np.Spec.NodePool.Replicas).To(Equal(int32(2)))
				Expect(np.Spec.NodePool.Platform.AWS.InstanceProfile).To(Equal("cluster1-ROSA-Worker-Role"))
				Expect(*np.Spec.NodePool.Platform.AWS.Subnet.ID).To(Equal("subnet-abc123"))
				return created, nil
			})

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{Replicas: 2, Subnet: "subnet-abc123"}, []string{"my-np"})
	})

	It("fails when name is not specified", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newCreateMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{}, nil)
		}).To(Panic())
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _, _ := newCreateMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{Name: "my-np"}, nil)
		}).To(Panic())
	})

	It("rejects the invalid name from 56786 before calling the API", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		// No calls are expected, including cluster lookup and node pool creation.
		t.RosaRuntime.HyperFleetClient = hfmocks.NewMockInterface(ctrl)
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "anything%^#@", Replicas: 2,
			}, nil)
		}).To(PanicWith("exit"))
	})

	It("rejects the unsupported instance type from 56786 before creating a node pool", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		t.RosaRuntime.HyperFleetClient = hf
		client := awsmocks.NewMockEc2ApiClient(ctrl)
		client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(&ec2.DescribeSubnetsOutput{
			Subnets: []ec2types.Subnet{{VpcId: awssdk.String("vpc-cluster"), AvailabilityZone: awssdk.String("us-east-1a")}},
		}, nil)
		client.EXPECT().DescribeInstanceTypeOfferings(gomock.Any(), gomock.Any(), gomock.Any()).Return(&ec2.DescribeInstanceTypeOfferingsOutput{}, nil)
		newNodePoolEC2Client = func(config awssdk.Config) awsapi.Ec2ApiClient {
			Expect(config).To(Equal(t.RosaRuntime.AWSConfig))
			return client
		}
		// NodePools.Create has no expectation: reaching it fails this regression test.
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "anything", Replicas: 2, InstanceType: "wrong", Subnet: "subnet-worker",
			}, nil)
		}).To(PanicWith("exit"))
	})

	It("rejects the unsupported version from 56786 before AWS lookup or node pool creation", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		cluster.Status.Version = "5.0.0-ec.6"
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		t.RosaRuntime.HyperFleetClient = hf
		newNodePoolEC2Client = func(_ awssdk.Config) awsapi.Ec2ApiClient {
			Fail("version validation must precede AWS lookup")
			return nil
		}
		// No NodePools.Create expectation: duplicate names cannot mask this validation.
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "anything", Replicas: 2, Version: "4.12.1", Subnet: "subnet-worker",
			}, nil)
		}).To(PanicWith("exit"))
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{Name: "my-np"}, nil)
		}).To(Panic())
	})

	It("fails when cluster get fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		// Get is called in PostExpand (after subnet validation passes).
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(nil, fmt.Errorf("get failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "my-np", Subnet: "subnet-abc123",
			}, nil)
		}).To(Panic())
	})

	It("fails when instance profile cannot be derived (no AWS platform)", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: "cluster-uid"},
			Spec:       v1alpha1.ClusterSpec{},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "my-np", Subnet: "subnet-abc123",
			}, nil)
		}).To(Panic())
	})

	It("fails when subnet is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, nodePools := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.NodePoolList{}, nil)
		// Get is not called when no subnet can be selected: PreRequest fails before PostExpand.

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name: "my-np",
			}, nil)
		}).To(Panic())
	})

	It("fails when --disk-size is invalid", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() {
			runHyperfleetCreate(t.RosaRuntime, &mpOpts.CreateMachinepoolUserOptions{
				Name:         "my-np",
				Subnet:       "subnet-abc123",
				RootDiskSize: "invalid",
			}, nil)
		}).To(Panic())
	})

	It("returns error when --disk-size and --size conflict", func() {
		h := &hyperfleetNodePoolCreate{
			userOptions: &mpOpts.CreateMachinepoolUserOptions{
				RootDiskSize: "75GiB",
			},
			clusterKey: "cluster1",
			clusterUID: "cluster-uid",
		}
		obj := &v1alpha1.NodePool{
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Platform: v1alpha1.NodePoolPlatform{
						AWS: &hypershiftv1beta1.AWSNodePoolPlatform{
							RootVolume: &hypershiftv1beta1.Volume{Size: 50},
						},
					},
				},
			},
		}
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		t.RosaRuntime.HyperFleetClient = hf

		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).To(MatchError(ContainSubstring("--disk-size and --size cannot be used together")))
	})

	It("bridges --additional-security-group-ids into AWS.SecurityGroups", func() {
		h := &hyperfleetNodePoolCreate{
			userOptions: &mpOpts.CreateMachinepoolUserOptions{
				SecurityGroupIds: []string{"sg-1", "sg-2"},
			},
			clusterKey: "cluster1",
			clusterUID: "cluster-uid",
		}
		obj := &v1alpha1.NodePool{Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
			Platform: v1alpha1.NodePoolPlatform{AWS: &hypershiftv1beta1.AWSNodePoolPlatform{
				InstanceType: "m5.xlarge", Subnet: hypershiftv1beta1.AWSResourceReference{ID: awssdk.String("subnet-abc123")},
			}},
		}}}

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		t.RosaRuntime.HyperFleetClient = hf

		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).NotTo(HaveOccurred())

		sg1, sg2 := "sg-1", "sg-2"
		Expect(obj.Spec.NodePool.Platform.AWS.SecurityGroups).To(Equal(
			[]hypershiftv1beta1.AWSResourceReference{{ID: &sg1}, {ID: &sg2}},
		))
	})

	It("leaves AWS.SecurityGroups unset when no security group IDs are supplied", func() {
		h := &hyperfleetNodePoolCreate{
			userOptions: &mpOpts.CreateMachinepoolUserOptions{},
			clusterKey:  "cluster1",
			clusterUID:  "cluster-uid",
		}
		obj := &v1alpha1.NodePool{Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
			Platform: v1alpha1.NodePoolPlatform{AWS: &hypershiftv1beta1.AWSNodePoolPlatform{
				InstanceType: "m5.xlarge", Subnet: hypershiftv1beta1.AWSResourceReference{ID: awssdk.String("subnet-abc123")},
			}},
		}}}

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		cluster := makeCluster("cluster1", "cluster-uid")
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		t.RosaRuntime.HyperFleetClient = hf

		err := h.PostExpand(context.Background(), t.RosaRuntime, nil, obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(obj.Spec.NodePool.Platform.AWS.SecurityGroups).To(BeEmpty())
	})

	It("returns an error when too many AWS tags are supplied", func() {
		tags := make([]string, maxAWSResourceTags+1)
		for i := range tags {
			tags[i] = fmt.Sprintf("tag%d:value", i)
		}
		h := &hyperfleetNodePoolCreate{
			userOptions: &mpOpts.CreateMachinepoolUserOptions{Tags: tags},
		}

		err := h.validateAndParseUserOptions()
		Expect(err).To(MatchError("Invalid machine pool AWS tags: Resource has too many AWS tags"))
	})
})

var _ = Describe("HyperFleet nodepool metadata creation", func() {
	BeforeEach(mockMachinePoolAWSValidation)
	It("serializes customer labels and taints in their mutable API fields", func() {
		t := test.NewTestRuntime()
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters, _ := newCreateMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(makeCluster("cluster1", "cluster-uid"), nil)
		h := &hyperfleetNodePoolCreate{clusterUID: "cluster-uid", userOptions: &mpOpts.CreateMachinepoolUserOptions{
			Labels: "team=test,empty=", Taints: "team=test:NoSchedule,empty=:NoExecute",
		}}
		pool := &v1alpha1.NodePool{Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
			Platform: v1alpha1.NodePoolPlatform{AWS: &hypershiftv1beta1.AWSNodePoolPlatform{
				InstanceType: "m5.xlarge", Subnet: hypershiftv1beta1.AWSResourceReference{ID: awssdk.String("subnet-abc123")},
			}},
		}}}
		Expect(h.PostExpand(context.Background(), t.RosaRuntime, &hfNodePoolInput, pool)).To(Succeed())
		Expect(pool.Spec.Labels).To(Equal(map[string]string{"team": "test", "empty": ""}))
		Expect(pool.Spec.NodePool.NodeLabels).To(BeEmpty())
		data, err := json.Marshal(pool)
		Expect(err).NotTo(HaveOccurred())
		var body map[string]any
		Expect(json.Unmarshal(data, &body)).To(Succeed())
		spec := body["spec"].(map[string]any)
		Expect(spec["labels"]).To(Equal(map[string]any{"team": "test", "empty": ""}))
		Expect(spec["nodePool"].(map[string]any)["taints"]).To(ConsistOf(
			map[string]any{"key": "team", "value": "test", "effect": "NoSchedule"},
			map[string]any{"key": "empty", "effect": "NoExecute"},
		))
	})
})

func mockMachinePoolAWSValidation() {
	original := newNodePoolEC2Client
	DeferCleanup(func() { newNodePoolEC2Client = original })
	client := awsmocks.NewMockEc2ApiClient(gomock.NewController(GinkgoT()))
	client.EXPECT().DescribeSubnets(gomock.Any(), gomock.Any()).Return(&ec2.DescribeSubnetsOutput{
		Subnets: []ec2types.Subnet{{VpcId: awssdk.String("vpc-cluster"), AvailabilityZone: awssdk.String("us-east-1a")}},
	}, nil).AnyTimes()
	client.EXPECT().DescribeInstanceTypeOfferings(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, input *ec2.DescribeInstanceTypeOfferingsInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error) {
			return &ec2.DescribeInstanceTypeOfferingsOutput{InstanceTypeOfferings: []ec2types.InstanceTypeOffering{{
				InstanceType: ec2types.InstanceType(input.Filters[1].Values[0]), Location: awssdk.String("us-east-1a"),
			}}}, nil
		}).AnyTimes()
	newNodePoolEC2Client = func(awssdk.Config) awsapi.Ec2ApiClient { return client }
}
