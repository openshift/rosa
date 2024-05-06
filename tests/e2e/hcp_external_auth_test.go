package e2e

import (
	"bytes"
	"strings"
	"time"

	"k8s.io/utils/strings/slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ciConfig "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Rosacli Testing", func() {

	var rosaClient *rosacli.Client
	var clusterService rosacli.ClusterService

	Describe("Hypershift external auth creation testing", func() {
		BeforeEach(func() {
			Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster

			By("Check cluster is hosted cluster and external auth config is enabled")
			hosted, err := clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			externalAuthProvider, err := clusterService.IsExternalAuthenticationEnabled(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if !hosted || !externalAuthProvider {
				Skip("This is only for external_auth_config enabled hypershift cluster")
			}
		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("create/list/describe/delete HCP cluster with break_glass_credentials via Rosa client can work well - [id:72899]",
			labels.High, labels.NonClassicCluster,
			func() {
				var resp bytes.Buffer
				var err error
				var userName string
				var breakGlassCredID string

				userNameList := []string{}

				reqBody := map[string][]string{
					"full":            []string{"--username", common.GenerateRandomName("userName1", 2), "--expiration", "15m"},
					"emptyUsername":   []string{"--expiration", "15m"},
					"emptyExpiration": []string{"--username", common.GenerateRandomName("userName2", 2)},
					"empty":           []string{},
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
					Expect(textData).To(ContainSubstring("INFO: Successfully created a break glass credential for cluster '%s'", clusterID))

					By("List the break-glass-credentials of the cluster")
					breakGlassCredList, err := rosaClient.BreakGlassCredential.ListBreakGlassCredentialsAndReflect(clusterID)
					Expect(err).ToNot(HaveOccurred())
					for _, breabreakGlassCred := range breakGlassCredList.BreakGlassCredentials {
						if !slices.Contains(userNameList, breabreakGlassCred.Username) && breabreakGlassCred.Status != "revoked" && breabreakGlassCred.Status != "awaiting_revocation" {
							userName = breabreakGlassCred.Username
							breakGlassCredID = breabreakGlassCred.ID
							userNameList = append(userNameList, userName)
							Eventually(rosaClient.BreakGlassCredential.WaitForBreakGlassCredentialToStatus(clusterID, "issued", userName), time.Minute*2, time.Second*10).Should(BeTrue())
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
				Expect(textData).To(ContainSubstring("INFO: Successfully requested revocation for all break glass credentials from cluster '%s'", clusterID))

				By("Check the break-glass-credential status is revoked")
				for _, userName = range userNameList {
					Eventually(rosaClient.BreakGlassCredential.WaitForBreakGlassCredentialToStatus(clusterID, "revoked", userName), time.Minute*4, time.Second*10).Should(BeTrue())
				}
			})
	})

	Describe("Hypershift external auth validation testing", func() {
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

		It("validation for HCP cluster break_glass_credentials create/list/describe/delete can work well via ROSA client - [id:73018]",
			labels.Medium,
			func() {
				By("Create/list/revoke break-glass-credential to non-HCP cluster")
				hosted, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				if !hosted {
					By("Create a break-glass-credential to the cluster")
					userName := common.GenerateRandomName("bgc-user-classic", 2)

					resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, "--username", userName, "--expiration", "2h")
					Expect(err).To(HaveOccurred())
					textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).To(ContainSubstring("ERR: external authentication provider is only supported for Hosted Control Planes"))

					By("List the break-glass-credentials of the cluster")
					resp, err = rosaClient.BreakGlassCredential.ListBreakGlassCredentials(clusterID)
					Expect(err).To(HaveOccurred())
					textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).To(ContainSubstring("ERR: external authentication provider is only supported for Hosted Control Planes"))

					By("Revoke the break-glass-credentials of the cluster")
					resp, err = rosaClient.BreakGlassCredential.DeleteBreakGlassCredential(clusterID)
					Expect(err).To(HaveOccurred())
					textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
					Expect(textData).To(ContainSubstring("ERR: external authentication provider is only supported for Hosted Control Planes"))
				}

				if hosted {
					By("Create/list/revoke break-glass-credential to external-auth-providers-enabled not enable")
					externalAuthProvider, err := clusterService.IsExternalAuthenticationEnabled(clusterID)
					Expect(err).ToNot(HaveOccurred())
					if !externalAuthProvider {
						By("Create a break-glass-credential to the cluster")
						userName := common.GenerateRandomName("bgc-user-non-external", 2)

						resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, "--username", userName, "--expiration", "2h")
						Expect(err).To(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("ERR: External authentication configuration is not enabled for cluster '%s'", clusterID))

						By("List the break-glass-credentials of the cluster")
						resp, err = rosaClient.BreakGlassCredential.ListBreakGlassCredentials(clusterID)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("ERR: External authentication configuration is not enabled for cluster '%s'", clusterID))

						By("Revoke the break-glass-credentials of the cluster")
						resp, err = rosaClient.BreakGlassCredential.DeleteBreakGlassCredential(clusterID)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("ERR: External authentication configuration is not enabled for cluster '%s'", clusterID))

					} else if externalAuthProvider {

						By("Create break-glass-credential with invalid --username")
						userName := common.GenerateRandomName("bgc-user_", 2)

						resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, "--username", userName, "--expiration", "2h")
						Expect(err).To(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("ERR: failed to create a break glass credential for cluster '%s': The username '%s' must respect the regexp '^[a-zA-Z0-9-.]*$'", clusterID, userName))

						By("Create break-glass-credential with invalid --expiration")
						userName = common.GenerateRandomName("bgc-user-invalid-exp", 2)
						expirationTime := "2may"

						resp, err = rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, "--username", userName, "--expiration", expirationTime)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring(`invalid argument "%s" for "--expiration" flag`, expirationTime))

						By("Create break-glass-credential with invalid expiration")
						userName = common.GenerateRandomName("bgc-user-exp-1s", 2)

						resp, err = rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, "--username", userName, "--expiration", "1s")
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("ERR: failed to create a break glass credential for cluster '%s': Expiration needs to be at least 10 minutes from now", clusterID))
					}
				}
			})

		It("validation should work when create/list/delete idp and user/admin to external_auth_config cluster via ROSA client - [id:71946]",
			labels.Medium, labels.NonClassicCluster,
			func() {

				By("Check if hcp cluster external_auth_providers is enabled")
				isExternalAuthEnabled, err := clusterService.IsExternalAuthenticationEnabled(clusterID)
				Expect(err).ToNot(HaveOccurred())

				if !isExternalAuthEnabled {
					Skip("This case is for HCP clusters with External Auth")
				}

				By("Create admin on --external-auth-providers-enabled cluster")
				output, err := rosaClient.User.CreateAdmin(clusterID)
				Expect(err).To(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: Creating the 'cluster-admin' user is not supported for clusters with external authentication configured."))

				By("Delete admin on --external-auth-providers-enabled cluster")
				output, err = rosaClient.User.DeleteAdmin(clusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: Deleting the 'cluster-admin' user is not supported for clusters with external authentication configured."))

				By("Create idp on --external-auth-providers-enabled cluster")
				idpName := common.GenerateRandomName("cluster-idp", 2)
				output, err = rosaClient.IDP.CreateIDP(clusterID, idpName, "--type", "htpasswd")
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: Adding IDP is not supported for clusters with external authentication configured."))

				By("Delete idp on --external-auth-providers-enabled cluster")
				output, err = rosaClient.IDP.DeleteIDP(clusterID, idpName)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: Deleting IDP is not supported for clusters with external authentication configured."))

				By("List user on --external-auth-providers-enabled cluster")
				_, output, err = rosaClient.User.ListUsers(clusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: Listing cluster users is not supported for clusters with external authentication configured."))

				By("List idps on --external-auth-providers-enabled cluster")
				_, output, err = rosaClient.IDP.ListIDP(clusterID)
				Expect(err).To(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("ERR: Listing identity providers is not supported for clusters with external authentication configured."))
			})

		It("validation should work when create cluster with external_auth_config via ROSA client - [id:73755]",
			labels.Medium, labels.Day1Validation,
			func() {
				isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).To(BeNil())
				isExternalAuthEnabled, err := clusterService.IsExternalAuthenticationEnabled(clusterID)
				Expect(err).ToNot(HaveOccurred())

				rosalCommand, err := config.RetrieveClusterCreationCommand(ciConfig.Test.CreateCommandFile)
				Expect(err).To(BeNil())

				if !isHostedCP {
					By("Create non-HCP cluster with --external-auth-providers-enabled")
					clusterName := common.GenerateRandomName("classic-71946", 2)
					operatorPrefix := common.GenerateRandomName("classic-oper", 2)
					replacingFlags := map[string]string{
						"-c":                     clusterName,
						"--cluster-name":         clusterName,
						"--domain-prefix":        clusterName,
						"--operator-role-prefix": operatorPrefix,
					}
					rosalCommand.ReplaceFlagValue(replacingFlags)
					rosalCommand.AddFlags("--dry-run", "--external-auth-providers-enabled", "-y")
					output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
					Expect(err).To(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("ERR: External authentication configuration is only supported for a Hosted Control Plane cluster."))
				} else {
					By("Create HCP cluster with --external-auth-providers-enabled and cluster version lower than 4.15")
					clusterName := common.GenerateRandomName("cluster-71946", 2)
					operatorPrefix := common.GenerateRandomName("cluster-oper", 2)

					cg := rosalCommand.GetFlagValue("--channel-group", true)
					if cg == "" {
						cg = rosacli.VersionChannelGroupStable
					}
					versionList, err := rosaClient.Version.ListAndReflectVersions(cg, isHostedCP)
					Expect(err).To(BeNil())
					Expect(versionList).ToNot(BeNil())
					previousVersionsList, err := versionList.FindNearestBackwardMinorVersion("4.14", 0, true)
					Expect(err).ToNot(HaveOccurred())
					foundVersion := previousVersionsList.Version
					replacingFlags := map[string]string{
						"-c":                     clusterName,
						"--cluster-name":         clusterName,
						"--domain-prefix":        clusterName,
						"--operator-role-prefix": operatorPrefix,
						"--version":              foundVersion,
					}
					rosalCommand.ReplaceFlagValue(replacingFlags)
					if !isExternalAuthEnabled {
						rosalCommand.AddFlags("--dry-run", "--external-auth-providers-enabled", "-y")
					} else {
						rosalCommand.AddFlags("--dry-run", "-y")
					}
					output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
					Expect(err).To(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("External authentication is only supported in version '4.15.0' or greater, current cluster version is '%s'", foundVersion))
				}
			})
	})
})
