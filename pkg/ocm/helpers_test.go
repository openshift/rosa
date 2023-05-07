package ocm

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Http tokens", func() {
	Context("Http tokens variable validations", func() {
		It("OK: Validates successfully http tokens required", func() {
			err := ValidateHttpTokensValue(string(cmv1.HttpTokenStateRequired))
			Expect(err).NotTo(HaveOccurred())
		})
		It("OK: Validates successfully http tokens optional", func() {
			err := ValidateHttpTokensValue(string(cmv1.HttpTokenStateOptional))
			Expect(err).NotTo(HaveOccurred())
		})
		It("OK: Validates successfully http tokens empty string", func() {
			err := ValidateHttpTokensValue("")
			Expect(err).NotTo(HaveOccurred())
		})
		It("Error: Validates error for http tokens bad string", func() {
			err := ValidateHttpTokensValue("dummy")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("http-tokens value should be one of '%s', '%s'",
				cmv1.HttpTokenStateRequired, cmv1.HttpTokenStateOptional)))
		})
	})

})
