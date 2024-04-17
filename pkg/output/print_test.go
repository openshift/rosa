package output

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Print Functions test", func() {
	Context("PrintBool", func() {
		It("Prints true bool", func() {
			Expect(PrintBool(true)).To(Equal(Yes))
		})

		It("Prints false bool", func() {
			Expect(PrintBool(false)).To(Equal(No))
		})
	})

	Context("Print String Slice", func() {
		It("Returns empty string for empty slice", func() {
			Expect(PrintStringSlice(make([]string, 0))).To(Equal(EmptySlice))
		})

		It("Returns slice elements correctly separated", func() {
			values := []string{"foo", "bar"}
			Expect(PrintStringSlice(values)).To(Equal("foo, bar"))
		})
	})
})
