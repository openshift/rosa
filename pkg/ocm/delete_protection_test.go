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

var _ = Describe("DeleteProtection", func() {
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

	It("updates delete protection successfully", func() {
		body, err := marshalDeleteProtection(true)
		Expect(err).NotTo(HaveOccurred())

		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, body),
		)

		deleteProtection, err := cmv1.NewDeleteProtection().Enabled(true).Build()
		Expect(err).NotTo(HaveOccurred())

		err = ocmClient.UpdateClusterDeleteProtection(clusterId, deleteProtection)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns an error when the update API responds with a server error", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusForbidden, `{
				"kind": "Error",
				"id": "403",
				"href": "/api/clusters_mgmt/v1/errors/403",
				"code": "CLUSTERS-MGMT-403",
				"reason": "forbidden"
			}`),
		)

		deleteProtection, err := cmv1.NewDeleteProtection().Enabled(true).Build()
		Expect(err).NotTo(HaveOccurred())

		err = ocmClient.UpdateClusterDeleteProtection(clusterId, deleteProtection)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("forbidden"))
	})
})

func marshalDeleteProtection(enabled bool) (string, error) {
	deleteProtection, err := cmv1.NewDeleteProtection().Enabled(enabled).Build()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := cmv1.MarshalDeleteProtection(deleteProtection, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}
