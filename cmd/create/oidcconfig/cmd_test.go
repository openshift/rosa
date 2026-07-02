package oidcconfig

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/rosa/oidcconfigs"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Create OIDC Config", func() {
	Context("getOidcConfigStrategy", func() {
		var input *oidcconfigs.OidcConfigInput

		BeforeEach(func() {
			input = &oidcconfigs.OidcConfigInput{}
			args.rawFiles = false
			args.managed = false
		})

		It("returns raw strategy when rawFiles is true", func() {
			args.rawFiles = true

			strategy, err := getOidcConfigStrategy(interactive.ModeAuto, input)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(BeAssignableToTypeOf(&CreateUnmanagedOidcConfigRawStrategy{}))
		})

		It("returns managed auto strategy when managed is true", func() {
			args.managed = true

			strategy, err := getOidcConfigStrategy(interactive.ModeAuto, input)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(BeAssignableToTypeOf(&CreateManagedOidcConfigAutoStrategy{}))
		})

		It("returns unmanaged auto strategy for auto mode", func() {
			strategy, err := getOidcConfigStrategy(interactive.ModeAuto, input)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(BeAssignableToTypeOf(&CreateUnmanagedOidcConfigAutoStrategy{}))
		})

		It("returns unmanaged manual strategy for manual mode", func() {
			strategy, err := getOidcConfigStrategy(interactive.ModeManual, input)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategy).To(BeAssignableToTypeOf(&CreateUnmanagedOidcConfigManualStrategy{}))
		})

		It("returns error for invalid mode", func() {
			_, err := getOidcConfigStrategy("invalid", input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid mode"))
		})
	})

	Context("ManagedOidcConfigAutoStrategy.executeNoExit", func() {
		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
		})

		It("returns the OIDC config ID on success", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, `{
				"kind": "OidcConfig",
				"id": "managed-oidc-123",
				"managed": true,
				"issuer_url": "https://oidc.managed.example.com"
			}`))

			input := &oidcconfigs.OidcConfigInput{}
			strategy := &CreateManagedOidcConfigAutoStrategy{oidcConfigInput: input}

			id, err := strategy.executeNoExit(t.RosaRuntime)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("managed-oidc-123"))
		})

		It("returns error when CreateOidcConfig fails", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError, `{
				"kind": "Error",
				"id": "500",
				"href": "/api/clusters_mgmt/v1/errors/500",
				"code": "CLUSTERS-MGMT-500",
				"reason": "internal error"
			}`))

			input := &oidcconfigs.OidcConfigInput{}
			strategy := &CreateManagedOidcConfigAutoStrategy{oidcConfigInput: input}

			_, err := strategy.executeNoExit(t.RosaRuntime)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("managed OIDC Configuration"))
		})
	})
})
