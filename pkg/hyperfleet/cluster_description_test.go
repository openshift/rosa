package hyperfleet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"

	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hfscheme "github.com/openshift-online/rosa-hyperfleet-api/clientset/generated/scheme"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
)

var _ = Describe("GetClusterDescription", func() {
	DescribeTable("preserves trust-bundle presence without exposing its contents",
		func(spec, expected string) {
			client := clusterDescriptionTestClient(`{"metadata":{"name":"hf-proxy-e8","uid":"cluster-uid"},"spec":` + spec + `}`)
			description, err := GetClusterDescription(context.Background(), client, "cluster-uid")
			Expect(err).NotTo(HaveOccurred())
			Expect(description.Cluster.Name).To(Equal("hf-proxy-e8"))
			Expect(string(description.Cluster.UID)).To(Equal("cluster-uid"))
			Expect(description.AdditionalTrustBundle).To(Equal(expected))
		},
		Entry("redacted bundle", `{"additionalTrustBundle":"REDACTED"}`, "REDACTED"),
		Entry("unexpected unredacted contents", `{"additionalTrustBundle":"PEM contents must stay private"}`, "REDACTED"),
		Entry("empty bundle", `{"additionalTrustBundle":""}`, ""),
		Entry("null bundle", `{"additionalTrustBundle":null}`, ""),
		Entry("older response with no bundle field", `{}`, ""),
	)

	It("rejects a bundle response of the wrong type", func() {
		client := clusterDescriptionTestClient(`{"spec":{"additionalTrustBundle":true}}`)
		_, err := GetClusterDescription(context.Background(), client, "cluster-uid")
		Expect(err).To(MatchError(ContainSubstring("failed to decode cluster proxy or trust bundle")))
	})

	It("reports malformed cluster JSON", func() {
		client := clusterDescriptionTestClient(`{"spec":`)
		_, err := GetClusterDescription(context.Background(), client, "cluster-uid")
		Expect(err).To(MatchError(ContainSubstring("failed to decode cluster response")))
	})
})

func clusterDescriptionTestClient(response string) *hfmocks.MockInterface {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer GinkgoRecover()
		Expect(req.Method).To(Equal(http.MethodGet))
		Expect(req.URL.Path).To(Equal("/clusters/cluster-uid"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(response))
		Expect(err).NotTo(HaveOccurred())
	}))
	DeferCleanup(server.Close)
	baseURL, err := url.Parse(server.URL)
	Expect(err).NotTo(HaveOccurred())
	client, err := rest.NewRESTClient(baseURL, "", rest.ClientContentConfig{Negotiator: runtime.NewClientNegotiator(hfscheme.Codecs.WithoutConversion(), schema.GroupVersion{Version: "v1"})}, nil, server.Client())
	Expect(err).NotTo(HaveOccurred())
	ctrl := gomock.NewController(GinkgoT())
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1)
	v1.EXPECT().RESTClient().Return(client)
	return hf
}
