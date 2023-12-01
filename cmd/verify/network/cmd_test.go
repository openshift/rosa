package network

import (
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
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

//nolint:lll
var _ = Describe("verify network", func() {
	var ssoServer, apiServer *ghttp.Server

	var cmd *cobra.Command
	var r *rosa.Runtime

	mockCluster, _ := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
		c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
		c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
		c.State(cmv1.ClusterStateReady)
	})
	clustersSuccess := test.FormatClusterList([]*cmv1.Cluster{mockCluster})

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
	var subnetsComplete = `
	{
		"page": 1,
		"size": 2,
		"total": 2,
		"items": [
		  {
			"href": "/api/clusters_mgmt/v1/network_verifications/subnet-0b761d44d3d9a4663/",
			"id": "subnet-0b761d44d3d9a4663",
			"state": "passed"
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
	var subnetPendingSuccess = `
	{
		"href": "/api/clusters_mgmt/v1/network_verifications/subnet-0b761d44d3d9a4663/",
		"id": "subnet-0b761d44d3d9a4663",
		"state": "pending"
	}
	`
	var subnetRunningSuccess = `
	{
		"href": "/api/clusters_mgmt/v1/network_verifications/subnet-0b761d44d3d9a4663/",
		"id": "subnet-0b761d44d3d9a4663",
		"state": "running"
	}
	`
	var subnetPassedSuccess = `
	{
		"href": "/api/clusters_mgmt/v1/network_verifications/subnet-0f87f640e56934cbc/",
		"id": "subnet-0f87f640e56934cbc",
		"state": "passed"
	}
	` // #nosec G101
	var successOutputPendingComplete = `INFO: subnet-0b761d44d3d9a4663: pending
INFO: subnet-0f87f640e56934cbc: passed
INFO: Run the following command to wait for verification to all subnets to complete:
rosa verify network --watch --status-only --region us-east-1 --subnet-ids subnet-0b761d44d3d9a4663,subnet-0f87f640e56934cbc
`
	var successOutputRunningComplete = `INFO: subnet-0b761d44d3d9a4663: running
INFO: subnet-0f87f640e56934cbc: passed
INFO: Run the following command to wait for verification to all subnets to complete:
rosa verify network --watch --status-only --region us-east-1 --subnet-ids subnet-0b761d44d3d9a4663,subnet-0f87f640e56934cbc
`
	var successOutputComplete = `INFO: subnet-0b761d44d3d9a4663: passed
INFO: subnet-0f87f640e56934cbc: passed
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

			mockCluster, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
				c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
				c.State(state)
			})
			Expect(err).To(BeNil())
			clusterList := test.FormatClusterList([]*cmv1.Cluster{mockCluster})

			// GET /api/clusters_mgmt/v1/clusters
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					clusterList,
				),
			)
			// GET /api/clusters_mgmt/v1/network_verifications/subnetA
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					subnetPendingSuccess,
				),
			)
			// GET /api/clusters_mgmt/v1/network_verifications/subnetB
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					subnetPassedSuccess,
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
		Entry("installing state", cmv1.ClusterStateInstalling, successOutputPendingComplete),
		Entry("uninstalling state", cmv1.ClusterStateUninstalling, successOutputPendingComplete),
	)
	It("Succeeds if --cluster with --role-arn, prints watch command", func() {
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
				subnetPendingSuccess,
			),
		)
		// GET /api/clusters_mgmt/v1/network_verifications/subnetB
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetPassedSuccess,
			),
		)
		cmd.Flags().Set(clusterFlag, "tomckay-vpc")
		cmd.Flags().Set(roleArnFlag, "arn:aws:iam::765374464689:role/tomckay-Installer-Role")
		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal(successOutputPendingComplete))
	})
	It("Succeeds if --cluster with --role-arn, prints watch command", func() {
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
				subnetRunningSuccess,
			),
		)
		// GET /api/clusters_mgmt/v1/network_verifications/subnetB
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetPassedSuccess,
			),
		)
		cmd.Flags().Set(clusterFlag, "tomckay-vpc")
		cmd.Flags().Set(roleArnFlag, "arn:aws:iam::765374464689:role/tomckay-Installer-Role")
		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal(successOutputRunningComplete))
	})
	It("Succeeds if --cluster with --role-arn, no need to print watch command", func() {
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
				subnetsComplete,
			),
		)
		// GET /api/clusters_mgmt/v1/network_verifications/subnetB
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetPassedSuccess,
			),
		)
		// GET /api/clusters_mgmt/v1/network_verifications/subnetB
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetPassedSuccess,
			),
		)
		cmd.Flags().Set(clusterFlag, "tomckay-vpc")
		cmd.Flags().Set(roleArnFlag, "arn:aws:iam::765374464689:role/tomckay-Installer-Role")
		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal(successOutputComplete))
	})
	It("Succeeds if custom --tags are supplied", func() {
		// POST /api/clusters_mgmt/v1/network_verifications
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetsComplete,
			),
		)
		// GET /api/clusters_mgmt/v1/network_verifications/subnetB
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetPassedSuccess,
			),
		)
		// GET /api/clusters_mgmt/v1/network_verifications/subnetB
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				subnetPassedSuccess,
			),
		)
		cmd.Flags().Set(roleArnFlag, "arn:aws:iam::765374464689:role/tomckay-Installer-Role")
		cmd.Flags().Set("tags", "test val1, test2 val2")
		cmd.Flags().Set(subnetIDsFlag, "subnet-0b761d44d3d9a4663,subnet-0f87f640e56934cbc")
		cmd.Flags().Set("region", "us-east-1")
		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal(successOutputComplete))
	})
	It("Fails if --tags are formatted incorrectly", func() {
		cmd.Flags().Set(roleArnFlag, "arn:aws:iam::765374464689:role/tomckay-Installer-Role")
		cmd.Flags().Set("tags", "test val1 test2 val2")
		cmd.Flags().Set(subnetIDsFlag, "subnet-0b761d44d3d9a4663,subnet-0f87f640e56934cbc")
		cmd.Flags().Set("region", "us-east-1")
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("invalid tag format"))
	})
	It("Fails if --cluster with --hosted-cp flag", func() {
		// GET /api/clusters_mgmt/v1/clusters
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				clustersSuccess,
			),
		)
		cmd.Flags().Set(clusterFlag, "dle-vpc")
		cmd.Flags().Lookup(hostedCpFlag).Changed = true
		err := runWithRuntime(r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring(
				"'--hosted-cp' flag is not required when running the network verifier with cluster"))
	})
})
