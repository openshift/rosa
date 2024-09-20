package e2e

import (
	"fmt"
	"path"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	utilConfig "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/occli"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Cluster preparation", labels.Feature.Cluster, func() {
	It("by profile",
		labels.Runtime.Day1,
		labels.Critical,
		func() {
			client := rosacli.NewClient()
			profile := profilehandler.LoadProfileYamlFileByENV()
			cluster, err := profilehandler.CreateClusterByProfile(profile, client, config.Test.GlobalENV.WaitSetupClusterReady)
			Expect(err).ToNot(HaveOccurred())
			log.Logger.Infof("Cluster prepared successfully with id %s", cluster.ID)

			if profile.ClusterConfig.HCP && profile.ClusterConfig.NetworkType == "other" {
				clusterID = cluster.ID
				profilehandler.WaitForClusterReady(client, clusterID, config.Test.GlobalENV.ClusterWaitingTime)

				clusterService := client.Cluster
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetails, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())
				if clusterDetails.ExternalAuthentication == "Enabled" {
					//create break-glass-credential to get kubeconfig
					_, err := client.BreakGlassCredential.CreateBreakGlassCredential(clusterID)
					Expect(err).To(BeNil())
					breakGlassCredList, err := client.BreakGlassCredential.ListBreakGlassCredentialsAndReflect(clusterID)
					Expect(err).To(BeNil())
					testDir := config.Test.OutputDir
					kubeconfigFile := path.Join(testDir, fmt.Sprintf("%s.kubeconfig", clusterID))

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
					ocClient := occli.NewOCClient(kubeconfigFile)
					err = utilConfig.DeployCilium(ocClient, podCIDR, hostPrefix, testDir, kubeconfigFile)
					Expect(err).ToNot(HaveOccurred())
					log.Logger.Infof("Deploy cilium for HCP cluster: %s successfully ", cluster.ID)
				} else {
					utilConfig.GetKubeconfigDummyFunc()
				}
			}
		})

	It("to wait for cluster ready",
		labels.Runtime.Day1Readiness,
		func() {
			clusterDetail, err := profilehandler.ParserClusterDetail()
			Expect(err).ToNot(HaveOccurred())
			client := rosacli.NewClient()
			profilehandler.WaitForClusterReady(client, clusterDetail.ClusterID, config.Test.GlobalENV.ClusterWaitingTime)
		})
})
