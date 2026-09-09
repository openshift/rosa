package accountroles

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	awsClient "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
)

const (
	// Empty versions response: no versions available
	versionsResponse = `{"kind":"VersionList","page":1,"size":0,"total":0,"items":[]}`
	// Valid versions response with a single version
	validVersionsResponse = `{"kind":"VersionList","page":1,"size":1,"total":1,"items":[` +
		`{"kind":"Version","id":"openshift-v4.14.0","raw_id":"4.14.0",` +
		`"enabled":true,"default":true,"rosa_enabled":true,"channel_group":"stable"}]}`
)

const cmdTestExternalID = "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"

var _ = Describe("validateAccountRolesSTSExternalID", func() {
	It("accepts a valid external-id", func() {
		err := validateAccountRolesSTSExternalID(cmdTestExternalID)
		Expect(err).NotTo(HaveOccurred(), "valid external-id should pass validation")
	})

	It("accepts an empty external-id", func() {
		err := validateAccountRolesSTSExternalID("")
		Expect(err).NotTo(HaveOccurred(), "empty external-id should pass validation")
	})

	It("rejects an invalid external-id", func() {
		err := validateAccountRolesSTSExternalID("x")
		Expect(err).To(HaveOccurred(), "invalid external-id should fail validation")
	})
})

var _ = Describe("runWithRuntime", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		args = struct {
			prefix              string
			permissionsBoundary string
			path                string
			version             string
			channelGroup        string
			managed             bool
			forcePolicyCreation bool
			hostedCP            bool
			classic             bool
			route53RoleArn      string
			vpcEndpointRoleArn  string
			externalID          string
		}{}
		interactive.SetEnabled(false)
		interactive.SetModeKey("")
	})

	It("returns error when GetMode fails", func() {
		interactive.SetModeKey("invalid-mode")
		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Invalid mode"))
	})

	It("returns error when GetEnv fails", func() {
		tmpDir := GinkgoT().TempDir()
		configFile := filepath.Join(tmpDir, "ocm.json")
		Expect(os.WriteFile(configFile, []byte("{invalid-json"), 0600)).To(Succeed())
		GinkgoT().Setenv("OCM_CONFIG", configFile)

		args.classic = true
		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to determine OCM environment"))
	})

	Context("with valid env", func() {
		BeforeEach(func() {
			tmpDir := GinkgoT().TempDir()
			configFile := filepath.Join(tmpDir, "ocm.json")
			cfg := `{"url":"https://api.stage.openshift.com","access_token":"test","client_id":"test","token_url":"https://sso.test"}`
			Expect(os.WriteFile(configFile, []byte(cfg), 0600)).To(Succeed())
			GinkgoT().Setenv("OCM_CONFIG", configFile)
		})

		It("returns error when force-policy-creation is used with managed policies", func() {
			args.classic = true
			args.managed = true
			args.forcePolicyCreation = true
			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("forcing creation of policies only works for unmanaged policies"))
		})

		It("returns error when AWS credentials validation fails", func() {
			args.classic = true

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(false, nil)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AWS credentials are invalid"))
		})

		It("returns error when GetPolicyVersion fails", func() {
			args.classic = true

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed())
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})

			// GetPolicyVersion -> GetLatestVersion -> GetVersions: return error
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusInternalServerError, `{"kind":"Error","id":"500","reason":"versions unavailable"}`),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting version"))
		})

		It("returns error when prefix exceeds 32 characters", func() {
			args.classic = true
			args.prefix = strings.Repeat("a", 33)

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed())
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, validVersionsResponse),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a prefix with no more than 32 characters"))
		})

		It("returns error when prefix has invalid characters", func() {
			args.classic = true
			args.prefix = "bad prefix!"

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed())
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, validVersionsResponse),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid role prefix matching"))
		})

		It("returns error when non-HCP prefix ends with -HCP", func() {
			args.classic = true
			args.prefix = "MyPrefix-HCP"

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed())
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, validVersionsResponse),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("the '-HCP' suffix is reserved for hosted CP managed policies"))
		})
	})
})
