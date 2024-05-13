package ocm

import (
	"bytes"
	"context"
	"fmt"
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

	clusterId   = "foo"
	kubeletName = "test"
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

		kubeletConfig, exists, err := ocmClient.GetClusterKubeletConfig(clusterId)

		Expect(err).To(BeNil())
		Expect(kubeletConfig).To(Not(BeNil()))
		Expect(kubeletConfig.HREF()).To(Equal(kubeletHref))
		Expect(kubeletConfig.PodPidsLimit()).To(Equal(podPidsLimit))
		Expect(exists).To(BeTrue())
	})

	It("Returns nil when KubeletConfig does not exist", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound,
				body,
			),
		)
		kubeletConfig, exists, err := ocmClient.GetClusterKubeletConfig(clusterId)
		Expect(err).To(BeNil())
		Expect(kubeletConfig).To(BeNil())
		Expect(exists).To(BeFalse())
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

	It("Creates KubeletConfig", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusCreated,
				body,
			),
		)

		args := KubeletConfigArgs{podPidsLimit, kubeletName}
		kubeletConfig, err := ocmClient.CreateKubeletConfig(clusterId, args)
		Expect(kubeletConfig.Name()).To(Equal(kubeletName))

		Expect(kubeletConfig).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())

	})

	It("Fails to create KubeletConfig if one exists", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusBadRequest,
				body,
			),
		)

		args := KubeletConfigArgs{podPidsLimit, kubeletName}
		_, err := ocmClient.CreateKubeletConfig(clusterId, args)
		Expect(err).To(HaveOccurred())
	})

	It("Updates KubeletConfig", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				body,
			),
		)

		args := KubeletConfigArgs{podPidsLimit, kubeletName}
		kubeletConfig, err := ocmClient.UpdateKubeletConfig(context.Background(), clusterId, kubeletId, args)

		Expect(kubeletConfig).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())
	})

	It("Fails to update KubeletConfig", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusBadRequest,
				body,
			),
		)

		args := KubeletConfigArgs{podPidsLimit, kubeletName}
		_, err := ocmClient.UpdateKubeletConfig(context.Background(), clusterId, kubeletId, args)
		Expect(err).To(HaveOccurred())
	})

	Context("List KubeletConfigs", func() {

		It("Lists all kubeletconfigs", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, createKubeletConfigList(false)))

			response, err := ocmClient.ListKubeletConfigs(context.Background(), clusterId)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(HaveLen(1))

			config := response[0]
			Expect(config.Name()).To(Equal(kubeletName))
			Expect(config.ID()).To(Equal(kubeletId))
			Expect(config.PodPidsLimit()).To(Equal(podPidsLimit))
		})

		It("Returns an empty list if no KubeletConfigs", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, createKubeletConfigList(true)))

			response, err := ocmClient.ListKubeletConfigs(context.Background(), clusterId)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(HaveLen(0))
		})

		It("Returns an error if failing to list KubeletConfigs", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusInternalServerError, createKubeletConfigList(true)))
			_, err := ocmClient.ListKubeletConfigs(context.Background(), clusterId)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Find KubeletConfig By Name", func() {
		It("Returns the KubeletConfig when it exists", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, createKubeletConfigList(false)))

			response, exists, err := ocmClient.FindKubeletConfigByName(context.Background(), clusterId, kubeletName)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(exists).To(BeTrue())

			Expect(response.Name()).To(Equal(kubeletName))
			Expect(response.ID()).To(Equal(kubeletId))
			Expect(response.PodPidsLimit()).To(Equal(podPidsLimit))
		})

		It("Returns nil KubeletConfig if it doesn't exist", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK, createKubeletConfigList(false)))

			response, exists, err := ocmClient.FindKubeletConfigByName(context.Background(), clusterId, "notExisting")
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(BeNil())
			Expect(exists).To(BeFalse())
		})

		It("Returns an error if failing to list KubeletConfigs", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusInternalServerError, createKubeletConfigList(true)))
			response, exists, err := ocmClient.FindKubeletConfigByName(context.Background(), clusterId, kubeletName)
			Expect(response).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("Returns an error if the name specified is empty", func() {
			response, exists, err := ocmClient.FindKubeletConfigByName(context.Background(), clusterId, "")
			Expect(response).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

})

func createKubeletConfig() (string, error) {
	builder := &cmv1.KubeletConfigBuilder{}
	kubeletConfig, err := builder.PodPidsLimit(podPidsLimit).ID(kubeletId).HREF(kubeletHref).Name(kubeletName).Build()
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

func createKubeletConfigList(empty bool) string {
	var json bytes.Buffer
	var configs []*cmv1.KubeletConfig

	if !empty {
		builder := &cmv1.KubeletConfigBuilder{}
		kubeletConfig, err := builder.PodPidsLimit(podPidsLimit).ID(kubeletId).HREF(kubeletHref).Name(kubeletName).Build()
		Expect(err).NotTo(HaveOccurred())
		configs = []*cmv1.KubeletConfig{kubeletConfig}
	}

	cmv1.MarshalKubeletConfigList(configs, &json)

	return fmt.Sprintf(`
	{
		"kind": "KubeletConfigList",
		"page": 1,
		"size": %d,
		"total": %d,
		"items": %s
	}`, len(configs), len(configs), json.String())
}
