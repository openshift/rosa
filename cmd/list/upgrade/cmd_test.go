package upgrade

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/openshift/rosa/pkg/test"
)

const noUpgradeOutput = `VERSION  NOTES
`
const ongoingUpgradeOutput = `VERSION  NOTES
4.12.26  recommended
4.12.25  pending for 2023-06-02 12:30 UTC
`
const upgradeAvailableOutput = `VERSION  NOTES
4.12.26  recommended
4.12.25  
`

var _ = Describe("List upgrade", func() {
	Context("Format scheduled upgrade", func() {
		It("Node pool upgrade is scheduled", func() {
			availableUpgrade := "4.12.19"
			nowUTC := time.Now().UTC()
			upgradeState := cmv1.NewUpgradePolicyState().Value("scheduled")
			npUpgradePolicy, err := cmv1.NewNodePoolUpgradePolicy().ID("id1").Version("4.12.19").
				State(upgradeState).NextRun(nowUTC).Build()
			Expect(err).To(BeNil())
			notes := formatScheduledUpgradeHypershift(availableUpgrade, npUpgradePolicy)
			Expect(notes).To(Equal(fmt.Sprintf("scheduled for %s", nowUTC.Format("2006-01-02 15:04 MST"))))
		})
		It("Nothing scheduled for this node pool", func() {
			availableUpgrade := "4.12.18"
			npUpgradePolicy, err := cmv1.NewNodePoolUpgradePolicy().ID("id1").Version("4.12.19").Build()
			Expect(err).To(BeNil())
			notes := formatScheduledUpgradeHypershift(availableUpgrade, npUpgradePolicy)
			Expect(notes).To(Equal(""))
		})
	})
	Context("Latest rev in minor", func() {
		It("Find latest minor", func() {
			currentVersion := "4.12.16"
			availableUpgrades := []string{"4.12.17", "4.12.18", "4.13.0"}
			latestRev := latestInCurrentMinor(currentVersion, availableUpgrades)
			Expect(latestRev).To(Equal("4.12.18"))
		})
		It("Only upgrades to a major available", func() {
			currentVersion := "4.12.16"
			availableUpgrades := []string{"4.13.0"}
			latestRev := latestInCurrentMinor(currentVersion, availableUpgrades)
			Expect(latestRev).To(Equal("4.12.16"))
		})
	})
	Context("List upgrades command", func() {
		var testRuntime test.TestingRuntime
		var nodePoolName = "nodepool85"

		mockClusterError, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateError)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})
		Expect(err).To(BeNil())
		var hypershiftClusterNotReady = test.FormatClusterList([]*cmv1.Cluster{mockClusterError})

		mockClusterReady, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.Version(cmv1.NewVersion().RawID("4.12.26").ChannelGroup("stable").
				ID("4.12.26").Enabled(true).AvailableUpgrades("4.12.27"))
		})
		Expect(err).To(BeNil())
		var hypershiftClusterReady = test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		mockClassicCluster, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(false))
		})
		Expect(err).To(BeNil())
		var classicCluster = test.FormatClusterList([]*cmv1.Cluster{mockClassicCluster})

		// nolint:lll
		const versionListResponse = `{
					  "kind": "VersionList",
					  "page": 1,
					  "size": 3,
					  "total": 3,
					  "items": [
						{
						  "kind": "Version",
						  "id": "openshift-v4.12.26",
						  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.12.26",
						  "raw_id": "4.12.26",
						  "enabled": true,
						  "default": true,
						  "channel_group": "stable",
						  "rosa_enabled": true,
						  "hosted_control_plane_enabled": true,
						  "end_of_life_timestamp": "2024-05-17T00:00:00Z",
						  "ami_overrides": [
							{
							  "product": {
								"kind": "ProductLink",
								"id": "rosa",
								"href": "/api/clusters_mgmt/v1/products/rosa"
							  },
							  "region": {
								"kind": "CloudRegionLink",
								"id": "us-east-2",
								"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-2"
							  },
							  "ami": "ami-0e677f92eb4180cc0"
							},
							{
							  "product": {
								"kind": "ProductLink",
								"id": "rosa",
								"href": "/api/clusters_mgmt/v1/products/rosa"
							  },
							  "region": {
								"kind": "CloudRegionLink",
								"id": "us-east-1",
								"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-1"
							  },
							  "ami": "ami-00354720d36d019f9"
							}
						  ],
						  "release_image": "quay.io/openshift-release-dev/ocp-release@sha256:8d72f29227418d2ae12ee52e25cce9edef7cd645bdaea02410a89fe8a0ec6a47"
						},
						{
						  "kind": "Version",
						  "id": "openshift-v4.12.25",
						  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.12.25",
						  "raw_id": "4.12.25",
						  "enabled": true,
						  "default": false,
						  "channel_group": "stable",
						  "available_upgrades": [
							"4.12.26"
						  ],
						  "rosa_enabled": true,
						  "hosted_control_plane_enabled": true,
						  "end_of_life_timestamp": "2024-05-17T00:00:00Z",
						  "ami_overrides": [
							{
							  "product": {
								"kind": "ProductLink",
								"id": "rosa",
								"href": "/api/clusters_mgmt/v1/products/rosa"
							  },
							  "region": {
								"kind": "CloudRegionLink",
								"id": "us-east-1",
								"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-1"
							  },
							  "ami": "ami-00354720d36d019f9"
							},
							{
							  "product": {
								"kind": "ProductLink",
								"id": "rosa",
								"href": "/api/clusters_mgmt/v1/products/rosa"
							  },
							  "region": {
								"kind": "CloudRegionLink",
								"id": "us-east-2",
								"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-2"
							  },
							  "ami": "ami-0e677f92eb4180cc0"
							}
						  ],
						  "release_image": "quay.io/openshift-release-dev/ocp-release@sha256:5a4fb052cda1d14d1e306ce87e6b0ded84edddaa76f1cf401bcded99cef2ad84"
						},
						{
						  "kind": "Version",
						  "id": "openshift-v4.12.24",
						  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.12.24",
						  "raw_id": "4.12.24",
						  "enabled": true,
						  "default": false,
						  "channel_group": "stable",
						  "available_upgrades": [
							"4.12.25",
							"4.12.26"
						  ],
						  "rosa_enabled": true,
						  "hosted_control_plane_enabled": true,
						  "end_of_life_timestamp": "2024-05-17T00:00:00Z",
						  "ami_overrides": [
							{
							  "product": {
								"kind": "ProductLink",
								"id": "rosa",
								"href": "/api/clusters_mgmt/v1/products/rosa"
							  },
							  "region": {
								"kind": "CloudRegionLink",
								"id": "us-east-2",
								"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-2"
							  },
							  "ami": "ami-0e677f92eb4180cc0"
							},
							{
							  "product": {
								"kind": "ProductLink",
								"id": "rosa",
								"href": "/api/clusters_mgmt/v1/products/rosa"
							  },
							  "region": {
								"kind": "CloudRegionLink",
								"id": "us-east-1",
								"href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-1"
							  },
							  "ami": "ami-00354720d36d019f9"
							}
						  ],
						  "release_image": "quay.io/openshift-release-dev/ocp-release@sha256:b0b11eedf91175459b5d7aefcf3936d0cabf00f01ced756677483f5f26227328"
						}
					  ]
					}`

		// nolint:lll
		const nodePoolResponse = `{
						  "kind": "NodePool",
						  "href": "/api/clusters_mgmt/v1/clusters/243nmgjr5v2q9rn5sf3456euj2lcq5tn/node_pools/workers",
						  "id": "workers",
						  "replicas": 2,
						  "auto_repair": true,
						  "aws_node_pool": {
							"instance_type": "m5.xlarge",
							"instance_profile": "rosa-service-managed-integration-243nmgjr5v2q9rn5sf3456euj2lcq5tn-ad-int1-worker",
							"tags": {
							  "api.openshift.com/environment": "integration",
							  "api.openshift.com/id": "243nmgjr5v2q9rn5sf3456euj2lcq5tn",
							  "api.openshift.com/legal-entity-id": "1jIHnIbrnLH9kQD57W0BuPm78f1",
							  "api.openshift.com/name": "ad-int1",
							  "api.openshift.com/nodepool-hypershift": "ad-int1-workers",
							  "api.openshift.com/nodepool-ocm": "workers",
							  "red-hat-clustertype": "rosa",
							  "red-hat-managed": "true"
							}
						  },
						  "availability_zone": "us-west-2a",
						  "subnet": "subnet-0e3a4046c1c2f1078",
						  "status": {
							"current_replicas": 0,
							"message": "WaitingForAvailableMachines: NodeProvisioning"
						  },
						  "version": {
							"kind": "VersionLink",
							"id": "openshift-v4.12.%s",
							"href": "/api/clusters_mgmt/v1/versions/openshift-v4.12.%s"
						  },
						  "tuning_configs": []
						}`
		BeforeEach(func() {
			testRuntime.InitRuntime()
		})
		It("Fails if cluster is not hypershift and we are using hypershift specific flags", func() {
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(
				ContainSubstring("The '--machinepool' option is only supported for Hosted Control Planes"))
		})
		It("Fails if cluster is not ready", func() {
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterNotReady))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Cluster 'cluster1' is not yet ready"))
		})

		It("Cluster is ready and node pool no upgrade available", func() {
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			// A node pool
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				fmt.Sprintf(nodePoolResponse, "26", "26")))
			// No existing policy upgrade
			testRuntime.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "NodePoolUpgradePolicyList",
						"page": 1,
						"size": 0,
						"total": 0,
						"items": []
				}`,
				),
			)
			// available versions
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, versionListResponse))
			stdout, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(stdout).To(Equal(noUpgradeOutput))
			Expect(err).To(BeNil())
		})

		It("Cluster is ready and node pool can be upgraded with 2 upgrades available", func() {
			format.TruncatedDiff = true
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			// A node pool
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				fmt.Sprintf(nodePoolResponse, "24", "24")))
			// No existing policy upgrade
			testRuntime.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "NodePoolUpgradePolicyList",
						"page": 1,
						"size": 0,
						"total": 0,
						"items": []
				}`,
				),
			)
			// available versions
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, versionListResponse))
			stdout, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(stdout).To(Equal(upgradeAvailableOutput))
			Expect(err).To(BeNil())
		})

		It("Cluster is ready and node pool can be upgraded with 2 upgrades available, 1 is scheduled", func() {
			format.TruncatedDiff = false
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			// A node pool
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				fmt.Sprintf(nodePoolResponse, "24", "24")))
			// An existing policy upgrade
			// nolint:lll
			testRuntime.ApiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "NodePoolUpgradePolicyList",
						"page": 1,
						"size": 1,
						"total": 1,
						"items": [
							{
							"kind": "NodePoolUpgradePolicy",
							"id": "a33c8cae-013f-11ee-a3b2-acde48001122",
							"href": "/api/clusters_mgmt/v1/clusters/243nmgjr5v2q9rn5sf3456euj2lcq5tn/node_pools/upgrade_policies/a33c8cae-013f-11ee-a3b2-acde48001122",
							"schedule_type": "manual",
							"upgrade_type": "NodePool",
							"version": "4.12.25",
							"next_run": "2023-06-02T12:30:00Z",
							"cluster_id": "243nmgjr5v2q9rn5sf3456euj2lcq5tn",
							"enable_minor_version_upgrades": false,
							"creation_timestamp": "2023-06-02T14:18:52.828589+02:00",
							"last_update_timestamp": "2023-06-02T14:18:52.828589+02:00",
							"state": {
							"value": "pending",
							"description": "Upgrade policy defined, pending scheduling."
							}
						}
					]
				}`,
				),
			)
			// available versions
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, versionListResponse))
			stdout, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(stdout).To(Equal(ongoingUpgradeOutput))
			Expect(err).To(BeNil())
		})
	})
})
