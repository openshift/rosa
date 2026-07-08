package idp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete IDP", func() {
	Context("Args validator", func() {
		It("rejects zero arguments", func() {
			err := Cmd.Args(nil, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected exactly one"))
		})

		It("accepts exactly one argument", func() {
			err := Cmd.Args(nil, []string{"github-1"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects two arguments", func() {
			err := Cmd.Args(nil, []string{"github-1", "extra"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected exactly one"))
		})
	})
})
