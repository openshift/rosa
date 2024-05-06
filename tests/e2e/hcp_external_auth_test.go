package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
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
				By("Retrieve help for create/list/describe/delete break-glass-credential")
				_, err := rosaClient.BreakGlassCredential.RetrieveHelpForCreate()
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.BreakGlassCredential.RetrieveHelpForDescribe()
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.BreakGlassCredential.RetrieveHelpForList()
				Expect(err).ToNot(HaveOccurred())
				_, err = rosaClient.BreakGlassCredential.RetrieveHelpForDelete()
				Expect(err).ToNot(HaveOccurred())

				By("Create a break-glass-credential to the cluster")
				userName := common.GenerateRandomName("bgc-username", 2)
				createTime := time.Now().UTC()
				expiredTime := createTime.Add(2 * time.Hour).Format("Jan _2 2006 15:04 MST")

				resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, userName, "--expiration", "2h")
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).To(ContainSubstring("INFO: Successfully created a break glass credential for cluster '%s'", clusterID))

				By("List the break-glass-credentials of the cluster")
				breakGlassCredList, err := rosaClient.BreakGlassCredential.ListBreakGlassCredentialsAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				isPresent, breakGlassCredential := breakGlassCredList.IsPresent(userName)
				Expect(isPresent).To(BeTrue())
				Eventually(rosaClient.BreakGlassCredential.WaitForBreakGlassCredentialToStatus(clusterID, "issued", userName), time.Minute*2, time.Second*10).Should(BeTrue())

				By("Retrieve the break-glass-credential id")
				breakGlassCredID := breakGlassCredential.ID

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

				By("Delete the break-glass-credential")
				resp, err = rosaClient.BreakGlassCredential.DeleteBreakGlassCredential(clusterID)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				Expect(textData).To(ContainSubstring("INFO: Successfully requested revocation for all break glass credentials from cluster '%s'", clusterID))

				By("Check the break-glass-credential status is revoked")
				Eventually(rosaClient.BreakGlassCredential.WaitForBreakGlassCredentialToStatus(clusterID, "revoked", userName), time.Minute*4, time.Second*10).Should(BeTrue())
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

		It("Validation for HCP cluster break_glass_credentials create/list/describe/delete can work well via ROSA client - [id:73018]",
			labels.Medium,
			func() {
				By("Create/list/revoke break-glass-credential to non-HCP cluster")
				hosted, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				if !hosted {
					By("Create a break-glass-credential to the cluster")
					userName := common.GenerateRandomName("bgc-user-classic", 2)

					resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, userName, "--expiration", "2h")
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

						resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, userName, "--expiration", "2h")
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

						resp, err := rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, userName, "--expiration", "2h")
						Expect(err).To(HaveOccurred())
						textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("ERR: failed to create a break glass credential for cluster '%s': The username '%s' must respect the regexp '^[a-zA-Z0-9-.]*$'", clusterID, userName))

						By("Create break-glass-credential with invalid --expiration")
						userName = common.GenerateRandomName("bgc-user-invalid-exp", 2)
						expirationTime := "2may"

						resp, err = rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, userName, "--expiration", expirationTime)
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring(`invalid argument "%s" for "--expiration" flag`, expirationTime))

						By("Create break-glass-credential with invalid expiration")
						userName = common.GenerateRandomName("bgc-user-exp-1s", 2)

						resp, err = rosaClient.BreakGlassCredential.CreateBreakGlassCredential(clusterID, userName, "--expiration", "1s")
						Expect(err).To(HaveOccurred())
						textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
						Expect(textData).To(ContainSubstring("ERR: failed to create a break glass credential for cluster '%s': Expiration needs to be at least 10 minutes from now", clusterID))
					}
				}
			})
	})
})
