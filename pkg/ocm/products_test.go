package ocm

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

// nolint
const hcpTechnologyPreviewBody = `{
	"kind": "ProductTechnologyPreview",
 	"href": "/api/clusters_mgmt/v1/products/rosa/technology_previews/hcp",
	"start_date": "2022-12-01T00:01:00Z",
	"end_date": "2023-12-05T00:01:00Z",
	"additional_text": "Tech preview message"
}`

var _ = Describe("Technology preview features", func() {
	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client

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
	})

	AfterEach(func() {
		// Close the servers:
		ssoServer.Close()
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("Expects a message for hcp in tech preview", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hcpTechnologyPreviewBody,
			),
		)

		beforeRelease, err := time.Parse(time.RFC3339, "2023-12-03T00:01:00Z")
		Expect(err).ToNot(HaveOccurred())

		message, err := ocmClient.GetTechnologyPreviewMessage("hcp", beforeRelease)
		Expect(err).ToNot(HaveOccurred())

		Expect(message).To(Equal("Tech preview message"))
	})

	It("Expects no message for hcp in GA", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hcpTechnologyPreviewBody,
			),
		)

		afterRelease, err := time.Parse(time.RFC3339, "2023-12-06T00:01:00Z")
		Expect(err).ToNot(HaveOccurred())

		message, err := ocmClient.GetTechnologyPreviewMessage("hcp", afterRelease)
		Expect(err).ToNot(HaveOccurred())

		Expect(message).To(BeEmpty())
	})

	It("Expects no message if no technology preview", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				"",
			),
		)

		beforeRelease, err := time.Parse(time.RFC3339, "2023-12-03T00:01:00Z")
		Expect(err).ToNot(HaveOccurred())

		message, err := ocmClient.GetTechnologyPreviewMessage("hcp", beforeRelease)
		Expect(err).ToNot(HaveOccurred())
		Expect(message).To(BeEmpty())
	})

	It("Expects an error if bad product", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusBadRequest,
				"Product 'bad-product' doesn't exist",
			),
		)

		beforeRelease, err := time.Parse(time.RFC3339, "2023-12-03T00:01:00Z")
		Expect(err).ToNot(HaveOccurred())

		message, err := ocmClient.GetTechnologyPreviewMessage("bad-product", beforeRelease)
		Expect(err).To(HaveOccurred())
		Expect(message).To(BeEmpty())
	})

})
