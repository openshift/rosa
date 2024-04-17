package ocm

import (
	"bytes"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

const (
	externalAuthId = "test-external-auth"
)

var _ = Describe("ExternalAuthConfig", Ordered, func() {

	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client
	var body string
	var externalAuth *cmv1.ExternalAuth

	BeforeEach(func() {
		// Create the servers:
		ssoServer = MakeTCPServer()
		apiServer = MakeTCPServer()
		apiServer.SetAllowUnhandledRequests(true)
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

		// Create the token:
		accessToken := MakeTokenString("Bearer", 15*time.Minute)

		// Prepare the server:
		ssoServer.AppendHandlers(
			RespondWithAccessToken(accessToken),
		)
		// Prepare the logger:
		logger, err := logging.NewGoLoggerBuilder().
			Debug(true).
			Build()
		Expect(err).To(BeNil())
		// Set up the connection with the fake config
		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens(accessToken).
			URL(apiServer.URL()).
			Build()
		// Initialize client object
		Expect(err).To(BeNil())
		ocmClient = &Client{ocm: connection}

		externalAuth, body, err = CreateExternalAuthConfig()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Close the servers:
		ssoServer.Close()
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("Gets ExternalAuthConfig when it exists", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				body,
			),
		)

		externalAuth, success, err := ocmClient.GetExternalAuth(clusterId, externalAuthId)

		Expect(err).To(BeNil())
		Expect(externalAuth).To(Not(BeNil()))
		Expect(externalAuth.ID()).To(Equal(externalAuthId))
		Expect(success).To(Equal(true))
	})

	It("Gets all externalAuths when they exist", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				body,
			),
		)

		externalAuths, err := ocmClient.GetExternalAuths(clusterId)

		Expect(err).To(BeNil())
		Expect(externalAuths).To(Not(BeNil()))
	})

	It("Returns nil when ExternalAuthConfig does not exist", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				body,
			),
		)
		externalAuth, success, err := ocmClient.GetExternalAuth(clusterId, externalAuthId)
		Expect(err).To(BeNil())
		Expect(externalAuth).To(BeNil())
		Expect(success).To(Equal(false))
	})

	It("Deletes ExternalAuthConfig", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusNoContent, ""),
		)

		err := ocmClient.DeleteExternalAuth(clusterId, externalAuthId)
		Expect(err).To(BeNil())
	})

	It("Fails to Delete ExternalAuthConfig if none exists", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				body,
			),
		)

		err := ocmClient.DeleteExternalAuth(clusterId, externalAuthId)
		Expect(err).NotTo(BeNil())
	})

	It("Creates ExternalAuthConfig", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusCreated,
				body,
			),
		)

		externalAuth, err := ocmClient.CreateExternalAuth(clusterId, externalAuth)

		Expect(externalAuth).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())
	})

	It("Fails to create ExternalAuthConfig if one exists", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusBadRequest,
				body,
			),
		)

		_, err := ocmClient.CreateExternalAuth(clusterId, externalAuth)
		Expect(err).To(HaveOccurred())
	})

})

func CreateExternalAuthConfig() (*cmv1.ExternalAuth, string, error) {
	builder := &cmv1.ExternalAuthBuilder{}
	externalAuthConfig, err := builder.ID(externalAuthId).
		Issuer(cmv1.NewTokenIssuer().URL("https://test.com").Audiences("abc")).
		Build()
	if err != nil {
		return &cmv1.ExternalAuth{}, "", err
	}

	var buf bytes.Buffer
	err = cmv1.MarshalExternalAuth(externalAuthConfig, &buf)
	if err != nil {
		return &cmv1.ExternalAuth{}, "", err
	}

	return externalAuthConfig, buf.String(), nil
}
