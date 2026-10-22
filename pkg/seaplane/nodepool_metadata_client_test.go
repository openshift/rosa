// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"context"
	"encoding/json"
	"io"
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
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	seaplanescheme "github.com/openshift-online/rosa-seaplane-api/clientset/generated/scheme"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	"github.com/openshift-online/rosa-seaplane-api/clientset/transport"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"

	seaplanemocks "github.com/openshift/rosa/pkg/seaplane/mocks"
)

func metadataTestClient(handler http.HandlerFunc) (*seaplanemocks.MockInterface, *seaplanemocks.MockNodePoolInterface) {
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
	config := rest.ClientContentConfig{Negotiator: runtime.NewClientNegotiator(seaplanescheme.Codecs.WithoutConversion(), schema.GroupVersion{Version: "v1"})}
	client, err := rest.NewRESTClient(baseURL, "", config, nil, httpClient)
	Expect(err).NotTo(HaveOccurred())
	ctrl := gomock.NewController(GinkgoT())
	seaplaneClient, v1, pools := seaplanemocks.NewMockInterface(ctrl), seaplanemocks.NewMockV1alpha1PublicInterface(ctrl), seaplanemocks.NewMockNodePoolInterface(ctrl)
	seaplaneClient.EXPECT().SeaplaneV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().NodePools().Return(pools).AnyTimes()
	v1.EXPECT().RESTClient().Return(client).AnyTimes()
	return seaplaneClient, pools
}

func nodePoolWithPlatformManagedFields() *v1alpha1.NodePool {
	replicas := int32(2)
	return &v1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers", UID: types.UID("pool-uid")},
		Spec: v1alpha1.NodePoolSpec{
			NodePool: v1alpha1.NodePoolSpecPassthrough{
				Replicas: &replicas,
				Platform: v1alpha1.NodePoolPlatform{Type: hypershiftv1beta1.AWSPlatform},
				Management: hypershiftv1beta1.NodePoolManagement{
					UpgradeType: hypershiftv1beta1.UpgradeTypeReplace,
				},
			},
		},
	}
}

