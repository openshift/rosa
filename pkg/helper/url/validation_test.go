package url

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestURLValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "URL Validation Suite")
}

var _ = Describe("ValidateURLCredentials", func() {
	Context("when URL is missing scheme", func() {
		It("returns error for URL without scheme", func() {
			err := ValidateURLCredentials("example.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("URL is missing scheme (expected '://')"))
		})

		It("returns error for URL with partial scheme", func() {
			err := ValidateURLCredentials("http:/example.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("URL is missing scheme (expected '://')"))
		})
	})

	Context("when URL has no credentials", func() {
		It("returns nil for URL without @", func() {
			err := ValidateURLCredentials("http://example.com")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns nil for URL with port but no credentials", func() {
			err := ValidateURLCredentials("http://example.com:8080")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when URL has valid credentials", func() {
		It("returns nil for URL with username only", func() {
			err := ValidateURLCredentials("http://user@example.com")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns nil for URL with username and password", func() {
			err := ValidateURLCredentials("http://user:pass@example.com")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns nil for URL with empty password", func() {
			err := ValidateURLCredentials("http://user:@example.com")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when username contains invalid characters", func() {
		DescribeTable("returns error for invalid username character",
			func(url string, expectedChar rune) {
				err := ValidateURLCredentials(url)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("username contains invalid character '" + string(expectedChar) + "'"))
			},
			Entry("slash in username", "http://us/er:pass@example.com", '/'),
			Entry("question mark in username", "http://us?er:pass@example.com", '?'),
			Entry("hash in username", "http://us#er:pass@example.com", '#'),
		)
	})

	Context("when password contains invalid characters", func() {
		DescribeTable("returns error for invalid password character",
			func(url string, expectedChar rune) {
				err := ValidateURLCredentials(url)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("password contains invalid character '" + string(expectedChar) + "'"))
			},
			Entry("slash in password", "http://user:pa/ss@example.com", '/'),
			Entry("question mark in password", "http://user:pa?ss@example.com", '?'),
			Entry("hash in password", "http://user:pa#ss@example.com", '#'),
			Entry("bracket in password", "http://user:pa[ss@example.com", '['),
		)
	})

	Context("when URL has multiple @ signs", func() {
		It("returns error indicating @ in password", func() {
			err := ValidateURLCredentials("http://user:p@ss@example.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("password contains invalid character '@'"))
		})
	})
})
