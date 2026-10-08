package cluster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hfscheme "github.com/openshift-online/rosa-hyperfleet-api/clientset/generated/scheme"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

func newDescribeClusterMocks(ctrl *gomock.Controller, clusterUID string) (*hfmocks.MockInterface, *hfmocks.MockClusterInterface) {
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	if clusterUID != "" {
		nodePools := hfmocks.NewMockNodePoolInterface(ctrl)
		v1.EXPECT().NodePools(clusterUID).Return(nodePools).AnyTimes()
		nodePools.EXPECT().List(gomock.Any(), gomock.Any()).
			Return(&v1alpha1.NodePoolList{}, nil).AnyTimes()
	}
	return hf, clusters
}

func expectDescribeClusterGet(hf *hfmocks.MockInterface, response any, statusCode int) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer GinkgoRecover()
		Expect(req.Method).To(Equal(http.MethodGet))
		Expect(req.URL.Path).To(Equal("/clusters/cluster-uid"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		Expect(json.NewEncoder(w).Encode(response)).To(Succeed())
	}))
	DeferCleanup(server.Close)
	baseURL, err := url.Parse(server.URL)
	Expect(err).NotTo(HaveOccurred())
	client, err := rest.NewRESTClient(baseURL, "", rest.ClientContentConfig{Negotiator: runtime.NewClientNegotiator(hfscheme.Codecs.WithoutConversion(), schema.GroupVersion{Version: "v1"})}, nil, server.Client())
	Expect(err).NotTo(HaveOccurred())
	v1 := hf.HyperfleetV1alpha1().(*hfmocks.MockV1alpha1PublicInterface)
	v1.EXPECT().RESTClient().Return(client).AnyTimes()
}

var _ = Describe("runHyperfleetDescribe (cluster)", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("describes a cluster on the success path (text output)", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")

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
		expectDescribeClusterGet(hf, cluster, http.StatusOK)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetDescribe(t.RosaRuntime, nil, nil)
	})

	It("resolves cluster key from argv when cluster flag is not set", func() {
		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		expectDescribeClusterGet(hf, cluster, http.StatusOK)

		t.RosaRuntime.HyperFleetClient = hf
		cmd := &cobra.Command{Use: "test"}
		ocm.AddOptionalClusterFlag(cmd)
		runHyperfleetDescribe(t.RosaRuntime, cmd, []string{"cluster1"})
	})

	It("describes a cluster in JSON output format", func() {
		output.SetOutput("json")
		DeferCleanup(func() { output.SetOutput("") })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")

		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		expectDescribeClusterGet(hf, cluster, http.StatusOK)

		t.RosaRuntime.HyperFleetClient = hf
		runHyperfleetDescribe(t.RosaRuntime, nil, nil)
	})

	It("fails when cluster key is not set", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })
		ocm.SetClusterKey("")
		DeferCleanup(func() { ocm.SetClusterKey("cluster1") })

		ctrl := gomock.NewController(GinkgoT())
		hf, _ := newDescribeClusterMocks(ctrl, "")
		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetDescribe(t.RosaRuntime, nil, nil) }).To(Panic())
	})

	It("fails when cluster cannot be resolved", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.ClusterList{}, nil)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetDescribe(t.RosaRuntime, nil, nil) }).To(Panic())
	})

	It("fails when get cluster fails", func() {
		orig := exitFn
		exitFn = func(_ int) { panic("exit") }
		DeferCleanup(func() { exitFn = orig })

		ctrl := gomock.NewController(GinkgoT())
		hf, clusters := newDescribeClusterMocks(ctrl, "cluster-uid")
		cluster := &v1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1", UID: types.UID("cluster-uid")},
		}
		clusters.EXPECT().List(gomock.Any(), gomock.Any()).Return(
			&v1alpha1.ClusterList{Items: []v1alpha1.Cluster{*cluster}}, nil)
		expectDescribeClusterGet(hf, map[string]string{"message": "get failed"}, http.StatusBadRequest)

		t.RosaRuntime.HyperFleetClient = hf
		Expect(func() { runHyperfleetDescribe(t.RosaRuntime, nil, nil) }).To(Panic())
	})
})
