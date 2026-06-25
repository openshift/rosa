package region

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func TestRegion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Region Suite")
}

var _ = Describe("Region", func() {
	var (
		previousRegion    string
		previousAwsRegion string
	)

	BeforeEach(func() {
		previousRegion = region
		previousAwsRegion = os.Getenv("AWS_REGION")
		region = ""
		Expect(os.Setenv("AWS_REGION", "")).To(Succeed())
	})

	AfterEach(func() {
		region = previousRegion
		Expect(os.Setenv("AWS_REGION", previousAwsRegion)).To(Succeed())
	})

	It("registers the region flag with an empty default", func() {
		flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

		AddFlag(flags)

		flag := flags.Lookup("region")
		Expect(flag).NotTo(BeNil())
		Expect(flag.DefValue).To(Equal(""))
	})

	It("prefers the flag value over the environment variable", func() {
		region = "us-east-1"
		Expect(os.Setenv("AWS_REGION", "eu-west-1")).To(Succeed())

		Expect(Region()).To(Equal("us-east-1"))
	})

	It("falls back to the AWS_REGION environment variable", func() {
		Expect(os.Setenv("AWS_REGION", "eu-west-1")).To(Succeed())

		Expect(Region()).To(Equal("eu-west-1"))
	})

	It("returns an empty string when neither flag nor environment is set", func() {
		Expect(Region()).To(Equal(""))
	})

	It("treats an escaped empty flag value as unset", func() {
		region = "\"\""
		Expect(os.Setenv("AWS_REGION", "sa-east-1")).To(Succeed())

		Expect(Region()).To(Equal("sa-east-1"))
	})

	It("treats an escaped empty environment value as empty", func() {
		Expect(os.Setenv("AWS_REGION", "\"\"")).To(Succeed())

		Expect(Region()).To(Equal(""))
	})
})
