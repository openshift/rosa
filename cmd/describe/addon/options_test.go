package addon

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test delete addon options", func() {
	var args *DescribeAddonUserOptions
	var options *DescribeAddonOptions
	BeforeEach(func() {
		options = NewDescribeAddonOptions()
	})
	Context("Describe Addon User Options", func() {
		It("Creates default options", func() {
			args = NewDescribeAddonUserOptions()
			Expect(args.addon).To(Equal(""))
		})
	})
	Context("Describe Addon Options", func() {
		It("Create args from options using Bind (also tests MachinePool func)", func() {
			// Set value then bind
			testAddon := "test"
			args.addon = testAddon
			err := options.Bind(args, []string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(options.Addon()).To(Equal(testAddon))
		})
		It("Fail to bind args due to empty addon name", func() {
			args.addon = ""
			err := options.Bind(args, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("You need to specify a machine pool name"))
		})
		It("Fail to bind args due to invalid addon name", func() {
			args.addon = "%asd"
			err := options.Bind(args, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected a valid identifier for the machine pool"))
			args.addon = "1asd"
			err = options.Bind(args, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected a valid identifier for the machine pool"))
			args.addon = "asd123$"
			err = options.Bind(args, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected a valid identifier for the machine pool"))
		})
		It("Test Bind with argv instead of normal args (single arg, no flag for addon)", func() {
			argv := []string{"test-id"}
			args.addon = ""
			options.Bind(args, argv)
			Expect(options.Addon()).To(Equal(argv[0]))
		})
	})
})
