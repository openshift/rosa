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

	When("isHostedCPSupportedRegion", func() {
		It("2 active service cluster in region", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
					  "kind": "ProvisionShardList",
					  "page": 1,
					  "size": 2,
					  "total": 2,
					  "items": [
						{
						  "kind": "ProvisionShard",
						  "id": "123",
						  "href": "/api/clusters_mgmt/v1/provision_shards/123",
						  "hypershift_config": {
							"server": "https://api.123.org:6443",
							"kubeconfig": "**********"
						  },
						  "aws_base_domain": "123.org",
						  "status": "active",
						  "region": {
							"kind": "CloudRegion",
							"id": "us-west-2",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-2"
						  },
						  "cloud_provider": {
							"kind": "CloudProvider",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "management_cluster": "mc1"
						},
						{
						  "kind": "ProvisionShard",
						  "id": "456",
						  "href": "/api/clusters_mgmt/v1/provision_shards/456",
						  "hypershift_config": {
							"server": "https://api2.123.org:6443",
							"kubeconfig": "**********"
						  },
						  "aws_base_domain": "123.org",
						  "status": "active",
						  "region": {
							"kind": "CloudRegion",
							"id": "us-west-2",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-2"
						  },
						  "cloud_provider": {
							"kind": "CloudProvider",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "management_cluster": "mc4"
						}
					  ]
					}`,
				),
			)
			region, err := cmv1.NewCloudRegion().
				ID("us-west-2").Build()
			Expect(err).To(BeNil())
			exists, err := ocmClient.isHostedCPSupportedRegion(region)
			Expect(err).To(BeNil())
			// Region has available Service Clusters
			Expect(exists).To(BeTrue())
		})
		It("1 active service cluster in region", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
					  "kind": "ProvisionShardList",
					  "page": 1,
					  "size": 1,
					  "total": 1,
					  "items": [
						{
						  "kind": "ProvisionShard",
						  "id": "123",
						  "href": "/api/clusters_mgmt/v1/provision_shards/123",
						  "hypershift_config": {
							"server": "https://api.123.org:6443",
							"kubeconfig": "**********"
						  },
						  "aws_base_domain": "123.org",
						  "status": "active",
						  "region": {
							"kind": "CloudRegion",
							"id": "us-west-2",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-2"
						  },
						  "cloud_provider": {
							"kind": "CloudProvider",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "management_cluster": "hs-mc-ccb16elad"
						}
					  ]
					}`,
				),
			)
			region, err := cmv1.NewCloudRegion().
				ID("us-west-2").Build()
			Expect(err).To(BeNil())
			exists, err := ocmClient.isHostedCPSupportedRegion(region)
			Expect(err).To(BeNil())
			// Region has available Service Clusters
			Expect(exists).To(BeTrue())
		})
		It("0 active service clusters in region", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
					  "kind": "ProvisionShardList",
					  "page": 1,
					  "size": 0,
					  "total": 0,
					  "items": []
					}`,
				),
			)
			region, err := cmv1.NewCloudRegion().
				ID("us-west-2").Build()
			Expect(err).To(BeNil())
			exists, err := ocmClient.isHostedCPSupportedRegion(region)
			Expect(err).To(BeNil())
			// Region has no available Service Clusters
			Expect(exists).To(BeFalse())
		})
		It("CS replies in error", func() {
			// No handler registered, we get a 500, check we handle it
			region, err := cmv1.NewCloudRegion().
				ID("us-west-2").Build()
			Expect(err).To(BeNil())
			exists, err := ocmClient.isHostedCPSupportedRegion(region)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("failed to get Provison Shards: expected response " +
				"content type 'application/json' but received '' and content ''"))
			Expect(exists).To(BeFalse())
		})
	})
	When("ListHostedCPSupportedRegion", func() {
		It("2 active service cluster in 1 region", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
					  "kind": "ProvisionShardList",
					  "page": 1,
					  "size": 2,
					  "total": 2,
					  "items": [
						{
						  "kind": "ProvisionShard",
						  "id": "123",
						  "href": "/api/clusters_mgmt/v1/provision_shards/123",
						  "hypershift_config": {
							"server": "https://api.123.org:6443",
							"kubeconfig": "**********"
						  },
						  "aws_base_domain": "123.org",
						  "status": "active",
						  "region": {
							"kind": "CloudRegion",
							"id": "us-west-2",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-2"
						  },
						  "cloud_provider": {
							"kind": "CloudProvider",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "management_cluster": "mc1"
						},
						{
						  "kind": "ProvisionShard",
						  "id": "456",
						  "href": "/api/clusters_mgmt/v1/provision_shards/456",
						  "hypershift_config": {
							"server": "https://api2.123.org:6443",
							"kubeconfig": "**********"
						  },
						  "aws_base_domain": "123.org",
						  "status": "active",
						  "region": {
							"kind": "CloudRegion",
							"id": "us-west-2",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-2"
						  },
						  "cloud_provider": {
							"kind": "CloudProvider",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "management_cluster": "mc4"
						}
					  ]
					}`,
				),
			)
			regions, err := ocmClient.ListHostedCPSupportedRegion()
			Expect(err).To(BeNil())
			Expect(regions).Should(HaveLen(1))
			Expect(regions).Should(HaveKey("us-west-2"))
		})
		It("2 active service cluster in 2 regions", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
					  "kind": "ProvisionShardList",
					  "page": 1,
					  "size": 2,
					  "total": 2,
					  "items": [
						{
						  "kind": "ProvisionShard",
						  "id": "123",
						  "href": "/api/clusters_mgmt/v1/provision_shards/123",
						  "hypershift_config": {
							"server": "https://api.123.org:6443",
							"kubeconfig": "**********"
						  },
						  "aws_base_domain": "123.org",
						  "status": "active",
						  "region": {
							"kind": "CloudRegion",
							"id": "us-west-2",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-2"
						  },
						  "cloud_provider": {
							"kind": "CloudProvider",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "management_cluster": "mc1"
						},
						{
						  "kind": "ProvisionShard",
						  "id": "456",
						  "href": "/api/clusters_mgmt/v1/provision_shards/456",
						  "hypershift_config": {
							"server": "https://api2.123.org:6443",
							"kubeconfig": "**********"
						  },
						  "aws_base_domain": "123.org",
						  "status": "active",
						  "region": {
							"kind": "CloudRegion",
							"id": "us-west-1",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-1"
						  },
						  "cloud_provider": {
							"kind": "CloudProvider",
							"id": "aws",
							"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						  },
						  "management_cluster": "mc4"
						}
					  ]
					}`,
				),
			)
			regions, err := ocmClient.ListHostedCPSupportedRegion()
			Expect(err).To(BeNil())
			Expect(regions).Should(HaveLen(2))
			Expect(regions).Should(HaveKey("us-west-2"))
			Expect(regions).Should(HaveKey("us-west-1"))
		})
		It("0 active service clusters", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
					  "kind": "ProvisionShardList",
					  "page": 1,
					  "size": 0,
					  "total": 0,
					  "items": []
					}`,
				),
			)
			regions, err := ocmClient.ListHostedCPSupportedRegion()
			Expect(err).To(BeNil())
			// Region has no available Service Clusters
			Expect(regions).To(BeEmpty())
		})
		It("CS replies in error", func() {
			// No handler registered, we get a 500, check we handle it
			regions, err := ocmClient.ListHostedCPSupportedRegion()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("failed to get Provison Shards: expected response " +
				"content type 'application/json' but received '' and content ''"))
			Expect(regions).To(BeEmpty())
		})
	})
})
