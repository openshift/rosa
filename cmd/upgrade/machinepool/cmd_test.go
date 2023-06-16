package machinepool

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/openshift/rosa/pkg/test"
)

const (
	scheduleTime        = "10:00"
	invalidScheduleDate = "25h December"
	validScheduleDate   = "2023-12-25"
	invalidVersionError = `Expected a valid machine pool version: A valid version number must be specified
Valid versions: 4.12.26 4.12.25 4.12.24`
)

var _ = Describe("Upgrade machine pool", func() {
	Context("Upgrade machine pool command", func() {
		var testRuntime test.TestingRuntime
		var nodePoolName = "nodepool85"
		mockClusterError, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateError)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})
		Expect(err).To(BeNil())
		hypershiftClusterNotReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterError})

		mockClusterReady, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})
		Expect(err).To(BeNil())
		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

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

		// nolint:lll
		const nodePoolUpgradePolicy = `{
  "kind": "NodePoolUpgradePolicyList",
  "page": 1,
  "size": 1,
  "total": 1,
  "items": [
    {
      "kind": "NodePoolUpgradePolicy",
      "id": "e2800d05-3534-11ee-b9bc-0a580a811709",
      "href": "/api/clusters_mgmt/v1/clusters/25f96obptkqc5mh9vdc779jiqb3sihnn/node_pools/workers/upgrade_policies/e2800d05-3534-11ee-b9bc-0a580a811709",
      "schedule_type": "manual",
      "upgrade_type": "NodePool",
      "version": "4.12.25",
      "next_run": "2023-08-07T15:22:00Z",
      "cluster_id": "25f96obptkqc5mh9vdc779jiqb3sihnn",
      "node_pool_id": "workers",
      "enable_minor_version_upgrades": true,
      "creation_timestamp": "2023-08-07T15:12:54.967835Z",
      "last_update_timestamp": "2023-08-07T15:12:54.967835Z",
      "state": {
        "value": "scheduled",
        "description": "Upgrade scheduled."
      }
    }
  ]
}`

		const noNodePoolUpgradePolicy = `{
  "kind": "NodePoolUpgradePolicyList",
  "page": 1,
  "size": 0,
  "total": 0,
  "items": []
}`

		BeforeEach(func() {
			testRuntime.InitRuntime()
		})
		It("Fails if cluster is not ready", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterNotReady))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd, []string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Cluster 'cluster1' is not yet ready"))
		})
		It("Cluster is ready but node pool not found", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd, []string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				"Failed to get scheduled upgrades for machine pool 'nodepool85': " +
					"Machine pool 'nodepool85' does not exist for hosted cluster 'cluster1'"))
		})
		It("Cluster is ready and there is a scheduled upgraded", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
			_, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(ContainSubstring(
				"WARN: There is already a scheduled upgrade to version 4.12.25 on 2023-08-07 15:22 UTC"))
		})
		It("Cluster is ready and there is a scheduled upgraded", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
			_, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(ContainSubstring(
				"WARN: There is already a scheduled upgrade to version 4.12.25 on 2023-08-07 15:22 UTC"))
		})
		It("Cluster is ready and there is no scheduled upgraded but schedule date is invalid -> fail", func() {
			args.scheduleTime = scheduleTime
			args.scheduleDate = invalidScheduleDate
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, versionListResponse))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				"Failed to build schedule upgrade for machine pool nodepool85 in cluster 'cluster1': " +
					"schedule date should use the format 'yyyy-mm-dd'"))
			Expect(stdout).To(BeEmpty())
			Expect(stderr).To(BeEmpty())
		})
		It("Cluster is ready and there is no scheduled upgraded and an invalid version is specified -> fail",
			func() {
				args.scheduleTime = scheduleTime
				args.scheduleDate = validScheduleDate
				Cmd.Flags().Set("version", "4.13.26")
				Cmd.Flags().Set("interactive", "false")
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, versionListResponse))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{nodePoolName})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(invalidVersionError))
				Expect(stderr).To(BeEmpty())
				Expect(stdout).To(BeEmpty())
			})
		It("Cluster is ready and there is no scheduled upgraded and a version is specified -> success", func() {
			args.scheduleTime = scheduleTime
			args.scheduleDate = validScheduleDate
			Cmd.Flags().Set("version", "4.12.26")
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, versionListResponse))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring(
				"Upgrade successfully scheduled for the machine pool 'nodepool85' on cluster 'cluster1"))
		})
		It("Cluster is ready and there is no scheduled upgraded -> success", func() {
			args.scheduleTime = scheduleTime
			args.scheduleDate = validScheduleDate
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, versionListResponse))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring(
				"Upgrade successfully scheduled for the machine pool 'nodepool85' on cluster 'cluster1"))
		})
	})
})
