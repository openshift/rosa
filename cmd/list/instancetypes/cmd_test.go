package instancetypes

import (
	"net/http"
	"time"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

//nolint:lll
var _ = Describe("list instance-types", func() {
	var (
		ssoServer, apiServer *ghttp.Server

		cmd *cobra.Command
		r   *rosa.Runtime

		// GET /api/accounts_mgmt/v1/current_account
		currentAccount = `
	{
		"kind": "Account",
		"organization": {
		  "href": "/api/accounts_mgmt/v1/organizations/123abc",
		  "id": "123abc",
		  "kind": "Organization"
		}
	  }
	`
		// GET /api/accounts_mgmt/v1/organizations/123abc/quota_cost
		orgQuota = `
	{
		"items": [
		  {
			"allowed": 20000,
			"consumed": 0,
			"href": "/api/accounts_mgmt/v1/organizations/123abc/quota_cost",
			"kind": "QuotaCost",
			"organization_id": "123abc",
			"quota_id": "compute.node|gpu|byoc|moa-self-supported",
			"related_resources": [
			  {
				"availability_zone_type": "any",
				"billing_model": "any",
				"byoc": "byoc",
				"cloud_provider": "aws",
				"cost": 1,
				"product": "MOA",
				"resource_name": "t4-gpu-48",
				"resource_type": "compute.node"
			  }
			]
		  }
		]
	  }
	`

		regionsSuccess = `
	{
		"kind": "CloudRegionList",
		"page": 1,
		"size": 31,
		"total": 31,
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
			"supports_hypershift": true
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
			"supports_hypershift": true
		  }
		]
	}
	`
		machinesSuccess = `
	{
		"kind": "MachineTypeList",
		"page": 1,
		"size": 2,
		"total": 2,
		"items": [
		  {
			"kind": "MachineType",
			"name": "dl1.24xlarge - Accelerated Computing",
			"category": "accelerated_computing",
			"size": "24xlarge",
			"id": "dl1.24xlarge",
			"href": "/api/clusters_mgmt/v1/machine_types/dl1.24xlarge",
			"memory": {
			  "value": 824633720832,
			  "unit": "B"
			},
			"cpu": {
			  "value": 96,
			  "unit": "vCPU"
			},
			"cloud_provider": {
			  "kind": "CloudProviderLink",
			  "id": "aws",
			  "href": "/api/clusters_mgmt/v1/cloud_providers/aws"
			},
			"ccs_only": true,
			"generic_name": "d1-gaudi-24x"
		  },
		  {
			"kind": "MachineType",
			"name": "g4dn.12xlarge - Accelerated Computing (4 GPUs)",
			"category": "accelerated_computing",
			"size": "12xlarge",
			"id": "g4dn.12xlarge",
			"href": "/api/clusters_mgmt/v1/machine_types/g4dn.12xlarge",
			"memory": {
			  "value": 206158430208,
			  "unit": "B"
			},
			"cpu": {
			  "value": 48,
			  "unit": "vCPU"
			},
			"cloud_provider": {
			  "kind": "CloudProviderLink",
			  "id": "aws",
			  "href": "/api/clusters_mgmt/v1/cloud_providers/aws"
			},
			"ccs_only": true,
			"generic_name": "t4-gpu-48"
		  }
		]
	}
	`
		machinesEmptySuccess = `
	{
		"kind": "MachineTypeList",
		"page": 1,
		"size": 0,
		"total": 0,
		"items": [
		]
	}
	`
		regionSuccessOutput = `INFO: Using fake_installer_arn for the Installer role
ID             CATEGORY               CPU_CORES  MEMORY
g4dn.12xlarge  accelerated_computing  48         192.0 GiB
`
		mockAwsClient *aws.MockClient
	)

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

		ctrl := gomock.NewController(GinkgoT())
		mockAwsClient = aws.NewMockClient(ctrl)
		r.AWSClient = mockAwsClient
		mockAwsClient.EXPECT().GetAWSAccessKeys().Return(&aws.AccessKey{
			AccessKeyID:     "abc123",
			SecretAccessKey: "abc123",
		}, nil).AnyTimes()

		DeferCleanup(r.Cleanup)
	})

	AfterEach(func() {
		ssoServer.Close()
		apiServer.Close()
	})

	It("Succeeds with --region", func() {

		cmd.Flags().Set("region", "us-east-1")
		cmd.Flags().Set("win-li", "false")

		mockAwsClient.EXPECT().FindRoleARNs(aws.InstallerAccountRole, "").Return([]string{"fake_installer_arn"}, nil)

		// POST /api/clusters_mgmt/v1/aws_inquiries/machine_types
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				regionsSuccess,
			),
		)

		// POST /api/clusters_mgmt/v1/aws_inquiries/machine_types
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				machinesSuccess,
			),
		)

		// GET /api/accounts_mgmt/v1/current_account
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				currentAccount,
			),
		)

		// GET /api/accounts_mgmt/v1/organizations/123abc/quota_cost
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				orgQuota,
			),
		)

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal(regionSuccessOutput))
	})

	It("Handles unknown --region", func() {

		cmd.Flags().Set("region", "us-east-xyz")
		cmd.Flags().Set("win-li", "false")

		mockAwsClient.EXPECT().FindRoleARNs(aws.InstallerAccountRole, "").Return([]string{"fake_installer_arn"}, nil)

		// POST /api/clusters_mgmt/v1/aws_inquiries/machine_types
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				regionsSuccess,
			),
		)

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("region 'us-east-xyz' not found"))
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal("INFO: Using fake_installer_arn for the Installer role\n"))
	})

	It("Succeeds", func() {

		cmd.Flags().Set("output", "yaml")

		// GET /api/clusters_mgmt/v1/machine_types
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				machinesSuccess,
			),
		)

		// GET /api/accounts_mgmt/v1/current_account
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				currentAccount,
			),
		)

		// GET /api/accounts_mgmt/v1/organizations/123abc/quota_cost
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				orgQuota,
			),
		)

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(ContainSubstring("d1-gaudi-24x"))
		Expect(stdout).To(ContainSubstring("t4-gpu-48"))
	})

	It("Succeeds with zero results", func() {

		// GET /api/clusters_mgmt/v1/machine_types
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				machinesEmptySuccess,
			),
		)

		// GET /api/accounts_mgmt/v1/current_account
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				currentAccount,
			),
		)

		// GET /api/accounts_mgmt/v1/organizations/123abc/quota_cost
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				orgQuota,
			),
		)

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, r, cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("there are no machine types supported for your account. Contact Red Hat support"))
		Expect(stderr).To(Equal(""))
		Expect(stdout).To(Equal(""))
	})
})
