/*
Copyright (c) 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocmrole

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	r         *rosa.Runtime
	ocmClient *ocm.Client
	awsClient *aws.MockClient
	ctrl      *gomock.Controller

	testAccountID = "111111111111"
)

func TestOCMRoleAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OCM Role API Suite")
}

var _ = Describe("OCM Role API", func() {
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		awsClient = aws.NewMockClient(ctrl)

		logger, err := logging.NewGoLoggerBuilder().
			Debug(false).
			Build()
		Expect(err).To(BeNil())

		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens("test-token").
			URL("http://fake.api").
			Build()
		Expect(err).To(BeNil())
		ocmClient = ocm.NewClientWithConnection(connection)

		r = &rosa.Runtime{
			Reporter:  reporter.CreateReporter(),
			OCMClient: ocmClient,
			AWSClient: awsClient,
		}
		r.Creator = &aws.Creator{
			AccountID: testAccountID,
			ARN:       fmt.Sprintf("arn:aws:iam::%s:user/test", testAccountID),
			Partition: "aws",
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("GetOrCreateOCMRole validation", func() {
		It("should fail when runtime is nil", func() {
			_, _, err := GetOrCreateOCMRole(nil, "test-prefix", ProfileStandard, "", "/", false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("runtime cannot be nil"))
		})

		It("should fail when AWS client is nil", func() {
			r.AWSClient = nil

			_, _, err := GetOrCreateOCMRole(r, "test-prefix", ProfileStandard, "", "/", false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AWS client not initialized"))
		})

		It("should fail when creator is nil", func() {
			r.Creator = nil

			_, _, err := GetOrCreateOCMRole(r, "test-prefix", ProfileStandard, "", "/", false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("creator not initialized"))
		})

		It("should fail when reporter is nil", func() {
			r.Reporter = nil

			_, _, err := GetOrCreateOCMRole(r, "test-prefix", ProfileStandard, "", "/", false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("reporter not initialized"))
		})

		It("should fail when OCM client is nil", func() {
			r.OCMClient = nil

			_, _, err := GetOrCreateOCMRole(r, "test-prefix", ProfileStandard, "", "/", false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("OCM client not initialized"))
		})

		It("should fail when prefix is empty", func() {
			_, _, err := GetOrCreateOCMRole(r, "", ProfileStandard, "", "/", false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("prefix cannot be empty"))
		})

		It("should fail when profile is invalid", func() {
			_, _, err := GetOrCreateOCMRole(r, "test-prefix", "InvalidProfile", "", "/", false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("profile must be one of"))
		})
	})

	Context("GetOrCreateOCMRole behavior (CAPA dependency scenarios)", func() {
		var ssoServer, apiServer *ghttp.Server
		var testRuntime *rosa.Runtime
		var tmpdir string

		BeforeEach(func() {
			var err error

			// Create temp directory for OCM config
			tmpdir, err = os.MkdirTemp("", ".ocm-config-*")
			Expect(err).ToNot(HaveOccurred())
			os.Setenv("OCM_CONFIG", tmpdir+"/ocm_config.json")

			// Create mock OCM servers
			ssoServer = MakeTCPServer()
			apiServer = MakeTCPServer()

			ssoServer.AppendHandlers(
				RespondWithAccessAndRefreshTokens(
					MakeTokenString("Bearer", 15*time.Minute),
					MakeTokenString("Refresh", 15*time.Minute),
				),
			)

			logger, err := logging.NewGoLoggerBuilder().Debug(false).Build()
			Expect(err).ToNot(HaveOccurred())

			connection, err := sdk.NewConnectionBuilder().
				Logger(logger).
				Tokens(MakeTokenString("Bearer", 15*time.Minute)).
				URL(apiServer.URL()).
				TokenURL(ssoServer.URL()).
				Build()
			Expect(err).ToNot(HaveOccurred())

			// Save OCM config so GetEnv() can read it
			config.Save(&config.Config{
				URL:          "https://api.openshift.com",
				AccessToken:  MakeTokenString("Bearer", 15*time.Minute),
				RefreshToken: MakeTokenString("Refresh", 15*time.Minute),
			})

			testRuntime = &rosa.Runtime{
				Reporter:  reporter.CreateReporter(),
				OCMClient: ocm.NewClientWithConnection(connection),
				AWSClient: awsClient,
				Creator: &aws.Creator{
					AccountID: testAccountID,
					ARN:       fmt.Sprintf("arn:aws:iam::%s:user/test", testAccountID),
					Partition: "aws",
				},
			}
		})

		AfterEach(func() {
			ssoServer.Close()
			apiServer.Close()
			os.Setenv("OCM_CONFIG", "")
			if tmpdir != "" {
				os.RemoveAll(tmpdir)
			}
		})

		It("should return existing role with created=false (idempotency)", func() {
			roleName := "test-prefix-OCM-Role-12345678"
			existingARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", testAccountID, roleName)

			// Mock OCM GetCurrentOrganization API
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{
					"id": "test-org-id",
					"organization": {
						"id": "test-org-id",
						"external_id": "12345678"
					}
				}`),
			)

			// Mock AWS: role exists as standard profile
			awsClient.EXPECT().CheckRoleExists(roleName).Return(true, existingARN, nil)
			awsClient.EXPECT().IsAdminRole(roleName).Return(false, nil)
			awsClient.EXPECT().IsNoConsoleRole(roleName).Return(false, nil)

			roleARN, created, err := GetOrCreateOCMRole(testRuntime, "test-prefix", ProfileStandard, "", "/", false)

			Expect(err).ToNot(HaveOccurred())
			Expect(created).To(BeFalse(), "should return created=false when role exists")
			Expect(roleARN).To(Equal(existingARN))
		})

		It("should fail when no-console policy is missing (orphan prevention)", func() {
			roleName := "test-prefix-OCM-Role-12345678"

			// Mock OCM GetCurrentOrganization API
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{
					"id": "test-org-id",
					"organization": {
						"id": "test-org-id",
						"external_id": "12345678"
					}
				}`),
			)

			awsClient.EXPECT().CheckRoleExists(roleName).Return(false, "", nil)

			// Mock OCM GetPolicies API - return policies WITHOUT no-console policy
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{
					"items": [
						{
							"id": "sts_ocm_permission_policy",
							"details": "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
						}
					]
				}`),
			)

			roleARN, created, err := GetOrCreateOCMRole(testRuntime, "test-prefix", ProfileNoConsole, "", "/", false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no-console OCM role profile is not yet enabled"))
			Expect(created).To(BeFalse())
			Expect(roleARN).To(BeEmpty())
		})
	})
})
