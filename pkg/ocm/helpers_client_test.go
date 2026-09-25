// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"encoding/json"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/reporter"
)

// fakeReporter records warnings/debug messages for assertion, without printing anything.
type fakeReporter struct {
	warnings []string
	debugs   []string
}

func (f *fakeReporter) Errorf(format string, args ...any) error { return fmt.Errorf(format, args...) }
func (f *fakeReporter) Warnf(format string, args ...any) {
	f.warnings = append(f.warnings, fmt.Sprintf(format, args...))
}
func (f *fakeReporter) Infof(format string, args ...any) {}
func (f *fakeReporter) Debugf(format string, args ...any) {
	f.debugs = append(f.debugs, fmt.Sprintf(format, args...))
}
func (f *fakeReporter) IsTerminal() bool { return false }

var _ reporter.Logger = &fakeReporter{}

var _ = Describe("Helpers API client behavior", func() {
	var apiServer *ghttp.Server
	var ocmClient *Client
	var expectedRequests int

	appendHandlers := func(requestCount int, handlers ...http.HandlerFunc) {
		expectedRequests += requestCount
		apiServer.AppendHandlers(handlers...)
	}

	BeforeEach(func() {
		apiServer = MakeTCPServer()
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)
		ocmClient = buildTestOCMClient(apiServer.URL())
		expectedRequests = 0
	})

	AfterEach(func() {
		Expect(apiServer.ReceivedRequests()).To(HaveLen(expectedRequests))
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("handles missing current account in GetCurrentOrganization", func() {
		appendHandlers(1,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusNotFound, `{"reason":"not found"}`),
			),
		)

		var (
			id         string
			externalID string
			err        error
		)
		Expect(func() {
			id, externalID, err = ocmClient.GetCurrentOrganization()
		}).NotTo(Panic())

		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(BeEmpty())
		Expect(externalID).To(BeEmpty())
	})

	It("returns current account body when GetCurrentAccount succeeds", func() {
		appendHandlers(1,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1","external_id":"ext-1"}
				}`),
			),
		)

		account, err := ocmClient.GetCurrentAccount()
		Expect(err).NotTo(HaveOccurred())
		Expect(account).NotTo(BeNil())
		Expect(account.ID()).To(Equal("acct-1"))
		Expect(account.Organization().ID()).To(Equal("org-1"))
		Expect(account.Organization().ExternalID()).To(Equal("ext-1"))
	})

	It("returns organization identifiers when GetCurrentOrganization succeeds", func() {
		appendHandlers(1,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1","external_id":"ext-1"}
				}`),
			),
		)

		id, externalID, err := ocmClient.GetCurrentOrganization()
		Expect(err).NotTo(HaveOccurred())
		Expect(id).To(Equal("org-1"))
		Expect(externalID).To(Equal("ext-1"))
	})

	It("links a role to organization labels when account is not present yet", func() {
		roleARN := "arn:aws:iam::123456789012:role/first-role"
		appendHandlers(2,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK, `{"key":"sts_ocm_role","value":""}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/accounts_mgmt/v1/organizations/org-1/labels"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]string{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("key", "sts_ocm_role"))
					Expect(payload).To(HaveKeyWithValue("value", roleARN))
				},
				RespondWithJSON(http.StatusCreated, `{"key":"sts_ocm_role","value":"arn:aws:iam::123456789012:role/first-role"}`),
			),
		)

		linked, err := ocmClient.LinkOrgToRole("org-1", roleARN)
		Expect(err).NotTo(HaveOccurred())
		Expect(linked).To(BeTrue())
	})

	It("rejects linking a second role for the same aws account", func() {
		appendHandlers(1,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK, `{"key":"sts_ocm_role","value":"arn:aws:iam::123456789012:role/existing-role"}`),
			),
		)

		linked, err := ocmClient.LinkOrgToRole("org-1", "arn:aws:iam::123456789012:role/new-role")
		Expect(err).To(HaveOccurred())
		Expect(linked).To(BeFalse())
		Expect(err.Error()).To(ContainSubstring("Only one role can be linked per AWS account per organization"))
	})

	It("treats linking the same organization role as idempotent", func() {
		roleARN := "arn:aws:iam::123456789012:role/existing-role"
		appendHandlers(1,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK, `{"key":"sts_ocm_role","value":"arn:aws:iam::123456789012:role/existing-role"}`),
			),
		)

		linked, err := ocmClient.LinkOrgToRole("org-1", roleARN)
		Expect(err).NotTo(HaveOccurred())
		Expect(linked).To(BeFalse())
	})

	It("unlinks organization ocm role by deleting label when last entry is removed", func() {
		roleARN := "arn:aws:iam::123456789012:role/only-role"
		appendHandlers(2,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK, `{"key":"sts_ocm_role","value":"arn:aws:iam::123456789012:role/only-role"}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodDelete, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK, `{}`),
			),
		)

		err := ocmClient.UnlinkOCMRoleFromOrg("org-1", roleARN)
		Expect(err).NotTo(HaveOccurred())
	})

	It("avoids creating duplicate user role links on an account", func() {
		appendHandlers(1,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/accounts/acct-1/labels/sts_user_role"),
				RespondWithJSON(http.StatusOK, `{"key":"sts_user_role","value":"arn:aws:iam::123456789012:role/existing"}`),
			),
		)

		err := ocmClient.LinkAccountRole("acct-1", "arn:aws:iam::123456789012:role/existing")
		Expect(err).NotTo(HaveOccurred())
	})

	It("does not warn when a properly formatted OCM role is linked to the AWS account", func() {
		appendHandlers(2,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1","external_id":"ext-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK,
					`{"key":"sts_ocm_role","value":"arn:aws:iam::123456789012:role/prefix-OCM-Role-ext-1"}`),
			),
		)

		fake := &fakeReporter{}
		ocmClient.WarnIfOCMRoleNotLinked(fake, "123456789012")

		Expect(fake.warnings).To(BeEmpty())
	})

	It("warns when a role is linked to the account but its ARN doesn't match "+
		"the -OCM-Role-$EXTERNAL_ORG_ID format for the current org", func() {
		appendHandlers(2,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1","external_id":"ext-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK,
					`{"key":"sts_ocm_role","value":"arn:aws:iam::123456789012:role/legacy-role-name"}`),
			),
		)

		fake := &fakeReporter{}
		ocmClient.WarnIfOCMRoleNotLinked(fake, "123456789012")

		Expect(fake.warnings).To(HaveLen(1))
		Expect(fake.warnings[0]).To(ContainSubstring("Missing or unlinked OCM role"))
	})

	It("warns when the AWS account has no OCM role linked", func() {
		appendHandlers(2,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1","external_id":"ext-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK,
					`{"key":"sts_ocm_role","value":"arn:aws:iam::999999999999:role/other-account-role"}`),
			),
		)

		fake := &fakeReporter{}
		ocmClient.WarnIfOCMRoleNotLinked(fake, "123456789012")

		Expect(fake.warnings).To(HaveLen(1))
		Expect(fake.warnings[0]).To(ContainSubstring("Missing or unlinked OCM role"))
		Expect(fake.warnings[0]).To(ContainSubstring("https://access.redhat.com/articles/7137057"))
	})

	It("warns with account-scoped wording when no cluster exists yet", func() {
		appendHandlers(2,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1","external_id":"ext-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusOK,
					`{"key":"sts_ocm_role","value":"arn:aws:iam::999999999999:role/other-account-role"}`),
			),
		)

		fake := &fakeReporter{}
		ocmClient.WarnIfOCMRoleNotLinkedForAccount(fake, "123456789012")

		Expect(fake.warnings).To(HaveLen(1))
		Expect(fake.warnings[0]).NotTo(ContainSubstring("this cluster"))
		Expect(fake.warnings[0]).To(ContainSubstring("account 123456789012"))
		Expect(fake.warnings[0]).To(ContainSubstring("https://access.redhat.com/articles/7137057"))
	})

	It("does not warn or fail when the org labels lookup is forbidden", func() {
		appendHandlers(2,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/current_account"),
				RespondWithJSON(http.StatusOK, `{
					"id":"acct-1",
					"organization":{"id":"org-1","external_id":"ext-1"}
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/organizations/org-1/labels/sts_ocm_role"),
				RespondWithJSON(http.StatusForbidden, `{
					"kind": "Error",
					"id": "403",
					"href": "/api/accounts_mgmt/v1/errors/403",
					"code": "ACCT-MGMT-403",
					"reason": "forbidden"
				}`),
			),
		)

		fake := &fakeReporter{}
		Expect(func() {
			ocmClient.WarnIfOCMRoleNotLinked(fake, "123456789012")
		}).NotTo(Panic())

		Expect(fake.warnings).To(BeEmpty())
	})

	It("unlinks a user role from account labels and updates remaining roles", func() {
		appendHandlers(2,
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/accounts/acct-1/labels/sts_user_role"),
				RespondWithJSON(http.StatusOK, `{
					"key":"sts_user_role",
					"value":"arn:aws:iam::123456789012:role/remove,arn:aws:iam::123456789012:role/keep"
				}`),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPatch, "/api/accounts_mgmt/v1/accounts/acct-1/labels/sts_user_role"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]string{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("key", "sts_user_role"))
					Expect(payload).To(HaveKeyWithValue("value", "arn:aws:iam::123456789012:role/keep"))
				},
				RespondWithJSON(http.StatusOK, `{
					"key":"sts_user_role",
					"value":"arn:aws:iam::123456789012:role/keep"
				}`),
			),
		)

		err := ocmClient.UnlinkUserRoleFromAccount("acct-1", "arn:aws:iam::123456789012:role/remove")
		Expect(err).NotTo(HaveOccurred())
	})
})
