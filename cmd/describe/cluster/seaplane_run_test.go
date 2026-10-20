package cluster

import (
	"errors"
	"net/http"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	seaplanemocks "github.com/openshift/rosa/pkg/seaplane/mocks"
	"github.com/openshift/rosa/pkg/test"
)

func newDescribeClusterMocks(ctrl *gomock.Controller, _ string) (*seaplanemocks.MockInterface, *seaplanemocks.MockClusterInterface) {
	seaplaneClient := seaplanemocks.NewMockInterface(ctrl)
	v1 := seaplanemocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := seaplanemocks.NewMockClusterInterface(ctrl)
	seaplaneClient.EXPECT().SeaplaneV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	nodePools := seaplanemocks.NewMockNodePoolInterface(ctrl)
	v1.EXPECT().NodePools().Return(nodePools).AnyTimes()
	nodePools.EXPECT().List(gomock.Any(), gomock.Any()).
		Return(&v1alpha1.NodePoolList{}, nil).AnyTimes()
	return seaplaneClient, clusters
}

func expectDescribeClusterGet(clusters *seaplanemocks.MockClusterInterface, response any, statusCode int) {
	if statusCode != http.StatusOK {
		clusters.EXPECT().Get(gomock.Any(), "cluster1", platform.GetOptions{}).Return(nil, errors.New("get failed"))
		return
	}
	cluster, ok := response.(*v1alpha1.Cluster)
	Expect(ok).To(BeTrue())
	clusters.EXPECT().Get(gomock.Any(), cluster.Name, platform.GetOptions{}).Return(cluster, nil)
}

var _ = Describe("runSeaplaneDescribe (cluster)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("describes a cluster on the success path (text output)", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
			Spec: v1alpha1.ClusterSpec{
				HostedCluster: v1alpha1.HostedClusterSpecPassthrough{
					IssuerURL: "https://oidc.example.com",
					Platform: v1alpha1.PlatformSpec{
						AWS: &hypershiftv1beta1.AWSPlatformSpec{Region: "us-east-1"},
					},
				},
			},
			Status: v1alpha1.ClusterStatus{Phase: v1alpha1.ClusterPhaseReady, Version: "4.17.0"},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		expectDescribeClusterGet(clusters, cluster, http.StatusOK)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneDescribe(t.RosaRuntime, nil, nil)
	})

	It("resolves cluster key from argv when cluster flag is not set", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		expectDescribeClusterGet(clusters, cluster, http.StatusOK)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		cmd := &cobra.Command{Use: "test"}
		ocm.AddOptionalClusterFlag(cmd)
		runSeaplaneDescribe(t.RosaRuntime, cmd, []string{"cluster1"})
	})

	It("describes a cluster in JSON output format", func() {
		output.SetOutput("json")
		DeferCleanup(func() { output.SetOutput("") })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		expectDescribeClusterGet(clusters, cluster, http.StatusOK)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		runSeaplaneDescribe(t.RosaRuntime, nil, nil)
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _ := newDescribeClusterMocks(ctrl, "")
		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneDescribe(t.RosaRuntime, nil, nil) }).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneDescribe(t.RosaRuntime, nil, nil) }).To(Panic())
	})

	It("fails when get cluster fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")
		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		expectDescribeClusterGet(clusters, map[string]string{"message": "get failed"}, http.StatusBadRequest)

		t.RosaRuntime.SeaplaneClient = seaplaneClient
		Expect(func() { runSeaplaneDescribe(t.RosaRuntime, nil, nil) }).To(Panic())
	})
})
