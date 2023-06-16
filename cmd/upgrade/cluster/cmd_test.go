package cluster

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
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
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

	var hypershiftClusterNotReady = `
{
  "kind": "ClusterList",
  "page": 1,
  "size": 1,
  "total": 1,
  "items": [
    {
      "kind": "Cluster",
      "id": "241p9e7ve372j8li66cdegd7t90ro5e4",
      "href": "/api/clusters_mgmt/v1/clusters/241p9e7ve372j8li66cdegd7t90ro5e4",
      "name": "cluster1",
      "external_id": "1cd93e5d-24e5-4c12-b3c5-2710ecd77e07",
      "display_name": "cluster1",
      "creation_timestamp": "2023-05-30T13:49:18.526324+02:00",
      "activity_timestamp": "2023-05-30T13:49:18.526324+02:00",
      "state": "error",
      "status": {
        "state": "error",
        "description": "Manifest work deletion is stuck",
        "dns_ready": true,
        "oidc_ready": true,
        "provision_error_message": "",
        "provision_error_code": "",
        "configuration_mode": "full",
        "limited_support_reason_count": 0
      },
      "hypershift": {
        "enabled": true
      }
    }
  ]
}`
	var hypershiftClusterReady = `
{
  "kind": "ClusterList",
  "page": 1,
  "size": 1,
  "total": 1,
  "items": [
    {
      "kind": "Cluster",
      "id": "241p9e7ve372j8li66cdegd7t90ro5e4",
      "href": "/api/clusters_mgmt/v1/clusters/241p9e7ve372j8li66cdegd7t90ro5e4",
      "name": "cluster1",
      "external_id": "1cd93e5d-24e5-4c12-b3c5-2710ecd77e07",
      "display_name": "cluster1",
      "creation_timestamp": "2023-05-30T13:49:18.526324+02:00",
      "activity_timestamp": "2023-05-30T13:49:18.526324+02:00",
      "cloud_provider": {
        "kind": "CloudProviderLink",
        "id": "aws",
        "href": "/api/clusters_mgmt/v1/cloud_providers/aws"
      },
      "subscription": {
        "kind": "SubscriptionLink",
        "id": "2QVmw1VcHO01lKSstX90BuhkkoP",
        "href": "/api/accounts_mgmt/v1/subscriptions/2QVmw1VcHO01lKSstX90BuhkkoP"
      },
      "region": {
        "kind": "CloudRegionLink",
        "id": "us-west-2",
        "href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-west-2"
      },
      "console": {
        "url": "https://console-openshift-console.apps.00un.hypershift.sdev.devshift.net"
      },
      "api": {
        "url": "https://aac3dfec32fdd455ba1d649e4b80a144-2bd3de32e0108b79.elb.us-west-2.amazonaws.com:6443",
        "listening": "external"
      },
      "nodes": {
        "compute": 2,
        "availability_zones": [
           "us-west-2a"
        ],
        "compute_machine_type": {
          "kind": "MachineTypeLink",
          "id": "m5.xlarge",
          "href": "/api/clusters_mgmt/v1/machine_types/m5.xlarge"
        }
      },
      "state": "ready",
        "status": {
        "state": "ready",
        "description": "",
        "dns_ready": true,
        "oidc_ready": true,
        "provision_error_message": "",
        "provision_error_code": "",
        "configuration_mode": "full",
        "limited_support_reason_count": 0
      },
      "node_drain_grace_period": {
        "value": 60,
        "unit": "minutes"
      },
      "etcd_encryption": false,
      "billing_model": "marketplace-aws",
      "disable_user_workload_monitoring": false,
      "managed_service": {
        "enabled": false,
        "managed": false
      },
      "hypershift": {
        "enabled": true
      },
      "byo_oidc": {
        "enabled": true
      },
      "delete_protection": {
        "href": "/api/clusters_mgmt/v1/clusters/241p9e7ve372j8li66cdegd7t90ro5e4/delete_protection",
        "enabled": false
      },
	  "version": {
		"kind": "Version",
		"id": "openshift-v4.12.18",
		"href": "/api/clusters_mgmt/v1/versions/openshift-v4.12.18",
		"raw_id": "4.12.18",
		"channel_group": "stable",
		"available_upgrades": [
		   "4.12.19"
		],
		"end_of_life_timestamp": "2024-03-17T00:00:00Z"
	  }
    }
  ]
}`
	var classicCluster = `
{
  "kind": "ClusterList",
  "page": 1,
  "size": 1,
  "total": 1,
  "items": [
    {
      "kind": "Cluster",
      "id": "241p9e7ve372j8li66cdegd7t90ro5e4",
      "href": "/api/clusters_mgmt/v1/clusters/241p9e7ve372j8li66cdegd7t90ro5e4",
      "name": "cluster1",
      "external_id": "1cd93e5d-24e5-4c12-b3c5-2710ecd77e07",
      "display_name": "cluster1",
      "creation_timestamp": "2023-05-30T13:49:18.526324+02:00",
      "activity_timestamp": "2023-05-30T13:49:18.526324+02:00",
      "state": "ready",
      "hypershift": {
        "enabled": false
      }
    }
  ]
}`

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
