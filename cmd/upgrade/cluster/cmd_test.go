package cluster

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
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
	"github.com/spf13/cobra"
)

var _ = Describe("Upgrade", Ordered, func() {
	var ssoServer, apiServer *ghttp.Server

	var cmd *cobra.Command
	var r *rosa.Runtime
	const cronSchedule = "* * * * *"
	const timeSchedule = "10:00"
	const dateSchedule = "2023-06-01"
	var clusterNotFound = `
	{
	  "kind": "Error",
	  "id": "404",
	  "href": "/api/clusters_mgmt/v1/errors/404",
	  "code": "CLUSTERS-MGMT-404",
	  "reason": "Cluster 'cluster1' not found",
	  "operation_id": "8f4c6a3e-4d40-41fd-9288-60ee670ef846"
	}`
	var emptyClusterList = `
	{
		"kind": "ClusterList",
		"page": 1,
		"size": 1,
		"total": 0,
		"items": []
	}`

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
		ocmClient := ocm.NewClientWithConnection(connection)
		cmd = &cobra.Command{
			Use:   "cluster",
			Short: "Upgrade cluster",
			Long:  "Upgrade cluster to a new available version",
			Example: `  # Interactively schedule an upgrade on the cluster named "mycluster"
  rosa upgrade cluster --cluster=mycluster --interactive

  # Schedule a cluster upgrade within the hour
  rosa upgrade cluster -c mycluster --version 4.5.20`,
			Run: run,
		}
		ocm.SetClusterKey("cluster1")
		r = rosa.NewRuntime()
		r.OCMClient = ocmClient
		r.Creator = &aws.Creator{
			ARN:       "fake",
			AccountID: "123",
			IsSTS:     false,
		}
		DeferCleanup(r.Cleanup)
	})
	AfterEach(func() {
		// Close the servers:
		ssoServer.Close()
		apiServer.Close()
	})
	It("Fails if flag is missing", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterNotReady,
			),
		)
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("The '--control-plane' option is currently mandatory for Hosted Control Plane"))
	})
	It("Fails if cluster is not hypershift and we are using hypershift specific flags", func() {
		args.controlPlane = true
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				classicCluster,
			),
		)
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("The '--control-plane' option is only supported for Hosted Control Planes"))
	})

	It("Fails if cluster is not hypershift and we are using hypershift specific flags", func() {
		args.controlPlane = false
		args.schedule = cronSchedule
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				classicCluster,
			),
		)
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("The '--schedule' option is only supported for Hosted Control Planes"))
	})
	It("Fails if we are using minor version flag for manual upgrades", func() {
		args.controlPlane = true
		args.schedule = ""
		args.allowMinorVersionUpdates = true
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("The '--allow-minor-version-upgrades' " +
			"option needs to be used with --schedule"))
	})
	It("Fails if we are mixing scheduling type flags", func() {
		args.controlPlane = true
		args.schedule = cronSchedule
		args.scheduleDate = "31 Jan"
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("The '--schedule-date' and '--schedule-time' " +
			"options are mutually exclusive with '--schedule'"))
	})
	It("Fails if we are mixing automatic scheduling and version flags", func() {
		args.controlPlane = true
		args.schedule = cronSchedule
		args.scheduleDate = ""
		args.version = "4.13.0"
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("The '--schedule' " +
			"option is mutually exclusive with '--version'"))
	})
	It("Fails if cluster is not ready", func() {
		args.controlPlane = true
		args.allowMinorVersionUpdates = false
		args.schedule = ""
		args.scheduleDate = ""
		args.version = ""
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterNotReady,
			),
		)
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("Cluster 'cluster1' is not yet ready"))
	})
	It("Cluster is ready but no upgrade type specified", func() {
		args.controlPlane = true
		args.schedule = cronSchedule
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		// No existing policy upgrade
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
			"kind": "ControlPlaneUpgradePolicyList",
				"page": 1,
				"size": 0,
				"total": 0,
				"items": []
		}`,
			),
		)
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		// Missing the upgrade type
		Expect(err.Error()).To(ContainSubstring("Failed to find available upgrades"))
	})
	It("Cluster is ready but existing upgrade scheduled", func() {
		args.controlPlane = true
		args.schedule = cronSchedule
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		// An existing policy upgrade
		// nolint:lll
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
						"kind": "ControlPlaneUpgradePolicyList",
						"page": 1,
						"size": 1,
						"total": 1,
						"items": [
							{
							"kind": "ControlPlaneUpgradePolicy",
							"id": "a33c8cae-013f-11ee-a3b2-acde48001122",
							"href": "/api/clusters_mgmt/v1/clusters/243nmgjr5v2q9rn5sf3456euj2lcq5tn/control_plane/upgrade_policies/a33c8cae-013f-11ee-a3b2-acde48001122",
							"schedule": "30 12 * * *",
							"schedule_type": "automatic",
							"upgrade_type": "ControlPlane",
							"version": "4.12.18",
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
		err := runWithRuntime(r, cmd)
		// No error, it will just exit
		Expect(err).To(BeNil())
	})
	It("Cluster is ready and with automatic scheduling but bad cron format", func() {
		args.controlPlane = true
		// not a valid cron
		args.schedule = "* a"
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		// No existing policy upgrade
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
			"kind": "ControlPlaneUpgradePolicyList",
				"page": 1,
				"size": 0,
				"total": 0,
				"items": []
		}`,
			),
		)

		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		// Missing the upgrade type
		Expect(err.Error()).To(ContainSubstring("Schedule '* a' is not a valid cron expression"))
	})

	It("Cluster is ready and with manual scheduling but bad format", func() {
		args.controlPlane = true
		args.schedule = ""
		// Not a valid date format
		args.scheduleDate = "Jan 23"
		args.scheduleTime = timeSchedule
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		// No existing policy upgrade
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
			"kind": "ControlPlaneUpgradePolicyList",
				"page": 1,
				"size": 0,
				"total": 0,
				"items": []
		}`,
			),
		)

		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		// Missing the upgrade type
		Expect(err.Error()).To(ContainSubstring(
			"schedule date should use the format 'yyyy-mm-dd'\n   Schedule time should use the format 'HH:mm'"))
	})
	It("Cluster is ready and with manual scheduling but no upgrades available", func() {
		args.controlPlane = true
		args.schedule = ""
		// Not a valid date format
		args.scheduleDate = dateSchedule
		args.scheduleTime = timeSchedule
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		// No existing policy upgrade
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
				"kind": "ControlPlaneUpgradePolicyList",
				"page": 1,
				"size": 0,
				"total": 0,
				"items": []
				}`,
			),
		)
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
					  "kind": "Version",
					  "id": "openshift-v4.13.0",
					  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.13.0",
					  "raw_id": "4.13.0",
					  "enabled": true,
					  "default": false,
					  "channel_group": "stable",
					  "rosa_enabled": true,
					  "hosted_control_plane_enabled": true,
					  "release_image": "quay.io/openshift-release-dev/ocp-release@sha256:5"
					}`))

		err := runWithRuntime(r, cmd)
		// No upgrades, just return
		Expect(err).To(BeNil())
	})
	It("Cluster is ready and with manual scheduling and available upgrades but a wrong version in input", func() {
		args.controlPlane = true
		args.schedule = ""
		// Not a valid date format
		args.scheduleDate = dateSchedule
		args.scheduleTime = timeSchedule
		// The version we want to update to
		args.version = "4.13.4"
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		// No existing policy upgrade
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
				"kind": "ControlPlaneUpgradePolicyList",
				"page": 1,
				"size": 0,
				"total": 0,
				"items": []
				}`,
			),
		)
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
					  "kind": "Version",
					  "id": "openshift-v4.13.0",
					  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.13.0",
					  "raw_id": "4.13.0",
					  "enabled": true,
					  "default": false,
					  "channel_group": "stable",
					  "available_upgrades": [
						 "4.13.1"
					  ],
					  "rosa_enabled": true,
					  "hosted_control_plane_enabled": true,
					  "release_image": "quay.io/openshift-release-dev/ocp-release@sha256:4"
					}`))
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
					  "kind": "Version",
					  "id": "openshift-v4.13.1",
					  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.13.1",
					  "raw_id": "4.13.1",
					  "enabled": true,
					  "default": false,
					  "channel_group": "stable",
					  "available_upgrades": [
						 "4.13.2"
					  ],
					  "rosa_enabled": true,
					  "hosted_control_plane_enabled": true,
					  "release_image": "quay.io/openshift-release-dev/ocp-release@sha256:3"
					}`))

		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("Expected a valid version to upgrade cluster to.\nValid versions: [4.13.1]"))
	})
	It("Cluster is ready and with manual scheduling and one available upgrade", func() {
		args.controlPlane = true
		args.schedule = ""
		// Not a valid date format
		args.scheduleDate = dateSchedule
		args.scheduleTime = timeSchedule
		// The version we want to update to
		args.version = "4.13.1"
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterReady,
			),
		)
		// No existing policy upgrade
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
				"kind": "ControlPlaneUpgradePolicyList",
				"page": 1,
				"size": 0,
				"total": 0,
				"items": []
				}`,
			),
		)
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
					  "kind": "Version",
					  "id": "openshift-v4.13.0",
					  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.13.0",
					  "raw_id": "4.13.0",
					  "enabled": true,
					  "default": false,
					  "channel_group": "stable",
					  "available_upgrades": [
						 "4.13.1"
					  ],
					  "rosa_enabled": true,
					  "hosted_control_plane_enabled": true,
					  "release_image": "quay.io/openshift-release-dev/ocp-release@sha256:2"
					}`))
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				`{
					  "kind": "Version",
					  "id": "openshift-v4.13.1",
					  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.13.1",
					  "raw_id": "4.13.1",
					  "enabled": true,
					  "default": false,
					  "channel_group": "stable",
					  "available_upgrades": [
						 "4.13.2"
					  ],
					  "rosa_enabled": true,
					  "hosted_control_plane_enabled": true,
					  "release_image": "quay.io/openshift-release-dev/ocp-release@sha256:1"
					}`))
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNoContent, ""))
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusCreated, `{
					  "kind": "ControlPlaneUpgradePolicy",
					  "enable_minor_version_upgrades": false,
					  "schedule": "35 12 * * *",
					  "schedule_type": "automatic",
					  "upgrade_type": "ControlPlane"
					}`))
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound, clusterNotFound))
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK, emptyClusterList))
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("There is no cluster with identifier or name"))
	})
})
