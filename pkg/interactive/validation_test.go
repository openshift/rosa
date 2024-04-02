package interactive

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation", func() {
	Context("MinValue", func() {
		It("Fails validation if the answer is less than the minimum", func() {
			validator := MinValue(50)
			err := validator("25")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'25' is less than the permitted minimum of '50'"))
		})

		It("Fails validation if the answer is not an integer", func() {
			validator := MinValue(50)
			err := validator("hello")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("please enter an integer value, you entered 'hello'"))
		})

		It("Fails validation if the answer is not a string", func() {
			validator := MinValue(50)
			err := validator(45)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings, got int"))
		})

		It("Passes validation if the answer is greater than the min", func() {
			validator := MinValue(50)
			err := validator("55")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes validation if the answer is equal to the min", func() {
			validator := MinValue(50)
			err := validator("50")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("MaxValue", func() {
		It("Fails validation if the answer is greater than the maximum", func() {
			validator := MaxValue(50)
			err := validator("52")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'52' is greater than the permitted maximum of '50'"))
		})

		It("Fails validation if the answer is not an integer", func() {
			validator := MaxValue(50)
			err := validator("hello")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("please enter an integer value, you entered 'hello'"))
		})

		It("Fails validation if the answer is not a string", func() {
			validator := MaxValue(50)
			err := validator(45)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings, got int"))
		})

		It("Passes validation if the answer is less than the max", func() {
			validator := MaxValue(50)
			err := validator("49")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes validation if the answer is equal to the max", func() {
			validator := MaxValue(50)
			err := validator("50")
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Context("GitHub Hostname", func() {
		It("Fails validation if hostname is 'https://github.com'", func() {
			err := IsValidHostname("https://github.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"'https://github.com' hostname must be a valid DNS subdomain or IP address"),
			)
		})
		It("Passes validation if hostname is 'domain.customer.com'", func() {
			err := IsValidHostname("domain.customer.com")
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
