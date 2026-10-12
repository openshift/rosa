package hyperfleet

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
)

var _ = Describe("WithNodePoolUpgrade", func() {
	It("adds the rolling update settings while preserving the SDK request and caller's object", func() {
		replicas := int32(3)
		pool := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "workers"},
			Spec: v1alpha1.NodePoolSpec{
				DisplayName: "Workers",
				NodePool:    v1alpha1.NodePoolSpecPassthrough{ClusterName: "cluster1", Replicas: &replicas},
			},
		}
		original := pool.DeepCopy()
		data, err := json.Marshal(pool)
		Expect(err).NotTo(HaveOccurred())
		expected := map[string]any{}
		Expect(json.Unmarshal(data, &expected)).To(Succeed())

		hf, _ := nodePoolUpdateTestClient(func(w http.ResponseWriter, req *http.Request) {
			Expect(req.Method).To(Equal(http.MethodPost))
			Expect(req.URL.Path).To(Equal("/nodepools"))
			Expect(req.URL.Query().Get("clusterId")).To(Equal("cluster-uid"))
			body := map[string]any{}
			Expect(json.NewDecoder(req.Body).Decode(&body)).To(Succeed())
			spec := body["spec"].(map[string]any)
			Expect(spec["maxSurge"]).To(Equal("2"))
			Expect(spec["maxUnavailable"]).To(Equal("20%"))
			delete(spec, "maxSurge")
			delete(spec, "maxUnavailable")
			Expect(body).To(Equal(expected))
			_, err := w.Write([]byte(`{"metadata":{"name":"workers","uid":"pool-uid"}}`))
			Expect(err).NotTo(HaveOccurred())
		})

		created, err := WithNodePoolUpgrade(hf, &NodePoolUpgrade{MaxSurge: "2", MaxUnavailable: "20%"}).
			HyperfleetV1alpha1().NodePools("cluster-cluster-uid").
			Create(context.Background(), pool, platform.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(string(created.UID)).To(Equal("pool-uid"))
		Expect(pool).To(Equal(original))
	})

	It("delegates non-create operations to the original SDK client", func() {
		hf, pools := nodePoolUpdateTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected HTTP request") })
		pools.EXPECT().Get(gomock.Any(), "pool-uid", platform.GetOptions{}).Return(&v1alpha1.NodePool{}, nil)
		_, err := WithNodePoolUpgrade(hf, &NodePoolUpgrade{MaxSurge: "1"}).HyperfleetV1alpha1().
			NodePools("cluster-cluster-uid").Get(context.Background(), "pool-uid", platform.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects missing node pools before sending a request", func() {
		hf, _ := nodePoolUpdateTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected HTTP request") })
		_, err := WithNodePoolUpgrade(hf, &NodePoolUpgrade{MaxSurge: "1"}).HyperfleetV1alpha1().
			NodePools("cluster-cluster-uid").Create(context.Background(), nil, platform.CreateOptions{})
		Expect(err).To(MatchError("node pool is required"))
	})

	It("preserves API error details", func() {
		hf, _ := nodePoolUpdateTestClient(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(
				`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"invalid max surge","code":400}`))
			Expect(err).NotTo(HaveOccurred())
		})
		_, err := WithNodePoolUpgrade(hf, &NodePoolUpgrade{MaxSurge: "1"}).HyperfleetV1alpha1().
			NodePools("cluster-cluster-uid").
			Create(context.Background(), &v1alpha1.NodePool{}, platform.CreateOptions{})
		Expect(err).To(MatchError("invalid max surge"))
	})

	It("reports malformed API responses", func() {
		hf, _ := nodePoolUpdateTestClient(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte("{"))
			Expect(err).NotTo(HaveOccurred())
		})
		_, err := WithNodePoolUpgrade(hf, &NodePoolUpgrade{MaxSurge: "1"}).HyperfleetV1alpha1().
			NodePools("cluster-cluster-uid").
			Create(context.Background(), &v1alpha1.NodePool{}, platform.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring("failed to decode node pool create")))
	})
})
