package ocm

import (
	"bytes"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

const (
	breakGlassCredentialId = "test-break-glass-credential"
)

var _ = Describe("BreakGlassCredential", func() {

	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client
	var body string
	var breakGlassCredential *cmv1.BreakGlassCredential

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

		breakGlassCredential, body, err = CreateBreakGlassCredential()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Close the servers:
		ssoServer.Close()
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("OK: gets all BreakGlassCredentials when exist", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				body,
			),
		)

		breakGlassCredentials, err := ocmClient.GetBreakGlassCredentials(clusterId)
		Expect(err).To(BeNil())
		Expect(breakGlassCredentials).To(Not(BeNil()))
	})

	It("KO: fails to get all BreakGlassCredentials", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusBadRequest,
				body,
			),
		)

		_, err := ocmClient.GetBreakGlassCredentials(clusterId)
		Expect(err).To(HaveOccurred())
	})

	It("OK: gets BreakGlassCredential when it exists", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				body,
			),
		)

		breakGlassCredential, err := ocmClient.GetBreakGlassCredential(clusterId, breakGlassCredentialId)

		Expect(err).To(BeNil())
		Expect(breakGlassCredential).To(Not(BeNil()))
		Expect(breakGlassCredential.ID()).To(Equal(breakGlassCredentialId))
	})

	It("KO: returns error when BreakGlassCredential id does not exist", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				body,
			),
		)
		breakGlassCredential, err := ocmClient.GetBreakGlassCredential(clusterId, breakGlassCredentialId)
		Expect(err).Should(HaveOccurred())
		Expect(breakGlassCredential).To(BeNil())
	})

	It("OK: deletes BreakGlassCredential successfully", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusNoContent, ""),
		)

		err := ocmClient.DeleteBreakGlassCredentials(clusterId)
		Expect(err).To(BeNil())
	})

	It("KO: fails to delete BreakGlassCredential if none exists", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				body,
			),
		)

		err := ocmClient.DeleteBreakGlassCredentials(clusterId)
		Expect(err).NotTo(BeNil())
	})

	It("OK: creates BreakGlassCredential successfully", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusCreated,
				body,
			),
		)

		breakGlassCredential, err := ocmClient.CreateBreakGlassCredential(clusterId, breakGlassCredential)

		Expect(breakGlassCredential).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())
	})

	It("KO: fails to create BreakGlassCredential if one exists", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusBadRequest,
				body,
			),
		)

		_, err := ocmClient.CreateBreakGlassCredential(clusterId, breakGlassCredential)
		Expect(err).To(HaveOccurred())
	})

	It("OK: Successfully gets PollKubeconfig if one exists", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				body,
			),
		)
		_, err := ocmClient.PollKubeconfig(clusterId, breakGlassCredential.ID(), time.Millisecond*100, time.Second*5)
		Expect(err).ToNot(HaveOccurred())
	})

	It("KO: fails to get PollKubeconfig", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				body,
			),
		)
		_, err := ocmClient.PollKubeconfig(clusterId, breakGlassCredential.ID(), time.Millisecond*100, time.Second*5)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(
			"Failed to poll kubeconfig for cluster 'foo' with break glass credential 'test-break-glass-credential': " +
				"expected response content type 'application/json' but received '' and content ''"))
	})

})

func CreateBreakGlassCredential() (*cmv1.BreakGlassCredential, string, error) {
	builder := &cmv1.BreakGlassCredentialBuilder{}
	breakGlassCredentialConfig, err := builder.ID(breakGlassCredentialId).Build()
	if err != nil {
		return &cmv1.BreakGlassCredential{}, "", err
	}

	var buf bytes.Buffer
	err = cmv1.MarshalBreakGlassCredential(breakGlassCredentialConfig, &buf)
	if err != nil {
		return &cmv1.BreakGlassCredential{}, "", err
	}

	return breakGlassCredentialConfig, buf.String(), nil
}
