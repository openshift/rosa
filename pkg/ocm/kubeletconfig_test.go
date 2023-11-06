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
	podPidsLimit = 5000
	kubeletHref  = "/api/clusters_mgmt/cmv1/clusters/foo/kubelet_config"

	kubeletId = "bar"

	clusterId = "foo"
)

var _ = Describe("KubeletConfig", Ordered, func() {
	var ssoServer, apiServer *ghttp.Server
	var ocmClient *Client
	var body string

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

		body, err = createKubeletConfig()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Close the servers:
		ssoServer.Close()
		apiServer.Close()
		Expect(ocmClient.Close()).To(Succeed())
	})

	It("Gets KubeletConfig when it exists", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				body,
			),
		)

		kubeletConfig, err := ocmClient.GetClusterKubeletConfig(clusterId)

		Expect(err).To(BeNil())
		Expect(kubeletConfig).To(Not(BeNil()))
		Expect(kubeletConfig.HREF()).To(Equal(kubeletHref))
		Expect(kubeletConfig.PodPidsLimit()).To(Equal(podPidsLimit))
	})

	It("Returns nil when KubeletConfig does not exist", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				body,
			),
		)
		kubeletConfig, err := ocmClient.GetClusterKubeletConfig(clusterId)
		Expect(err).To(BeNil())
		Expect(kubeletConfig).To(BeNil())
	})

	It("Deletes KubeletConfig", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusNoContent, ""),
		)

		err := ocmClient.DeleteKubeletConfig(clusterId)
		Expect(err).To(BeNil())
	})

	It("Fails to Delete KubeletConfig if none exists", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				body,
			),
		)

		err := ocmClient.DeleteKubeletConfig(clusterId)
		Expect(err).NotTo(BeNil())
	})

})

func createKubeletConfig() (string, error) {
	builder := &cmv1.KubeletConfigBuilder{}
	kubeletConfig, err := builder.PodPidsLimit(podPidsLimit).ID(kubeletId).HREF(kubeletHref).Build()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = cmv1.MarshalKubeletConfig(kubeletConfig, &buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
