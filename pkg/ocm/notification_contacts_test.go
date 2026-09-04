package ocm

import (
	"context"
	"io"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

const subscriptionId = "sub123"
const notificationContactsPath = "/api/accounts_mgmt/v1/subscriptions/sub123/notification_contacts"

var _ = Describe("NotificationContacts", func() {
	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client

	BeforeEach(func() {
		ssoServer = MakeTCPServer()
		apiServer = MakeTCPServer()
		apiServer.SetAllowUnhandledRequests(true)
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

		accessToken := MakeTokenString("Bearer", 15*time.Minute)
		ssoServer.AppendHandlers(
			RespondWithAccessToken(accessToken),
		)

		logger, err := logging.NewGoLoggerBuilder().
			Debug(true).
			Build()
		Expect(err).NotTo(HaveOccurred())

		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens(accessToken).
			URL(apiServer.URL()).
			Build()
		Expect(err).NotTo(HaveOccurred())
		ocmClient = &Client{ocm: connection}
	})

	AfterEach(func() {
		ssoServer.Close()
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	Describe("UpdateSubscriptionNotificationContacts", func() {
		It("adds contacts when none exist", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
						"items": []interface{}{},
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, notificationContactsPath),
					func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(body)).To(ContainSubstring(`"account_identifier":"user1"`))
					},
					ghttp.RespondWithJSONEncoded(http.StatusCreated, map[string]interface{}{
						"id": "acc1", "username": "user1",
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, notificationContactsPath),
					func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(body)).To(ContainSubstring(`"account_identifier":"user2"`))
					},
					ghttp.RespondWithJSONEncoded(http.StatusCreated, map[string]interface{}{
						"id": "acc2", "username": "user2",
					}),
				),
			)

			err := ocmClient.UpdateSubscriptionNotificationContacts(context.Background(), subscriptionId, []string{"user1", "user2"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes contacts not in the desired list", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
						"items": []interface{}{
							map[string]string{"id": "acc1", "username": "user1"},
							map[string]string{"id": "acc2", "username": "user2"},
						},
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, notificationContactsPath+"/acc2"),
					ghttp.RespondWith(http.StatusNoContent, ""),
				),
			)

			err := ocmClient.UpdateSubscriptionNotificationContacts(context.Background(), subscriptionId, []string{"user1"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("clears all contacts with an empty list", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
						"items": []interface{}{
							map[string]string{"id": "acc1", "username": "user1"},
						},
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, notificationContactsPath+"/acc1"),
					ghttp.RespondWith(http.StatusNoContent, ""),
				),
			)

			err := ocmClient.UpdateSubscriptionNotificationContacts(context.Background(), subscriptionId, []string{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error when the list request fails", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusForbidden, map[string]interface{}{
						"kind": "Error", "reason": "forbidden",
					}),
				),
			)

			err := ocmClient.UpdateSubscriptionNotificationContacts(context.Background(), subscriptionId, []string{"user1"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can't read notification contacts"))
		})

		It("returns an error when adding a contact fails", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
						"items": []interface{}{},
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusBadRequest, map[string]interface{}{
						"kind": "Error", "reason": "invalid username",
					}),
				),
			)

			err := ocmClient.UpdateSubscriptionNotificationContacts(context.Background(), subscriptionId, []string{"bad_user"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can't add notification contact"))
		})

		It("deduplicates contacts and sends only one POST per unique username", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
						"items": []interface{}{},
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusCreated, map[string]interface{}{
						"id": "acc1", "username": "user1",
					}),
				),
			)

			err := ocmClient.UpdateSubscriptionNotificationContacts(
				context.Background(), subscriptionId, []string{"user1", "user1", "user1"})
			Expect(err).NotTo(HaveOccurred())
			Expect(apiServer.ReceivedRequests()).To(HaveLen(2))
		})

		It("does not delete contacts when adding a replacement fails", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
						"items": []interface{}{
							map[string]string{"id": "acc1", "username": "old_user"},
						},
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusBadRequest, map[string]interface{}{
						"kind": "Error", "reason": "invalid username",
					}),
				),
			)

			err := ocmClient.UpdateSubscriptionNotificationContacts(
				context.Background(), subscriptionId, []string{"new_user"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can't add notification contact"))
			Expect(apiServer.ReceivedRequests()).To(HaveLen(2))
		})
	})

	Describe("GetSubscriptionNotificationContacts", func() {
		It("returns sorted usernames", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
						"items": []interface{}{
							map[string]string{"id": "acc2", "username": "zeta_user"},
							map[string]string{"id": "acc1", "username": "alpha_user"},
						},
					}),
				),
			)

			contacts, err := ocmClient.GetSubscriptionNotificationContacts(context.Background(), subscriptionId)
			Expect(err).NotTo(HaveOccurred())
			Expect(contacts).To(Equal([]string{"alpha_user", "zeta_user"}))
		})

		It("returns nil when no contacts exist", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
						"items": []interface{}{},
					}),
				),
			)

			contacts, err := ocmClient.GetSubscriptionNotificationContacts(context.Background(), subscriptionId)
			Expect(err).NotTo(HaveOccurred())
			Expect(contacts).To(BeNil())
		})

		It("returns an error when the API responds with a server error", func() {
			apiServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, notificationContactsPath),
					ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, map[string]interface{}{
						"kind": "Error", "reason": "internal server error",
					}),
				),
			)

			contacts, err := ocmClient.GetSubscriptionNotificationContacts(context.Background(), subscriptionId)
			Expect(err).To(HaveOccurred())
			Expect(contacts).To(BeNil())
		})
	})
})
