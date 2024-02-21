package ocm

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Test oidc config filtering", func() {
	var configs []*v1.OidcConfig

	BeforeEach(func() {
		c1, err := v1.NewOidcConfig().ID("292929ssti8i2beq6vivt256bj0ju15dm").
			IssuerUrl("https://test.org/292929ssti8i2beq6vivt256bj0ju15dm").Managed(true).Build()
		Expect(err).ToNot(HaveOccurred())
		c2, err := v1.NewOidcConfig().ID("212121ssti8i2beq6vivt256bj0ju15dm").
			IssuerUrl("https://test2.org/212121ssti8i2beq6vivt256bj0ju15dm").Managed(true).Build()
		Expect(err).ToNot(HaveOccurred())
		c3, err := v1.NewOidcConfig().ID("202020ssti8i2beq6vivt256bj0ju15dm").
			IssuerUrl("https://test3.org/202020ssti8i2beq6vivt256bj0ju15dm").Managed(false).Build()
		Expect(err).ToNot(HaveOccurred())
		c4, err := v1.NewOidcConfig().ID("151515ssti8i2beq6vivt256bj0ju15dm").
			IssuerUrl("https://test4.org/151515ssti8i2beq6vivt256bj0ju15dm").Managed(false).Build()
		Expect(err).ToNot(HaveOccurred())
		configs = []*v1.OidcConfig{c1, c2, c3, c4}
	})

	It("Managed filter", func() {
		resp := filterOidcConfigs(configs, "292929")
		Expect(len(resp)).To(Equal(1))
		resp = filterOidcConfigs(configs, "212121")
		Expect(len(resp)).To(Equal(1))
	})
	It("Unmanaged filter", func() {
		resp := filterOidcConfigs(configs, "test3")
		Expect(len(resp)).To(Equal(1))
		resp = filterOidcConfigs(configs, "test4")
		Expect(len(resp)).To(Equal(1))
	})
	It("No filter", func() {
		resp := filterOidcConfigs(configs, "")
		Expect(len(resp)).To(Equal(4))
	})
})
