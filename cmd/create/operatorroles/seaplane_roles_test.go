// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package operatorroles

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/seaplane"
)

func TestWorkerTrustPolicyParses(t *testing.T) {
	doc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":["ec2.amazonaws.com"]},"Action":"sts:AssumeRole"}]}`
	if _, err := aws.ParsePolicyDocument(doc); err != nil {
		t.Fatal(err)
	}
}

func TestGetHCPOperatorRolesMatchesSuffixes(t *testing.T) {
	roles := getHCPOperatorRoles()
	suffixes := seaplane.OperatorRoleSuffixes()
	if len(roles) != len(suffixes) {
		t.Fatalf("expected %d roles, got %d", len(suffixes), len(roles))
	}
	for i, suffix := range suffixes {
		if roles[i].Name != suffix {
			t.Fatalf("role[%d]: expected name %q, got %q", i, suffix, roles[i].Name)
		}
	}
}

var _ = Describe("oidcIssuerDomainFromUrl", func() {
	It("derives host and path from a CloudFront issuer with a path", func() {
		domain, err := oidcIssuerDomainFromUrl(
			"https://d1y318c688czkz.cloudfront.net/db9ed6aaa41e4797968f134b49935645")
		Expect(err).NotTo(HaveOccurred())
		Expect(domain).To(Equal("d1y318c688czkz.cloudfront.net/db9ed6aaa41e4797968f134b49935645"))
	})

	It("derives host and path from an S3 issuer with a path", func() {
		domain, err := oidcIssuerDomainFromUrl("https://rh-oidc.s3.us-east-1.amazonaws.com/my-prefix")
		Expect(err).NotTo(HaveOccurred())
		Expect(domain).To(Equal("rh-oidc.s3.us-east-1.amazonaws.com/my-prefix"))
	})

	It("drops query strings, matching how the OIDC provider is registered", func() {
		domain, err := oidcIssuerDomainFromUrl("https://rh-oidc.s3.us-east-1.amazonaws.com/my-prefix?x=1")
		Expect(err).NotTo(HaveOccurred())
		Expect(domain).To(Equal("rh-oidc.s3.us-east-1.amazonaws.com/my-prefix"))
	})

	It("rejects non-https issuer URLs", func() {
		_, err := oidcIssuerDomainFromUrl("http://rh-oidc.s3.us-east-1.amazonaws.com/my-prefix")
		Expect(err).To(MatchError(ContainSubstring("https:// scheme")))
	})

	It("rejects invalid issuer URLs", func() {
		_, err := oidcIssuerDomainFromUrl("not a url")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("runSeaplaneCreateOperatorRoles validation", func() {
	var originalArgs = args
	var exitCode int
	var exited bool

	BeforeEach(func() {
		originalArgs = args
		exitCode = 0
		exited = false
		seaplaneExitFn = func(code int) {
			exitCode = code
			exited = true
		}
	})

	AfterEach(func() {
		args = originalArgs
		seaplaneExitFn = func(code int) { os.Exit(code) }
	})

	It("rejects an empty OIDC config ID", func() {
		args.prefix = "test-prefix"
		args.oidcConfigId = ""
		runSeaplaneCreateOperatorRoles(rosa.NewRuntime())
		Expect(exited).To(BeTrue())
		Expect(exitCode).To(Equal(1))
	})

	It("rejects a flag-like OIDC config ID swallowed from a following flag", func() {
		args.prefix = "test-prefix"
		args.oidcConfigId = "--mode"
		runSeaplaneCreateOperatorRoles(rosa.NewRuntime())
		Expect(exited).To(BeTrue())
		Expect(exitCode).To(Equal(1))
	})
})
