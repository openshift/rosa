package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Input Validators", Ordered, func() {
	Context("int validator", func() {
		It("succeeds if got an empty string", func() {
			Expect(IntValidator("")).To(BeNil())
		})

		It("raises an error if it fails to parse the input into an integer", func() {
			Expect(IntValidator("something")).ToNot(BeNil())
		})

		It("raises an error if it got a float", func() {
			Expect(IntValidator("1.0")).ToNot(BeNil())
		})

		It("successfully parses an input that contains an integer", func() {
			Expect(IntValidator("1")).To(BeNil())
		})
	})

	Context("non-negative int validator", func() {
		It("succeeds if got an empty string", func() {
			Expect(NonNegativeIntValidator("")).To(BeNil())
		})

		It("raises an error if it fails to parse the input into an integer", func() {
			Expect(NonNegativeIntValidator("something")).ToNot(BeNil())
		})

		It("raises an error if it got a float", func() {
			Expect(NonNegativeIntValidator("1.0")).ToNot(BeNil())
		})

		It("raises an error if it got a negative number", func() {
			Expect(NonNegativeIntValidator("-1")).ToNot(BeNil())
		})

		It("successfully parses an input that contains an integer", func() {
			Expect(NonNegativeIntValidator("1")).To(BeNil())
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
