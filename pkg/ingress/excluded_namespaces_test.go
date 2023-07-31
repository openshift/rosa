package ingress

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetExcludedNamespaces", func() {
	When("input is empty", func() {
		It("should return empty slice", func() {
			output := GetExcludedNamespaces("")
			Expect(0).To(Equal(len(output)))
		})
	})
	When("input doesn't contain spaces", func() {
		It("should return correct array", func() {
			output := GetExcludedNamespaces("stage,dev,int")
			Expect([]string{"stage", "dev", "int"}).To(Equal(output))
		})
	})
	When("input contain spaces", func() {
		It("should return correct array", func() {
			output := GetExcludedNamespaces("stage, dev, int")
			Expect([]string{"stage", "dev", "int"}).To(Equal(output))
		})
	})
})
