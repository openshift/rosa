package breakglasscredential

import (
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Break glass credential", func() {
	Context("CreateBreakGlassConfig", func() {
		It("Returns the expected value", func() {
			args := BreakGlassCredentialArgs{
				username:           "abc",
				expirationDuration: time.Hour,
			}
			credential, err := CreateBreakGlassConfig(&args)
			Expect(err).NotTo(HaveOccurred())
			Expect(credential.Username()).To(Equal("abc"))
			Expect(credential.ExpirationTimestamp()).To(Equal(time.Now().Add(time.Hour).Round(time.Second)))
		})
	})

	Context("FormatBreakGlassCredentialOutput", func() {
		It("Should not fail", func() {
			credential := test.BuildBreakGlassCredential()
			_, err := FormatBreakGlassCredentialOutput(credential)
			Expect(err).To(Not(HaveOccurred()))
		})
	})
})
