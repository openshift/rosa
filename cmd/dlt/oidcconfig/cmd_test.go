package oidcconfig

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/interactive"
)

var _ = Describe("Delete OIDC Config", func() {
	Context("getOidcConfigStrategy", func() {
		It("returns managed strategy when input is managed", func() {
			input := OidcConfigInput{Managed: true}

			strategy, err := getOidcConfigStrategy(interactive.ModeAuto, input)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(BeAssignableToTypeOf(&deleteManagedOidcConfigStrategy{}))
		})

		It("returns unmanaged auto strategy for auto mode", func() {
			input := OidcConfigInput{Managed: false, BucketName: "test-bucket"}

			strategy, err := getOidcConfigStrategy(interactive.ModeAuto, input)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(BeAssignableToTypeOf(&deleteUnmanagedOidcConfigAutoStrategy{}))
		})

		It("returns unmanaged manual strategy for manual mode", func() {
			input := OidcConfigInput{Managed: false, BucketName: "test-bucket"}

			strategy, err := getOidcConfigStrategy(interactive.ModeManual, input)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(BeAssignableToTypeOf(&deleteUnmanagedOidcConfigManualStrategy{}))
		})

		It("returns error for invalid mode", func() {
			input := OidcConfigInput{Managed: false}

			_, err := getOidcConfigStrategy("invalid", input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid mode"))
		})
	})
})
