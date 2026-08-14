/*
Copyright (c) 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	Context("when URL has no scheme separator", func() {
		It("returns nil for URL without scheme", func() {
			err := ValidateURLCredentials("example.com")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns nil for URL with partial scheme", func() {
			err := ValidateURLCredentials("http:/example.com")
			Expect(err).NotTo(HaveOccurred())
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

var _ = Describe("Parse helpers", func() {
	Describe("ValidateHTTPSQueueURL", func() {
		It("accepts a valid HTTPS queue URL", func() {
			err := ValidateHTTPSQueueURL("https://sqs.us-east-1.amazonaws.com/123456789012/rosa-spot-queue")
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects malformed URLs", func() {
			err := ValidateHTTPSQueueURL("not-a-url")
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("rejects invalid queue URL shapes",
			func(queueURL string, expectedError string) {
				err := ValidateHTTPSQueueURL(queueURL)
				Expect(err).To(MatchError(expectedError))
			},
			Entry("non-https scheme", "http://sqs.us-east-1.amazonaws.com/123456789012/queue",
				"expect URL 'http://sqs.us-east-1.amazonaws.com/123456789012/queue' to use an 'https://' scheme"),
			Entry("missing host", "https:///123456789012/queue",
				"expect URL 'https:///123456789012/queue' to include a host"),
			Entry("userinfo", "https://user:pass@sqs.us-east-1.amazonaws.com/123456789012/queue",
				"expect URL 'https://user:pass@sqs.us-east-1.amazonaws.com/123456789012/queue' to not include user info"),
			Entry("query string", "https://sqs.us-east-1.amazonaws.com/123456789012/queue?debug=true",
				"expect URL 'https://sqs.us-east-1.amazonaws.com/123456789012/queue?debug=true' to not include a query string"),
			Entry("fragment", "https://sqs.us-east-1.amazonaws.com/123456789012/queue#anchor",
				"expect URL 'https://sqs.us-east-1.amazonaws.com/123456789012/queue#anchor' to not include a fragment"),
		)
	})

	Describe("Parse", func() {
		It("accepts a valid IPv6 host literal", func() {
			parsedURL, err := Parse("http://[::1]:8080")
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedURL.Host).To(Equal("[::1]:8080"))
		})

		It("rejects an IPv6 literal that is not at the start of the host", func() {
			_, err := Parse("http://example.com[::1]:8080")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid IP-literal"))
		})

		It("accepts a valid IPv6 host literal after userinfo", func() {
			parsedURL, err := Parse("http://user:pass@[::1]:8080")
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedURL.Host).To(Equal("[::1]:8080"))
		})
	})

	Describe("ParseRequestURI", func() {
		It("accepts a valid absolute path", func() {
			parsedURL, err := ParseRequestURI("/api/clusters")
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedURL.Path).To(Equal("/api/clusters"))
		})

		It("rejects an IPv6 literal that is not at the start of the host", func() {
			_, err := ParseRequestURI("https://api.example.com[::1]")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid IP-literal"))
		})
	})
})
