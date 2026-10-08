package hyperfleet

import (
	"context"
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
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/transport"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
)

func metadataTestClient(handler http.HandlerFunc) (*hfmocks.MockInterface, *hfmocks.MockNodePoolInterface) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer GinkgoRecover()
		w.Header().Set("Content-Type", "application/json")
		handler(w, req)
	}))
	DeferCleanup(server.Close)
	baseURL, err := url.Parse(server.URL)
	Expect(err).NotTo(HaveOccurred())
	httpClient := server.Client()
	httpClient.Transport = transport.NewAdapter(httpClient.Transport)
	config := rest.ClientContentConfig{Negotiator: runtime.NewClientNegotiator(hfscheme.Codecs.WithoutConversion(), schema.GroupVersion{Version: "v1"})}
	client, err := rest.NewRESTClient(baseURL, "", config, nil, httpClient)
	Expect(err).NotTo(HaveOccurred())
	ctrl := gomock.NewController(GinkgoT())
	hf, v1, pools := hfmocks.NewMockInterface(ctrl), hfmocks.NewMockV1alpha1PublicInterface(ctrl), hfmocks.NewMockNodePoolInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().NodePools("cluster-cluster-uid").Return(pools).AnyTimes()
	v1.EXPECT().RESTClient().Return(client).AnyTimes()
	return hf, pools
}

var _ = Describe("Nodepool metadata updates", func() {
	DescribeTable("preserves request fields and explicit clearing", func(labelsChanged, taintsChanged, empty bool) {
		replicas := int32(3)
		pool := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "workers", UID: types.UID("pool-uid")}, Spec: v1alpha1.NodePoolSpec{
			DisplayName: "Workers", Labels: map[string]string{"team": "test"},
			NodePool: v1alpha1.NodePoolSpecPassthrough{Replicas: &replicas, Taints: []hypershiftv1beta1.Taint{{Key: "team", Value: "test", Effect: "NoSchedule"}}},
		}}
		if empty {
			pool.Spec.Labels = map[string]string{}
			pool.Spec.NodePool.Taints = nil
		}
		original := pool.DeepCopy()
		data, err := json.Marshal(pool)
		Expect(err).NotTo(HaveOccurred())
		expected := map[string]any{}
		Expect(json.Unmarshal(data, &expected)).To(Succeed())
		if empty {
			if labelsChanged {
				expected["spec"].(map[string]any)["labels"] = nil
			}
			if taintsChanged {
				expected["spec"].(map[string]any)["nodePool"].(map[string]any)["taints"] = nil
			}
		}
		hf, _ := metadataTestClient(func(w http.ResponseWriter, req *http.Request) {
			Expect(req.Method).To(Equal(http.MethodPut))
			Expect(req.URL.Path).To(Equal("/nodepools/pool-uid"))
			Expect(req.URL.Query().Get("clusterId")).To(Equal("cluster-uid"))
			body := map[string]any{}
			Expect(json.NewDecoder(req.Body).Decode(&body)).To(Succeed())
			Expect(body).To(Equal(expected))
			_, err := w.Write([]byte(`{"metadata":{"name":"workers","uid":"pool-uid"}}`))
			Expect(err).NotTo(HaveOccurred())
		})
		updated, err := WithNodePoolMetadataUpdates(hf, labelsChanged, taintsChanged).HyperfleetV1alpha1().NodePools("cluster-cluster-uid").Update(context.Background(), pool, platform.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Name).To(Equal("workers"))
		Expect(pool).To(Equal(original))
	},
		Entry("both fields set", true, true, false), Entry("labels only", true, false, false),
		Entry("taints only", false, true, false), Entry("clear both", true, true, true),
		Entry("clear labels without clearing taints", true, false, true), Entry("clear taints without clearing labels", false, true, true),
	)
	It("delegates reads to the SDK", func() {
		hf, pools := metadataTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected HTTP request") })
		pools.EXPECT().Get(gomock.Any(), "pool-uid", platform.GetOptions{}).Return(&v1alpha1.NodePool{}, nil)
		_, err := WithNodePoolMetadataUpdates(hf, true, true).HyperfleetV1alpha1().NodePools("cluster-cluster-uid").Get(context.Background(), "pool-uid", platform.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
	DescribeTable("rejects pools without a UID", func(pool *v1alpha1.NodePool) {
		hf, _ := metadataTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected HTTP request") })
		_, err := WithNodePoolMetadataUpdates(hf, true, true).HyperfleetV1alpha1().NodePools("cluster-cluster-uid").Update(context.Background(), pool, platform.UpdateOptions{})
		Expect(err).To(MatchError("node pool UID is required"))
	}, Entry("nil pool", (*v1alpha1.NodePool)(nil)), Entry("empty UID", &v1alpha1.NodePool{}))
	It("preserves API error details", func() {
		hf, _ := metadataTestClient(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"invalid labels","code":400}`))
			Expect(err).NotTo(HaveOccurred())
		})
		_, err := WithNodePoolMetadataUpdates(hf, true, false).HyperfleetV1alpha1().NodePools("cluster-cluster-uid").Update(context.Background(), &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{UID: "pool-uid"}}, platform.UpdateOptions{})
		Expect(err).To(MatchError("invalid labels"))
	})
	It("reports malformed API responses", func() {
		hf, _ := metadataTestClient(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte("{"))
			Expect(err).NotTo(HaveOccurred())
		})
		_, err := WithNodePoolMetadataUpdates(hf, true, false).HyperfleetV1alpha1().NodePools("cluster-cluster-uid").Update(context.Background(), &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{UID: "pool-uid"}}, platform.UpdateOptions{})
		Expect(err).To(MatchError(ContainSubstring("failed to decode node pool update")))
	})
})
