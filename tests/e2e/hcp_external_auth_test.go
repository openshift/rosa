package e2e

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/utils/strings/slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
)

var _ = Describe("External auth provider", labels.Feature.ExternalAuthProvider, func() {

	var rosaClient *rosacli.Client
	var clusterService rosacli.ClusterService

	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()
		clusterService = rosaClient.Cluster
	})
	AfterEach(func() {
		By("Clean remaining resources")
		err := rosaClient.CleanResources(clusterID)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("creation testing", func() {
		BeforeEach(func() {
			By("Skip testing if the cluster is not a HCP cluster")
			hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if !hostedCluster {
				SkipNotHosted()
			}

			By("Check if hcp cluster external_auth_providers is enabled")
			externalAuthProvider, err := clusterService.IsExternalAuthenticationEnabled(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if !externalAuthProvider {
				SkipTestOnFeature("external auth provider")
			}
		})

		It("to create/list/describe/delete HCP cluster with break_glass_credentials can work well - [id:72899]",
			labels.High, labels.Runtime.Day2,
			func() {
				var resp bytes.Buffer
				var err error
				var userName string
				var breakGlassCredID string

				userNameList := []string{}

				reqBody := map[string][]string{
					"full":            {"--username", helper.GenerateRandomName("userName1", 2), "--expiration", "15m"},
					"emptyUsername":   {"--expiration", "15m"},
					"emptyExpiration": {"--username", helper.GenerateRandomName("userName2", 2)},
					"empty":           {},
				}

				By("Retrieve help for create/list/describe/delete break-glass-credential")
				_, err = rosaClient.BreakGlassCredential.RetrieveHelpForCreate()
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.BreakGlassCredential.RetrieveHelpForDescribe()
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.BreakGlassCredential.RetrieveHelpForList()
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.BreakGlassCredential.RetrieveHelpForDelete()
				Expect(err).ToNot(HaveOccurred())

				for key, value := range reqBody {
					By("Create a break-glass-credential to the cluster")
					resp, err = rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, value...)
					createTime := time.Now().UTC()
					expiredTime := createTime.Add(15 * time.Minute).Format("Jan _2 2006 15:04 MST")
					if key == "emptyExpiration" || key == "empty" {
						expiredTime = createTime.Add(24 * time.Hour).Format("Jan _2 2006 15:04 MST")
					}
					Expect(err).ToNot(HaveOccurred())
					textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).
						To(ContainSubstring(
							"INFO: Successfully created a break glass credential for cluster '%s'",
							clusterID))

					By("List the break-glass-credentials of the cluster")
					breakGlassCredList, err := rosaClient.BreakGlassCredential.ListBreakGlassCredentialsAndReflect(clusterID)
					Expect(err).ToNot(HaveOccurred())
					for _, breabreakGlassCred := range breakGlassCredList.BreakGlassCredentials {
						if !slices.Contains(userNameList, breabreakGlassCred.Username) &&
							breabreakGlassCred.Status != "revoked" &&
							breabreakGlassCred.Status != "awaiting_revocation" {

							userName = breabreakGlassCred.Username
							breakGlassCredID = breabreakGlassCred.ID
							userNameList = append(userNameList, userName)
							Eventually(
								rosaClient.BreakGlassCredential.WaitForBreakGlassCredentialToStatus(
									clusterID,
									"issued",
									userName),
								time.Minute*2,
								time.Second*10,
							).Should(BeTrue())
						}
					}

					By("Describe the break-glass-credential")
					output, err := rosaClient.BreakGlassCredential.DescribeBreakGlassCredentialsAndReflect(clusterID, breakGlassCredID)
					Expect(err).ToNot(HaveOccurred())
					// expiration timestamp changes from 4 to 5 seconds ahead, so have to remove second from time stamp
					expirationTime, err := time.Parse("Jan _2 2006 15:04:05 MST", output.ExpireAt)
					Expect(err).ToNot(HaveOccurred())
					Expect(output.ID).To(Equal(breakGlassCredID))
					Expect(expirationTime.Format("Jan _2 2006 15:04 MST")).To(Equal(expiredTime))
					Expect(output.Username).To(Equal(userName))
					Expect(output.Status).To(Equal("issued"))

					By("Get the issued credential")
					_, err = rosaClient.BreakGlassCredential.GetIssuedCredential(clusterID, breakGlassCredID)
					Expect(err).ToNot(HaveOccurred())
				}

				By("Delete the break-glass-credential")
				resp, err = rosaClient.BreakGlassCredential.DeleteBreakGlassCredential(clusterID)
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"INFO: Successfully requested revocation for all break glass credentials from cluster '%s'",
						clusterID))

				By("Check the break-glass-credential status is revoked")
				for _, userName = range userNameList {
					Eventually(
						rosaClient.BreakGlassCredential.WaitForBreakGlassCredentialToStatus(
							clusterID,
							"revoked",
							userName,
						),
						time.Minute*4,
						time.Second*10,
					).Should(BeTrue())
				}
			})

		It("create/list/describe/delete external_auth for a HCP cluster can work well via rosa client - [id:72536]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				var (
					consoleClientID      = "abc"
					consoleClientSecrect = "efgh"
					issuerURL            = "https://local.com"
					issuerAudience       = "abc"
					ca                   = "----BEGIN CERTIFICATE-----MIIDNTCCAh2gAwIBAgIUAegBu2L2aoOizuGxf/" +
						"fxBCU10oswDQYJKoZIhvcNAQELS3nCXMvI8q0E-----END CERTIFICATE-----"
					groupClaim               = "groups"
					userNameClaim            = "email"
					claimValidationRuleClaim = "claim1:rule1"
				)

				claimRule := strings.Split(claimValidationRuleClaim, ":")
				providerNameList := []string{}
				var providerName string

				caPath, err := helper.CreateTempFileWithContent(ca)
				defer os.Remove(caPath)
				Expect(err).ToNot(HaveOccurred())

				reqBody := map[string][]string{
					"simple": {
						"--name", helper.GenerateRandomName("provider1", 2),
						"--issuer-url", issuerURL,
						"--issuer-audiences", issuerAudience,
						"--claim-mapping-username-claim", userNameClaim,
						"--claim-mapping-groups-claim", groupClaim,
						"--claim-validation-rule", claimValidationRuleClaim,
					},
					"with_ca": {
						"--name", helper.GenerateRandomName("provider2", 2),
						"--issuer-url", issuerURL,
						"--issuer-audiences", issuerAudience,
						"--claim-mapping-username-claim", userNameClaim,
						"--claim-mapping-groups-claim", groupClaim,
						"--issuer-ca-file", caPath,
					},
					"with_client_parameters": {
						"--name", helper.GenerateRandomName("provider3", 2),
						"--issuer-url", issuerURL,
						"--issuer-audiences", issuerAudience,
						"--claim-mapping-username-claim", userNameClaim,
						"--claim-mapping-groups-claim", groupClaim,
						"--console-client-id", consoleClientID,
						"--console-client-secret", consoleClientSecrect,
					},
				}

				By("Check help message for create/list/describe/delete external_auth_provider")
				_, err = rosaClient.ExternalAuthProvider.CreateExternalAuthProvider(clusterID, "-h")
				Expect(err).ToNot(HaveOccurred())

				_, err = rosaClient.ExternalAuthProvider.RetrieveHelpForList()
				Expect(err).ToNot(HaveOccurred())

				_, err = rosaClient.ExternalAuthProvider.RetrieveHelpForDescribe()
				Expect(err).ToNot(HaveOccurred())

				_, err = rosaClient.ExternalAuthProvider.RetrieveHelpForDelete()
				Expect(err).ToNot(HaveOccurred())

				for key, value := range reqBody {
					By("Create external auth provider to the cluster")
					output, err := rosaClient.ExternalAuthProvider.CreateExternalAuthProvider(clusterID, value...)
					Expect(err).ToNot(HaveOccurred())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).
						To(ContainSubstring(
							"INFO: Successfully created an external authentication provider for cluster '%s'. "+
								"It can take a few minutes for the creation of an external authentication provider to become fully effective.",
							clusterID))

					By("List external auth providers of the cluster")
					externalAuthProviderList, err := rosaClient.ExternalAuthProvider.ListExternalAuthProviderAndReflect(clusterID)
					Expect(err).ToNot(HaveOccurred())

					for _, externalAuthProvider := range externalAuthProviderList.ExternalAuthProviders {
						if !slices.Contains(providerNameList, externalAuthProvider.Name) {
							providerName = externalAuthProvider.Name
							Expect(providerName).ToNot(BeEmpty())
							Expect(externalAuthProvider.IssuerUrl).To(Equal(issuerURL))
							providerNameList = append(providerNameList, providerName)
						}
					}

					By("Describe external auth provider of the cluster")
					externalAuthProviderDesc, err :=
						rosaClient.ExternalAuthProvider.DescribeExternalAuthProviderAndReflect(clusterID, providerName)
					Expect(err).ToNot(HaveOccurred())
					Expect(externalAuthProviderDesc.ID).To(Equal(providerName))
					Expect(externalAuthProviderDesc.ClusterID).To(Equal(clusterID))
					Expect(externalAuthProviderDesc.IssuerAudiences[0]).To(Equal(issuerAudience))
					Expect(externalAuthProviderDesc.IssuerUrl).To(Equal(issuerURL))
					Expect(externalAuthProviderDesc.ClaimMappingsGroup).To(Equal(groupClaim))
					Expect(externalAuthProviderDesc.ClaimMappingsUserName).To(Equal(userNameClaim))
					if key == "simple" {
						Expect(externalAuthProviderDesc.ClaimValidationRules[0]).To(Equal(fmt.Sprintf("Claim:%s", claimRule[0])))
						Expect(externalAuthProviderDesc.ClaimValidationRules[1]).To(Equal(fmt.Sprintf("Value:%s", claimRule[1])))
					}
					if key == "with_client_parameters" {
						Expect(externalAuthProviderDesc.ConsoleClientID).To(Equal(consoleClientID))
					}

					By("Delete external auth provider of the cluster")
					output, err = rosaClient.ExternalAuthProvider.DeleteExternalAuthProvider(clusterID, providerName)
					Expect(err).ToNot(HaveOccurred())
					textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
					Expect(textData).
						To(ContainSubstring(
							"INFO: Successfully deleted external authentication provider '%s' from cluster '%s'",
							providerName,
							clusterID))
				}
			})
	})

	Describe("validation testing", func() {
		It("to validate create/list/describe/delete break_glass_credentials can work well - [id:73018]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Create/list/revoke break-glass-credential to non-HCP cluster")
				hosted, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				if !hosted {
					By("Create a break-glass-credential to the cluster")
					userName := helper.GenerateRandomName("bgc-user-classic", 2)

					resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(
						clusterID,
						"--username", userName,
						"--expiration", "2h")
					Expect(err).To(HaveOccurred())
					textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).
						To(ContainSubstring(
							"ERR: external authentication provider is only supported for Hosted Control Planes"))

					By("List the break-glass-credentials of the cluster")
					resp, err = rosaClient.BreakGlassCredential.ListBreakGlassCredentials(clusterID)
					Expect(err).To(HaveOccurred())
					textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).
						To(ContainSubstring(
							"ERR: external authentication provider is only supported for Hosted Control Planes"))

					By("Revoke the break-glass-credentials of the cluster")
					resp, err = rosaClient.BreakGlassCredential.DeleteBreakGlassCredential(clusterID)
					Expect(err).To(HaveOccurred())
					textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).
						To(ContainSubstring(
							"ERR: external authentication provider is only supported for Hosted Control Planes"))
				}

				if hosted {
					By("Create/list/revoke break-glass-credential to external-auth-providers-enabled not enable")
					externalAuthProvider, err := clusterService.IsExternalAuthenticationEnabled(clusterID)
					Expect(err).ToNot(HaveOccurred())
					if !externalAuthProvider {
						By("Create a break-glass-credential to the cluster")
						userName := helper.GenerateRandomName("bgc-user-non-external", 2)

						resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(
							clusterID,
							"--username", userName,
							"--expiration", "2h")
						Expect(err).To(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(ContainSubstring(
								"ERR: External authentication configuration is not enabled for cluster '%s'",
								clusterID))

						By("List the break-glass-credentials of the cluster")
						resp, err = rosaClient.BreakGlassCredential.ListBreakGlassCredentials(clusterID)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(ContainSubstring(
								"ERR: External authentication configuration is not enabled for cluster '%s'",
								clusterID))

						By("Revoke the break-glass-credentials of the cluster")
						resp, err = rosaClient.BreakGlassCredential.DeleteBreakGlassCredential(clusterID)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(ContainSubstring(
								"ERR: External authentication configuration is not enabled for cluster '%s'",
								clusterID))

					} else if externalAuthProvider {

						By("Create break-glass-credential with invalid --username")
						userName := helper.GenerateRandomName("bgc-user_", 2)

						resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(
							clusterID,
							"--username", userName,
							"--expiration", "2h")
						Expect(err).To(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(ContainSubstring("The username '%s' must respect the regexp '^[a-zA-Z0-9-.]*$'",
								userName))

						By("Create break-glass-credential with invalid --expiration")
						userName = helper.GenerateRandomName("bgc-user-invalid-exp", 2)
						expirationTime := "2may"

						resp, err = rosaClient.BreakGlassCredential.CreateBreakGlassCredential(
							clusterID,
							"--username", userName,
							"--expiration", expirationTime)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(ContainSubstring(
								`invalid argument "%s" for "--expiration" flag`,
								expirationTime))

						By("Create break-glass-credential with invalid expiration")
						userName = helper.GenerateRandomName("bgc-user-exp-1s", 2)

						resp, err = rosaClient.BreakGlassCredential.CreateBreakGlassCredential(
							clusterID,
							"--username", userName,
							"--expiration", "1s")
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).
							To(ContainSubstring(
								"Expiration needs to be at least 10 minutes from now"))
					}
				}
			})

		It("to validate create/list/delete idp and user/admin to external_auth_config cluster can work well - [id:71946]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Skip testing if the cluster is not a HCP cluster")
				hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if !hostedCluster {
					SkipNotHosted()
				}

				By("Check if hcp cluster external_auth_providers is enabled")
				isExternalAuthEnabled, err := clusterService.IsExternalAuthenticationEnabled(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if !isExternalAuthEnabled {
					SkipTestOnFeature("external auth provider")
				}

				By("Create admin on --external-auth-providers-enabled cluster")
				output, err := rosaClient.User.CreateAdmin(clusterID)
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Creating the 'cluster-admin' user is not supported for clusters with external authentication configured."))

				By("Delete admin on --external-auth-providers-enabled cluster")
				output, err = rosaClient.User.DeleteAdmin(clusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Deleting the 'cluster-admin' user is not supported for clusters with external authentication configured."))

				By("Create idp on --external-auth-providers-enabled cluster")
				idpName := helper.GenerateRandomName("cluster-idp", 2)
				output, err = rosaClient.IDP.CreateIDP(clusterID, idpName, "--type", "htpasswd")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Adding IDP is not supported for clusters with external authentication configured."))

				By("Delete idp on --external-auth-providers-enabled cluster")
				output, err = rosaClient.IDP.DeleteIDP(clusterID, idpName)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Deleting IDP is not supported for clusters with external authentication configured."))

				By("List user on --external-auth-providers-enabled cluster")
				_, output, err = rosaClient.User.ListUsers(clusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Listing cluster users is not supported for clusters with external authentication configured."))

				By("List idps on --external-auth-providers-enabled cluster")
				_, output, err = rosaClient.IDP.ListIDP(clusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).
					To(ContainSubstring(
						"ERR: Listing identity providers is not supported for clusters with external authentication configured."))
			})

		It("to validate HCP cluster creation/list with external auth - [id:72602]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Create non HCP cluster with external_auths")
				hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if !hostedCluster {
					output, err := rosaClient.ExternalAuthProvider.CreateExternalAuthProvider(clusterID)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(
						ContainSubstring("ERR: external authentication provider is only supported for Hosted Control Planes"))

					output, err = rosaClient.ExternalAuthProvider.ListExternalAuthProvider(clusterID)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(
						ContainSubstring("ERR: external authentication provider is only supported for Hosted Control Planes"))
					return
				}

				By("Create external_provider  to HCP cluster that external_auth_config is not enable")
				isExternalAuthEnabled, err := clusterService.IsExternalAuthenticationEnabled(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if !isExternalAuthEnabled {
					output, err := rosaClient.ExternalAuthProvider.CreateExternalAuthProvider(clusterID,
						"--name", "test")
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(
						ContainSubstring("ERR: External authentication configuration is not enabled for cluster"))

					By("List external_provider to HCP cluster that external_auth_config is not enable")
					output, err = rosaClient.ExternalAuthProvider.ListExternalAuthProvider(clusterID)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).Should(
						ContainSubstring("ERR: External authentication configuration is not enabled for cluster"))
					return
				}
			})
	})
})
