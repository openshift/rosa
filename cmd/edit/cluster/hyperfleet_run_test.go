package cluster

import (
	"context"
	"fmt"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/spf13/cobra"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
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
	})

	AfterEach(func() {
		args.expirationDuration = 0
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
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		clusters.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(cluster, nil)

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
		clusters.EXPECT().Get(gomock.Any(), "cluster-uid", gomock.Any()).Return(cluster, nil)
		clusters.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("update failed"))

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetEdit(t.RosaRuntime, makeExpirationCmd(true)) }).To(Panic())
	})
})
