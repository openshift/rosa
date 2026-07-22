package debug

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func TestDebug(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Debug Suite")
}

var _ = Describe("Debug", func() {
	var previousEnabled bool

	BeforeEach(func() {
		previousEnabled = enabled
		enabled = false
	})

	AfterEach(func() {
		enabled = previousEnabled
	})

	It("registers the debug flag with false as the default", func() {
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

		AddFlag(flags)

		flag := flags.Lookup("debug")
		Expect(flag).NotTo(BeNil())
		Expect(flag.DefValue).To(Equal("false"))
	})

	It("enables debug mode when the parsed debug flag is true", func() {
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
		AddFlag(flags)

		Expect(flags.Parse([]string{"--debug"})).To(Succeed())

		Expect(Enabled()).To(BeTrue())
	})

	It("tracks debug mode through SetEnabled and Enabled", func() {
		Expect(Enabled()).To(BeFalse())

		SetEnabled(true)
		Expect(Enabled()).To(BeTrue())

		SetEnabled(false)
		Expect(Enabled()).To(BeFalse())
	})
})
