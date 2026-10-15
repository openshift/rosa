package oidcconfig

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	clientpathbind "github.com/openshift-online/rosa-hyperfleet-api/clientset/pathbind"

	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("hyperfleetOidcConfigCreate PreRequest", func() {
	var originalArgs = args

	BeforeEach(func() {
		originalArgs = args
	})

	AfterEach(func() {
		args = originalArgs
	})

	It("generates a unique name and sets the required managed type", func() {
		args.managed = true
		args.userPrefix = ""
		args.installerRoleArn = ""

		runtime := test.NewTestRuntime()
		input := &hfpathbind.OidcConfigCreateInput{}
		handler := &hyperfleetOidcConfigCreate{}
		Expect(handler.PreRequest(context.Background(), runtime.RosaRuntime, input)).To(Succeed())
		Expect(input.Name).To(MatchRegexp(`^managed-oidc-[a-f0-9]{16}$`))
		Expect(input.Type).To(Equal("managed"))

		obj := &v1alpha1.OidcConfig{}
		Expect(clientpathbind.Expand(context.Background(), *input, obj)).To(Succeed())
		Expect(handler.PostExpand(context.Background(), runtime.RosaRuntime, input, obj)).To(Succeed())
		Expect(obj.Name).To(Equal(input.Name))
		Expect(obj.Spec.Type).To(Equal("managed"))
	})

	It("uses the supplied prefix as the resource name", func() {
		args.managed = true
		args.userPrefix = "e2e-managed"
		args.installerRoleArn = ""

		runtime := test.NewTestRuntime()
		input := &hfpathbind.OidcConfigCreateInput{}
		Expect((&hyperfleetOidcConfigCreate{}).PreRequest(context.Background(), runtime.RosaRuntime, input)).To(Succeed())
		Expect(input.Name).To(Equal("e2e-managed"))
		Expect(input.Type).To(Equal("managed"))
	})
})

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
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-id",
			},
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
