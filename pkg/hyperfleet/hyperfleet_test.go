package hyperfleet

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
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
		Entry("cleartext HTTP", "http://abc123.execute-api.us-east-1.amazonaws.com", "", true),
	)
})

var _ = Describe("CheckRegionConflict", func() {
	It("returns nil when regions match", func() {
		warn, err := CheckRegionConflict("us-east-1", "https://abc.execute-api.us-east-1.amazonaws.com")
		Expect(err).ToNot(HaveOccurred())
		Expect(warn).To(BeEmpty())
	})

	It("returns an error when explicit region differs from URL region", func() {
		_, err := CheckRegionConflict("us-west-2", "https://abc.execute-api.us-east-1.amazonaws.com")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("us-west-2"))
		Expect(err.Error()).To(ContainSubstring("us-east-1"))
	})

	It("returns a warning when URL has no extractable region", func() {
		warn, err := CheckRegionConflict("us-east-1", "https://example.com")
		Expect(err).ToNot(HaveOccurred())
		Expect(warn).To(ContainSubstring("cannot verify region"))
	})
})

var _ = Describe("SetURL and Reset", func() {
	BeforeEach(Reset)
	AfterEach(Reset)

	It("sets the URL when empty", func() {
		SetURL("https://example.execute-api.us-east-1.amazonaws.com")
		Expect(hyperfleetURL).To(Equal("https://example.execute-api.us-east-1.amazonaws.com"))
	})

	It("does not overwrite an existing URL", func() {
		hyperfleetURL = "https://first.execute-api.us-east-1.amazonaws.com"
		SetURL("https://second.execute-api.us-east-1.amazonaws.com")
		Expect(hyperfleetURL).To(Equal("https://first.execute-api.us-east-1.amazonaws.com"))
	})

	It("does not set a cleartext HTTP URL", func() {
		SetURL("http://abc123.execute-api.us-east-1.amazonaws.com")
		Expect(hyperfleetURL).To(BeEmpty())
	})

	It("Reset clears the URL", func() {
		hyperfleetURL = "https://something.execute-api.us-east-1.amazonaws.com"
		urlFromFlag = true
		Reset()
		Expect(hyperfleetURL).To(BeEmpty())
		Expect(FromFlag()).To(BeFalse())
	})
})

var _ = Describe("Enabled and ExplicitURL", func() {
	BeforeEach(Reset)
	AfterEach(Reset)

	It("reports disabled when URL is empty", func() {
		Expect(Enabled()).To(BeFalse())
		Expect(ExplicitURL()).To(BeEmpty())
	})

	It("reports enabled when URL is set", func() {
		hyperfleetURL = "https://abc.execute-api.us-east-1.amazonaws.com"
		Expect(Enabled()).To(BeTrue())
		Expect(ExplicitURL()).To(Equal("https://abc.execute-api.us-east-1.amazonaws.com"))
		Expect(FromFlag()).To(BeFalse())
	})
})

var _ = Describe("FromFlag", func() {
	BeforeEach(Reset)
	AfterEach(Reset)

	It("is false after SetURL", func() {
		SetURL("https://abc.execute-api.us-east-1.amazonaws.com")
		Expect(FromFlag()).To(BeFalse())
		Expect(ExplicitURL()).To(Equal("https://abc.execute-api.us-east-1.amazonaws.com"))
	})

	It("is true after SetFromFlag", func() {
		SetFromFlag("https://abc.execute-api.us-east-1.amazonaws.com")
		Expect(FromFlag()).To(BeTrue())
		Expect(ExplicitURL()).To(Equal("https://abc.execute-api.us-east-1.amazonaws.com"))
	})
})

var _ = Describe("ComputeRolesRef", func() {
	DescribeTable("builds all seven ARNs for each partition",
		func(partition, wantPrefix string) {
			ref := ComputeRolesRef("my-cluster", "123456789012", partition)
			Expect(ref.IngressARN).To(Equal(wantPrefix + "my-cluster-ingress"))
			Expect(ref.KubeCloudControllerARN).To(Equal(wantPrefix + "my-cluster-cloud-controller-manager"))
			Expect(ref.StorageARN).To(Equal(wantPrefix + "my-cluster-ebs-csi"))
			Expect(ref.ImageRegistryARN).To(Equal(wantPrefix + "my-cluster-image-registry"))
			Expect(ref.NetworkARN).To(Equal(wantPrefix + "my-cluster-network-config"))
			Expect(ref.ControlPlaneOperatorARN).To(Equal(wantPrefix + "my-cluster-control-plane-operator"))
			Expect(ref.NodePoolManagementARN).To(Equal(wantPrefix + "my-cluster-node-pool-management"))
		},
		Entry("commercial", "aws", "arn:aws:iam::123456789012:role/"),
		Entry("GovCloud", "aws-us-gov", "arn:aws-us-gov:iam::123456789012:role/"),
	)
})

var _ = Describe("ComputeInstanceProfile", func() {
	It("appends the worker role suffix", func() {
		Expect(ComputeInstanceProfile("my-cluster")).To(Equal("my-cluster-ROSA-Worker-Role"))
	})
})

var _ = Describe("InstanceProfileFromRolesRef", func() {
	It("extracts prefix from NodePoolManagementARN and returns the instance profile name", func() {
		ref := hypershiftv1beta1.AWSRolesRef{
			NodePoolManagementARN: "arn:aws:iam::123456789012:role/my-cluster-node-pool-management",
		}
		Expect(InstanceProfileFromRolesRef(ref)).To(Equal("my-cluster-ROSA-Worker-Role"))
	})

	It("returns empty string when NodePoolManagementARN has unexpected format", func() {
		ref := hypershiftv1beta1.AWSRolesRef{NodePoolManagementARN: "bad-arn"}
		Expect(InstanceProfileFromRolesRef(ref)).To(BeEmpty())
	})

	It("returns empty string when NodePoolManagementARN is empty", func() {
		Expect(InstanceProfileFromRolesRef(hypershiftv1beta1.AWSRolesRef{})).To(BeEmpty())
	})
})
