package ocm

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("Helpers API client behavior", func() {
	var apiServer *ghttp.Server
	var ocmClient *Client

	BeforeEach(func() {
		apiServer = MakeTCPServer()
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)
		ocmClient = buildTestOCMClient(apiServer.URL())
	})

	AfterEach(func() {
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("handles missing current account in GetCurrentOrganization", func() {
		apiServer.AppendHandlers(
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
		apiServer.AppendHandlers(
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
	})

	It("links a role to organization labels when account is not present yet", func() {
		roleARN := "arn:aws:iam::123456789012:role/first-role"
		apiServer.AppendHandlers(
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
		apiServer.AppendHandlers(
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

	It("unlinks organization ocm role by deleting label when last entry is removed", func() {
		roleARN := "arn:aws:iam::123456789012:role/only-role"
		apiServer.AppendHandlers(
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
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/accounts/acct-1/labels/sts_user_role"),
				RespondWithJSON(http.StatusOK, `{"key":"sts_user_role","value":"arn:aws:iam::123456789012:role/existing"}`),
			),
		)

		err := ocmClient.LinkAccountRole("acct-1", "arn:aws:iam::123456789012:role/existing")
		Expect(err).NotTo(HaveOccurred())
	})

	It("unlinks a user role from account labels and updates remaining roles", func() {
		apiServer.AppendHandlers(
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
