package bootstrap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bsOpts "github.com/openshift/rosa/pkg/options/bootstrap"
	"github.com/openshift/rosa/pkg/reporter"
)

var _ = Describe("CreateMachinepoolOptions", func() {
	var (
		bootstrapOptions *BootstrapOptions
		userOptions      *bsOpts.BootstrapUserOptions
	)

	BeforeEach(func() {
		bootstrapOptions = NewBootstrapOptions()
		userOptions = NewBootstrapUserOptions()
	})

	Context("NewBootstrapUserOptions", func() {
		It("should create default user options", func() {
			Expect(userOptions.Params).To(Equal([]string{}))
		})
	})

	Context("NewBootstrapOptions", func() {
		It("should create default bootstrap options", func() {
			Expect(bootstrapOptions.reporter).To(BeAssignableToTypeOf(&reporter.Object{}))
			Expect(bootstrapOptions.args).To(BeAssignableToTypeOf(&bsOpts.BootstrapUserOptions{}))
		})
	})

	Context("Bootstrap", func() {
		It("should return the args field", func() {
			bootstrapOptions.args = userOptions
			Expect(bootstrapOptions.Bootstrap()).To(Equal(userOptions))
		})
	})
})
