package cache

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConvertToStringSlice", func() {
	When("input is a slice of strings containing semantic versions", func() {
		It("should extract and return the strings", func() {
			input := []string{"1.0.0", "2.0.0", "3.0.0"}
			extracted, ok, err := ConvertToStringSlice(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(extracted).To(Equal(input))
		})
	})

	When("input is a slice containing non-string values", func() {
		It("should only extract string values", func() {
			input := []interface{}{"1.0.0", 2, "3.0.0", 4}
			expected := []string{"1.0.0", "3.0.0"}
			extracted, ok, err := ConvertToStringSlice(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(extracted).To(Equal(expected))
		})
	})

	When("input is not a slice", func() {
		It("should return an error", func() {
			input := "not a slice"
			_, ok, err := ConvertToStringSlice(input)
			Expect(err).To(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(err.Error()).To(Equal("input is not a slice"))
		})
	})

	When("input contains values with invalid kinds", func() {
		It("should skip those values and return the rest", func() {
			input := []interface{}{"1.0.0", 2, "3.0.0", 4, complex(1, 2)}
			expected := []string{"1.0.0", "3.0.0"}
			extracted, ok, err := ConvertToStringSlice(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(extracted).To(Equal(expected))
		})
	})
})

var _ = Describe("pathExists", func() {
	It("should correctly identify non-existing path", func() {
		exists, err := pathExists("/non-existing-path")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
	})

	It("should correctly identify existing path", func() {
		tmpDir := os.TempDir()
		exists, err := pathExists(tmpDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
	})
})
