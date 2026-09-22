package cluster

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled         func() bool
		origDescribeCluster func(*cobra.Command, []string)
	)

	BeforeEach(func() {
		origEnabled = hfEnabled
		origDescribeCluster = hfDescribeCluster
	})

	AfterEach(func() {
		hfEnabled = origEnabled
		hfDescribeCluster = origDescribeCluster
	})

	It("routes to hfDescribeCluster when hyperfleet is enabled", func() {
		called := false
		hfEnabled = func() bool { return true }
		hfDescribeCluster = func(_ *cobra.Command, _ []string) { called = true }

		run(Cmd, nil)

		Expect(called).To(BeTrue())
	})
})

var _ = Describe("hfClusterToMap", func() {
	buildCluster := func() *v1alpha1.Cluster {
		subnetID := "subnet-abc123"
		expiry := metav1.NewTime(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
		return &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-cluster",
				UID:  types.UID("cluster-uid-123"),
			},
			Spec: v1alpha1.ClusterSpec{
				ExpirationTimestamp: &expiry,
				HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
					IssuerURL: "https://oidc.example.com/issuer",
					Platform: v1alpha1.PlatformSpec{
						AWS: &hypershiftv1beta1.AWSPlatformSpec{
							Region: "us-east-1",
							RolesRef: hypershiftv1beta1.AWSRolesRef{
								IngressARN:              "arn:aws:iam::123456789012:role/my-cluster-ingress",
								ImageRegistryARN:        "arn:aws:iam::123456789012:role/my-cluster-image-registry",
								StorageARN:              "arn:aws:iam::123456789012:role/my-cluster-ebs-csi",
								NetworkARN:              "arn:aws:iam::123456789012:role/my-cluster-network-config",
								KubeCloudControllerARN:  "arn:aws:iam::123456789012:role/my-cluster-cloud-controller-manager",
								ControlPlaneOperatorARN: "arn:aws:iam::123456789012:role/my-cluster-control-plane-operator",
								NodePoolManagementARN:   "arn:aws:iam::123456789012:role/my-cluster-node-pool-management",
							},
							CloudProviderConfig: &hypershiftv1beta1.AWSCloudProviderConfig{
								VPC: "vpc-def456",
								Subnet: &hypershiftv1beta1.AWSResourceReference{
									ID: &subnetID,
								},
							},
						},
					},
				},
			},
			Status: v1alpha1.ClusterStatus{
				Phase:   v1alpha1.ClusterPhaseReady,
				Version: "4.17.0",
				ControlPlaneEndpoint: hypershiftv1beta1.APIEndpoint{
					Host: "api.my-cluster.example.com",
					Port: 6443,
				},
				PlacementRef: &v1alpha1.PlacementReference{
					ManagementCluster: "mc01",
				},
				Conditions: []metav1.Condition{
					{
						Type:    "Available",
						Status:  "True",
						Reason:  "AsExpected",
						Message: "All good",
					},
				},
			},
		}
	}

	It("maps all core fields", func() {
		c := buildCluster()
		m := hfClusterToMap(c, nil, nil, "")

		Expect(m["id"]).To(Equal("cluster-uid-123"))
		Expect(m["name"]).To(Equal("my-cluster"))
		Expect(m["control_plane"]).To(Equal("ROSA Service Hosted"))
		Expect(m["state"]).To(Equal("ready"))
		version, ok := m["version"].(map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(version["raw_id"]).To(Equal("4.17.0"))
		Expect(m["region"]).To(Equal("us-east-1"))
		Expect(m["vpc"]).To(Equal("vpc-def456"))
		Expect(m["subnet"]).To(Equal("subnet-abc123"))
		Expect(m["api_url"]).To(Equal("https://api.my-cluster.example.com:6443"))
		Expect(m["management_cluster"]).To(Equal("mc01"))
	})

	It("maps spec fields", func() {
		c := buildCluster()
		m := hfClusterToMap(c, nil, nil, "")

		spec, ok := m["spec"].(map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(spec["oidc_issuer"]).To(Equal("https://oidc.example.com/issuer"))

		roles, ok := spec["roles_ref"].(map[string]string)
		Expect(ok).To(BeTrue())
		Expect(roles["ingressARN"]).To(Equal("arn:aws:iam::123456789012:role/my-cluster-ingress"))
		Expect(roles["storageARN"]).To(Equal("arn:aws:iam::123456789012:role/my-cluster-ebs-csi"))
	})

	It("maps conditions", func() {
		c := buildCluster()
		m := hfClusterToMap(c, nil, nil, "")

		conds, ok := m["conditions"].([]map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(conds).To(HaveLen(1))
		Expect(conds[0]["type"]).To(Equal("Available"))
		Expect(conds[0]["status"]).To(Equal("True"))
	})

	It("includes expiration when set", func() {
		c := buildCluster()
		m := hfClusterToMap(c, nil, nil, "")
		Expect(m).To(HaveKey("expiration"))
	})

	It("handles nil AWS spec gracefully", func() {
		c := buildCluster()
		c.Spec.HostedCluster.Platform.AWS = nil
		m := hfClusterToMap(c, nil, nil, "")
		Expect(m).NotTo(HaveKey("region"))
		Expect(m).NotTo(HaveKey("vpc"))
		Expect(m).NotTo(HaveKey("subnet"))

		spec := m["spec"].(map[string]interface{})
		roles := spec["roles_ref"].(map[string]string)
		Expect(roles).To(BeEmpty())
	})

	It("maps nodes.compute_machine_type from default node pool instance type", func() {
		c := buildCluster()
		m := hfClusterToMap(c, nil, nil, "m5.xlarge")
		nodes, ok := m["nodes"].(map[string]interface{})
		Expect(ok).To(BeTrue())
		cmt, ok := nodes["compute_machine_type"].(map[string]interface{})
		Expect(ok).To(BeTrue())
		Expect(cmt["id"]).To(Equal("m5.xlarge"))
	})

	It("omits EC2 metadata tokens when the API omits them", func() {
		c := buildCluster()
		m := hfClusterToMap(c, nil, nil, "")
		aws := m["aws"].(map[string]interface{})
		Expect(aws).NotTo(HaveKey("ec2_metadata_http_tokens"))
		Expect(hfClusterToString(c, nil, nil)).NotTo(ContainSubstring("EC2 Metadata Http Tokens:"))
	})
})

var _ = Describe("hfDefaultNodePoolInstanceTypeFromList", func() {
	It("does not treat a non-worker pool as the default", func() {
		list := &v1alpha1.NodePoolList{Items: []v1alpha1.NodePool{{
			ObjectMeta: metav1.ObjectMeta{Name: "custom"},
			Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
				Platform: v1alpha1.NodePoolPlatform{
					AWS: &hypershiftv1beta1.AWSNodePoolPlatform{InstanceType: "m7i.xlarge"},
				},
			}},
		}}}
		Expect(hfDefaultNodePoolInstanceTypeFromList(list)).To(BeEmpty())
	})
})

var _ = Describe("hfClusterToString", func() {
	It("lists operator role ARNs like OCM describe output", func() {
		arn := "arn:aws:iam::123456789012:role/my-cluster-ingress"
		out := hfClusterToString(&v1alpha1.Cluster{
			Spec: v1alpha1.ClusterSpec{
				HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
					Platform: v1alpha1.PlatformSpec{
						AWS: &hypershiftv1beta1.AWSPlatformSpec{
							RolesRef: hypershiftv1beta1.AWSRolesRef{IngressARN: arn},
						},
					},
				},
			},
		}, nil, nil)
		Expect(out).To(ContainSubstring("Operator IAM Roles:\n - " + arn + "\n"))
		Expect(out).NotTo(ContainSubstring("Ingress:"))
	})
})
