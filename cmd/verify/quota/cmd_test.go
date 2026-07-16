package quota

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("verify quota", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		Expect(os.Setenv("AWS_REGION", "us-east-1")).To(Succeed())
		DeferCleanup(os.Unsetenv, "AWS_REGION")
	})

	It("Succeeds when quota validation passes", func() {
		mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
		mockClient.EXPECT().ValidateQuota().Return(true, nil)

		err := runWithRuntime(t.RosaRuntime)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Prints no info messages when not running in a terminal", func() {
		mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
		mockClient.EXPECT().ValidateQuota().Return(true, nil)

		stdout, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(BeEmpty())
	})

	It("Returns error and prints failure details when ValidateQuota fails with error", func() {
		mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
		mockClient.EXPECT().ValidateQuota().Return(false, fmt.Errorf("not enough vCPU"))

		stdout, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Insufficient AWS quotas"))
		Expect(stderr).To(ContainSubstring("not enough vCPU"))
		Expect(stdout).To(BeEmpty())
	})

	It("Returns error when ValidateQuota returns false without error", func() {
		mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
		mockClient.EXPECT().ValidateQuota().Return(false, nil)

		stdout, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Insufficient AWS quotas"))
		Expect(stdout).To(BeEmpty())
	})

	It("Rejects extra arguments via cobra.NoArgs", func() {
		Expect(Cmd.Args).NotTo(BeNil())
		err := Cmd.Args(Cmd, []string{"unexpected"})
		Expect(err).To(HaveOccurred())
	})
})
