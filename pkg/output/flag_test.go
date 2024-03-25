package output

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Output flag", func() {

	BeforeEach(func() {
		SetOutput("")
	})

	AfterEach(func() {
		SetOutput("")
	})

	It("Adds flag to command", func() {

		cmd := &cobra.Command{}
		Expect(cmd.Flag(FLAG_NAME)).To(BeNil())

		AddFlag(cmd)

		flag := cmd.Flag(FLAG_NAME)
		Expect(flag).NotTo(BeNil())
		Expect(flag.Name).To(Equal(FLAG_NAME))
		Expect(flag.Shorthand).To(Equal(FLAG_SHORTHAND))
		Expect(flag.Value.String()).To(Equal(""))
		Expect(flag.Usage).To(Equal("Output format. Allowed formats are [json yaml]"))
	})

	It("Has a completion function", func() {
		args, directive := completion(nil, nil, "")
		Expect(len(args)).To(Equal(2))
		Expect(args).To(ContainElements(JSON, YAML))

		Expect(directive).To(Equal(cobra.ShellCompDirectiveDefault))
	})

	It("Has flag", func() {
		Expect(HasFlag()).To(BeFalse())
		SetOutput(JSON)
		Expect(HasFlag()).To(BeTrue())
		Expect(Output()).To(Equal(JSON))
	})

	It("Does not have flag", func() {
		Expect(HasFlag()).To(BeFalse())
	})

})
