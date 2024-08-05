package version

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RosaVersionOptions", func() {
	var (
		o            *RosaVersionOptions
		userOptions  RosaVersionUserOptions
		expectedArgs RosaVersionUserOptions
	)

	Describe("BindAndValidate", func() {
		When("valid options are provided", func() {
			It("should correctly bind and validate the options", func() {
				o = &RosaVersionOptions{}
				userOptions = RosaVersionUserOptions{
					verbose:    true,
					clientOnly: true,
				}
				expectedArgs = RosaVersionUserOptions{
					verbose:    true,
					clientOnly: true,
				}
				o.BindAndValidate(&userOptions)
				Expect(o.args).To(Equal(&expectedArgs))
			})
		})

		When("empty options are provided", func() {
			It("should not change the default options", func() {
				o = &RosaVersionOptions{}
				o.BindAndValidate(&RosaVersionUserOptions{})
				Expect(o.args).To(Equal(&RosaVersionUserOptions{}))
			})
		})
	})
})
