package interactive

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation", func() {
	Context("Min", func() {
		It("Fails validation if the answer is less than the minimum", func() {
			validator := Min(50)
			err := validator("25")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'25' is less than the permitted minimum of '50'"))
		})

		It("Fails validation if the answer is not an integer", func() {
			validator := Min(50)
			err := validator("hello")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("please enter an integer value, you entered 'hello'"))
		})

		It("Fails validation if the answer is not a string", func() {
			validator := Min(50)
			err := validator(45)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings, got int"))
		})

		It("Passes validation if the answer is greater than the min", func() {
			validator := Min(50)
			err := validator("55")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes validation if the answer is equal to the min", func() {
			validator := Min(50)
			err := validator("50")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
