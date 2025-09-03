package imagemirror

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/reporter"
)

var _ = Describe("EditImageMirrorOptions", func() {
	var (
		imageMirrorOptions *EditImageMirrorOptions
		userOptions        *EditImageMirrorUserOptions
	)

	BeforeEach(func() {
		imageMirrorOptions = NewEditImageMirrorOptions()
		userOptions = NewEditImageMirrorUserOptions()
	})

	Context("NewEditImageMirrorUserOptions", func() {
		It("should create default user options", func() {
			Expect(userOptions.Id).To(Equal(""))
			Expect(userOptions.Mirrors).To(BeEmpty())
		})
	})

	Context("NewEditImageMirrorOptions", func() {
		It("should create default image mirror options", func() {
			Expect(imageMirrorOptions.reporter).To(BeAssignableToTypeOf(&reporter.Object{}))
			Expect(imageMirrorOptions.args).To(BeAssignableToTypeOf(&EditImageMirrorUserOptions{}))
		})

		It("should initialize args with default values", func() {
			Expect(imageMirrorOptions.args.Id).To(Equal(""))
			Expect(imageMirrorOptions.args.Mirrors).To(BeEmpty())
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
			args.Mirrors = []string{"new-mirror1.com", "new-mirror2.com"}

			Expect(imageMirrorOptions.args.Id).To(Equal("test-mirror-id"))
			Expect(imageMirrorOptions.args.Mirrors).To(Equal([]string{"new-mirror1.com", "new-mirror2.com"}))
		})
	})
})
