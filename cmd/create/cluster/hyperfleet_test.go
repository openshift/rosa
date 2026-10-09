package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2svc "github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hfscheme "github.com/openshift-online/rosa-hyperfleet-api/clientset/generated/scheme"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	pkgaws "github.com/openshift/rosa/pkg/aws"
	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled    func() bool
		origRunCluster func(*cobra.Command)
	)

	BeforeEach(func() {
		origEnabled = hyperfleetEnabled
		origRunCluster = hfCreateCluster
	})

	AfterEach(func() {
		hyperfleetEnabled = origEnabled
		hfCreateCluster = origRunCluster
	})

	It("routes to hfCreateCluster when hyperfleet is enabled", func() {
		called := false
		hyperfleetEnabled = func() bool { return true }
		hfCreateCluster = func(*cobra.Command) { called = true }

		dispatch(nil, nil)

		Expect(called).To(BeTrue())
	})
})

var _ = Describe("rejectUnsupportedHyperfleetCreateFlags", func() {
	DescribeTable("checks effective values of explicitly set flags", func(name, value string, boolean, reject bool) {
		cmd := &cobra.Command{}
		if boolean {
			cmd.Flags().Bool(name, name == "sts", "")
		} else {
			cmd.Flags().String(name, "", "")
		}
		Expect(cmd.Flags().Set(name, value)).To(Succeed())
		err := rejectUnsupportedHyperfleetCreateFlags(cmd)
		if reject {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("--" + name))
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("classic boolean enabled", "non-sts", "true", true, true),
		Entry("classic boolean disabled", "non-sts", "false", true, false),
		Entry("STS disabled", "sts", "false", true, true),
		Entry("STS enabled", "sts", "true", true, false),
		Entry("classic string supplied", "controlplane-iam-role-arn", "arn:aws:iam::123456789012:role/control-plane", false, true),
		Entry("classic string unchanged", "controlplane-iam-role-arn", "", false, false),
	)

	It("rejects all explicitly set classic-only flags", func() {
		cmd := &cobra.Command{}
		var disableUWM bool
		var controlPlaneRoleARN string
		cmd.Flags().BoolVar(&disableUWM, "disable-workload-monitoring", false, "")
		cmd.Flags().StringVar(&controlPlaneRoleARN, "controlplane-iam-role-arn", "", "")
		Expect(cmd.Flags().Set("disable-workload-monitoring", "true")).To(Succeed())
		Expect(cmd.Flags().Set(
			"controlplane-iam-role-arn", "arn:aws:iam::123456789012:role/control-plane",
		)).To(Succeed())

		err := rejectUnsupportedHyperfleetCreateFlags(cmd)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not supported for HyperFleet cluster creation"))
		Expect(err.Error()).To(ContainSubstring("--disable-workload-monitoring"))
		Expect(err.Error()).To(ContainSubstring("--controlplane-iam-role-arn"))
	})

	It("allows create when classic-only flags are unset", func() {
		cmd := &cobra.Command{}
		var disableUWM bool
		cmd.Flags().BoolVar(&disableUWM, "disable-workload-monitoring", false, "")
		cmd.Flags().String("controlplane-iam-role-arn", "", "")

		Expect(rejectUnsupportedHyperfleetCreateFlags(cmd)).To(Succeed())
	})

	It("allows nil command", func() {
		Expect(rejectUnsupportedHyperfleetCreateFlags(nil)).To(Succeed())
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
			networkType         string
			noCni               bool
			version             string
			tags                []string
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
		origArgs.networkType = args.networkType
		origArgs.noCni = args.noCni
		origArgs.version = args.version
		origArgs.tags = args.tags

		exited = false
		hfExitFn = func(int) { exited = true }

		args.clusterName = "test-cluster"
		args.operatorRolesPrefix = "test-cluster"
		args.subnetIDs = []string{"subnet-abc123"}
		args.networkType = ""
		args.noCni = false
		args.version = "quay.io/openshift-release-dev/ocp-release:5.0.0-ec.6-multi"
		args.tags = nil
		hfClusterInput.Tags = ""

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
		args.networkType = origArgs.networkType
		args.noCni = origArgs.noCni
		args.version = origArgs.version
		args.tags = origArgs.tags
		hfClusterInput.Tags = ""
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
		Expect(capturedCluster.Spec.HostedCluster.Release.Image).To(Equal(args.version))
		rolesRef := capturedCluster.Spec.HostedCluster.Platform.AWS.RolesRef
		Expect(rolesRef.IngressARN).To(HavePrefix("arn:aws-us-gov:iam::"),
			"role ARNs must use the GovCloud partition")
		Expect(rolesRef.NodePoolManagementARN).To(HavePrefix("arn:aws-us-gov:iam::"),
			"role ARNs must use the GovCloud partition")
	})

	It("sends user defined AWS tags as spec.tags", func() {
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

		args.tags = []string{"cost-center:eng", "owner:platform"}
		stubSubnets("vpc-123", "us-gov-east-1a")
		t.RosaRuntime.HyperFleetClient = hf

		runHyperfleet(t.RosaRuntime)

		Expect(exited).To(BeFalse())
		Expect(capturedCluster).NotTo(BeNil())
		Expect(capturedCluster.Spec.Tags).To(Equal(map[string]string{
			"cost-center": "eng",
			"owner":       "platform",
		}))
	})

	It("leaves spec.tags unset when no tags are supplied", func() {
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
		Expect(capturedCluster.Spec.Tags).To(BeEmpty())
	})

	It("exits when more tags than the Platform API allows are supplied", func() {
		tags := make([]string, maxHyperfleetClusterTags+1)
		for i := range tags {
			tags[i] = fmt.Sprintf("tag%d:value", i)
		}
		args.tags = tags
		stubSubnets("vpc-123", "us-gov-east-1a")

		runHyperfleet(t.RosaRuntime)
		Expect(exited).To(BeTrue())
	})

	It("exits when a tag is malformed", func() {
		args.tags = []string{"no-delimiter"}
		stubSubnets("vpc-123", "us-gov-east-1a")

		runHyperfleet(t.RosaRuntime)
		Expect(exited).To(BeTrue())
	})

	It("exits when cluster name is missing", func() {
		args.clusterName = ""
		runHyperfleet(&rosa.Runtime{Reporter: t.RosaRuntime.Reporter})
		Expect(exited).To(BeTrue())
	})

	It("sends proxy settings in the serialized create request", func() {
		originalArgs := args
		DeferCleanup(func() { args = originalArgs })
		args.httpProxy = "http://10.0.0.226:8080"
		args.httpsProxy = "https://10.0.0.226:8080"
		args.noProxySlice = []string{"quay.io"}

		ctrl := gomock.NewController(GinkgoT())
		hf := hfmocks.NewMockInterface(ctrl)
		v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
		clusters := hfmocks.NewMockClusterInterface(ctrl)
		hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
		v1.EXPECT().Clusters().Return(clusters).AnyTimes()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer GinkgoRecover()
			Expect(req.Method).To(Equal(http.MethodPost))
			Expect(req.URL.Path).To(Equal("/clusters"))
			var request map[string]any
			Expect(json.NewDecoder(req.Body).Decode(&request)).To(Succeed())
			spec := request["spec"].(map[string]any)
			hosted := spec["hostedCluster"].(map[string]any)
			configuration := hosted["configuration"].(map[string]any)
			Expect(configuration["proxy"]).To(Equal(map[string]any{
				"httpProxy":  "http://10.0.0.226:8080",
				"httpsProxy": "https://10.0.0.226:8080",
				"noProxy":    "quay.io",
			}))
			Expect(hosted["platform"].(map[string]any)["type"]).To(Equal("AWS"))
			Expect(request).NotTo(HaveKey("proxy"), "the top-level proxy field is response-only")
			w.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(w).Encode(request)).To(Succeed())
		}))
		DeferCleanup(server.Close)
		baseURL, err := url.Parse(server.URL)
		Expect(err).NotTo(HaveOccurred())
		client, err := rest.NewRESTClient(baseURL, "", rest.ClientContentConfig{Negotiator: runtime.NewClientNegotiator(hfscheme.Codecs.WithoutConversion(), schema.GroupVersion{Version: "v1"})}, nil, server.Client())
		Expect(err).NotTo(HaveOccurred())
		v1.EXPECT().RESTClient().Return(client).AnyTimes()
		stubSubnets("vpc-123", "us-east-1a")
		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleet(t.RosaRuntime)
		Expect(exited).To(BeFalse())
		Expect(t.RosaRuntime.HyperFleetClient).To(BeIdenticalTo(hf))
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

	It("rejects an invalid network type before creating a cluster", func() {
		args.networkType = "invalid-network-type"
		handler := &hyperfleetClusterCreate{}
		err := handler.PreRequest(context.Background(), t.RosaRuntime, &hfpathbind.ClusterCreateInput{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("expected a valid network type"))
	})

	It("rejects no CNI together with a network type before creating a cluster", func() {
		args.noCni = true
		args.networkType = "OVNKubernetes"
		handler := &hyperfleetClusterCreate{}
		err := handler.PreRequest(context.Background(), t.RosaRuntime, &hfpathbind.ClusterCreateInput{})
		Expect(err).To(MatchError("--no-cni and --network-type are mutually exclusive parameters"))
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

var _ = Describe("parseHyperfleetClusterTags", func() {
	It("returns nil for no tags", func() {
		tags, err := parseHyperfleetClusterTags(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(tags).To(BeNil())
	})

	It("parses colon delimited tags", func() {
		tags, err := parseHyperfleetClusterTags([]string{"cost-center:eng", "owner:platform"})
		Expect(err).NotTo(HaveOccurred())
		Expect(tags).To(Equal(map[string]string{"cost-center": "eng", "owner": "platform"}))
	})

	It("parses space delimited tags", func() {
		tags, err := parseHyperfleetClusterTags([]string{"cost-center eng", "owner platform"})
		Expect(err).NotTo(HaveOccurred())
		Expect(tags).To(Equal(map[string]string{"cost-center": "eng", "owner": "platform"}))
	})

	It("accepts the maximum supported number of tags", func() {
		input := make([]string, maxHyperfleetClusterTags)
		for i := range input {
			input[i] = fmt.Sprintf("tag%d:value", i)
		}
		tags, err := parseHyperfleetClusterTags(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(tags).To(HaveLen(maxHyperfleetClusterTags))
	})

	It("rejects more tags than the Platform API allows", func() {
		input := make([]string, maxHyperfleetClusterTags+1)
		for i := range input {
			input[i] = fmt.Sprintf("tag%d:value", i)
		}
		_, err := parseHyperfleetClusterTags(input)
		Expect(err).To(MatchError(ContainSubstring("a maximum of 23 tags is supported")))
	})

	It("rejects duplicate tag keys", func() {
		_, err := parseHyperfleetClusterTags([]string{"owner:a", "owner:b"})
		Expect(err).To(MatchError(ContainSubstring("user tag keys must be unique")))
	})

	It("rejects a tag without a delimiter", func() {
		_, err := parseHyperfleetClusterTags([]string{"no-delimiter"})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Platform API custom DNS cluster inputs", func() {
	DescribeTable("maps existing DNS flags into the cluster specification", func(baseDomain, privateZone string) {
		originalDomain, originalZone := args.baseDomain, args.privateHostedZoneID
		DeferCleanup(func() { args.baseDomain, args.privateHostedZoneID = originalDomain, originalZone })
		args.baseDomain, args.privateHostedZoneID = baseDomain, privateZone
		r := rosa.NewRuntime()
		r.Creator = &pkgaws.Creator{AccountID: "123456789012", Partition: "aws"}
		obj := &v1alpha1.Cluster{}
		obj.Spec.HostedCluster.Platform.AWS = &hypershiftv1beta1.AWSPlatformSpec{}
		handler := &hyperfleetClusterCreate{}
		Expect(handler.PostExpand(context.Background(), r, &hfpathbind.ClusterCreateInput{OperatorRolesPrefix: "test"}, obj)).To(Succeed())
		Expect(obj.Spec.HostedCluster.DNS.BaseDomain).To(Equal(strings.TrimSpace(baseDomain)))
		Expect(obj.Spec.HostedCluster.DNS.PrivateZoneID).To(Equal(privateZone))
		Expect(obj.Spec.HostedCluster.DNS.BaseDomainPrefix).To(BeNil())
		Expect(obj.Spec.HostedCluster.DNS.PublicZoneID).To(BeEmpty())
	}, Entry("reserved custom domain", " abcd.0.example.com ", "Z012345"), Entry("service-assigned domain", "", ""))
})
