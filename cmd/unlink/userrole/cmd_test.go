package userrole

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
)

const (
	testAccountID = "acct-123"
	testRoleARN   = "arn:aws:iam::123456789012:role/ManagedOpenshift-User-Role"
)

var currentAccountResponse = `{
	"kind": "Account",
	"id": "acct-123",
	"username": "testuser",
	"organization": {
		"id": "org-123",
		"kind": "Organization"
	}
}`

var _ = Describe("unlink user-role", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		args = struct {
			roleArn   string
			accountID string
		}{}
		interactive.SetEnabled(false)
		Expect(Cmd.Flag("yes").Value.Set("false")).To(Succeed())
	})

	Context("runWithRuntime", func() {
		It("returns error when role ARN is empty", func() {
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, currentAccountResponse),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid user role ARN to unlink from the current account"))
		})

		It("successfully unlinks role when OCM API call succeeds", func() {
			args.roleArn = testRoleARN
			args.accountID = testAccountID
			Expect(Cmd.Flag("yes").Value.Set("true")).To(Succeed())

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, fmt.Sprintf(`{"key":"sts_user_role","value":"%s"}`, testRoleARN)),
				RespondWithJSON(http.StatusOK, `{}`),
			)

			stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, t.RosaRuntime, Cmd)
			Expect(err).NotTo(HaveOccurred())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring("Successfully unlinked role ARN"))
			Expect(stdout).To(ContainSubstring(testRoleARN))
			Expect(stdout).To(ContainSubstring(testAccountID))
		})
	})
})
