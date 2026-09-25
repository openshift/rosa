// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package accountroles

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	awsClient "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/test"
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

var _ = Describe("getPolicyVersion", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
	})

	It("requests HCP versions for hosted control plane roles", func() {
		version, err := cmv1.NewVersion().
			ID("openshift-v5.0.0-candidate").
			RawID("5.0.0").
			Enabled(true).
			ROSAEnabled(true).
			ChannelGroup("candidate").
			Build()
		Expect(err).NotTo(HaveOccurred(), "expected the HCP version fixture to build")
		t.ApiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/versions"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query().Get("product")).To(Equal(ocm.HcpProduct),
						"expected HCP-only account roles to request product=hcp")
				},
				RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{version})),
			),
		)

		policyVersion, err := getPolicyVersion(
			t.RosaRuntime.OCMClient, "5.0", "candidate", false, true)

		Expect(err).NotTo(HaveOccurred(), "expected HCP policy-version lookup to succeed")
		Expect(policyVersion).To(Equal("5.0"), "expected HCP policy version 5.0")
	})

	It("rejects an HCP-only version when creating both role topologies", func() {
		version, err := cmv1.NewVersion().
			ID("openshift-v4.22.0-candidate").
			RawID("4.22.0").
			Enabled(true).
			ROSAEnabled(true).
			ChannelGroup("candidate").
			Build()
		Expect(err).NotTo(HaveOccurred(), "expected the Classic version fixture to build")
		t.ApiServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/versions"),
				func(_ http.ResponseWriter, request *http.Request) {
					Expect(request.URL.Query()).NotTo(HaveKey("product"),
						"expected dual-topology account roles to use the default product")
				},
				RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{version})),
			),
		)

		_, err = getPolicyVersion(t.RosaRuntime.OCMClient, "5.0", "candidate", true, true)

		Expect(err).To(HaveOccurred(), "expected an HCP-only version to be rejected for Classic roles")
	})

	It("returns version lookup errors", func() {
		t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
			`{"kind":"Error","code":"CLUSTERS-MGMT-500","reason":"internal error"}`))

		_, err := getPolicyVersion(t.RosaRuntime.OCMClient, "5.0", "candidate", false, true)

		Expect(err).To(HaveOccurred(), "expected HCP version lookup errors to be returned")
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
		}{
			prefix:       Cmd.Flags().Lookup("prefix").DefValue,
			channelGroup: Cmd.Flags().Lookup("channel-group").DefValue,
		}
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

		It("returns shared VPC flag validation errors", func() {
			args.hostedCP = true
			args.vpcEndpointRoleArn = "arn:aws:iam::123456789012:role/vpc-endpoint"

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("Must supply 'route53-role-arn' flag")),
				"expected a missing Route 53 role ARN to be rejected")
		})

		It("returns invalid shared VPC role ARN errors", func() {
			args.hostedCP = true
			args.vpcEndpointRoleArn = "invalid"
			args.route53RoleArn = "arn:aws:iam::123456789012:role/route53"

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("expected a valid policy ARN for vpc-endpoint-role-arn")),
				"expected an invalid VPC endpoint role ARN to be rejected")

			args.vpcEndpointRoleArn = "arn:aws:iam::123456789012:role/vpc-endpoint"
			args.route53RoleArn = "invalid"
			err = runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("expected a valid policy ARN for route53-role-arn")),
				"expected an invalid Route 53 role ARN to be rejected")
		})

		It("rejects unmanaged hosted CP policies", func() {
			args.hostedCP = true
			Cmd.Flag("hosted-cp").Changed = true
			Cmd.Flag("managed-policies").Changed = true
			DeferCleanup(func() {
				Cmd.Flag("hosted-cp").Changed = false
				Cmd.Flag("managed-policies").Changed = false
			})

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError("setting `hosted-cp` as unmanaged policies is not supported"),
				"expected unmanaged hosted CP policies to be rejected")
		})

		It("rejects classic managed policies in production", func() {
			configFile := filepath.Join(GinkgoT().TempDir(), "ocm.json")
			cfg := `{"url":"https://api.openshift.com","access_token":"test","client_id":"test","token_url":"https://sso.test"}`
			Expect(os.WriteFile(configFile, []byte(cfg), 0600)).To(Succeed(),
				"expected the production OCM config fixture to be written")
			GinkgoT().Setenv("OCM_CONFIG", configFile)

			args.classic = true
			args.managed = true
			Cmd.Flag("managed-policies").Changed = true
			DeferCleanup(func() { Cmd.Flag("managed-policies").Changed = false })

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError("classic ROSA managed policies are not supported in this environment"),
				"expected classic managed policies to be rejected in production")
		})

		It("returns error when AWS credentials validation fails", func() {
			args.classic = true

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(false, nil)
			preflightRan := false

			err := runWithRuntime(t.RosaRuntime, Cmd, func() { preflightRan = true })
			Expect(err).To(HaveOccurred(), "invalid AWS credentials should fail")
			Expect(err.Error()).To(ContainSubstring("AWS credentials are invalid"),
				"expected invalid AWS credentials error")
			Expect(preflightRan).To(BeFalse(), "preflight checks must wait for valid AWS credentials")
		})

		It("returns AWS credential validation errors", func() {
			args.classic = true

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(false, errors.New("credentials unavailable"))

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("error validating AWS credentials: credentials unavailable")),
				"expected AWS credential validation errors to be returned")
		})

		It("returns interactive prefix errors", func() {
			args.classic = true
			interactive.SetEnabled(true)
			DeferCleanup(func() { interactive.SetEnabled(false) })

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("expected a valid role prefix")),
				"expected interactive prefix input errors to be returned")
		})

		DescribeTable("returns argument validation errors",
			func(configure func(), expected string) {
				args.classic = true
				configure()

				mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

				Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed(),
					"expected auto mode to be configured")
				DeferCleanup(func() {
					interactive.SetModeKey("")
					Cmd.Flag("mode").Changed = false
				})

				err := runWithRuntime(t.RosaRuntime, Cmd)
				Expect(err).To(MatchError(ContainSubstring(expected)),
					"expected invalid account-role arguments to be rejected")
			},
			Entry("permissions boundary", func() { args.permissionsBoundary = "invalid" },
				"expected a valid policy ARN for permissions boundary"),
			Entry("role path", func() { args.path = "invalid" },
				"the specified value for path is invalid"),
			Entry("external ID", func() { args.externalID = "invalid external id" },
				"expected a valid STS external ID"),
		)

		It("rejects forced policy creation in manual mode", func() {
			args.classic = true
			args.forcePolicyCreation = true

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
				"expected manual mode to be configured")
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError("forcing creation of policies only works in auto mode"),
				"expected forced policy creation to require auto mode")
		})

		DescribeTable("returns automatic role creation errors",
			func(roleError error, expected string) {
				args.classic = true
				args.managed = true
				args.version = "4.22"
				Cmd.Flag("classic").Changed = true
				DeferCleanup(func() { Cmd.Flag("classic").Changed = false })

				mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
				mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
				mockAWS.EXPECT().CheckRoleExists(gomock.Any()).Return(false, "", roleError)

				Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed(),
					"expected auto mode to be configured")
				DeferCleanup(func() {
					interactive.SetModeKey("")
					Cmd.Flag("mode").Changed = false
				})
				appendPolicyAndVersionResponses(t, "4.22.0")

				err := runWithRuntime(t.RosaRuntime, Cmd)
				Expect(err).To(MatchError(ContainSubstring(expected)),
					"expected account-role creation errors to be returned")
			},
			Entry("regular errors", errors.New("role creation failed"), "role creation failed"),
			Entry("throttling errors", errors.New("Throttling: retry later"), "Throttling: retry later"),
		)

		It("returns an error when no role topology is selected", func() {
			args.version = "4.22"
			Cmd.Flag("classic").Changed = true
			Cmd.Flag("hosted-cp").Changed = true
			DeferCleanup(func() {
				Cmd.Flag("classic").Changed = false
				Cmd.Flag("hosted-cp").Changed = false
			})

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed(),
				"expected auto mode to be configured")
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})
			appendPolicyAndVersionResponses(t, "4.22.0")

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError("failed to initialize account role creator"),
				"expected an empty topology selection to be rejected")
		})

		It("rejects shared VPC roles in FedRAMP mode", func() {
			args.hostedCP = true
			args.version = "5.0"
			args.vpcEndpointRoleArn = "arn:aws:iam::123456789012:role/vpc-endpoint"
			args.route53RoleArn = "arn:aws:iam::123456789012:role/route53"
			Cmd.Flag("hosted-cp").Changed = true
			fedramp.Enable()
			DeferCleanup(func() {
				Cmd.Flag("hosted-cp").Changed = false
				fedramp.Disable()
			})

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed(),
				"expected auto mode to be configured")
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})
			appendPolicyAndVersionResponses(t, "5.0.0")

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError("HCP shared VPC not supported while using a govcloud region"),
				"expected shared VPC roles to be rejected in FedRAMP mode")
		})

		It("creates manual role commands", func() {
			args.classic = true
			args.version = "4.22"
			Cmd.Flag("classic").Changed = true
			DeferCleanup(func() { Cmd.Flag("classic").Changed = false })

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
			Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
				"expected manual mode to be configured")
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})
			appendPolicyAndVersionResponses(t, "4.22.0")
			isolatePolicyFiles()

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).NotTo(HaveOccurred(), "expected manual role commands to be generated")
		})

		It("returns manual policy file generation errors", func() {
			args.classic = true
			args.version = "4.22"
			Cmd.Flag("classic").Changed = true
			DeferCleanup(func() { Cmd.Flag("classic").Changed = false })

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
			Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
				"expected manual mode to be configured")
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})
			appendPolicyAndVersionResponses(t, "4.22.0")

			workingDir, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred(), "expected the current working directory to be available")
			policyDir := GinkgoT().TempDir()
			for _, name := range []string{
				"sts_installer_trust_policy.json",
				"sts_instance_controlplane_trust_policy.json",
				"sts_instance_worker_trust_policy.json",
				"sts_support_trust_policy.json",
			} {
				Expect(os.Mkdir(filepath.Join(policyDir, name), 0750)).To(Succeed(),
					"expected a directory to block policy file creation")
			}
			Expect(os.Chdir(policyDir)).To(Succeed(),
				"expected policy files to be isolated in a temporary directory")
			DeferCleanup(func() {
				Expect(os.Chdir(workingDir)).To(Succeed(),
					"expected the original working directory to be restored")
			})

			err = runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("error generating the policy files")),
				"expected policy file generation errors to be returned")
		})

		It("returns manual command generation errors", func() {
			args.hostedCP = true
			args.version = "5.0"
			Cmd.Flag("hosted-cp").Changed = true
			DeferCleanup(func() { Cmd.Flag("hosted-cp").Changed = false })

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
			Expect(Cmd.Flags().Set("mode", "manual")).To(Succeed(),
				"expected manual mode to be configured")
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})
			appendPolicyAndVersionResponses(t, "5.0.0")
			isolatePolicyFiles()

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred(), "expected manual command generation errors to be returned")
		})

		It("rejects an empty creation mode", func() {
			args.classic = true
			args.version = "4.22"
			Cmd.Flag("classic").Changed = true
			Cmd.Flag("mode").Changed = true
			DeferCleanup(func() {
				Cmd.Flag("classic").Changed = false
				Cmd.Flag("mode").Changed = false
			})

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)
			appendPolicyAndVersionResponses(t, "4.22.0")

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(MatchError(ContainSubstring("invalid mode")),
				"expected an empty account-role creation mode to be rejected")
		})

		It("returns error when GetPolicyVersion fails", func() {
			args.classic = true
			args.prefix = "test-prefix"

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed())
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})

			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK,
					`{"kind":"AWSSTSPolicyList","page":1,"size":0,"total":0,"items":[]}`),
				RespondWithJSON(http.StatusInternalServerError,
					`{"kind":"Error","id":"500","reason":"versions unavailable"}`),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting version"))
		})

		It("returns error when retrieving account role policies fails", func() {
			args.classic = true

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed())
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})

			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusInternalServerError,
				`{"kind":"Error","id":"500","reason":"policies unavailable"}`))

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred(), "expected policy retrieval failure to be returned")
			Expect(err.Error()).To(ContainSubstring("failed to retrieve account role policies"))
		})

		It("returns error when prefix exceeds 32 characters", func() {
			args.classic = true
			args.prefix = strings.Repeat("a", 33) // invalid length

			mockAWS := t.RosaRuntime.AWSClient.(*awsClient.MockClient)
			mockAWS.EXPECT().ValidateCredentials().Return(true, nil)

			Expect(Cmd.Flags().Set("mode", "auto")).To(Succeed())
			DeferCleanup(func() {
				interactive.SetModeKey("")
				Cmd.Flag("mode").Changed = false
			})

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
				RespondWithJSON(http.StatusOK,
					`{"kind":"AWSSTSPolicyList","page":1,"size":0,"total":0,"items":[]}`),
			)

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("the '-HCP' suffix is reserved for hosted CP managed policies"))
		})
	})
})

func appendPolicyAndVersionResponses(t *test.TestingRuntime, rawVersion string) {
	version, err := cmv1.NewVersion().
		ID("openshift-v" + rawVersion).
		RawID(rawVersion).
		Enabled(true).
		ROSAEnabled(true).
		ChannelGroup("stable").
		Build()
	Expect(err).NotTo(HaveOccurred(), "expected the version fixture to build")
	t.ApiServer.AppendHandlers(
		RespondWithJSON(http.StatusOK,
			`{"kind":"AWSSTSPolicyList","page":1,"size":0,"total":0,"items":[]}`),
		RespondWithJSON(http.StatusOK, test.FormatVersionList([]*cmv1.Version{version})),
	)
}

func isolatePolicyFiles() {
	workingDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred(), "expected the current working directory to be available")
	Expect(os.Chdir(GinkgoT().TempDir())).To(Succeed(),
		"expected policy files to be isolated in a temporary directory")
	DeferCleanup(func() {
		Expect(os.Chdir(workingDir)).To(Succeed(),
			"expected the original working directory to be restored")
	})
}
