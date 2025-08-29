package imagemirror

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/reporter"
)

var _ = Describe("DeleteImageMirrorOptions", func() {
	var (
		imageMirrorOptions *DeleteImageMirrorOptions
		userOptions        *DeleteImageMirrorUserOptions
	)

	BeforeEach(func() {
		imageMirrorOptions = NewDeleteImageMirrorOptions()
		userOptions = NewDeleteImageMirrorUserOptions()
	})

	Context("NewDeleteImageMirrorUserOptions", func() {
		It("should create default user options", func() {
			Expect(userOptions.Id).To(Equal(""))
			Expect(userOptions.Yes).To(BeFalse())
		})
	})

	Context("NewDeleteImageMirrorOptions", func() {
		It("should create default image mirror options", func() {
			Expect(imageMirrorOptions.reporter).To(BeAssignableToTypeOf(&reporter.Object{}))
			Expect(imageMirrorOptions.args).To(BeAssignableToTypeOf(&DeleteImageMirrorUserOptions{}))
		})

		It("should initialize args with default values", func() {
			Expect(imageMirrorOptions.args.Id).To(Equal(""))
			Expect(imageMirrorOptions.args.Yes).To(BeFalse())
		})
	})

	Context("Args", func() {
		It("should return the args field", func() {
			imageMirrorOptions.args = userOptions
			Expect(imageMirrorOptions.Args()).To(Equal(userOptions))
		})

		It("should allow modification of args", func() {
			args := imageMirrorOptions.Args()
			args.Id = "test-mirror-id"
			args.Yes = true

			Expect(imageMirrorOptions.args.Id).To(Equal("test-mirror-id"))
			Expect(imageMirrorOptions.args.Yes).To(BeTrue())
		})
	})
})
