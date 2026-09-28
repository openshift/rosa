package cluster

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/spf13/cobra"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
)

func newEditClusterMocks(ctrl *gomock.Controller) (*hfmocks.MockInterface, *hfmocks.MockClusterInterface) {
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	return hf, clusters
}

// makeExpirationCmd builds a minimal cobra command with the expiration flag wired
// to args.expirationDuration and optionally set to a value.
func makeExpirationCmd(setExpiration bool) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.Flags().DurationVar(&args.expirationDuration, "expiration", 0, "")
	if setExpiration {
		if err := cmd.Flags().Set("expiration", "1h"); err != nil {
			panic(err)
		}
	}
	return cmd
}

var _ = Describe("runHyperfleetEdit (cluster)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		hfClusterUpdateInput = hfpathbindZeroInput()
	})

	AfterEach(func() {
		args.expirationDuration = 0
		args.channelGroup = ""
		args.channel = ""
		hfClusterUpdateInput = hfpathbindZeroInput()
	})

	It("updates cluster expiration on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newEditClusterMocks(ctrl)

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			Spec:       v1alpha1.ClusterSpec{HostedCluster: v1alpha1.HostedClusterSpecPassthrough{}},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster-uid", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ interface{}) (*v1alpha1.Cluster, error) {
				var body map[string]any
				Expect(json.Unmarshal(data, &body)).To(Succeed())
				spec := body["spec"].(map[string]any)
				Expect(spec).To(HaveKey("expirationTimestamp"))
				Expect(spec).NotTo(HaveKey("hostedCluster"))
				Expect(spec).NotTo(HaveKey("oidcConfigId"))
				return cluster, nil
			})

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, makeExpirationCmd(true))
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _ := newEditClusterMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetEdit(t.RosaRuntime, makeExpirationCmd(true)) }).To(Panic())
	})

	It("fails when no supported flags are changed", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newEditClusterMocks(ctrl)
		// ResolveClusterUID is called before PreRequest validates flags.
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}}}, nil)
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetEdit(t.RosaRuntime, makeExpirationCmd(false)) }).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newEditClusterMocks(ctrl)
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetEdit(t.RosaRuntime, makeExpirationCmd(true)) }).To(Panic())
	})

	It("fails when cluster update fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newEditClusterMocks(ctrl)

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster-uid", types.MergePatchType, gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("update failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetEdit(t.RosaRuntime, makeExpirationCmd(true)) }).To(Panic())
	})

	It("updates cluster channel group on the success path", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newEditClusterMocks(ctrl)

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			Spec:       v1alpha1.ClusterSpec{HostedCluster: v1alpha1.HostedClusterSpecPassthrough{}},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		clusters.EXPECT().Patch(gomock.Any(), "cluster-uid", types.MergePatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ interface{}) (*v1alpha1.Cluster, error) {
				var body map[string]any
				Expect(json.Unmarshal(data, &body)).To(Succeed())
				spec := body["spec"].(map[string]any)
				hc := spec["hostedCluster"].(map[string]any)
				Expect(hc["channel"]).To(Equal("candidate"))
				props := spec["properties"].(map[string]any)
				Expect(props["channel_group"]).To(Equal("candidate"))
				Expect(spec).NotTo(HaveKey("oidcConfigId"))
				return cluster, nil
			})

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetEdit(t.RosaRuntime, makeChannelGroupCmd("candidate"))
	})

	It("rejects unsupported channel group", func() {
		Expect(validateHyperfleetChannelArgsFor("fakecg")).To(MatchError(ContainSubstring("Unsupported channel group")))
	})

	It("rejects nightly channel group without a version catalog", func() {
		Expect(validateHyperfleetChannelArgsFor("nightly")).To(MatchError(
			ContainSubstring("is not available for the desired channel group")))
	})
})

func makeChannelGroupCmd(group string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.Flags().StringVar(&args.channelGroup, "channel-group", "", "")
	if err := cmd.Flags().Set("channel-group", group); err != nil {
		panic(err)
	}
	return cmd
}

func validateHyperfleetChannelArgsFor(group string) error {
	orig := args.channelGroup
	args.channelGroup = group
	defer func() { args.channelGroup = orig }()
	return validateHyperfleetChannelArgs()
}

func hfpathbindZeroInput() hfpathbind.ClusterUpdateInput {
	return hfpathbind.ClusterUpdateInput{}
}
