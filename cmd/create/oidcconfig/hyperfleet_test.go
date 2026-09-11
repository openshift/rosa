package oidcconfig

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("hyperfleetOidcConfigCreate PostResponse", func() {
	AfterEach(func() {
		output.SetOutput("")
	})

	It("creates the IAM OIDC provider even when --output json is set", func() {
		t := test.NewTestRuntime()
		output.SetOutput("json")

		called := false
		orig := createOidcProviderFn
		createOidcProviderFn = func(ctx context.Context, r *rosa.Runtime, oidcConfig *v1alpha1.OidcConfig, mode string) error {
			called = true
			Expect(oidcConfig.Spec.IssuerUrl).NotTo(BeEmpty())
			Expect(mode).To(Equal(interactive.ModeAuto))
			return nil
		}
		defer func() { createOidcProviderFn = orig }()

		interactive.SetModeKey(interactive.ModeAuto)

		cfg := &v1alpha1.OidcConfig{
			Name: "test-id",
			Spec: v1alpha1.OidcConfigSpec{
				Type:      "managed",
				IssuerUrl: "https://example.cloudfront.net/test-id",
			},
		}

		err := (&hyperfleetOidcConfigCreate{}).PostResponse(context.Background(), t.RosaRuntime, cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeTrue())
	})
})
