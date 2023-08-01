package network

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
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

//nolint:lll
var _ = Describe("verify network", func() {
	var ssoServer, apiServer *ghttp.Server

	var cmd *cobra.Command
	var r *rosa.Runtime

	var clustersSuccess = `
	{
		"kind": "ClusterList",
		"page": 1,
		"size": 1,
		"total": 1,
		"items": [
		  {
			"kind": "Cluster",
			"id": "24vf9iitg3p6tlml88iml6j6mu095mh8",
			"href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8",
			"name": "tomckay-vpc",
			"external_id": "a2db0ba0-0418-4da2-a3d0-1c976ba1ed0b",
			"infra_id": "tomckay-vpc-n7mtk",
			"display_name": "tomckay-vpc",
			"creation_timestamp": "2023-07-14T12:42:53.549435Z",
			"activity_timestamp": "2023-07-14T12:42:53.549435Z",
			"cloud_provider": {
			  "kind": "CloudProviderLink",
			  "id": "aws",
			  "href": "/api/clusters_mgmt/v1/cloud_providers/aws"
			},
			"subscription": {
			  "kind": "SubscriptionLink",
			  "id": "2SZ00THE7B78LS8E0CATLf49QZr",
			  "href": "/api/accounts_mgmt/v1/subscriptions/2SZ00THE7B78LS8E0CATLf49QZr"
			},
			"region": {
			  "kind": "CloudRegionLink",
			  "id": "us-east-1",
			  "href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-1"
			},
			"console": {
			  "url": "https://console-openshift-console.apps.tomckay-vpc.kh8d.i1.devshift.org"
			},
			"api": {
			  "url": "https://api.tomckay-vpc.kh8d.i1.devshift.org:6443",
			  "listening": "external"
			},
			"nodes": {
			  "master": 3,
			  "infra": 2,
			  "compute": 2,
			  "availability_zones": [
				"us-east-1a"
			  ],
			  "compute_machine_type": {
				"kind": "MachineTypeLink",
				"id": "m5.xlarge",
				"href": "/api/clusters_mgmt/v1/machine_types/m5.xlarge"
			  },
			  "infra_machine_type": {
				"kind": "MachineTypeLink",
				"id": "r5.xlarge",
				"href": "/api/clusters_mgmt/v1/machine_types/r5.xlarge"
			  }
			},
			"state": "ready",
			"flavour": {
			  "kind": "FlavourLink",
			  "id": "osd-4",
			  "href": "/api/clusters_mgmt/v1/flavours/osd-4"
			},
			"groups": {
			  "kind": "GroupListLink",
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/groups"
			},
			"properties": {
			  "rosa_cli_version": "1.2.22",
			  "rosa_creator_arn": "arn:aws:iam::765374464689:user/tomckay@redhat.com"
			},
			"aws": {
			  "subnet_ids": [
				"subnet-0b761d44d3d9a4663",
				"subnet-0f87f640e56934cbc"
			  ],
			  "private_link": false,
			  "private_link_configuration": {
				"kind": "PrivateLinkConfigurationLink",
				"href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/aws/private_link_configuration"
			  },
			  "sts": {
				"enabled": true,
				"role_arn": "arn:aws:iam::765374464689:role/tomckay-Installer-Role",
				"support_role_arn": "arn:aws:iam::765374464689:role/tomckay-Support-Role",
				"oidc_endpoint_url": "https://rh-oidc-dev.s3.us-east-1.amazonaws.com/24vf9iitg3p6tlml88iml6j6mu095mh8",
				"operator_iam_roles": [
				  {
					"id": "",
					"name": "cloud-credentials",
					"namespace": "openshift-ingress-operator",
					"role_arn": "arn:aws:iam::765374464689:role/tomckay-vpc-t8q3-openshift-ingress-operator-cloud-credentials",
					"service_account": ""
				  },
				  {
					"id": "",
					"name": "ebs-cloud-credentials",
					"namespace": "openshift-cluster-csi-drivers",
					"role_arn": "arn:aws:iam::765374464689:role/tomckay-vpc-t8q3-openshift-cluster-csi-drivers-ebs-cloud-credent",
					"service_account": ""
				  },
				  {
					"id": "",
					"name": "cloud-credentials",
					"namespace": "openshift-cloud-network-config-controller",
					"role_arn": "arn:aws:iam::765374464689:role/tomckay-vpc-t8q3-openshift-cloud-network-config-controller-cloud",
					"service_account": ""
				  },
				  {
					"id": "",
					"name": "aws-cloud-credentials",
					"namespace": "openshift-machine-api",
					"role_arn": "arn:aws:iam::765374464689:role/tomckay-vpc-t8q3-openshift-machine-api-aws-cloud-credentials",
					"service_account": ""
				  },
				  {
					"id": "",
					"name": "cloud-credential-operator-iam-ro-creds",
					"namespace": "openshift-cloud-credential-operator",
					"role_arn": "arn:aws:iam::765374464689:role/tomckay-vpc-t8q3-openshift-cloud-credential-operator-cloud-crede",
					"service_account": ""
				  },
				  {
					"id": "",
					"name": "installer-cloud-credentials",
					"namespace": "openshift-image-registry",
					"role_arn": "arn:aws:iam::765374464689:role/tomckay-vpc-t8q3-openshift-image-registry-installer-cloud-creden",
					"service_account": ""
				  }
				],
				"instance_iam_roles": {
				  "master_role_arn": "arn:aws:iam::765374464689:role/tomckay-ControlPlane-Role",
				  "worker_role_arn": "arn:aws:iam::765374464689:role/tomckay-Worker-Role"
				},
				"auto_mode": false,
				"operator_role_prefix": "tomckay-vpc-t8q3",
				"managed_policies": false
			  },
			  "tags": {
				"red-hat-clustertype": "rosa",
				"red-hat-managed": "true"
			  },
			  "audit_log": {
				"role_arn": ""
			  },
			  "ec2_metadata_http_tokens": ""
			},
			"dns": {
			  "base_domain": "kh8d.i1.devshift.org"
			},
			"network": {
			  "type": "OVNKubernetes",
			  "machine_cidr": "10.0.0.0/16",
			  "service_cidr": "172.30.0.0/16",
			  "pod_cidr": "10.128.0.0/14",
			  "host_prefix": 23
			},
			"external_configuration": {
			  "kind": "ExternalConfiguration",
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/external_configuration",
			  "syncsets": {
				"kind": "SyncsetListLink",
				"href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/external_configuration/syncsets"
			  },
			  "labels": {
				"kind": "LabelListLink",
				"href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/external_configuration/labels"
			  },
			  "manifests": {
				"kind": "ManifestListLink",
				"href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/external_configuration/manifests"
			  }
			},
			"multi_az": false,
			"managed": true,
			"ccs": {
			  "enabled": true,
			  "disable_scp_checks": false
			},
			"version": {
			  "kind": "Version",
			  "id": "openshift-v4.13.4",
			  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.13.4",
			  "raw_id": "4.13.4",
			  "channel_group": "stable",
			  "end_of_life_timestamp": "2024-09-17T00:00:00Z"
			},
			"identity_providers": {
			  "kind": "IdentityProviderListLink",
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/identity_providers"
			},
			"aws_infrastructure_access_role_grants": {
			  "kind": "AWSInfrastructureAccessRoleGrantLink",
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/aws_infrastructure_access_role_grants"
			},
			"addons": {
			  "kind": "AddOnInstallationListLink",
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/addons"
			},
			"ingresses": {
			  "kind": "IngressListLink",
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/ingresses"
			},
			"machine_pools": {
			  "kind": "MachinePoolListLink",
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/machine_pools"
			},
			"inflight_checks": {
			  "kind": "InflightCheckListLink",
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/inflight_checks"
			},
			"product": {
			  "kind": "ProductLink",
			  "id": "rosa",
			  "href": "/api/clusters_mgmt/v1/products/rosa"
			},
			"status": {
			  "state": "ready",
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
			"billing_model": "standard",
			"disable_user_workload_monitoring": false,
			"managed_service": {
			  "enabled": false,
			  "managed": false
			},
			"hypershift": {
			  "enabled": false
			},
			"byo_oidc": {
			  "enabled": false
			},
			"delete_protection": {
			  "href": "/api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/delete_protection",
			  "enabled": false
			}
		  }
		]
	}
	`

	var subnetsSuccess = `
	{
		"page": 1,
		"size": 2,
		"total": 2,
		"items": [
		  {
			"href": "/api/clusters_mgmt/v1/network_verifications/subnet-0b761d44d3d9a4663/",
			"id": "subnet-0b761d44d3d9a4663",
			"state": "pending"
		  },
		  {
			"href": "/api/clusters_mgmt/v1/network_verifications/subnet-0f87f640e56934cbc/",
			"id": "subnet-0f87f640e56934cbc",
			"state": "passed"
		  }
		],
		"cloud_provider_data": {

		}
	}
	`
	var subnetASuccess = `
	{
		"href": "/api/clusters_mgmt/v1/network_verifications/subnet-0b761d44d3d9a4663/",
		"id": "subnet-0b761d44d3d9a4663",
		"state": "pending"
	}
	`
	var subnetBSuccess = `
	{
		"href": "/api/clusters_mgmt/v1/network_verifications/subnet-0f87f640e56934cbc/",
		"id": "subnet-0f87f640e56934cbc",
		"state": "passed"
	}
	`
	var successOutputPendingComplete = `INFO: subnet-0b761d44d3d9a4663: pending
INFO: subnet-0f87f640e56934cbc: passed
INFO: Run the following command to wait for verification to all subnets to complete:
rosa verify network --watch --status-only --region us-east-1 --subnet-ids subnet-0b761d44d3d9a4663,subnet-0f87f640e56934cbc
`
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
			Debug(false).
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

		cmd = makeCmd()
		initFlags(cmd)

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
		ssoServer.Close()
		apiServer.Close()
	})

	It("Fails if neither --cluster nor --subnet-ids", func() {
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("At least one subnet IDs is required"))
	})
	It("Fails if no --region without --cluster", func() {
		cmd.Flags().Set(subnetIDsFlag, "abc,def")
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("Region is required"))
	})
	It("Fails if no --role-arn without --cluster", func() {
		cmd.Flags().Set(subnetIDsFlag, "abc,def")
		cmd.Flags().Set("region", "us-east1")
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("role-arn is required"))
	})
	DescribeTable("Test --cluster with various statuses",
		func(state cmv1.ClusterState, expected string) {
			cmd.Flags().Lookup(statusOnlyFlag).Changed = true
			cmd.Flags().Set(clusterFlag, "tomckay-vpc")

			var clusterJson map[string]any
			json.Unmarshal([]byte(clustersSuccess), &clusterJson)
			clusterJson["items"].([]interface{})[0].(map[string]interface{})["status"].(map[string]interface{})["state"] = state
			updatedCluster, _ := json.Marshal(clusterJson)

			// GET /api/clusters_mgmt/v1/clusters
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					string(updatedCluster),
				),
			)
			// GET /api/clusters_mgmt/v1/network_verifications/subnetA
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					subnetASuccess,
				),
			)
			// GET /api/clusters_mgmt/v1/network_verifications/subnetB
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					subnetBSuccess,
				),
			)
			stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
			Expect(err).To(BeNil())
			Expect(stderr).To(Equal(""))
			Expect(stdout).To(Equal(expected))
		},
		Entry("ready state", cmv1.ClusterStateReady, successOutputPendingComplete),
		Entry("error state", cmv1.ClusterStateError, successOutputPendingComplete),
		Entry("hibernating state", cmv1.ClusterStateHibernating, successOutputPendingComplete),
		Entry("hibernating state", cmv1.ClusterStateInstalling, successOutputPendingComplete),
		Entry("hibernating state", cmv1.ClusterStateUninstalling, successOutputPendingComplete),
	)
	It("Succeeds if --cluster with --role-arn", func() {
		// GET /api/clusters_mgmt/v1/clusters
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				clustersSuccess,
			),
		)
		// POST /api/clusters_mgmt/v1/network_verifications
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetsSuccess,
			),
		)
		// GET /api/clusters_mgmt/v1/network_verifications/subnetA
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetASuccess,
			),
		)
		// GET /api/clusters_mgmt/v1/network_verifications/subnetB
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetBSuccess,
			),
		)
		cmd.Flags().Set(clusterFlag, "tomckay-vpc")
		cmd.Flags().Set(roleArnFlag, "arn:aws:iam::765374464689:role/tomckay-Installer-Role")
		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal(successOutputPendingComplete))
	})
})
