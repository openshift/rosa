package ocm

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
)

// nolint
const responseBody = `{
  "kind": "TuningConfigList",
  "href": "/api/clusters_mgmt/v1/clusters/236p9gd58dekekrtaugvok47p4smp8lg/tuning_configs",
  "page": 1,
  "size": 2,
  "total": 2,
  "items": [
    {
      "kind": "TuningConfig",
      "href": "/api/clusters_mgmt/v1/clusters/236p9gd58dekekrtaugvok47p4smp8lg/tuning_configs/a86ed592-df57-11ed-8c8b-acde48001122",
      "id": "a86ed592-df57-11ed-8c8b-acde48001122",
      "name": "tuned-2",
      "spec": {
        "profile": [
          {
            "data": "[main]\nsummary=Custom OpenShift profile\ninclude=openshift-node\n\n[sysctl]\nvm.dirty_ratio=\"55\"\n",
            "name": "tuned-1-profile"
          }
        ],
        "recommend": [
          {
            "priority": 20,
            "profile": "tuned-1-profile"
          }
        ]
      }
    },
    {
      "kind": "TuningConfig",
      "href": "/api/clusters_mgmt/v1/clusters/236p9gd58dekekrtaugvok47p4smp8lg/tuning_configs/27eebbf0-df5f-11ed-8c8b-acde48001122",
      "id": "27eebbf0-df5f-11ed-8c8b-acde48001122",
      "name": "tuned-22",
      "spec": {
        "profile": [
          {
            "data": "[main]\nsummary=Custom OpenShift profile\ninclude=openshift-node\n\n[sysctl]\nvm.dirty_ratio=\"45\"\n",
            "name": "tuned-2-profile"
          }
        ],
        "recommend": [
          {
            "priority": 20,
            "profile": "tuned-2-profile"
          }
        ]
      }
    }
  ]
}`

var _ = Describe("Tuning configs", Ordered, func() {
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

	It("Find a name inside the response", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				responseBody,
			),
		)
		tuningConfig, err := ocmClient.FindTuningConfigByName("id1", "tuned-2")
		Expect(err).To(BeNil())
		Expect(tuningConfig).To(Not(BeNil()))
		Expect(tuningConfig.ID()).To(Equal("a86ed592-df57-11ed-8c8b-acde48001122"))
	})
	It("Name not found", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				responseBody,
			),
		)
		tuningConfig, err := ocmClient.FindTuningConfigByName("id1", "tuned-not-existing")
		Expect(tuningConfig).To(BeNil())
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(ContainSubstring("does not exist on cluster"))
	})
	It("Extracts correctly names", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				responseBody,
			),
		)
		tuningConfigsNames, err := ocmClient.GetTuningConfigsName("id1")
		Expect(err).To(BeNil())
		Expect(tuningConfigsNames).Should(HaveLen(2))
		Expect(tuningConfigsNames).To(ContainElements("tuned-2", "tuned-22"))
	})
})
