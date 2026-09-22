package operatorroles

import (
	"fmt"
	"net/http"

	"go.uber.org/mock/gomock"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

const ebsCredRequestResponse = `{
	"kind": "STSCredentialRequestList",
	"page": 1,
	"size": 1,
	"total": 1,
	"items": [{
		"name": "ebs_csi_driver_operator",
		"operator": {
			"name": "ebs-cloud-credentials",
			"namespace": "openshift-cluster-csi-drivers"
		}
	}]
}`

var _ = Describe("create operator-roles by cluster key", func() {
	var (
		t       *test.TestingRuntime
		ctrl    *gomock.Controller
		mockAws *aws.MockClient
	)

	BeforeEach(func() {
		t = test.NewTestRuntime()
		ctrl = gomock.NewController(GinkgoT())
		mockAws = aws.NewMockClient(ctrl)
		t.RosaRuntime.AWSClient = mockAws
		interactive.SetEnabled(false)

		args.prefix = ""
		args.hostedCp = false
		args.installerRoleArn = ""
		args.permissionsBoundary = ""
		args.forcePolicyCreation = false
		args.oidcConfigId = ""
		args.sharedVpcRoleArn = ""
		args.channelGroup = ocm.DefaultChannelGroup
		args.vpcEndpointRoleArn = ""
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("validateOperatorRoles", func() {
		It("returns error when cluster has no operator IAM roles", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role"),
				))
			})
			missing, err := validateOperatorRoles(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("No Operator IAM roles found"))
			Expect(missing).To(BeEmpty())
		})

		It("returns empty list when all roles exist", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/test-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
							cmv1.NewOperatorIAMRole().
								Name("cloud-credentials").
								Namespace("openshift-cloud-network-config-controller").
								RoleARN("arn:aws:iam::123456789012:role/test-openshift-cloud-network-config-controller-cloud-credentials"),
						),
				))
			})

			mockAws.EXPECT().CheckRoleExists(
				"test-openshift-cluster-csi-drivers-ebs-cloud-credentials",
			).Return(true, "", nil)
			mockAws.EXPECT().CheckRoleExists(
				"test-openshift-cloud-network-config-controller-cloud-credentials",
			).Return(true, "", nil)

			missing, err := validateOperatorRoles(t.RosaRuntime, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(missing).To(BeEmpty())
		})

		It("returns missing roles when some do not exist", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/test-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
							cmv1.NewOperatorIAMRole().
								Name("cloud-credentials").
								Namespace("openshift-cloud-network-config-controller").
								RoleARN("arn:aws:iam::123456789012:role/test-openshift-cloud-network-config-controller-cloud-credentials"),
						),
				))
			})

			mockAws.EXPECT().CheckRoleExists(
				"test-openshift-cluster-csi-drivers-ebs-cloud-credentials",
			).Return(true, "", nil)
			mockAws.EXPECT().CheckRoleExists(
				"test-openshift-cloud-network-config-controller-cloud-credentials",
			).Return(false, "", nil)

			missing, err := validateOperatorRoles(t.RosaRuntime, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(missing).To(HaveLen(1))
			Expect(missing[0]).To(Equal(
				"test-openshift-cloud-network-config-controller-cloud-credentials"))
		})

		It("returns error when CheckRoleExists fails", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("arn:aws:iam::123456789012:role/test-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
						),
				))
			})

			mockAws.EXPECT().CheckRoleExists(gomock.Any()).
				Return(false, "", fmt.Errorf("AccessDenied: not authorized"))

			missing, err := validateOperatorRoles(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AccessDenied"))
			Expect(missing).To(BeEmpty())
		})

		It("returns error for operator IAM role with invalid ARN", func() {
			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role").
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN("not-a-valid-arn"),
						),
				))
			})

			missing, err := validateOperatorRoles(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(missing).To(BeEmpty())
		})
	})

	Describe("validateOperatorRolesMatchOidcProvider", func() {
		It("returns error when GetRoleByARN fails", func() {
			operatorRoleARN := "arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"
			installerARN := "arn:aws:iam::123456789012:role/test-prefix-Installer-Role"

			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN(installerARN).
						OidcConfig(cmv1.NewOidcConfig().
							ID("test-oidc-id").
							Reusable(true).
							IssuerUrl("https://oidc.example.com")).
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN(operatorRoleARN),
						),
				))
				c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
			})

			mockAws.EXPECT().GetRoleByARN(operatorRoleARN).
				Return(iamtypes.Role{}, fmt.Errorf("role not found"))

			err := validateOperatorRolesMatchOidcProvider(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("role not found"))
		})

		It("returns error when operator role path does not match installer role path", func() {
			installerARN := "arn:aws:iam::123456789012:role/custom-path/test-prefix-Installer-Role"
			operatorARN := "arn:aws:iam::123456789012:role/other-path/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"

			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN(installerARN).
						OidcConfig(cmv1.NewOidcConfig().
							ID("test-oidc-id").
							Reusable(true).
							IssuerUrl("https://oidc.example.com")).
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN(operatorARN),
						),
				))
				c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
			})

			mockAws.EXPECT().GetRoleByARN(operatorARN).
				Return(iamtypes.Role{
					Arn:                      awssdk.String(operatorARN),
					AssumeRolePolicyDocument: awssdk.String(`{}`),
				}, nil)

			err := validateOperatorRolesMatchOidcProvider(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not match the path from Installer Role"))
		})

		It("returns error when HasManagedPolicies fails", func() {
			operatorRoleARN := "arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"
			installerARN := "arn:aws:iam::123456789012:role/test-prefix-Installer-Role"
			issuerUrl := "https://oidc.example.com"

			cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.State(cmv1.ClusterStateReady)
				c.AWS(cmv1.NewAWS().STS(
					cmv1.NewSTS().
						RoleARN(installerARN).
						ManagedPolicies(true).
						OidcConfig(cmv1.NewOidcConfig().
							ID("test-oidc-id").
							Reusable(true).
							IssuerUrl(issuerUrl)).
						OperatorIAMRoles(
							cmv1.NewOperatorIAMRole().
								Name("ebs-cloud-credentials").
								Namespace("openshift-cluster-csi-drivers").
								RoleARN(operatorRoleARN),
						),
				))
				c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
			})

			assumePolicyDoc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow",` +
				`"Principal":{"Federated":"arn:aws:iam::123456789012:oidc-provider/oidc.example.com"},` +
				`"Action":"sts:AssumeRoleWithWebIdentity"}]}`

			mockAws.EXPECT().GetRoleByARN(operatorRoleARN).
				Return(iamtypes.Role{
					Arn:                      awssdk.String(operatorRoleARN),
					AssumeRolePolicyDocument: awssdk.String(assumePolicyDoc),
				}, nil)
			mockAws.EXPECT().HasManagedPolicies(operatorRoleARN).
				Return(false, fmt.Errorf("HasManagedPolicies failed"))

			err := validateOperatorRolesMatchOidcProvider(t.RosaRuntime, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("HasManagedPolicies failed"))
		})
	})

	Describe("handleOperatorRoleCreationByClusterKey", func() {
		BeforeEach(func() {
			t.RosaRuntime.Creator = &aws.Creator{
				ARN:       "arn:aws:iam::123456789012:user/test-user",
				AccountID: "123456789012",
				Partition: "aws",
			}
		})

		Context("auto mode", func() {
			It("returns nil when all operator roles exist and OIDC config is not reusable", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.State(cmv1.ClusterStateReady)
					c.AWS(cmv1.NewAWS().STS(
						cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-prefix-Installer-Role").
							OperatorRolePrefix("test-prefix").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								Reusable(false)).
							OperatorIAMRoles(
								cmv1.NewOperatorIAMRole().
									Name("ebs-cloud-credentials").
									Namespace("openshift-cluster-csi-drivers").
									RoleARN("arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
							),
					))
					c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
					c.Hypershift(cmv1.NewHypershift().Enabled(false))
				})
				t.SetCluster("test-cluster", cluster)

				mockAws.EXPECT().CheckRoleExists(gomock.Any()).
					Return(true, "", nil)

				// GetCredRequests
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
					`{"kind":"STSCredentialRequestList","page":1,"size":0,"total":0,"items":[]}`))

				err := handleOperatorRoleCreationByClusterKey(
					t.RosaRuntime, "production", "", interactive.ModeAuto,
					map[string]*cmv1.AWSSTSPolicy{}, "4.14.0", false,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			It("bypasses no-op and calls ForceEnsurePolicy when forcePolicyCreation is true", func() {
				args.forcePolicyCreation = true

				operatorPolicy, err := cmv1.NewAWSSTSPolicy().
					ID("operator_iam_role_policy").
					Details(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Federated":"arn:aws:iam::123456789012:oidc-provider/oidc.example.com"},"Action":"sts:AssumeRoleWithWebIdentity"}]}`).
					Build()
				Expect(err).NotTo(HaveOccurred())
				ebsOperatorPolicy, err := cmv1.NewAWSSTSPolicy().
					ID("openshift_ebs_csi_driver_operator_policy").
					Details(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}`).
					Build()
				Expect(err).NotTo(HaveOccurred())

				roleName := "test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"
				policyARN := "arn:aws:iam::123456789012:policy/" + roleName
				roleARN := "arn:aws:iam::123456789012:role/" + roleName

				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.State(cmv1.ClusterStateReady)
					c.AWS(cmv1.NewAWS().STS(
						cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-prefix-Installer-Role").
							OperatorRolePrefix("test-prefix").
							ManagedPolicies(false).
							OIDCEndpointURL("https://oidc.example.com").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								Reusable(false)).
							OperatorIAMRoles(
								cmv1.NewOperatorIAMRole().
									Name("ebs-cloud-credentials").
									Namespace("openshift-cluster-csi-drivers").
									RoleARN(roleARN),
							),
					))
					c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
					c.Hypershift(cmv1.NewHypershift().Enabled(false))
				})
				t.SetCluster("test-cluster", cluster)

				mockAws.EXPECT().CheckRoleExists(roleName).
					Return(true, "", nil)
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ebsCredRequestResponse))
				mockAws.EXPECT().GetAccountRoleVersion("test-prefix-Installer-Role").
					Return("4.14", nil)
				mockAws.EXPECT().ForceEnsurePolicy(gomock.Any(), gomock.Any(), "4.14.0", gomock.Any(), "").
					Return(policyARN, nil)
				mockAws.EXPECT().EnsureRole(gomock.Any(), roleName, gomock.Any(), "", "4.14", gomock.Any(), "", false).
					Return(roleARN, nil)
				mockAws.EXPECT().AttachRolePolicy(gomock.Any(), roleName, policyARN).
					Return(nil)

				err = handleOperatorRoleCreationByClusterKey(
					t.RosaRuntime, "production", "", interactive.ModeAuto,
					map[string]*cmv1.AWSSTSPolicy{
						"operator_iam_role_policy":                 operatorPolicy,
						"openshift_ebs_csi_driver_operator_policy": ebsOperatorPolicy,
					}, "4.14.0", false,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns error when OIDC provider validation fails for reusable config", func() {
				operatorRoleARN := "arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.State(cmv1.ClusterStateReady)
					c.AWS(cmv1.NewAWS().STS(
						cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-prefix-Installer-Role").
							OperatorRolePrefix("test-prefix").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								Reusable(true).
								IssuerUrl("https://oidc.example.com")).
							OperatorIAMRoles(
								cmv1.NewOperatorIAMRole().
									Name("ebs-cloud-credentials").
									Namespace("openshift-cluster-csi-drivers").
									RoleARN(operatorRoleARN),
							),
					))
					c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
					c.Hypershift(cmv1.NewHypershift().Enabled(false))
				})
				t.SetCluster("test-cluster", cluster)

				mockAws.EXPECT().CheckRoleExists(gomock.Any()).
					Return(true, "", nil)

				// GetCredRequests
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
					`{"kind":"STSCredentialRequestList","page":1,"size":0,"total":0,"items":[]}`))

				mockAws.EXPECT().GetRoleByARN(operatorRoleARN).
					Return(iamtypes.Role{}, fmt.Errorf("role not found in AWS"))

				err := handleOperatorRoleCreationByClusterKey(
					t.RosaRuntime, "production", "", interactive.ModeAuto,
					map[string]*cmv1.AWSSTSPolicy{}, "4.14.0", false,
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("role not found in AWS"))
			})

			It("returns nil when reusable OIDC config validates successfully", func() {
				operatorRoleARN := "arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"
				installerARN := "arn:aws:iam::123456789012:role/test-prefix-Installer-Role"
				issuerUrl := "https://oidc.example.com"
				assumePolicyDoc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow",` +
					`"Principal":{"Federated":"arn:aws:iam::123456789012:oidc-provider/oidc.example.com"},` +
					`"Action":"sts:AssumeRoleWithWebIdentity"}]}`

				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.State(cmv1.ClusterStateReady)
					c.AWS(cmv1.NewAWS().STS(
						cmv1.NewSTS().
							RoleARN(installerARN).
							OperatorRolePrefix("test-prefix").
							ManagedPolicies(true).
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								Reusable(true).
								IssuerUrl(issuerUrl)).
							OperatorIAMRoles(
								cmv1.NewOperatorIAMRole().
									Name("ebs-cloud-credentials").
									Namespace("openshift-cluster-csi-drivers").
									RoleARN(operatorRoleARN),
							),
					))
					c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
					c.Hypershift(cmv1.NewHypershift().Enabled(false))
				})
				t.SetCluster("test-cluster", cluster)

				mockAws.EXPECT().CheckRoleExists(gomock.Any()).
					Return(true, "", nil)
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
					`{"kind":"STSCredentialRequestList","page":1,"size":0,"total":0,"items":[]}`))
				mockAws.EXPECT().GetRoleByARN(operatorRoleARN).
					Return(iamtypes.Role{
						Arn:                      awssdk.String(operatorRoleARN),
						AssumeRolePolicyDocument: awssdk.String(assumePolicyDoc),
					}, nil)
				mockAws.EXPECT().HasManagedPolicies(operatorRoleARN).
					Return(true, nil)

				err := handleOperatorRoleCreationByClusterKey(
					t.RosaRuntime, "production", "", interactive.ModeAuto,
					map[string]*cmv1.AWSSTSPolicy{}, "4.14.0", false,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			It("identifies HCP cluster and uses hypershift credential requests", func() {
				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.State(cmv1.ClusterStateReady)
					c.AWS(cmv1.NewAWS().STS(
						cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-prefix-HCP-ROSA-Installer-Role").
							OperatorRolePrefix("test-prefix").
							ManagedPolicies(true).
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								Reusable(false)).
							OperatorIAMRoles(
								cmv1.NewOperatorIAMRole().
									Name("ebs-cloud-credentials").
									Namespace("openshift-cluster-csi-drivers").
									RoleARN("arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
							),
					))
					c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
					c.Hypershift(cmv1.NewHypershift().Enabled(true))
				})
				t.SetCluster("test-cluster", cluster)

				mockAws.EXPECT().CheckRoleExists(gomock.Any()).
					Return(true, "", nil)

				// GetCredRequests for HCP (is_hypershift=true)
				t.ApiServer.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet,
						"/api/clusters_mgmt/v1/aws_inquiries/sts_credential_requests"),
					ghttp.VerifyFormKV("is_hypershift", "true"),
					RespondWithJSON(http.StatusOK,
						`{"kind":"STSCredentialRequestList","page":1,"size":0,"total":0,"items":[]}`),
				))

				err := handleOperatorRoleCreationByClusterKey(
					t.RosaRuntime, "production", "", interactive.ModeAuto,
					map[string]*cmv1.AWSSTSPolicy{}, "4.14.0", false,
				)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("manual mode", func() {
			It("generates IAM commands from credential requests", func() {
				operatorPolicy, err := cmv1.NewAWSSTSPolicy().
					ID("operator_iam_role_policy").
					Details(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Federated":"arn:aws:iam::123456789012:oidc-provider/oidc.example.com"},"Action":"sts:AssumeRoleWithWebIdentity"}]}`).
					Build()
				Expect(err).NotTo(HaveOccurred())

				cluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.State(cmv1.ClusterStateReady)
					c.AWS(cmv1.NewAWS().STS(
						cmv1.NewSTS().
							RoleARN("arn:aws:iam::123456789012:role/test-prefix-Installer-Role").
							OperatorRolePrefix("test-prefix").
							OIDCEndpointURL("https://oidc.example.com").
							OidcConfig(cmv1.NewOidcConfig().
								ID("test-oidc-id").
								Reusable(false)).
							OperatorIAMRoles(
								cmv1.NewOperatorIAMRole().
									Name("ebs-cloud-credentials").
									Namespace("openshift-cluster-csi-drivers").
									RoleARN("arn:aws:iam::123456789012:role/test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"),
							),
					))
					c.Version(cmv1.NewVersion().ID("openshift-4.14.0").RawID("openshift-4.14.0"))
					c.Hypershift(cmv1.NewHypershift().Enabled(false))
				})
				t.SetCluster("test-cluster", cluster)

				mockAws.EXPECT().CheckRoleExists(gomock.Any()).
					Return(false, "", nil)
				mockAws.EXPECT().IsPolicyExists(gomock.Any()).
					Return(nil, fmt.Errorf("policy not found"))

				// GetCredRequests
				t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ebsCredRequestResponse))

				stdout, stderr, err := test.RunWithOutputCapture(
					func(_ *rosa.Runtime, _ *cobra.Command) error {
						return handleOperatorRoleCreationByClusterKey(
							t.RosaRuntime, "production", "", interactive.ModeManual,
							map[string]*cmv1.AWSSTSPolicy{
								"operator_iam_role_policy": operatorPolicy,
							}, "4.14.0", false,
						)
					}, t.RosaRuntime, &cobra.Command{})
				Expect(err).ToNot(HaveOccurred())
				Expect(stderr).To(BeEmpty())
				Expect(stdout).To(ContainSubstring("aws iam create-policy"))
				Expect(stdout).To(ContainSubstring("aws iam create-role"))
				Expect(stdout).To(ContainSubstring("aws iam attach-role-policy"))
				Expect(stdout).To(ContainSubstring("test-prefix-openshift-cluster-csi-drivers-ebs-cloud-credentials"))
			})
		})
	})

	Describe("computePolicyARN", func() {
		It("returns correct ARN with default prefix when prefix is empty", func() {
			creator := aws.Creator{
				Partition: "aws",
				AccountID: "123456789012",
			}
			result := computePolicyARN(creator, "", "ns1", "op1", "")
			Expect(result).To(Equal("arn:aws:iam::123456789012:policy/ManagedOpenShift-ns1-op1"))
		})

		It("returns correct ARN with custom prefix", func() {
			creator := aws.Creator{
				Partition: "aws",
				AccountID: "123456789012",
			}
			result := computePolicyARN(creator, "my-prefix", "openshift-cluster-csi-drivers",
				"ebs-cloud-credentials", "")
			Expect(result).To(HavePrefix("arn:aws:iam::123456789012:policy/my-prefix-"))
		})

		It("returns correct ARN with custom path", func() {
			creator := aws.Creator{
				Partition: "aws",
				AccountID: "123456789012",
			}
			result := computePolicyARN(creator, "test", "ns", "name", "/custom-path/")
			Expect(result).To(Equal("arn:aws:iam::123456789012:policy/custom-path/test-ns-name"))
		})

		It("returns correct ARN for govcloud partition", func() {
			creator := aws.Creator{
				Partition: "aws-us-gov",
				AccountID: "123456789012",
			}
			result := computePolicyARN(creator, "test", "ns", "name", "")
			Expect(result).To(HavePrefix("arn:aws-us-gov:iam::123456789012:policy/"))
		})
	})
})
