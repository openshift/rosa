package e2e

import (
	"path"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	utilConfig "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/occli"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("Cluster preparation", labels.Feature.Cluster, func() {
	It("by profile",
		labels.Runtime.Day1,
		labels.Critical,
		func() {
			client := rosacli.NewClient()
			profile := handler.LoadProfileYamlFileByENV()
			clusterHandler, err := handler.NewClusterHandler(client, profile)
			Expect(err).ToNot(HaveOccurred())
			err = clusterHandler.CreateCluster(config.Test.GlobalENV.WaitSetupClusterReady)
			Expect(err).ToNot(HaveOccurred())
			clusterID := clusterHandler.GetClusterDetail().ClusterID
			log.Logger.Infof("Cluster prepared successfully with id %s", clusterID)

		})

	It("to wait for cluster ready",
		labels.Runtime.Day1Readiness,
		func() {
			profile := handler.LoadProfileYamlFileByENV()
			client := rosacli.NewClient()
			clusterHandler, err := handler.NewClusterHandlerFromFilesystem(client, profile)
			Expect(err).ToNot(HaveOccurred())
			clusterHandler.WaitForClusterReady(config.Test.GlobalENV.ClusterWaitingTime)

			// For HCP cluster with other network type,it is required to set one configure:cilium
			if profile.ClusterConfig.HCP && profile.ClusterConfig.NetworkType == "other" {
				clusterID := clusterHandler.GetClusterDetail().ClusterID
				clusterService := client.Cluster
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetails, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				if clusterDetails.ExternalAuthentication == "Enabled" {
					// it is not support to create htpasswd for cluster with xternal auth enabled
					// create break-glass-credential to get kubeconfig
					_, err := client.BreakGlassCredential.CreateBreakGlassCredential(clusterID)
					Expect(err).To(BeNil())
					breakGlassCredList, err := client.BreakGlassCredential.ListBreakGlassCredentialsAndReflect(clusterID)
					Expect(err).To(BeNil())
					kubeconfigFile := path.Join(config.Test.OutputDir, "kubeconfig")

					By("Get the issued credential")
					for _, i := range breakGlassCredList.BreakGlassCredentials {
						Eventually(
							client.BreakGlassCredential.WaitForBreakGlassCredentialToStatus(
								clusterID,
								"issued",
								i.Username),
							time.Minute*1,
						).Should(BeTrue())
						output, err := client.BreakGlassCredential.GetIssuedCredential(clusterID, i.ID)
						Expect(err).ToNot(HaveOccurred())
						Expect(output.String()).ToNot(BeEmpty())
						_, err = helper.CreateFileWithContent(kubeconfigFile, output.String())
						Expect(err).ToNot(HaveOccurred())
						break
					}
					hostPrefix, podCIDR := "", ""
					for _, networkLine := range clusterDetails.Network {
						if value, containsKey := networkLine["Host Prefix"]; containsKey {
							hostPrefix = value
							break
						}
						if value, containsKey := networkLine["Pod CIDR"]; containsKey {
							podCIDR = value
							break
						}
					}
					By("Deploy cilium configures")
					ocClient, err := occli.NewOCClient(kubeconfigFile)
					Expect(err).ToNot(HaveOccurred())
					err = utilConfig.DeployCilium(ocClient, podCIDR, hostPrefix,
						config.Test.OutputDir, kubeconfigFile)
					Expect(err).ToNot(HaveOccurred())
					log.Logger.Infof("Deploy cilium for HCP cluster: %s successfully ", clusterID)
				} else {
					By("Create IDP to get kubeconfig")
					idpType := "htpasswd"
					idpName := "myhtpasswdKubeconf"
					name, password := clusterHandler.GetResourcesHandler().PrepareAdminUser()
					usersValue := name + ":" + password
					_, err := client.IDP.CreateIDP(clusterID, idpName,
						"--type", idpType,
						"--users", usersValue,
						"-y")
					Expect(err).ToNot(HaveOccurred())

					_, err = client.User.GrantUser(clusterID, "dedicated-admins", name)
					Expect(err).ToNot(HaveOccurred())

					helper.CreateFileWithContent(config.Test.ClusterIDPAdminUsernamePassword, usersValue)
				}
			}

		})
})
