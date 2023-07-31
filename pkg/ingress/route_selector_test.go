package ingress

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetRouteSelector", func() {
	When("input is empty", func() {
		It("should return nil", func() {
			output, err := GetRouteSelector("")
			Expect(err).To(Not(HaveOccurred()))
			Expect(0).To(Equal(len(output)))
		})
	})
	When("input doesn't contain spaces", func() {
		It("should return correct map", func() {
			output, err := GetRouteSelector("foo=bar,bar=foo")
			Expect(err).To(Not(HaveOccurred()))
			Expect("map[bar:foo foo:bar]").To(Equal(fmt.Sprintf("%v", output)))
		})
	})
	When("input contain spaces", func() {
		It("should return correct map", func() {
			output, err := GetRouteSelector("foo=bar, bar=foo")
			Expect(err).To(Not(HaveOccurred()))
			Expect("map[bar:foo foo:bar]").To(Equal(fmt.Sprintf("%v", output)))
		})
	})
	When("input has wrong delimiter", func() {
		It("should return error", func() {
			_, err := GetRouteSelector("foo:bar, bar:foo")
			Expect(err).To(HaveOccurred())
			Expect("Expected key=value format for label-match").To(Equal(err.Error()))
		})
	})
})
