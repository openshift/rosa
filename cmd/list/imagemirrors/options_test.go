package imagemirrors

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/reporter"
)

var _ = Describe("ListImageMirrorsOptions", func() {
	var (
		imageMirrorsOptions *ListImageMirrorsOptions
		userOptions         *ListImageMirrorsUserOptions
	)

	BeforeEach(func() {
		imageMirrorsOptions = NewListImageMirrorsOptions()
		userOptions = NewListImageMirrorsUserOptions()
	})

	Context("NewListImageMirrorsUserOptions", func() {
		It("should create default user options", func() {
			Expect(userOptions).To(BeAssignableToTypeOf(&ListImageMirrorsUserOptions{}))
		})
	})

	Context("NewListImageMirrorsOptions", func() {
		It("should create default image mirrors options", func() {
			Expect(imageMirrorsOptions.reporter).To(BeAssignableToTypeOf(&reporter.Object{}))
			Expect(imageMirrorsOptions.args).To(BeAssignableToTypeOf(&ListImageMirrorsUserOptions{}))
		})
	})

	Context("Args", func() {
		It("should return the args field", func() {
			imageMirrorsOptions.args = userOptions
			Expect(imageMirrorsOptions.Args()).To(Equal(userOptions))
		})
	})
})
