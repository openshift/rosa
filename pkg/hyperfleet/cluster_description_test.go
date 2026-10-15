package hyperfleet

import (
	"context"
	"encoding/json"
	"errors"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"

	hfmocks "github.com/openshift/rosa/pkg/hyperfleet/mocks"
)

var _ = Describe("GetClusterDescription", func() {
	DescribeTable("redacts trust-bundle contents from the typed response",
		func(bundle *string, expected string) {
			cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "hf-proxy-e8", UID: "cluster-uid"}}
			cluster.Spec.AdditionalTrustBundle = bundle
			client := clusterDescriptionTestClient(cluster, nil)
			description, err := GetClusterDescription(context.Background(), client, cluster.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(description.Cluster.Name).To(Equal("hf-proxy-e8"))
			Expect(string(description.Cluster.UID)).To(Equal("cluster-uid"))
			Expect(description.AdditionalTrustBundle).To(Equal(expected))
		},
		Entry("redacted bundle", stringPtr("REDACTED"), "REDACTED"),
		Entry("unexpected unredacted contents", stringPtr("PEM contents must stay private"), "REDACTED"),
		Entry("empty bundle", stringPtr(""), ""),
		Entry("absent bundle", (*string)(nil), ""),
	)

	It("uses the typed proxy projection", func() {
		cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "hf-proxy-e8", UID: "cluster-uid"}}
		cluster.Proxy = &v1alpha1.ClusterProxy{
			HTTPProxy: "http://proxy.example.com:8080",
			NoProxy:   "localhost,127.0.0.1",
		}
		client := clusterDescriptionTestClient(cluster, nil)
		description, err := GetClusterDescription(context.Background(), client, cluster.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(description.Proxy).To(Equal(&ClusterProxy{
			HTTPProxy: "http://proxy.example.com:8080",
			NoProxy:   "localhost,127.0.0.1",
		}))
	})

	It("returns typed API lookup errors", func() {
		cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "hf-proxy-e8"}}
		client := clusterDescriptionTestClient(cluster, errors.New("cluster unavailable"))
		_, err := GetClusterDescription(context.Background(), client, cluster.Name)
		Expect(err).To(MatchError("cluster unavailable"))
	})
})

var _ = Describe("DecodeClusterDescription", func() {
	It("redacts an unredacted trust bundle in raw responses", func() {
		data := []byte(`{"metadata":{"name":"hf-proxy-e8","uid":"cluster-uid"},"spec":{"additionalTrustBundle":"PEM contents must stay private"}}`)
		description, err := DecodeClusterDescription(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(description.AdditionalTrustBundle).To(Equal("REDACTED"))
	})

	It("rejects a bundle response of the wrong type", func() {
		_, err := DecodeClusterDescription([]byte(`{"spec":{"additionalTrustBundle":true}}`))
		Expect(err).To(MatchError(ContainSubstring("failed to decode cluster response")))
		var typeError *json.UnmarshalTypeError
		Expect(errors.As(err, &typeError)).To(BeTrue())
		Expect(typeError.Field).To(Equal("spec.additionalTrustBundle"))
		Expect(typeError.Value).To(Equal("bool"))
	})

	It("reports malformed cluster JSON", func() {
		_, err := DecodeClusterDescription([]byte(`{"spec":`))
		Expect(err).To(MatchError(ContainSubstring("failed to decode cluster response")))
	})
})

func clusterDescriptionTestClient(cluster *v1alpha1.Cluster, getErr error) *hfmocks.MockInterface {
	ctrl := gomock.NewController(GinkgoT())
	hf := hfmocks.NewMockInterface(ctrl)
	v1 := hfmocks.NewMockV1alpha1PublicInterface(ctrl)
	clusters := hfmocks.NewMockClusterInterface(ctrl)
	hf.EXPECT().HyperfleetV1alpha1().Return(v1).AnyTimes()
	v1.EXPECT().Clusters().Return(clusters).AnyTimes()
	clusters.EXPECT().Get(gomock.Any(), cluster.Name, platform.GetOptions{}).Return(cluster, getErr)
	return hf
}

func stringPtr(value string) *string { return &value }
