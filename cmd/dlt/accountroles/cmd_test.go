// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package accountroles

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	awsClient "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Delete account roles", func() {
	It("Deletes all account-roles if the user didn't specify topology", func() {
		deleteClassic, deleteHostedCP := setDeleteRoles(false, false)
		Expect(deleteClassic).To(Equal(true))
		Expect(deleteHostedCP).To(Equal(true))
	})
	It("Deletes only hosted CP account-roles if the user selected '--hosted-cp'", func() {
		deleteClassic, deleteHostedCP := setDeleteRoles(false, true)
		Expect(deleteClassic).To(Equal(false))
		Expect(deleteHostedCP).To(Equal(true))
	})
	It("Deletes only classic account-roles if the user selected '--classic'", func() {
		deleteClassic, deleteHostedCP := setDeleteRoles(true, false)
		Expect(deleteClassic).To(Equal(true))
		Expect(deleteHostedCP).To(Equal(false))
	})
	It("Deletes all account-roles if the user selected both '--classic' and '--hosted-cp'", func() {
		deleteClassic, deleteHostedCP := setDeleteRoles(true, true)
		Expect(deleteClassic).To(Equal(true))
		Expect(deleteHostedCP).To(Equal(true))
	})
})

var _ = Describe("runWithRuntime", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		args = struct {
			prefix                     string
			hostedCP                   bool
			classic                    bool
			deleteHcpSharedVpcPolicies bool
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

	Context("with mode set to manual and valid env", func() {
		BeforeEach(func() {
			Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed())
			setTestOCMConfig(GinkgoT())
		})

		AfterEach(func() {
			interactive.SetModeKey("")
			Cmd.Flag("mode").Changed = false
		})

		It("returns error when prefix exceeds 32 characters", func() {
			args.prefix = "a]b]c]d]e]f]g]h]i]j]k]l]m]n]o]p]q"
			// GetAllClusters
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				`{"kind":"ClusterList","items":[],"total":0}`))
			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a prefix with no more than 32 characters"))
		})

		It("returns error when prefix has invalid characters", func() {
			args.prefix = "bad prefix!"
			// GetAllClusters
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				`{"kind":"ClusterList","items":[],"total":0}`))
			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid role prefix matching"))
		})

		It("warns when no classic roles found for prefix", func() {
			args.prefix = "MyPrefix"
			args.classic = true
			Cmd.Flag("classic").Changed = true
			DeferCleanup(func() { Cmd.Flag("classic").Changed = false })

			// GetAllClusters
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				`{"kind":"ClusterList","items":[],"total":0}`))

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().GetAccountRoleForCurrentEnvWithPrefix(
				gomock.Any(), "MyPrefix", gomock.Any(),
			).Return([]awsClient.Role{}, nil)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("returns OCM environment errors", func() {
		Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
			"expected manual mode to be configured")
		DeferCleanup(func() {
			interactive.SetModeKey("")
			Cmd.Flag("mode").Changed = false
		})

		configFile := filepath.Join(GinkgoT().TempDir(), "ocm.json")
		Expect(os.WriteFile(configFile, []byte("{invalid-json"), 0600)).To(Succeed(),
			"expected the invalid OCM config fixture to be written")
		GinkgoT().Setenv("OCM_CONFIG", configFile)

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("error getting environment")),
			"expected OCM environment errors to be returned")
	})

	It("returns cluster lookup errors", func() {
		Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
			"expected manual mode to be configured")
		DeferCleanup(func() {
			interactive.SetModeKey("")
			Cmd.Flag("mode").Changed = false
		})
		setTestOCMConfig(GinkgoT())
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
			`{"kind":"Error","id":"500","reason":"clusters unavailable"}`))

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("error getting clusters")),
			"expected cluster lookup errors to be returned")
	})

	It("returns interactive prefix errors", func() {
		Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
			"expected manual mode to be configured")
		interactive.SetEnabled(true)
		DeferCleanup(func() {
			interactive.SetEnabled(false)
			interactive.SetModeKey("")
			Cmd.Flag("mode").Changed = false
		})
		setTestOCMConfig(GinkgoT())
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"kind":"ClusterList","items":[],"total":0}`))

		err := runWithRuntime(t.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("expected a valid role prefix")),
			"expected interactive prefix input errors to be returned")
	})

	DescribeTable("returns account-role lookup errors",
		func(flag string, configure func()) {
			Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
				"expected manual mode to be configured")
			configure()
			Cmd.Flag(flag).Changed = true
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
				Cmd.Flag(flag).Changed = false
			})
			setTestOCMConfig(GinkgoT())
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				`{"kind":"ClusterList","items":[],"total":0}`))

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().GetAccountRoleForCurrentEnvWithPrefix(
				gomock.Any(), "MyPrefix", gomock.Any(),
			).Return(nil, errors.New("role lookup failed"))

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("role lookup failed")),
				"expected account-role lookup errors to be returned")
		},
		Entry("for classic roles", "classic", func() {
			args.prefix = "MyPrefix"
			args.classic = true
		}),
		Entry("for hosted CP roles", "hosted-cp", func() {
			args.prefix = "MyPrefix"
			args.hostedCP = true
		}),
	)
})

var _ = Describe("buildCommand", func() {
	It("generates detach and delete commands for unmanaged policies", func() {
		roleNames := []string{"MyPrefix-Installer-Role"}
		policyMap := map[string][]awsClient.PolicyDetail{
			"MyPrefix-Installer-Role": {
				{PolicyType: awsClient.Attached, PolicyArn: "arn:aws:iam::123:policy/installer-policy"},
			},
		}
		arbitraryPolicyMap := map[string][]awsClient.PolicyDetail{}
		policiesOutput := []*iam.GetPolicyOutput{}

		result := buildCommand(roleNames, policyMap, arbitraryPolicyMap, false, policiesOutput)
		Expect(result).To(ContainSubstring("detach-role-policy"))
		Expect(result).To(ContainSubstring("delete-policy"))
		Expect(result).To(ContainSubstring("delete-role"))
		Expect(result).To(ContainSubstring("MyPrefix-Installer-Role"))
	})

	It("skips delete-policy for managed policies", func() {
		roleNames := []string{"MyPrefix-Installer-Role"}
		policyMap := map[string][]awsClient.PolicyDetail{
			"MyPrefix-Installer-Role": {
				{PolicyType: awsClient.Attached, PolicyArn: "arn:aws:iam::123:policy/installer-policy"},
			},
		}
		arbitraryPolicyMap := map[string][]awsClient.PolicyDetail{}
		policiesOutput := []*iam.GetPolicyOutput{}

		result := buildCommand(roleNames, policyMap, arbitraryPolicyMap, true, policiesOutput)
		Expect(result).To(ContainSubstring("detach-role-policy"))
		Expect(result).NotTo(ContainSubstring("delete-policy"))
		Expect(result).To(ContainSubstring("delete-role"))
	})
})

func setTestOCMConfig(t GinkgoTInterface) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "ocm.json")
	cfg := `{"url":"https://api.stage.openshift.com","access_token":"test","client_id":"test","token_url":"https://sso.test"}`
	Expect(os.WriteFile(configFile, []byte(cfg), 0600)).To(Succeed())
	t.Setenv("OCM_CONFIG", configFile)
}
