package ocm

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

var _ = Describe("Users", Ordered, func() {
	const (
		clusterID = "cluster-1"
		groupID   = "dedicated-admins"
		username  = "alice"
	)

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

	It("gets an existing user", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users/alice"),
				RespondWithJSON(http.StatusOK, `{"id":"user-1"}`),
			),
		)

		user, err := ocmClient.GetUser(clusterID, groupID, username)
		Expect(err).NotTo(HaveOccurred())
		Expect(user).NotTo(BeNil())
		Expect(user.ID()).To(Equal("user-1"))
	})

	It("returns nil,nil when user is not found", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users/missing"),
				RespondWithJSON(http.StatusNotFound, `{"reason":"not found"}`),
			),
		)

		user, err := ocmClient.GetUser(clusterID, groupID, "missing")
		Expect(err).NotTo(HaveOccurred())
		Expect(user).To(BeNil())
	})

	It("returns an error when user lookup fails with non-404 status", func() {
		for i := 0; i < 3; i++ {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users/alice"),
					RespondWithJSON(http.StatusInternalServerError, `{"reason":"internal error"}`),
				),
			)
		}

		user, err := ocmClient.GetUser(clusterID, groupID, username)
		Expect(err).To(HaveOccurred())
		Expect(user).To(BeNil())
		Expect(apiServer.ReceivedRequests()).NotTo(BeEmpty())
	})

	It("lists users", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users"),
				ghttp.VerifyFormKV("page", "1"),
				ghttp.VerifyFormKV("size", "-1"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "UserList",
					"page": 1,
					"size": 2,
					"total": 2,
					"items": [
						{"id":"user-1"},
						{"id":"user-2"}
					]
				}`),
			),
		)

		users, err := ocmClient.GetUsers(clusterID, groupID)
		Expect(err).NotTo(HaveOccurred())
		Expect(users).To(HaveLen(2))
		Expect(users[0].ID()).To(Equal("user-1"))
		Expect(users[1].ID()).To(Equal("user-2"))
	})

	It("creates users", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users"),
				func(_ http.ResponseWriter, request *http.Request) {
					payload := map[string]interface{}{}
					Expect(json.NewDecoder(request.Body).Decode(&payload)).To(Succeed())
					Expect(payload).To(HaveKeyWithValue("id", "user-3"))
				},
				RespondWithJSON(http.StatusCreated, `{"id":"user-3"}`),
			),
		)
		createInput, err := cmv1.NewUser().ID("user-3").Build()
		Expect(err).NotTo(HaveOccurred())

		created, err := ocmClient.CreateUser(clusterID, groupID, createInput)
		Expect(err).NotTo(HaveOccurred())
		Expect(created).NotTo(BeNil())
		Expect(created.ID()).To(Equal("user-3"))
	})

	It("deletes users", func() {
		apiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodDelete, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users/user-3"),
				RespondWithJSON(http.StatusNoContent, ``),
			),
		)

		err := ocmClient.DeleteUser(clusterID, groupID, "user-3")
		Expect(err).NotTo(HaveOccurred())
		Expect(apiServer.ReceivedRequests()).To(HaveLen(1))
	})

	It("returns an error when listing users fails", func() {
		for i := 0; i < 3; i++ {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users"),
					ghttp.VerifyFormKV("page", "1"),
					ghttp.VerifyFormKV("size", "-1"),
					RespondWithJSON(http.StatusInternalServerError, `{"reason":"internal error"}`),
				),
			)
		}

		users, err := ocmClient.GetUsers(clusterID, groupID)
		Expect(err).To(HaveOccurred())
		Expect(users).To(BeNil())
		Expect(apiServer.ReceivedRequests()).NotTo(BeEmpty())
	})

	It("returns an error when creating a user fails", func() {
		for i := 0; i < 3; i++ {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users"),
					RespondWithJSON(http.StatusForbidden, `{"reason":"forbidden"}`),
				),
			)
		}

		createInput, err := cmv1.NewUser().ID("user-3").Build()
		Expect(err).NotTo(HaveOccurred())

		created, err := ocmClient.CreateUser(clusterID, groupID, createInput)
		Expect(err).To(HaveOccurred())
		Expect(created).To(BeNil())
		Expect(apiServer.ReceivedRequests()).NotTo(BeEmpty())
	})

	It("returns an error when deleting a user fails", func() {
		for i := 0; i < 3; i++ {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, "/api/clusters_mgmt/v1/clusters/cluster-1/groups/dedicated-admins/users/user-3"),
					RespondWithJSON(http.StatusInternalServerError, `{"reason":"internal error"}`),
				),
			)
		}

		err := ocmClient.DeleteUser(clusterID, groupID, "user-3")
		Expect(err).To(HaveOccurred())
		Expect(apiServer.ReceivedRequests()).NotTo(BeEmpty())
	})
})
