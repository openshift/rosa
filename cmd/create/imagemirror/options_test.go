package imagemirror

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/reporter"
)

var _ = Describe("CreateImageMirrorOptions", func() {
	var (
		imageMirrorOptions *CreateImageMirrorOptions
		userOptions        *CreateImageMirrorUserOptions
	)

	BeforeEach(func() {
		imageMirrorOptions = NewCreateImageMirrorOptions()
		userOptions = NewCreateImageMirrorUserOptions()
	})

	Context("NewCreateImageMirrorUserOptions", func() {
		It("should create default user options", func() {
			Expect(userOptions.Type).To(Equal("digest"))
			Expect(userOptions.Source).To(Equal(""))
			Expect(userOptions.Mirrors).To(BeEmpty())
		})
	})

	Context("NewCreateImageMirrorOptions", func() {
		It("should create default image mirror options", func() {
			Expect(imageMirrorOptions.reporter).To(BeAssignableToTypeOf(&reporter.Object{}))
			Expect(imageMirrorOptions.args).To(BeAssignableToTypeOf(&CreateImageMirrorUserOptions{}))
		})

		It("should initialize args with default values", func() {
			Expect(imageMirrorOptions.args.Type).To(Equal("digest"))
			Expect(imageMirrorOptions.args.Source).To(Equal(""))
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
			args.Type = "tag"
			args.Source = "registry.example.com/team"
			args.Mirrors = []string{"mirror1.com", "mirror2.com"}

			Expect(imageMirrorOptions.args.Type).To(Equal("tag"))
			Expect(imageMirrorOptions.args.Source).To(Equal("registry.example.com/team"))
			Expect(imageMirrorOptions.args.Mirrors).To(Equal([]string{"mirror1.com", "mirror2.com"}))
		})
	})
})
