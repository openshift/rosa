package ocm

import (
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

var _ = Describe("Regions", Ordered, func() {
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

	When("getFilteredRegions", func() {
		It("Gets some regions", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
					  "kind": "CloudRegionList",
					  "page": 1,
					  "size": 4,
					  "total": 4,
					  "items": [
						{
						  "kind": "CloudRegion",
						  "id": "us-east-1",
						  "href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-1",
						  "display_name": "US East, N. Virginia",
						  "cloud_provider": {
							"kind": "CloudProviderLink",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "enabled": true,
						  "supports_multi_az": true,
						  "kms_location_name": "",
						  "kms_location_id": "",
						  "ccs_only": false,
						  "govcloud": false,
						  "supports_hypershift": false
						},
						{
						  "kind": "CloudRegion",
						  "id": "us-east-2",
						  "href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-2",
						  "display_name": "US East, Ohio",
						  "cloud_provider": {
							"kind": "CloudProviderLink",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "enabled": true,
						  "supports_multi_az": true,
						  "kms_location_name": "",
						  "kms_location_id": "",
						  "ccs_only": false,
						  "govcloud": false,
						  "supports_hypershift": false
						},
						{
						  "kind": "CloudRegion",
						  "id": "us-west-1",
						  "href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-1",
						  "display_name": "US West, N. California",
						  "cloud_provider": {
							"kind": "CloudProviderLink",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "enabled": true,
						  "supports_multi_az": false,
						  "kms_location_name": "",
						  "kms_location_id": "",
						  "ccs_only": false,
						  "govcloud": false,
						  "supports_hypershift": false
						},
						{
						  "kind": "CloudRegion",
						  "id": "us-west-2",
						  "href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-2",
						  "display_name": "US West, Oregon",
						  "cloud_provider": {
								"kind": "CloudProviderLink",
								"id": "aws",
								"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
							  },
							  "enabled": true,
							  "supports_multi_az": true,
							  "kms_location_name": "",
							  "kms_location_id": "",
							  "ccs_only": false,
							  "govcloud": false,
							  "supports_hypershift": true
							}
						  ]
						}`,
				),
			)
			cloudProviderData, err := cmv1.NewCloudProviderData().Build()
			Expect(err).ToNot(HaveOccurred())
			regions, err := ocmClient.getFilteredRegions(cloudProviderData)
			Expect(err).To(BeNil())
			Expect(regions).Should(HaveLen(4))
			Expect(regions[0].SupportsHypershift()).To(BeFalse())
		})
		It("Gets no region", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
					  "kind": "CloudRegionList",
					  "page": 1,
					  "size": 0,
					  "total": 0,
					  "items": []
					}`,
				),
			)
			cloudProviderData, err := cmv1.NewCloudProviderData().Build()
			Expect(err).ToNot(HaveOccurred())
			regions, err := ocmClient.getFilteredRegions(cloudProviderData)
			Expect(err).To(BeNil())
			// Region has available Service Clusters
			Expect(regions).Should(HaveLen(0))
		})
		It("CS replies in error", func() {
			// No handler registered, we get a 500, check we handle it
			cloudProviderData, err := cmv1.NewCloudProviderData().Build()
			Expect(err).ToNot(HaveOccurred())
			regions, err := ocmClient.getFilteredRegions(cloudProviderData)
			Expect(regions).Should(HaveLen(0))
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("expected response " +
				"content type 'application/json' but received '' and content ''"))
		})
	})
})
