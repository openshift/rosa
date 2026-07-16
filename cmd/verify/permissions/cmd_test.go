package permissions

import (
	"fmt"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

const stsPoliciesResponse = `{
	"kind": "AWSSTSPolicyList",
	"page": 1,
	"size": 0,
	"total": 0,
	"items": []
}`

var _ = Describe("verify permissions", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		os.Setenv("AWS_REGION", "us-east-1")
		DeferCleanup(os.Unsetenv, "AWS_REGION")
	})

	It("Succeeds when SCP validation passes", func() {
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, stsPoliciesResponse),
		)
		mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
		mockClient.EXPECT().ValidateSCP(nil, map[string]*cmv1.AWSSTSPolicy{}).Return(true, nil)

		stdout, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(ContainSubstring("AWS SCP policies ok"))
	})

	It("Returns error when GetPolicies API fails", func() {
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusInternalServerError, "{}"),
		)

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Failed to get 'osdscppolicy'"))
	})

	It("Returns error with throttling message when rate exceeded", func() {
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, stsPoliciesResponse),
		)
		mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
		mockClient.EXPECT().ValidateSCP(nil, map[string]*cmv1.AWSSTSPolicy{}).
			Return(false, fmt.Errorf("Throttling: Rate exceeded"))

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Unable to validate SCP policies"))
		Expect(stderr).To(ContainSubstring("Throttling: Rate exceeded. Please wait 3-5 minutes"))
	})

	It("Returns error with details when ValidateSCP fails without throttling", func() {
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, stsPoliciesResponse),
		)
		mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
		mockClient.EXPECT().ValidateSCP(nil, map[string]*cmv1.AWSSTSPolicy{}).
			Return(false, fmt.Errorf("access denied for policy simulation"))

		_, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Unable to validate SCP policies"))
		Expect(stderr).To(ContainSubstring("access denied for policy simulation"))
	})

	It("Continues with warning when ValidateSCP returns ok=false without error", func() {
		t.ApiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, stsPoliciesResponse),
		)
		mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
		mockClient.EXPECT().ValidateSCP(nil, map[string]*cmv1.AWSSTSPolicy{}).Return(false, nil)

		stdout, stderr, err := test.RunWithOutputCapture(
			func(r *rosa.Runtime, _ *cobra.Command) error {
				return runWithRuntime(r)
			}, t.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(ContainSubstring("Failed to validate SCP policies. Will try to continue anyway"))
		Expect(stdout).To(ContainSubstring("AWS SCP policies ok"))
	})

	It("Rejects extra arguments via cobra.NoArgs", func() {
		Expect(Cmd.Args).NotTo(BeNil())
		err := Cmd.Args(Cmd, []string{"unexpected"})
		Expect(err).To(HaveOccurred())
	})

	It("Has scp as a command alias", func() {
		Expect(Cmd.Aliases).To(ContainElement("scp"))
	})
})
