package color

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

func TestColor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Color Suite")
}

var _ = Describe("Color", func() {
	var previousColor string

	BeforeEach(func() {
		previousColor = color
		color = ""
	})

	AfterEach(func() {
		color = previousColor
	})

	It("registers the color flag with auto as default", func() {
		cmd := &cobra.Command{Use: "test"}

		AddFlag(cmd)

		flag := cmd.PersistentFlags().Lookup("color")
		Expect(flag).NotTo(BeNil())
		Expect(flag.DefValue).To(Equal("auto"))
	})

	It("returns the supported completion options", func() {
		values, directive := completion(&cobra.Command{Use: "test"}, nil, "")

		Expect(values).To(Equal([]string{"auto", "never", "always"}))
		Expect(directive).To(Equal(cobra.ShellCompDirectiveDefault))
	})

	It("disables color when set to never", func() {
		SetColor("never")
		Expect(UseColor()).To(BeFalse())
	})

	It("enables color when set to always", func() {
		SetColor("always")
		Expect(UseColor()).To(BeTrue())
	})

	It("treats unknown values the same as auto", func() {
		SetColor("auto")
		expected := UseColor()

		SetColor("unexpected")

		Expect(UseColor()).To(Equal(expected))
	})
})
