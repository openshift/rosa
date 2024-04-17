package interactive

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("containsString", func() {
	var (
		s     []string
		input string
	)

	BeforeEach(func() {
		s = []string{"a", "b", "c"}
	})

	Context("when the string is present", func() {
		It("should return true", func() {
			input = "a"
			Expect(containsString(s, input)).To(BeTrue())
		})
	})

	Context("when the string is not present", func() {
		It("should return false", func() {
			input = "d"
			Expect(containsString(s, input)).To(BeFalse())
		})
	})

	Context("when the slice is empty", func() {
		It("should return false", func() {
			s = []string{}
			input = "d"
			Expect(containsString(s, input)).To(BeFalse())
		})
	})

	Context("when the input string is empty", func() {
		It("should return false", func() {
			input = ""
			Expect(containsString(s, input)).To(BeFalse())
		})
	})
})
