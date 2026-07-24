package idp

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IDP Validators", func() {
	Context("validateGitlabHostURL", func() {
		It("accepts a valid HTTPS URL", func() {
			err := validateGitlabHostURL("https://gitlab.example.com")
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects a non-HTTPS URL", func() {
			err := validateGitlabHostURL("http://gitlab.example.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("https://"))
		})

		It("rejects a URL with query parameters", func() {
			err := validateGitlabHostURL("https://gitlab.example.com?foo=bar")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("query parameters"))
		})

		It("rejects an invalid URL", func() {
			err := validateGitlabHostURL("not-a-url")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("valid GitLab provider URL"))
		})

		It("rejects a URL with a fragment", func() {
			err := validateGitlabHostURL("https://gitlab.example.com#section")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("valid GitLab provider URL"))
		})
	})

	Context("validateGoogleHostedDomain", func() {
		It("accepts a valid domain", func() {
			err := validateGoogleHostedDomain("example.com")
			Expect(err).NotTo(HaveOccurred())
		})

		It("accepts an uppercase domain via normalization", func() {
			err := validateGoogleHostedDomain("Example.COM")
			Expect(err).NotTo(HaveOccurred())
		})

		It("accepts a max-length domain (253 chars)", func() {
			label := "a" + strings.Repeat("b", 61) + "c"
			domain := label + "." + label + "." + label + "." + label[:61]
			Expect(len(domain)).To(Equal(253))
			err := validateGoogleHostedDomain(domain)
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects a domain exceeding 253 chars", func() {
			label := strings.Repeat("a", 63)
			domain := label + "." + label + "." + label + "." + label + ".a"
			Expect(len(domain)).To(BeNumerically(">", 253))
			err := validateGoogleHostedDomain(domain)
			Expect(err).To(HaveOccurred())
		})

		It("rejects a domain with a leading hyphen", func() {
			err := validateGoogleHostedDomain("-example.com")
			Expect(err).To(HaveOccurred())
		})

		It("rejects a domain with a trailing hyphen", func() {
			err := validateGoogleHostedDomain("example-.com")
			Expect(err).To(HaveOccurred())
		})

		It("rejects a domain with underscores", func() {
			err := validateGoogleHostedDomain("ex_ample.com")
			Expect(err).To(HaveOccurred())
		})

		It("rejects a domain with a trailing dot", func() {
			err := validateGoogleHostedDomain("example.com.")
			Expect(err).To(HaveOccurred())
		})

		It("rejects an invalid domain", func() {
			err := validateGoogleHostedDomain("not a domain")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not valid"))
		})
	})

	Context("validateLdapURL", func() {
		It("accepts an ldap:// URL", func() {
			err := validateLdapURL("ldap://ldap.example.com/ou=users,dc=example,dc=com?uid")
			Expect(err).NotTo(HaveOccurred())
		})

		It("accepts an ldaps:// URL", func() {
			err := validateLdapURL("ldaps://ldap.example.com/ou=users,dc=example,dc=com?uid")
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects a non-LDAP scheme", func() {
			err := validateLdapURL("https://ldap.example.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ldap://"))
		})

		It("rejects a bare URL with no scheme", func() {
			err := validateLdapURL("ldap.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid LDAP URL"))
		})
	})

	Context("validateOpenidIssuerURL", func() {
		It("accepts a valid HTTPS URL", func() {
			err := validateOpenidIssuerURL("https://accounts.google.com")
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects a non-HTTPS URL", func() {
			err := validateOpenidIssuerURL("http://accounts.google.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("https://"))
		})

		It("rejects a URL with query parameters", func() {
			err := validateOpenidIssuerURL("https://accounts.google.com?foo=bar")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("query parameters"))
		})

		It("rejects an invalid URL", func() {
			err := validateOpenidIssuerURL("not-a-url")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("valid OpenID issuer URL"))
		})

		It("rejects a URL with a fragment", func() {
			err := validateOpenidIssuerURL("https://accounts.google.com#section")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("valid OpenID issuer URL"))
		})
	})
})
