package hyperfleet

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hfscheme "github.com/openshift-online/rosa-hyperfleet-api/clientset/generated/scheme"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
)

var _ = Describe("WithClusterProxy", func() {
	DescribeTable("adds proxy fields while preserving the SDK request and caller's object",
		func(includeProxy, includeBundle, existingConfiguration bool) {
			obj := &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Labels: map[string]string{"test": "proxy"}},
				Spec:       v1alpha1.ClusterSpec{HostedCluster: v1alpha1.HostedClusterSpecPassthrough{Channel: "candidate"}},
			}
			if existingConfiguration {
				obj.Spec.HostedCluster.Configuration = &v1alpha1.ClusterConfiguration{
					MachineConfig: &v1alpha1.MachineConfigSpec{},
				}
			}
			original := obj.DeepCopy()
			data, err := json.Marshal(obj)
			Expect(err).NotTo(HaveOccurred())
			var expected map[string]any
			Expect(json.Unmarshal(data, &expected)).To(Succeed())
			bundleFile := ""
			if includeBundle {
				bundleFile = filepath.Join(GinkgoT().TempDir(), "bundle.pem")
				Expect(os.WriteFile(bundleFile, []byte("test bundle contents"), 0600)).To(Succeed())
			}
			var proxy *ClusterProxy
			if includeProxy {
				proxy = &ClusterProxy{HTTPProxy: "http://proxy.example.com:8080", NoProxy: "quay.io"}
			}
			hf, _ := clusterProxyTestClient(func(w http.ResponseWriter, req *http.Request) {
				Expect(req.Method).To(Equal(http.MethodPost))
				Expect(req.URL.Path).To(Equal("/clusters"))
				var body map[string]any
				Expect(json.NewDecoder(req.Body).Decode(&body)).To(Succeed())
				spec := body["spec"].(map[string]any)
				hosted := spec["hostedCluster"].(map[string]any)
				if includeProxy {
					configuration := hosted["configuration"].(map[string]any)
					Expect(configuration["proxy"]).To(Equal(map[string]any{
						"httpProxy": "http://proxy.example.com:8080", "noProxy": "quay.io",
					}))
					delete(configuration, "proxy")
					if !existingConfiguration {
						delete(hosted, "configuration")
					}
				}
				if includeBundle {
					Expect(spec["additionalTrustBundle"]).To(Equal("test bundle contents"))
					delete(spec, "additionalTrustBundle")
				}
				Expect(body).To(Equal(expected))
				_, err := w.Write([]byte(`{"metadata":{"name":"test-cluster","uid":"cluster-uid"}}`))
				Expect(err).NotTo(HaveOccurred())
			})
			created, err := WithClusterProxy(hf, proxy, bundleFile).HyperfleetV1alpha1().Clusters().
				Create(context.Background(), obj, platform.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(created.UID)).To(Equal("cluster-uid"))
			Expect(obj).To(Equal(original))
		},
		Entry("proxy with existing configuration", true, false, true),
		Entry("proxy with no configuration", true, false, false),
		Entry("bundle only", false, true, true),
		Entry("proxy and bundle", true, true, true),
	)

	It("delegates non-create operations to the original SDK client", func() {
		hf, clusters := clusterProxyTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected request") })
		clusters.EXPECT().List(gomock.Any(), platform.ListOptions{}).Return(&v1alpha1.ClusterList{}, nil)
		_, err := WithClusterProxy(hf, nil, "").HyperfleetV1alpha1().Clusters().
			List(context.Background(), platform.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects missing clusters before sending a request", func() {
		hf, _ := clusterProxyTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected request") })
		_, err := WithClusterProxy(hf, nil, "").HyperfleetV1alpha1().Clusters().
			Create(context.Background(), nil, platform.CreateOptions{})
		Expect(err).To(MatchError("cluster is required"))
	})

	It("reports missing bundle files before sending a request", func() {
		hf, _ := clusterProxyTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected request") })
		_, err := WithClusterProxy(hf, nil, filepath.Join(GinkgoT().TempDir(), "missing.pem")).
			HyperfleetV1alpha1().Clusters().Create(context.Background(), &v1alpha1.Cluster{}, platform.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring("failed to read additional trust bundle file")))
	})

	It("preserves context cancellation", func() {
		hf, _ := clusterProxyTestClient(func(http.ResponseWriter, *http.Request) { Fail("unexpected request") })
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := WithClusterProxy(hf, nil, "").HyperfleetV1alpha1().Clusters().
			Create(ctx, &v1alpha1.Cluster{}, platform.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring("context canceled")))
	})

	It("reports malformed API responses", func() {
		hf, _ := clusterProxyTestClient(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(`{"spec":`))
			Expect(err).NotTo(HaveOccurred())
		})
		_, err := WithClusterProxy(hf, nil, "").HyperfleetV1alpha1().Clusters().
			Create(context.Background(), &v1alpha1.Cluster{}, platform.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring("failed to decode cluster response")))
	})
})

func clusterProxyTestClient(handler http.HandlerFunc) (*hfmocks.MockInterface, *hfmocks.MockClusterInterface) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer GinkgoRecover()
		w.Header().Set("Content-Type", "application/json")
		handler(w, req)
	}))
	DeferCleanup(server.Close)
	baseURL, err := url.Parse(server.URL)
	Expect(err).NotTo(HaveOccurred())
	client, err := rest.NewRESTClient(baseURL, "", rest.ClientContentConfig{Negotiator: runtime.NewClientNegotiator(hfscheme.Codecs.WithoutConversion(), schema.GroupVersion{Version: "v1"})}, nil, server.Client())
	Expect(err).NotTo(HaveOccurred())
	ctrl := gomock.NewController(GinkgoT())
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	v1.EXPECT().RESTClient().Return(client).AnyTimes()
	return hf, clusters
}
