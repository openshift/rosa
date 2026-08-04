package hyperfleet

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	reportertest "github.com/openshift/rosa/test/reporter"
)

func TestHyperfleet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "hyperfleet")
}

var _ = Describe("ExtractRegion", func() {
	DescribeTable("standard and GovCloud regions",
		func(rawURL, expected string, expectErr bool) {
			region, err := ExtractRegion(rawURL)
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(region).To(Equal(expected))
			}
		},
		Entry("standard region", "https://abc123.execute-api.us-east-1.amazonaws.com", "us-east-1", false),
		Entry("west region", "https://abc123.execute-api.us-west-2.amazonaws.com", "us-west-2", false),
		Entry("ap region", "https://abc123.execute-api.ap-southeast-1.amazonaws.com", "ap-southeast-1", false),
		Entry("govcloud east", "https://abc123.execute-api.us-gov-east-1.amazonaws.com", "us-gov-east-1", false),
		Entry("govcloud west", "https://abc123.execute-api.us-gov-west-1.amazonaws.com", "us-gov-west-1", false),
		Entry("no region in URL", "https://example.com/api", "", true),
		Entry("invalid URL", "://bad url", "", true),
	)
})

var _ = Describe("WarnOnMismatch", func() {
	It("does not warn when regions match", func() {
		warned := false
		r := &reportertest.FakeLogger{WarnFn: func(string, ...any) { warned = true }}
		WarnOnMismatch("us-east-1", "https://abc.execute-api.us-east-1.amazonaws.com", r)
		Expect(warned).To(BeFalse())
	})

	It("warns when explicit region differs from URL region", func() {
		warned := false
		r := &reportertest.FakeLogger{WarnFn: func(string, ...any) { warned = true }}
		WarnOnMismatch("us-west-2", "https://abc.execute-api.us-east-1.amazonaws.com", r)
		Expect(warned).To(BeTrue())
	})

	It("warns when URL has no extractable region", func() {
		warned := false
		r := &reportertest.FakeLogger{WarnFn: func(string, ...any) { warned = true }}
		WarnOnMismatch("us-east-1", "https://example.com", r)
		Expect(warned).To(BeTrue())
	})
})

var _ = Describe("SetURL and Reset", func() {
	BeforeEach(func() { hyperfleetURL = "" })
	AfterEach(func() { hyperfleetURL = "" })

	It("sets the URL when empty", func() {
		SetURL("https://example.execute-api.us-east-1.amazonaws.com")
		Expect(hyperfleetURL).To(Equal("https://example.execute-api.us-east-1.amazonaws.com"))
	})

	It("does not overwrite an existing URL", func() {
		hyperfleetURL = "https://first.execute-api.us-east-1.amazonaws.com"
		SetURL("https://second.execute-api.us-east-1.amazonaws.com")
		Expect(hyperfleetURL).To(Equal("https://first.execute-api.us-east-1.amazonaws.com"))
	})

	It("Reset clears the URL", func() {
		hyperfleetURL = "https://something.execute-api.us-east-1.amazonaws.com"
		Reset()
		Expect(hyperfleetURL).To(BeEmpty())
	})
})

var _ = Describe("Enabled and ExplicitURL", func() {
	BeforeEach(func() { hyperfleetURL = "" })
	AfterEach(func() { hyperfleetURL = "" })

	It("reports disabled when URL is empty", func() {
		Expect(Enabled()).To(BeFalse())
		Expect(ExplicitURL()).To(BeEmpty())
	})

	It("reports enabled when URL is set", func() {
		hyperfleetURL = "https://abc.execute-api.us-east-1.amazonaws.com"
		Expect(Enabled()).To(BeTrue())
		Expect(ExplicitURL()).To(Equal("https://abc.execute-api.us-east-1.amazonaws.com"))
	})
})

var _ = Describe("ComputeRolesRef", func() {
	It("builds all seven ARNs from prefix and account ID", func() {
		ref := ComputeRolesRef("my-cluster", "123456789012")
		Expect(ref.IngressARN).To(Equal("arn:aws:iam::123456789012:role/my-cluster-ingress"))
		Expect(ref.KubeCloudControllerARN).To(Equal("arn:aws:iam::123456789012:role/my-cluster-cloud-controller-manager"))
		Expect(ref.StorageARN).To(Equal("arn:aws:iam::123456789012:role/my-cluster-ebs-csi"))
		Expect(ref.ImageRegistryARN).To(Equal("arn:aws:iam::123456789012:role/my-cluster-image-registry"))
		Expect(ref.NetworkARN).To(Equal("arn:aws:iam::123456789012:role/my-cluster-network-config"))
		Expect(ref.ControlPlaneOperatorARN).To(Equal("arn:aws:iam::123456789012:role/my-cluster-control-plane-operator"))
		Expect(ref.NodePoolManagementARN).To(Equal("arn:aws:iam::123456789012:role/my-cluster-node-pool-management"))
	})
})

var _ = Describe("ComputeInstanceProfile", func() {
	It("appends the worker role suffix", func() {
		Expect(ComputeInstanceProfile("my-cluster")).To(Equal("my-cluster-ROSA-Worker-Role"))
	})
})