var _ = Describe("Nodepool metadata updates", func() {
	DescribeTable("preserves request fields and explicit clearing", func(labelsChanged, taintsChanged, empty bool) {
		replicas := int32(3)
		pool := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers", UID: types.UID("pool-uid")}, Spec: v1alpha1.NodePoolSpec{
			Labels:   map[string]string{"team": "test"},
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
		seaplaneClient, _ := metadataTestClient(func(w http.ResponseWriter, req *http.Request) {
			Expect(req.Method).To(Equal(http.MethodPut))
			Expect(req.URL.Path).To(Equal("/nodepools/my-cluster.workers"))
			Expect(req.URL.Query()).To(BeEmpty())
			body := map[string]any{}
			Expect(json.NewDecoder(req.Body).Decode(&body)).To(Succeed())
			Expect(body).To(Equal(expected))
			_, err := w.Write([]byte(`{"metadata":{"name":"my-cluster.workers","uid":"pool-uid"}}`))
			Expect(err).NotTo(HaveOccurred())
		})
		updated, err := WithNodePoolMetadataUpdates(seaplaneClient, labelsChanged, taintsChanged).SeaplaneV1alpha1().NodePools().Update(context.Background(), pool, platform.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Name).To(Equal("my-cluster.workers"))
		Expect(pool).To(Equal(original))
	},
		Entry("both fields set", true, true, false), Entry("labels only", true, false, false),
		Entry("taints only", false, true, false), Entry("clear both", true, true, true),
		Entry("clear labels without clearing taints", true, false, true), Entry("clear taints without clearing labels", false, true, true),
	)
	It("delegates reads to the SDK", func() {
		seaplaneClient, pools := metadataTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected HTTP request") })
		pools.EXPECT().Get(gomock.Any(), "my-cluster.workers", platform.GetOptions{}).Return(&v1alpha1.NodePool{}, nil)
		_, err := WithNodePoolMetadataUpdates(seaplaneClient, true, true).SeaplaneV1alpha1().NodePools().Get(context.Background(), "my-cluster.workers", platform.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
	DescribeTable("rejects pools without a name", func(pool *v1alpha1.NodePool) {
		seaplaneClient, _ := metadataTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected HTTP request") })
		_, err := WithNodePoolMetadataUpdates(seaplaneClient, true, true).SeaplaneV1alpha1().NodePools().Update(context.Background(), pool, platform.UpdateOptions{})
		Expect(err).To(MatchError("node pool name is required"))
	}, Entry("nil pool", (*v1alpha1.NodePool)(nil)), Entry("empty name", &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{UID: "pool-uid"}}))
	It("rejects updates without a UID", func() {
		seaplaneClient, _ := metadataTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected HTTP request") })
		pool := &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers"}}
		_, err := WithNodePoolMetadataUpdates(seaplaneClient, true, true).SeaplaneV1alpha1().NodePools().Update(context.Background(), pool, platform.UpdateOptions{})
		Expect(err).To(MatchError("node pool UID is required"))
	})
	It("preserves API error details", func() {
		seaplaneClient, _ := metadataTestClient(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"invalid labels","code":400}`))
			Expect(err).NotTo(HaveOccurred())
		})
		_, err := WithNodePoolMetadataUpdates(seaplaneClient, true, false).SeaplaneV1alpha1().NodePools().Update(context.Background(), &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers", UID: "pool-uid"}}, platform.UpdateOptions{})
		Expect(err).To(MatchError("invalid labels"))
	})
	It("includes the Platform API errors field in validation failures", func() {
		seaplaneClient, _ := metadataTestClient(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"NODEPOOLS-MGMT-VALIDATION-001: validation failed","code":400,"errors":[{"field":"spec.nodePool.replicas","message":"replicas exceeds the pool limit"}]}`))
			Expect(err).NotTo(HaveOccurred())
		})
		_, err := WithNodePoolMetadataUpdates(seaplaneClient, false, false).SeaplaneV1alpha1().NodePools().Update(
			context.Background(), &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers", UID: "pool-uid"}},
			platform.UpdateOptions{},
		)
		Expect(err).To(MatchError(ContainSubstring("replicas exceeds the pool limit")))
	})
	// TODO: Restore sparse PATCH expectations after the API raw-spec validation fix is deployed.
	It("preserves Platform API field errors from PATCH validation responses", func() {
		seaplaneClient, pools := metadataTestClient(func(w http.ResponseWriter, req *http.Request) {
			Expect(req.Method).To(Equal(http.MethodPatch))
			Expect(req.URL.Path).To(Equal("/nodepools/my-cluster.workers"))
			Expect(req.Header.Get("Content-Type")).To(ContainSubstring("merge-patch+json"))
			body, err := io.ReadAll(req.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(MatchJSON(`{"spec":{"nodePool":{"replicas":3,"platform":{"type":"AWS"},"management":{"upgradeType":"Replace","autoRepair":false}}}}`))

			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"NODEPOOLS-MGMT-VALIDATION-001: A validation error has occurred, check the errors field for more information","code":400,"errors":[{"field":"spec.nodePool.replicas","message":"replicas exceeds the pool limit"}]}`))
			Expect(err).NotTo(HaveOccurred())
		})
		pools.EXPECT().Get(gomock.Any(), "my-cluster.workers", platform.GetOptions{}).
			Return(nodePoolWithPlatformManagedFields(), nil)
		_, err := WithNodePoolMetadataUpdates(seaplaneClient, false, false).
			SeaplaneV1alpha1().NodePools().Patch(
			context.Background(), "my-cluster.workers", types.MergePatchType,
			[]byte(`{"spec":{"nodePool":{"replicas":3}}}`), platform.PatchOptions{},
		)
		Expect(err).To(MatchError(ContainSubstring(`"field":"spec.nodePool.replicas"`)))
		Expect(err).To(MatchError(ContainSubstring(`"message":"replicas exceeds the pool limit"`)))
	})
	It("decodes successful node pool PATCH responses", func() {
		seaplaneClient, pools := metadataTestClient(func(w http.ResponseWriter, req *http.Request) {
			Expect(req.Method).To(Equal(http.MethodPatch))
			Expect(req.URL.Path).To(Equal("/nodepools/my-cluster.workers"))
			body, err := io.ReadAll(req.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(MatchJSON(`{"spec":{"nodePool":{"replicas":3,"platform":{"type":"AWS"},"management":{"upgradeType":"Replace","autoRepair":false}}}}`))
			_, err = w.Write([]byte(`{"metadata":{"name":"my-cluster.workers","uid":"pool-uid"},"spec":{"nodePool":{"replicas":3}}}`))
			Expect(err).NotTo(HaveOccurred())
		})
		pools.EXPECT().Get(gomock.Any(), "my-cluster.workers", platform.GetOptions{}).
			Return(nodePoolWithPlatformManagedFields(), nil)
		updated, err := WithNodePoolMetadataUpdates(seaplaneClient, false, false).
			SeaplaneV1alpha1().NodePools().Patch(
			context.Background(), "my-cluster.workers", types.MergePatchType,
			[]byte(`{"spec":{"nodePool":{"replicas":3}}}`), platform.PatchOptions{},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Name).To(Equal("my-cluster.workers"))
		Expect(updated.UID).To(Equal(types.UID("pool-uid")))
		Expect(updated.Spec.NodePool.Replicas).NotTo(BeNil())
		Expect(*updated.Spec.NodePool.Replicas).To(Equal(int32(3)))
	})
	It("reports malformed API responses", func() {
		seaplaneClient, _ := metadataTestClient(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte("{"))
			Expect(err).NotTo(HaveOccurred())
		})
		_, err := WithNodePoolMetadataUpdates(seaplaneClient, true, false).SeaplaneV1alpha1().NodePools().Update(context.Background(), &v1alpha1.NodePool{ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers", UID: "pool-uid"}}, platform.UpdateOptions{})
		Expect(err).To(MatchError(ContainSubstring("failed to decode node pool update")))
	})
})
