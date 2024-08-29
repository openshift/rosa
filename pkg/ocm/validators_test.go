package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Input Validators", Ordered, func() {
	Context("int validator", func() {
		It("succeeds if got an empty string", func() {
			Expect(Int32Validator("")).To(BeNil())
		})

		It("raises an error if it fails to parse the input into an integer", func() {
			Expect(Int32Validator("something")).ToNot(BeNil())
		})

		It("raises an error if it got a float", func() {
			Expect(Int32Validator("1.0")).ToNot(BeNil())
		})

		It("raises an error if it got an integer out of range", func() {
			Expect(Int32Validator("1152000000000")).ToNot(BeNil())
		})

		It("successfully parses an input that contains an integer", func() {
			Expect(Int32Validator("1")).To(BeNil())
		})
	})

	Context("non-negative int validator", func() {
		It("succeeds if got an empty string", func() {
			Expect(NonNegativeInt32Validator("")).To(BeNil())
		})

		It("raises an error if it fails to parse the input into an integer", func() {
			Expect(NonNegativeInt32Validator("something")).ToNot(BeNil())
		})

		It("raises an error if it got a float", func() {
			Expect(NonNegativeInt32Validator("1.0")).ToNot(BeNil())
		})

		It("raises an error if it got an integer out of range", func() {
			Expect(NonNegativeInt32Validator("1152000000000")).ToNot(BeNil())
		})

		It("raises an error if it got a negative number", func() {
			Expect(NonNegativeInt32Validator("-1")).ToNot(BeNil())
		})

		It("successfully parses an input that contains an integer", func() {
			Expect(NonNegativeInt32Validator("1")).To(BeNil())
		})
	})

	Context("duration string validator", func() {
		It("succeeds if got an empty string", func() {
			Expect(PositiveDurationStringValidator("")).To(BeNil())
		})

		It("raises an error if the format is wrong", func() {
			Expect(PositiveDurationStringValidator("something")).ToNot(BeNil())
		})

		It("raises an error if got a negative duration", func() {
			Expect(PositiveDurationStringValidator("-1h")).ToNot(BeNil())
		})

		It("successfully parses a valid duration string", func() {
			Expect(PositiveDurationStringValidator("200h")).To(BeNil())
		})
	})

	Context("percentage validator", func() {
		It("succeeds if got an empty string", func() {
			Expect(PercentageValidator("")).To(BeNil())
		})

		It("raises an error if didn't got a number", func() {
			Expect(PercentageValidator("something")).ToNot(BeNil())
		})

		It("raises an error if got a number higher than 1.0", func() {
			Expect(PercentageValidator("1.1")).ToNot(BeNil())
		})

		It("raises an error if got a number lower than 0.0", func() {
			Expect(PercentageValidator("-0.1")).ToNot(BeNil())
		})

		It("successfully parses a valid percentage value", func() {
			Expect(PercentageValidator("0.4")).To(BeNil())
		})
	})
})
