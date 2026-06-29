package profile

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func TestProfile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Profile Suite")
}

var _ = Describe("Profile", func() {
	var (
		previousProfile    string
		previousAwsProfile string
	)

	BeforeEach(func() {
		previousProfile = profile
		previousAwsProfile = os.Getenv("AWS_PROFILE")
		profile = ""
		Expect(os.Setenv("AWS_PROFILE", "")).To(Succeed())
	})

	AfterEach(func() {
		profile = previousProfile
		Expect(os.Setenv("AWS_PROFILE", previousAwsProfile)).To(Succeed())
	})

	It("registers the profile flag with an empty default", func() {
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

		AddFlag(flags)

		flag := flags.Lookup("profile")
		Expect(flag).NotTo(BeNil())
		Expect(flag.DefValue).To(Equal(""))
	})

	It("prefers the flag value over the environment variable", func() {
		profile = "flag-profile"
		Expect(os.Setenv("AWS_PROFILE", "env-profile")).To(Succeed())

		Expect(Profile()).To(Equal("flag-profile"))
	})

	It("falls back to the AWS_PROFILE environment variable", func() {
		Expect(os.Setenv("AWS_PROFILE", "env-profile")).To(Succeed())

		Expect(Profile()).To(Equal("env-profile"))
	})

	It("returns an empty string when neither flag nor environment is set", func() {
		Expect(Profile()).To(Equal(""))
	})
})
