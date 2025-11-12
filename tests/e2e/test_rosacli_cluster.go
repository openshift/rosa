package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	ciConfig "github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = ginkgo.Describe("Edit cluster",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			clusterID      string
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			profile        *handler.Profile
		)

		ginkgo.BeforeEach(func() {
			ginkgo.By("Get the cluster")
			clusterID = config.GetClusterID()
			gomega.Expect(clusterID).ToNot(gomega.Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster

			ginkgo.By("Load the profile")
			profile = handler.LoadProfileYamlFileByENV()
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Clean the cluster")
			rosaClient.CleanResources(clusterID)
		})
		ginkgo.It("can edit cluster channel group - [id:81399]",
			labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
			func() {
				const STABLE_CHANNEL = "stable"
				const CANDIDATE_CHANNEL = "candidate"
				ginkgo.By("Check help message contains channel-group flag")
				output, err := clusterService.EditCluster("", "-h")
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("--channel-group"))

				ginkgo.By("Get original version and channel group")
				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())
				originalVersion := CD.OpenshiftVersion
				originalChannelGroup := CD.ChannelGroup

				ginkgo.By("Check if there is version in updating channel group")
				var upgradingChannelGroup string
				existingAvailabelVersion := false
				if originalChannelGroup == STABLE_CHANNEL {
					upgradingChannelGroup = CANDIDATE_CHANNEL
				} else {
					upgradingChannelGroup = STABLE_CHANNEL
				}
				versionService := rosaClient.Version
				hostedCP, err := clusterService.IsHostedCPCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				versionList, err := versionService.ListAndReflectJsonVersions(upgradingChannelGroup, hostedCP)
				gomega.Expect(err).To(gomega.BeNil())
				for _, version := range versionList {
					if version.RAWID == originalVersion {
						existingAvailabelVersion = true
						break
					}
					continue
				}

				ginkgo.By("Edit cluster with channel group")
				out, err := clusterService.EditCluster(
					clusterID,
					"--channel-group", upgradingChannelGroup,
					"-y",
				)
				defer func() {
					ginkgo.By("Recover the original channel group")
					_, err = clusterService.EditCluster(
						clusterID,
						"--channel-group", originalChannelGroup,
						"-y",
					)
					gomega.Expect(err).To(gomega.BeNil())
				}()

				if existingAvailabelVersion {
					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(out.String()).To(gomega.ContainSubstring("Updated cluster"))
				} else {
					gomega.Expect(err).ToNot(gomega.BeNil())
					gomega.Expect(out.String()).To(gomega.ContainSubstring("is not available for the desired channel group"))
				}

				ginkgo.By("Edit cluster with the channel group which has no available version")
				out, err = clusterService.EditCluster(
					clusterID,
					"--channel-group", "fakecg",
					"-y",
				)
				gomega.Expect(err).ToNot(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("is not available for the desired channel group"))
			})
		ginkgo.It("can check the description of the cluster - [id:34102]",
			labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
			func() {
				ginkgo.By("Describe cluster in text format")
				output, err := clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Describe cluster in json format")
				rosaClient.Runner.JsonFormat()
				jsonOutput, err := clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()

				ginkgo.By("Get OCM Environment")
				rosaClient.Runner.JsonFormat()
				userInfo, err := rosaClient.OCMResource.UserInfo()
				gomega.Expect(err).To(gomega.BeNil())
				rosaClient.Runner.UnsetFormat()
				ocmApi := userInfo.OCMApi

				ginkgo.By("Compare the text result with the json result")
				gomega.Expect(CD.ID).To(gomega.Equal(jsonData.DigString("id")))
				gomega.Expect(CD.ExternalID).To(gomega.Equal(jsonData.DigString("external_id")))
				gomega.Expect(CD.ChannelGroup).To(gomega.Equal(jsonData.DigString("version", "channel_group")))
				gomega.Expect(CD.DNS).To(gomega.Equal(
					jsonData.DigString("domain_prefix") + "." + jsonData.DigString("dns", "base_domain")))
				gomega.Expect(CD.AWSAccount).NotTo(gomega.BeEmpty())
				gomega.Expect(CD.APIURL).To(gomega.Equal(jsonData.DigString("api", "url")))
				gomega.Expect(CD.ConsoleURL).To(gomega.Equal(jsonData.DigString("console", "url")))
				gomega.Expect(CD.Region).To(gomega.Equal(jsonData.DigString("region", "id")))

				gomega.Expect(CD.State).To(gomega.Equal(jsonData.DigString("status", "state")))
				gomega.Expect(CD.Created).NotTo(gomega.BeEmpty())

				ginkgo.By("Get details page console url")
				consoleURL := helper.GetConsoleUrlBasedOnEnv(ocmApi)
				subscriptionID := jsonData.DigString("subscription", "id")
				if consoleURL != "" {
					gomega.Expect(CD.DetailsPage).To(gomega.Equal(consoleURL + subscriptionID))
				}

				if jsonData.DigBool("aws", "private_link") {
					gomega.Expect(CD.Private).To(gomega.Equal("Yes"))
				} else {
					gomega.Expect(CD.Private).To(gomega.Equal("No"))
				}

				if jsonData.DigBool("hypershift", "enabled") {
					//todo
				} else {
					if jsonData.DigBool("multi_az") {
						gomega.Expect(CD.MultiAZ).To(gomega.Equal(jsonData.DigBool("multi_az")))
					} else {
						gomega.Expect(CD.Nodes[0]["Control plane"]).To(gomega.Equal(int(jsonData.DigFloat("nodes", "master"))))
						gomega.Expect(CD.Nodes[1]["Infra"]).To(gomega.Equal(int(jsonData.DigFloat("nodes", "infra"))))
						gomega.Expect(CD.Nodes[2]["Compute"]).To(gomega.Equal(int(jsonData.DigFloat("nodes", "compute"))))
					}
				}

				gomega.Expect(CD.Network[1]["Service CIDR"]).To(gomega.Equal(jsonData.DigString("network", "service_cidr")))
				gomega.Expect(CD.Network[2]["Machine CIDR"]).To(gomega.Equal(jsonData.DigString("network", "machine_cidr")))
				gomega.Expect(CD.Network[3]["Pod CIDR"]).To(gomega.Equal(jsonData.DigString("network", "pod_cidr")))
				gomega.Expect(CD.Network[4]["Host Prefix"]).
					Should(gomega.ContainSubstring(strconv.FormatFloat(jsonData.DigFloat("network", "host_prefix"), 'f', -1, 64)))
				gomega.Expect(CD.InfraID).To(gomega.Equal(jsonData.DigString("infra_id")))
			})

		ginkgo.It("can restrict master API endpoint to direct, private connectivity or not - [id:38850]",
			labels.High, labels.Runtime.Day2, labels.FedRAMP,
			func() {
				ginkgo.By("Check the cluster is not private cluster")
				private, err := clusterService.IsPrivateCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				if private {
					SkipTestOnFeature("private")
				}
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				ginkgo.By("Edit cluster to private to true")
				out, err := clusterService.EditCluster(
					clusterID,
					"--private",
					"-y",
				)
				if !isSTS || isHostedCP {
					gomega.Expect(err).To(gomega.BeNil())
					textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
					gomega.Expect(textData).
						Should(gomega.ContainSubstring(
							"You are choosing to make your cluster API private. You will not be able to access your cluster"))
					gomega.Expect(textData).Should(gomega.ContainSubstring("Updated cluster '%s'", clusterID))
				} else {
					gomega.Expect(err).ToNot(gomega.BeNil())
					gomega.Expect(rosaClient.Parser.TextData.Input(out).Parse().Tip()).
						Should(gomega.ContainSubstring(
							"Failed to update cluster"))
				}
				defer func() {
					ginkgo.By("Edit cluster to private back to false")
					out, err = clusterService.EditCluster(
						clusterID,
						"--private=false",
						"-y",
					)
					gomega.Expect(err).To(gomega.BeNil())
					textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
					gomega.Expect(textData).Should(gomega.ContainSubstring("Updated cluster '%s'", clusterID))

					ginkgo.By("Describe cluster to check Private is true")
					output, err := clusterService.DescribeCluster(clusterID)
					gomega.Expect(err).To(gomega.BeNil())
					CD, err := clusterService.ReflectClusterDescription(output)
					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(CD.Private).To(gomega.Equal("No"))
				}()

				ginkgo.By("Describe cluster to check Private is true")
				output, err := clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())
				if !isSTS || isHostedCP {
					gomega.Expect(CD.Private).To(gomega.Equal("Yes"))
				} else {
					gomega.Expect(CD.Private).To(gomega.Equal("No"))
				}
			})

		// OCM-5231 caused the description parser issue
		ginkgo.It("can disable workload monitoring on/off - [id:45159]",
			labels.High, labels.Runtime.Day2, labels.FedRAMP,
			func() {
				ginkgo.By("Load the original cluster config")
				clusterConfig, err := config.ParseClusterProfile()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				ginkgo.By("Check the cluster UWM is in expected status")
				output, err := clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				// nolint
				expectedUWMValue := "Enabled"
				recoverUWMStatus := false
				if clusterConfig.DisableWorkloadMonitoring {
					expectedUWMValue = "Disabled"
					recoverUWMStatus = true
				}
				gomega.Expect(clusterDetail.UserWorkloadMonitoring).To(gomega.Equal(expectedUWMValue))
				defer clusterService.EditCluster(clusterID,
					fmt.Sprintf("--disable-workload-monitoring=%v", recoverUWMStatus),
					"-y")

				ginkgo.By("Disable the UWM")
				expectedUWMValue = "Disabled"
				_, err = clusterService.EditCluster(clusterID,
					"--disable-workload-monitoring",
					"-y")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				ginkgo.By("Check the disable result for cluster description")
				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				clusterDetail, err = clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(clusterDetail.UserWorkloadMonitoring).To(gomega.Equal(expectedUWMValue))

				ginkgo.By("Enable the UWM again")
				expectedUWMValue = "Enabled"
				_, err = clusterService.EditCluster(clusterID,
					"--disable-workload-monitoring=false",
					"-y")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				ginkgo.By("Check the disable result for cluster description")
				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				clusterDetail, err = clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(clusterDetail.UserWorkloadMonitoring).To(gomega.Equal(expectedUWMValue))
			})

		ginkgo.It("can edit privacy and workload monitoring via rosa-cli - [id:60275]",
			labels.Critical, labels.Runtime.Day2, labels.FedRAMP,
			func() {
				ginkgo.By("Check the cluster is private cluster")
				private, err := clusterService.IsPrivateCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				if !private {
					SkipTestOnFeature("private")
				}
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				isHostedCP, err := clusterService.IsHostedCPCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Run command to check help message of edit cluster")
				out, editErr := clusterService.EditCluster(clusterID, "-h")
				gomega.Expect(editErr).ToNot(gomega.HaveOccurred())
				gomega.Expect(out.String()).Should(gomega.ContainSubstring("rosa edit cluster [flags]"))
				gomega.Expect(out.String()).Should(gomega.ContainSubstring("rosa edit cluster -c mycluster --private"))
				gomega.Expect(out.String()).Should(gomega.ContainSubstring("rosa edit cluster -c mycluster --interactive"))

				ginkgo.By("Edit the cluster with '--private=false' flag")
				out, editErr = clusterService.EditCluster(
					clusterID,
					"--private=false",
					"-y",
				)

				defer func() {
					ginkgo.By("Edit cluster to private back to false")
					out, err := clusterService.EditCluster(
						clusterID,
						"--private",
						"-y",
					)
					gomega.Expect(err).To(gomega.BeNil())
					textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
					gomega.Expect(textData).Should(gomega.ContainSubstring("Updated cluster '%s'", clusterID))

					ginkgo.By("Describe cluster to check Private is true")
					output, err := clusterService.DescribeCluster(clusterID)
					gomega.Expect(err).To(gomega.BeNil())
					CD, err := clusterService.ReflectClusterDescription(output)
					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(CD.Private).To(gomega.Equal("Yes"))
				}()

				textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()

				output, err := clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())

				if isSTS && !isHostedCP {
					gomega.Expect(editErr).ToNot(gomega.BeNil())
					gomega.Expect(textData).Should(gomega.ContainSubstring("Failed to update cluster"))

					gomega.Expect(CD.Private).To(gomega.Equal("Yes"))
				} else {
					gomega.Expect(editErr).To(gomega.BeNil())
					gomega.Expect(textData).Should(gomega.ContainSubstring("Updated cluster '%s'", clusterID))

					gomega.Expect(CD.Private).To(gomega.Equal("No"))
				}

				ginkgo.By("Edit the cluster with '--private' flag")
				out, editErr = clusterService.EditCluster(
					clusterID,
					"--private",
					"-y",
				)
				gomega.Expect(editErr).To(gomega.BeNil())
				textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(textData).Should(gomega.ContainSubstring("You are choosing to make your cluster API private. " +
					"You will not be able to access your cluster until you edit network settings in your cloud provider. " +
					"To also change the privacy setting of the application router endpoints, use the 'rosa edit ingress' command."))
				gomega.Expect(textData).Should(gomega.ContainSubstring("Updated cluster '%s'", clusterID))

				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err = clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(CD.Private).To(gomega.Equal("Yes"))
			})

		// Excluded until bug on OCP-73161 is resolved
		ginkgo.It("can verify delete protection on a rosa cluster - [id:73161]",
			labels.High, labels.Runtime.Day2, labels.Exclude,
			func() {
				ginkgo.By("Get original delete protection value")
				output, err := clusterService.DescribeClusterAndReflect(clusterID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				originalDeleteProtection := output.EnableDeleteProtection

				ginkgo.By("Enable delete protection on the cluster")
				deleteProtection := constants.DeleteProtectionEnabled
				_, err = clusterService.EditCluster(clusterID, "--enable-delete-protection=true", "-y")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				defer clusterService.EditCluster(clusterID,
					fmt.Sprintf("--enable-delete-protection=%s", originalDeleteProtection), "-y")

				ginkgo.By("Check the enable result from cluster description")
				output, err = clusterService.DescribeClusterAndReflect(clusterID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(output.EnableDeleteProtection).To(gomega.Equal(deleteProtection))

				ginkgo.By("Attempt to delete cluster with delete protection enabled")
				out, err := clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(textData).Should(gomega.ContainSubstring(
					`Delete-protection has been activated on this cluster and 
				it cannot be deleted until delete-protection is disabled`))

				ginkgo.By("Disable delete protection on the cluster")
				deleteProtection = constants.DeleteProtectionDisabled
				_, err = clusterService.EditCluster(clusterID, "--enable-delete-protection=false", "-y")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				ginkgo.By("Check the disable result from cluster description")
				output, err = clusterService.DescribeClusterAndReflect(clusterID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(output.EnableDeleteProtection).To(gomega.Equal(deleteProtection))
			})

		// Excluded until bug on OCP-74656 is resolved
		ginkgo.It("can verify delete protection on a rosa cluster negative - [id:74656]",
			labels.Medium, labels.Runtime.Day2, labels.Exclude,
			func() {
				ginkgo.By("Enable delete protection with invalid values")
				resp, err := clusterService.EditCluster(clusterID,
					"--enable-delete-protection=aaa",
					"-y",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				gomega.Expect(textData).Should(
					gomega.ContainSubstring(`Error: invalid argument "aaa" for "--enable-delete-protection"`))

				resp, err = clusterService.EditCluster(clusterID, "--enable-delete-protection=", "-y")
				gomega.Expect(err).To(gomega.HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(resp).Parse().Tip()
				gomega.Expect(textData).Should(
					gomega.ContainSubstring(`Error: invalid argument "" for "--enable-delete-protection"`))
			})

		ginkgo.It("can edit proxy successfully - [id:46308]", labels.High, labels.Runtime.Day2, labels.FedRAMP,
			func() {
				ginkgo.By("Load the original cluster config")
				clusterConfig, err := config.ParseClusterProfile()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				var verifyProxy = func(httpProxy string, httpsProxy string, noProxy string, caFile string) {
					clusterDescription, err := clusterService.DescribeClusterAndReflect(clusterID)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())

					clusterHTTPProxy, clusterHTTPSProxy, clusterNoProxy := clusterService.DetectProxy(clusterDescription)
					gomega.Expect(httpProxy).To(gomega.Equal(clusterHTTPProxy), "http proxy not match")
					gomega.Expect(httpsProxy).To(gomega.Equal(clusterHTTPSProxy), "https proxy not match")
					gomega.Expect(noProxy).To(gomega.Equal(clusterNoProxy), "no proxy not match")
					if caFile == "" {
						gomega.Expect(clusterDescription.AdditionalTrustBundle).To(gomega.BeEmpty())
					} else {
						gomega.Expect(clusterDescription.AdditionalTrustBundle).To(gomega.Equal("REDACTED"))
					}
				}

				ginkgo.By("Check if cluster is BYOVPC")
				if !profile.ClusterConfig.ProxyEnabled {
					ginkgo.Skip("This feature only work for BYO VPC")
				}
				originalHttpProxy, originalHTTPSProxy, originalNoProxy, originalCAFile := "", "", "", ""
				if clusterConfig.Proxy.Enabled {
					originalHttpProxy,
						originalHTTPSProxy,
						originalNoProxy, originalCAFile =
						clusterConfig.Proxy.Http,
						clusterConfig.Proxy.Https,
						clusterConfig.Proxy.NoProxy,
						clusterConfig.Proxy.TrustBundleFile
				}

				ginkgo.By("Edit cluster with https_proxy, http_proxy, no_proxy and trust-bundle-file")
				updateHttpProxy := "http://example.com"
				updateHttpsProxy := "https://example.com"
				updatedNoProxy := "example.com"
				updatedCA := ""
				_, err = clusterService.EditCluster(clusterID,
					"--http-proxy", updateHttpProxy,
					"--https-proxy", updateHttpsProxy,
					"--no-proxy", updatedNoProxy,
					"--additional-trust-bundle-file", updatedCA,
				)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				defer clusterService.EditCluster(clusterID,
					"--http-proxy", originalHttpProxy,
					"--https-proxy", originalHTTPSProxy,
					"--no-proxy", originalNoProxy,
					"--additional-trust-bundle-file", originalCAFile,
				)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				verifyProxy(updateHttpProxy, updateHttpsProxy, updatedNoProxy, updatedCA)

				ginkgo.By("Edit cluster for removing cluster-wide proxy")
				updateHttpProxy = ""
				updateHttpsProxy = ""
				updatedNoProxy = ""
				updatedCA = originalCAFile
				_, err = clusterService.EditCluster(clusterID,
					"--http-proxy", updateHttpProxy,
					"--https-proxy", updateHttpsProxy,
					"--no-proxy", updatedNoProxy,
					"--additional-trust-bundle-file", updatedCA,
				)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				verifyProxy(updateHttpProxy, updateHttpsProxy, updatedNoProxy, updatedCA)

				ginkgo.By("Edit cluster with https_proxy and no_proxy with different valid value")
				updateHttpsProxy = "https://test-46308.com"
				updatedNoProxy = "rosacli-46308.com"
				_, err = clusterService.EditCluster(clusterID,
					"--https-proxy", updateHttpsProxy,
					"--no-proxy", updatedNoProxy,
				)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				verifyProxy(updateHttpProxy, updateHttpsProxy, updatedNoProxy, updatedCA)

				ginkgo.By("Edit cluster with only http_proxy")
				updateHttpProxy = "http://test-46308.com"
				_, err = clusterService.EditCluster(clusterID,
					"--http-proxy", updateHttpProxy,
				)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				verifyProxy(updateHttpProxy, updateHttpsProxy, updatedNoProxy, updatedCA)
			})
		ginkgo.It("Changing billing account for the cluster - [id:75921]",
			labels.High, labels.Runtime.Day2,
			func() {
				ginkgo.By("Check the help message of 'rosa edit cluster -h'")
				helpOutput, err := clusterService.EditCluster("", "-h")
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(helpOutput.String()).To(gomega.ContainSubstring("--billing-account"))

				ginkgo.By("Change the billing account for the cluster")
				output, err := clusterService.EditCluster(clusterID, "--billing-account", constants.ChangedBillingAccount)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(gomega.ContainSubstring("Updated cluster"))

				ginkgo.By("Check if billing account is changed")
				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				CD, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(CD.AWSBillingAccount).To(gomega.Equal(constants.ChangedBillingAccount))

				ginkgo.By("Create another machinepool without security groups and describe it")
				mpName := "mp-75921"
				_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
					"--replicas", "3",
					"-y",
				)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				defer func() {
					ginkgo.By("Remove the machine pool")
					_, _ = rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

					ginkgo.By("Change the billing account back")
					output, err := clusterService.EditCluster(clusterID, "--billing-account", constants.BillingAccount)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
					gomega.Expect(output.String()).Should(gomega.ContainSubstring("Updated cluster"))
				}()
			})

		ginkgo.It("Changing invalid billing account - [id:75922]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				ginkgo.By("Change the billing account with invalid value")
				output, err := clusterService.EditCluster(clusterID, "--billing-account", "qweD3")
				gomega.Expect(err).ToNot(gomega.BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).
					Should(gomega.ContainSubstring(
						"not valid. Rerun the command with a valid billing account number"))

				output, err = clusterService.EditCluster(clusterID, "--billing-account", "123")
				gomega.Expect(err).ToNot(gomega.BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).
					Should(gomega.ContainSubstring(
						"not valid. Rerun the command with a valid billing account number"))

				ginkgo.By("Change the billing account with an empty string")
				output, err = clusterService.EditCluster(clusterID, "--billing-account", " ")
				gomega.Expect(err).ToNot(gomega.BeNil())
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).
					Should(gomega.ContainSubstring(
						"not valid. Rerun the command with a valid billing account number"))

				ginkgo.By("Check the billing account is NOT changed")
				clusterConfig, err := config.ParseClusterProfile()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())
				if clusterConfig.BillingAccount != "" {
					gomega.Expect(CD.AWSBillingAccount).To(gomega.Equal(clusterConfig.BillingAccount))
				}
			})
	})
var _ = ginkgo.Describe("Edit cluster validation should", labels.Feature.Cluster, func() {
	defer ginkgo.GinkgoRecover()

	var (
		clusterID      string
		rosaClient     *rosacli.Client
		clusterService rosacli.ClusterService
		upgradeService rosacli.UpgradeService
	)

	ginkgo.BeforeEach(func() {
		ginkgo.By("Get the cluster")
		clusterID = config.GetClusterID()
		gomega.Expect(clusterID).ToNot(gomega.Equal(""), "ClusterID is required. Please export CLUSTER_ID")

		ginkgo.By("Init the client")
		rosaClient = rosacli.NewClient()
		clusterService = rosaClient.Cluster
		upgradeService = rosaClient.Upgrade
	})

	ginkgo.AfterEach(func() {
		ginkgo.By("Clean the cluster")
		rosaClient.CleanResources(clusterID)
	})
	ginkgo.It("can validate for deletion of upgrade policy of rosa cluster - [id:38787]",
		labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
		func() {
			ginkgo.By("Validate that deletion of upgrade policy for rosa cluster will work via rosacli")
			output, err := upgradeService.DeleteUpgrade()
			gomega.Expect(err).To(gomega.HaveOccurred())
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).Should(gomega.ContainSubstring(`required flag(s) "cluster" not set`))

			ginkgo.By("Delete an non-existent upgrade when cluster has no scheduled policy")
			output, err = upgradeService.DeleteUpgrade("-c", clusterID)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).Should(gomega.ContainSubstring(`There are no scheduled upgrades on cluster '%s'`, clusterID))

			ginkgo.By("Delete with unknown flag --interactive")
			output, err = upgradeService.DeleteUpgrade("-c", clusterID, "--interactive")
			gomega.Expect(err).To(gomega.HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).Should(gomega.ContainSubstring("Error: unknown flag: --interactive"))
		})

	ginkgo.It("can validate create/delete upgrade policies for HCP clusters - [id:73814]",
		labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
		func() {
			defer func() {
				_, err := upgradeService.DeleteUpgrade("-c", clusterID, "-y")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}()

			ginkgo.By("Skip testing if the cluster is not a HCP cluster")
			hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			if !hostedCluster {
				SkipNotHosted()
			}

			ginkgo.By("Upgrade cluster with invalid cluster id")
			invalidClusterID := helper.GenerateRandomString(30)
			output, err := upgradeService.Upgrade("-c", invalidClusterID)
			gomega.Expect(err).To(gomega.HaveOccurred())
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).
				To(gomega.ContainSubstring(
					"ERR: Failed to get cluster '%s': There is no cluster with identifier or name '%s'",
					invalidClusterID,
					invalidClusterID))

			ginkgo.By("Upgrade cluster with incorrect format of the date and time")
			output, err = upgradeService.Upgrade(
				"-c", clusterID,
				"--mode=auto",
				"--schedule-date=\"2024-06\"",
				"--schedule-time=\"09:00:12\"",
				"-y")
			gomega.Expect(err).To(gomega.HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).To(gomega.ContainSubstring("ERR: schedule date should use the format 'yyyy-mm-dd'"))

			ginkgo.By("Upgrade cluster using --schedule, --schedule-date and --schedule-time flags at the same time")
			output, err = upgradeService.Upgrade(
				"-c", clusterID,
				"--mode=auto",
				"--schedule-date=\"2024-06-24\"",
				"--schedule-time=\"09:00\"",
				"--schedule=\"5 5 * * *\"",
				"-y")
			gomega.Expect(err).To(gomega.HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).
				To(gomega.ContainSubstring(
					"ERR: The '--schedule-date' and '--schedule-time' options are mutually exclusive with '--schedule'"))

			ginkgo.By("Upgrade cluster using --schedule and --version flags at the same time")
			output, err = upgradeService.Upgrade(
				"-c", clusterID,
				"--mode=auto",
				"--schedule=\"5 5 * * *\"",
				"--version=4.15.10",
				"-y")
			gomega.Expect(err).To(gomega.HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).
				To(gomega.ContainSubstring(
					"ERR: The '--schedule' option is mutually exclusive with '--version'"))

			ginkgo.By("Upgrade cluster with value not match the cron epression")
			output, err = upgradeService.Upgrade(
				"-c", clusterID,
				"--mode=auto",
				"--schedule=\"5 5\"",
				"-y")
			gomega.Expect(err).To(gomega.HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).
				To(gomega.ContainSubstring(
					"ERR: Schedule '\"5 5\"' is not a valid cron expression"))

			ginkgo.By("Upgrade cluster with node_drain_grace_period")
			output, err = upgradeService.Upgrade(
				"-c", clusterID,
				"--mode=auto",
				"--schedule", "20 20 * * *",
				"--node-drain-grace-period", "60",
				"-y")
			gomega.Expect(err).To(gomega.HaveOccurred())
			textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).
				To(gomega.ContainSubstring(
					"ERR: node-drain-grace-period flag is not supported to hosted clusters"))
		})

	ginkgo.It("can validate cluster proxy well - [id:46310]", labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
		func() {
			ginkgo.By("Load the original cluster config")
			clusterConfig, err := config.ParseClusterProfile()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			ginkgo.By("Load the profile")
			profile := handler.LoadProfileYamlFileByENV()

			ginkgo.By("Skip if the cluster is no proxy setting")
			if !profile.ClusterConfig.ProxyEnabled {
				ginkgo.Skip("This feature only work for the cluster with proxy setting")
			}

			ginkgo.By("Edit cluster with invalid http_proxy set")
			if !profile.ClusterConfig.BYOVPC {
				output, err := clusterService.EditCluster(clusterID,
					"--http-proxy", "http://test-proxy.com",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("ERR: Cluster-wide proxy is not supported on clusters using the default VPC"))
				return
			}
			originalHttpProxy, originalHTTPSProxy, originalNoProxy, originalCAFile := "", "", "", ""
			if clusterConfig.Proxy != nil && clusterConfig.Proxy.Enabled {
				originalHttpProxy,
					originalHTTPSProxy,
					originalNoProxy, originalCAFile =
					clusterConfig.Proxy.Http,
					clusterConfig.Proxy.Https,
					clusterConfig.Proxy.NoProxy,
					clusterConfig.Proxy.TrustBundleFile
			}
			fmt.Println(originalHttpProxy, originalHTTPSProxy, originalNoProxy, originalCAFile)

			ginkgo.By("Edit cluster with invalid http_proxy not started with http")
			invalidHTTPProxy := map[string]string{
				"invalidvalue":           "ERR: Invalid http-proxy value 'invalidvalue'",
				"https://test-proxy.com": "ERR: Expected http-proxy to have an http:// scheme",
			}
			for illegalHttpProxy, errMessage := range invalidHTTPProxy {
				output, err := clusterService.EditCluster(clusterID,
					"--http-proxy", illegalHttpProxy,
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(gomega.ContainSubstring(errMessage))
			}

			ginkgo.By("Edit cluster with invalid https_proxy set")
			output, err := clusterService.EditCluster(clusterID,
				"--https-proxy", "invalid",
			)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(output.String()).Should(gomega.ContainSubstring(`ERR: parse "invalid": invalid URI for request`))

			ginkgo.By("Edit cluster with invalid no_proxy ")
			output, err = clusterService.EditCluster(clusterID,
				"--no-proxy", "*",
			)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(output.String()).Should(gomega.ContainSubstring(`ERR: expected a valid user no-proxy value`))

			ginkgo.By("Edit cluster with invalid additional_trust_bundle set")
			tempDir, err := os.MkdirTemp("", "*")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer os.RemoveAll(tempDir)
			tempFile, err := helper.CreateFileWithContent(path.Join(tempDir, "rosacli-46310"), "invalid CA")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			output, err = clusterService.EditCluster(clusterID,
				"--additional-trust-bundle-file", tempFile,
			)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(output.String()).Should(gomega.ContainSubstring(`ERR: Failed to parse additional trust bundle`))

			ginkgo.By("Edit wide-proxy cluster with invalid additional_trust_bundle set path")
			output, err = clusterService.EditCluster(clusterID,
				"--additional-trust-bundle-file", "/not/existing",
			)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(output.String()).Should(gomega.ContainSubstring(`ERR: open /not/existing: no such file or directory`))

			ginkgo.By("Edit cluster which is set no-proxy but others empty")
			output, err = clusterService.EditCluster(clusterID,
				"--http-proxy", "",
				"--https-proxy", "",
				"--no-proxy", "example.com",
			)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(output.String()).Should(
				gomega.ContainSubstring("ERR: Failed to update cluster"))

			ginkgo.By("Set all http settings to empty")
			output, err = clusterService.EditCluster(clusterID,
				"--http-proxy", "",
				"--https-proxy", "",
				"--no-proxy", "",
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer clusterService.EditCluster(clusterID,
				"--http-proxy", originalHttpProxy,
				"--https-proxy", originalHTTPSProxy,
				"--no-proxy", originalNoProxy,
			)
			ginkgo.By("Edit cluster which is not set http-proxy and http-proxy  with the command")
			output, err = clusterService.EditCluster(clusterID,
				"--no-proxy", "example.com",
			)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(output.String()).Should(
				gomega.ContainSubstring("ERR: Expected at least one of the following: http-proxy, https-proxy"))

		})

	ginkgo.It("can validate cluster registry config patching well - [id:77149]",
		labels.Medium, labels.Runtime.Day2, labels.FedRAMP,
		func() {
			ginkgo.By("edit non-hcp with registry config")
			hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			if !hostedCluster {
				output, err := clusterService.EditCluster(clusterID,
					"--registry-config-blocked-registries", "test.blocked.com")
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("ERR: Setting the registry config is only supported for hosted clusters"))
				return
			}

			ginkgo.By("patch hcp with --registry-config-blocked-registries and " +
				"--registry-config-allowed-registries at same time")
			output, err := clusterService.EditCluster(clusterID,
				"--registry-config-blocked-registries", "test.blocked.com",
				"--registry-config-allowed-registries", "test.com")
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(output.String()).Should(
				gomega.ContainSubstring("ERR: Allowed registries and blocked registries are mutually exclusive fields"))

			ginkgo.By("patch hcp with invalid value for --registry-config-allowed-registries-for-import flag")
			output, err = clusterService.EditCluster(clusterID,
				"--registry-config-allowed-registries-for-import", "test.com:stringType")
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(output.String()).Should(
				gomega.ContainSubstring("ERR: Expected valid allowed registries for import values"))

		})
})
var _ = ginkgo.Describe("Additional security groups validation",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			profilesMap    map[string]*handler.Profile
			profile        *handler.Profile
			clusterHandler handler.ClusterHandler
		)

		ginkgo.BeforeEach(func() {
			var err error

			// Init the client
			rosaClient = rosacli.NewClient()
			// Get a random profile
			// Use hcp profiles as the region will be used for both classic and hcp
			profilesMap = handler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-hcp.yaml"))
			profilesNames := make([]string, 0, len(profilesMap))
			for k, v := range profilesMap {
				if !v.ClusterConfig.SharedVPC || !v.ClusterConfig.AutoscalerEnabled {
					profilesNames = append(profilesNames, k)
				}
			}
			profile = profilesMap[profilesNames[helper.RandomInt(len(profilesNames))]]
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, profile)
			gomega.Expect(err).To(gomega.BeNil())
		})

		ginkgo.AfterEach(func() {
			clusterHandler.Destroy()
		})
		ginkgo.It("Create rosa cluster with additional security groups will validate well via rosacli - [id:68971]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				var (
					ocpVersionBelow4_14 string
					ocpVersion          string
					index               int
					flagName            string
					SGIdsMoreThanTen    = 11
					caseNumber          = "68971"
					clusterName         = "ocp-68971"
					securityGroups      = map[string]string{
						"--additional-infra-security-group-ids":         "sg-aisgi",
						"--additional-control-plane-security-group-ids": "sg-acpsgi",
						"--additional-compute-security-group-ids":       "sg-acsgi",
					}
					invalidSecurityGroups = map[string]string{
						"--additional-infra-security-group-ids":         "invalid",
						"--additional-control-plane-security-group-ids": "invalid",
						"--additional-compute-security-group-ids":       "invalid",
					}
				)

				ginkgo.By("Get cluster upgrade version")
				versionService := rosaClient.Version
				versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupCandidate, false)
				gomega.Expect(err).To(gomega.BeNil())
				defaultVersion := versionList.DefaultVersion()
				gomega.Expect(defaultVersion).ToNot(gomega.BeNil())
				ocpVersion = defaultVersion.Version

				pickedVersions, err := versionList.FilterVersionsSameMajorAndEqualOrLowerThanMinor(4, 13, false)
				gomega.Expect(err).To(gomega.BeNil())
				if len(pickedVersions.OpenShiftVersions) <= 0 {
					ginkgo.Skip("There is no version bellow 4.14.0, skip this case")
				}
				ocpVersionBelow4_14 = pickedVersions.OpenShiftVersions[0].Version

				ginkgo.By("Prepare a vpc for the testing")
				resourcesHandler := clusterHandler.GetResourcesHandler()
				_, err = resourcesHandler.PrepareVPC(caseNumber, "", true, false)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				subnetMap, err := resourcesHandler.PrepareSubnets([]string{}, false)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				//Check all subnets are created successfully and are in available state. If not, wait for them to be available
				subnetIDs := append(subnetMap["private"], subnetMap["public"]...)

				awsClient, err := aws_client.CreateAWSClient("", resourcesHandler.GetVPC().Region)
				gomega.Expect(err).To(gomega.BeNil())
				err = wait.PollUntilContextTimeout(
					context.Background(),
					30*time.Second,
					300*time.Second,
					false,
					func(context.Context) (bool, error) {
						subnetsDetail, err := awsClient.Ec2Client.DescribeSubnets(context.TODO(),
							&ec2.DescribeSubnetsInput{
								SubnetIds: subnetIDs,
							},
						)
						if err != nil {
							return false, nil
						}
						for _, subnet := range subnetsDetail.Subnets {
							if subnet.State != "available" {
								return false, nil
							}
						}
						return true, nil
					})
				helper.AssertWaitPollNoErr(err, "subnets are not available after 300s")

				ginkgo.By("Prepare additional security group ids for testing")
				sgIDs, err := resourcesHandler.PrepareAdditionalSecurityGroups(SGIdsMoreThanTen, caseNumber)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				subnetsFlagValue := strings.Join(append(subnetMap["private"], subnetMap["public"]...), ",")
				rosaclient := rosacli.NewClient()

				ginkgo.By("Try creating cluster with additional security groups but no subnet-ids")
				for additionalSecurityGroupFlag := range securityGroups {
					output, err, _ := rosaclient.Cluster.Create(
						clusterName,
						"--region", resourcesHandler.GetVPC().Region,
						"--replicas", "3",
						additionalSecurityGroupFlag, strings.Join(sgIDs, ","),
					)
					gomega.Expect(err).To(gomega.HaveOccurred())
					index = strings.Index(additionalSecurityGroupFlag, "a")
					flagName = additionalSecurityGroupFlag[index:]
					gomega.Expect(output.String()).To(gomega.ContainSubstring(
						"Setting the `%s` flag is only allowed for BYO VPC clusters",
						flagName))
				}

				ginkgo.By("Try creating cluster with additional security groups and ocp version lower than 4.14")
				for additionalSecurityGroupFlag := range securityGroups {
					output, err, _ := rosaclient.Cluster.Create(
						clusterName,
						"--region", resourcesHandler.GetVPC().Region,
						"--replicas", "3",
						"--subnet-ids", subnetsFlagValue,
						additionalSecurityGroupFlag, strings.Join(sgIDs, ","),
						"--version", ocpVersionBelow4_14,
						"--channel-group", rosacli.VersionChannelGroupCandidate,
						"-y",
					)
					gomega.Expect(err).To(gomega.HaveOccurred())
					index = strings.Index(additionalSecurityGroupFlag, "a")
					flagName = additionalSecurityGroupFlag[index:]
					gomega.Expect(output.String()).To(gomega.ContainSubstring(
						"Parameter '%s' is not supported prior to version '4.14.0'",
						flagName))
				}

				ginkgo.By("Try creating cluster with invalid additional security groups")
				for additionalSecurityGroupFlag, value := range invalidSecurityGroups {
					output, err, _ := rosaclient.Cluster.Create(
						clusterName,
						"--region", resourcesHandler.GetVPC().Region,
						"--replicas", "3",
						"--subnet-ids", subnetsFlagValue,
						additionalSecurityGroupFlag, value,
						"--version", ocpVersion,
						"--channel-group", rosacli.VersionChannelGroupCandidate,
					)
					gomega.Expect(err).To(gomega.HaveOccurred())
					gomega.Expect(output.String()).To(
						gomega.ContainSubstring("Security Group ID '%s' doesn't have 'sg-' prefix", value))
				}

				ginkgo.By("Try creating cluster with additional security groups with invalid and more than 10 SG ids")
				for additionalSecurityGroupFlag := range securityGroups {

					output, err, _ := rosaclient.Cluster.Create(
						clusterName,
						"--region", resourcesHandler.GetVPC().Region,
						"--replicas", "3",
						"--subnet-ids", subnetsFlagValue,
						additionalSecurityGroupFlag, strings.Join(sgIDs, ","),
						"--version", ocpVersion,
						"--channel-group", rosacli.VersionChannelGroupCandidate,
					)
					gomega.Expect(err).To(gomega.HaveOccurred())
					gomega.Expect(output.String()).To(gomega.ContainSubstring(
						"limit for Additional Security Groups is '10', but '11' have been supplied"),
					)
				}
			})
	})

var _ = ginkgo.Describe("Classic cluster creation validation",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			profilesMap    map[string]*handler.Profile
			profile        *handler.Profile
			clusterService rosacli.ClusterService
			clusterHandler handler.ClusterHandler
		)

		ginkgo.BeforeEach(func() {
			var err error

			// Init the client
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			// Get a random profile
			profilesMap = handler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-classic.yaml"))
			profilesNames := make([]string, 0, len(profilesMap))
			for k, v := range profilesMap {
				if !v.ClusterConfig.SharedVPC && !v.ClusterConfig.AutoscalerEnabled {
					profilesNames = append(profilesNames, k)
				}
			}
			profile = profilesMap[profilesNames[helper.RandomInt(len(profilesNames))]]
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, profile)
			gomega.Expect(err).To(gomega.BeNil())
		})

		ginkgo.AfterEach(func() {
			clusterHandler.Destroy()
		})

		ginkgo.It("to check the basic validation for the classic rosa cluster creation by the rosa cli - [id:38770]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Prepare creation command")
				var command string
				var rosalCommand config.Command
				profile.NamePrefix = helper.GenerateRandomName("ci38770", 2)

				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).To(gomega.BeNil())

				// nolint
				command = "rosa create cluster --cluster-name " + profile.ClusterConfig.Name + " " + strings.Join(flags, " ")
				rosalCommand = config.GenerateCommand(command)

				if !rosalCommand.CheckFlagExist("--compute-machine-type") {
					rosalCommand.AddFlags("--compute-machine-type", constants.DefaultInstanceType)
				}

				rosalCommand.AddFlags("--dry-run")
				originalClusterName := rosalCommand.GetFlagValue("--cluster-name", true)
				originalMachineType := rosalCommand.GetFlagValue("--compute-machine-type", true)
				originalRegion := rosalCommand.GetFlagValue("--region", true)
				if !rosalCommand.CheckFlagExist("--replicas") {
					rosalCommand.AddFlags("--replicas", "3")
				}
				originalReplicas := rosalCommand.GetFlagValue("--replicas", true)

				if rosalCommand.CheckFlagExist("--enable-autoscaling") {
					rosalCommand.DeleteFlag("--enable-autoscaling", false)
				}

				if rosalCommand.CheckFlagExist("--min-replicas") {
					rosalCommand.DeleteFlag("--min-replicas", true)
				}

				if rosalCommand.CheckFlagExist("--max-replicas") {
					rosalCommand.DeleteFlag("--max-replicas", true)
				}

				invalidClusterNames := []string{
					"1-test-1",
					"-test-cluster",
					"test-cluster-",
				}
				for _, cn := range invalidClusterNames {
					ginkgo.By("Check the validation for cluster-name " + cn)
					rosalCommand.ReplaceFlagValue(map[string]string{
						"--cluster-name": cn,
					})
					stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
					gomega.Expect(err).NotTo(gomega.BeNil())
					gomega.Expect(stdout.String()).
						To(gomega.ContainSubstring(
							"Cluster name must consist of no more than 54 lowercase alphanumeric characters or '-', " +
								"start with a letter, and end with an alphanumeric character"))
				}

				ginkgo.By("Check the validation for compute-machine-type")
				invalidMachineType := "not-exist-machine-type"
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--compute-machine-type": invalidMachineType,
					"--cluster-name":         originalClusterName,
				})
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(stdout.String()).To(gomega.ContainSubstring("is not supported for cloud provider"))

				ginkgo.By("Check the validation for replicas")
				invalidReplicasTypeErrorMap := map[string]string{
					"4.5":  "invalid argument \"4.5\" for \"--replicas\" flag",
					"five": "invalid argument \"five\" for \"--replicas\" flag",
				}
				for k, v := range invalidReplicasTypeErrorMap {
					rosalCommand.ReplaceFlagValue(map[string]string{
						"--compute-machine-type": originalMachineType,
						"--replicas":             k,
					})
					stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
					gomega.Expect(err).NotTo(gomega.BeNil())
					gomega.Expect(stdout.String()).To(gomega.ContainSubstring(v))
				}
				if rosalCommand.CheckFlagExist("--multi-az") {
					if !profile.ClusterConfig.AutoscalerEnabled {
						invalidReplicasErrorMapMultiAZ := map[string]string{
							"2":  "Multi AZ cluster requires at least 3 compute nodes",
							"0":  "Multi AZ cluster requires at least 3 compute nodes",
							"-3": "must be non-negative",
							"5":  "Multi AZ clusters require that the number of compute nodes be a multiple of 3",
						}
						for k, v := range invalidReplicasErrorMapMultiAZ {
							rosalCommand.ReplaceFlagValue(map[string]string{
								"--replicas": k,
							})
							stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
							gomega.Expect(err).NotTo(gomega.BeNil())
							gomega.Expect(stdout.String()).To(gomega.ContainSubstring(v))
						}
					}
				} else {
					if !profile.ClusterConfig.AutoscalerEnabled {
						invalidReplicasErrorMapSingeAZ := map[string]string{
							"1":  "requires at least 2 compute nodes",
							"0":  "requires at least 2 compute nodes",
							"-1": "must be non-negative",
						}
						for k, v := range invalidReplicasErrorMapSingeAZ {
							rosalCommand.ReplaceFlagValue(map[string]string{
								"--replicas": k,
							})
							stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
							gomega.Expect(err).NotTo(gomega.BeNil())
							gomega.Expect(stdout.String()).To(gomega.ContainSubstring(v))
						}
					}
				}
				ginkgo.By("Check the validation for region")
				invalidRegion := "not-exist-region"
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--region":   invalidRegion,
					"--replicas": originalReplicas,
				})
				stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(stdout.String()).To(gomega.ContainSubstring("Unsupported region"))

				ginkgo.By("Check the validation for invalid billing-account for classic sts cluster")
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--region": originalRegion,
				})
				rosalCommand.AddFlags("--billing-account", "123456789")
				stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).ToNot(gomega.BeNil())
				gomega.Expect(stdout.String()).
					ToNot(gomega.ContainSubstring(
						"Billing accounts are only supported for"))
				gomega.Expect(stdout.String()).
					To(gomega.ContainSubstring(
						"is not valid"))
			})

		ginkgo.It("can allow sts cluster installation with compatible policies - [id:45161]",
			labels.High, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Prepare creation command")
				var command string
				var rosalCommand config.Command
				profile.NamePrefix = helper.GenerateRandomName("ci45161", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).To(gomega.BeNil())

				command = "rosa create cluster --cluster-name " + profile.ClusterConfig.Name + " " + strings.Join(flags, " ")
				rosalCommand = config.GenerateCommand(command)

				if !profile.ClusterConfig.STS {
					SkipTestOnFeature("policy")
				}

				clusterName := "cluster-45161"
				operatorPrefix := "cluster-45161-asdf"

				ginkgo.By("Create cluster with one Y-1 version")
				ocmResourceService := rosaClient.OCMResource
				versionService := rosaClient.Version
				accountRoleList, _, err := ocmResourceService.ListAccountRole()
				gomega.Expect(err).To(gomega.BeNil())

				installerRole := rosalCommand.GetFlagValue("--role-arn", true)
				ar := accountRoleList.AccountRole(installerRole)
				gomega.Expect(ar).ToNot(gomega.BeNil())

				cg := rosalCommand.GetFlagValue("--channel-group", true)
				if cg == "" {
					cg = rosacli.VersionChannelGroupStable
				}

				versionList, err := versionService.ListAndReflectVersions(cg, rosalCommand.CheckFlagExist("--hosted-cp"))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(versionList).ToNot(gomega.BeNil())
				foundVersion, err := versionList.FindNearestBackwardMinorVersion(ar.OpenshiftVersion, 1, false)
				gomega.Expect(err).To(gomega.BeNil())
				var clusterVersion string
				if foundVersion == nil {
					ginkgo.Skip("No cluster version < y-1 found for compatibility testing")
				}
				clusterVersion = foundVersion.Version

				replacingFlags := map[string]string{
					"--version":               clusterVersion,
					"--cluster-name":          clusterName,
					"-c":                      clusterName,
					"--operator-roles-prefix": operatorPrefix,
					"--domain-prefix":         clusterName,
				}

				if rosalCommand.GetFlagValue("--https-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--https-proxy", true)
					gomega.Expect(err).To(gomega.BeNil())
				}
				if rosalCommand.GetFlagValue("--no-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--no-proxy", true)
					gomega.Expect(err).To(gomega.BeNil())
				}
				if rosalCommand.GetFlagValue("--http-proxy", true) != "" {
					err = rosalCommand.DeleteFlag("--http-proxy", true)
					gomega.Expect(err).To(gomega.BeNil())
				}
				if rosalCommand.CheckFlagExist("--base-domain") {
					rosalCommand.DeleteFlag("--base-domain", true)
				}

				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run")
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(stdout.String()).To(
					gomega.ContainSubstring(fmt.Sprintf("Creating cluster '%s' should succeed", clusterName)))
			})

		ginkgo.It("to validate to create the sts cluster with invalid tag - [id:56440]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-56440"

				ginkgo.By("Create cluster with invalid tag key")
				out, err := clusterService.CreateDryRun(
					clusterName, "--tags=~~~:cluster",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"expected a valid user tag key '~~~' matching ^[\\pL\\pZ\\pN_.:/=+\\-@]{1,128}$"))

				ginkgo.By("Create cluster with invalid tag value")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags=name:****",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"expected a valid user tag value '****' matching ^[\\pL\\pZ\\pN_.:/=+\\-@]{0,256}$"))

				ginkgo.By("Create cluster with duplicate tag key")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags=name:test1,op:clound,name:test2",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"invalid tags, user tag keys must be unique, duplicate key 'name' found"))

				ginkgo.By("Create cluster with invalid tag format")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags=test1,test2,test4",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"invalid tag format for tag '[test1]'. Expected tag format: 'key value'"))

				ginkgo.By("Create cluster with empty tag value")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags", "foo:",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"invalid tag format, tag key or tag value can not be empty"))

				ginkgo.By("Create cluster with invalid tag format")
				out, err = clusterService.CreateDryRun(
					clusterName, "--tags=name:gender:age",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"invalid tag format for tag '[name gender age]'. Expected tag format: 'key value'"))

			})

		ginkgo.It("Create cluster with invalid volume size [id:66372]",
			labels.Medium,
			labels.Runtime.Day1Negative,
			func() {

				minSize := constants.MinClassicDiskSize
				maxSize := constants.MaxDiskSize
				clusterName := helper.GenerateRandomName("ocp-66372", 2)
				client := rosacli.NewClient()

				ginkgo.By("Try a worker disk size that's too small")
				out, err := clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", fmt.Sprintf("%dGiB", minSize-1),
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout := client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(gomega.ContainSubstring(fmt.Sprintf(constants.DiskSizeErrRangeMsg, minSize-1, minSize, maxSize)))

				ginkgo.By("Try a worker disk size that's a little bigger")
				out, err = clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", fmt.Sprintf("%dGiB", maxSize+1),
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(gomega.ContainSubstring(fmt.Sprintf(constants.DiskSizeErrRangeMsg, maxSize+1, minSize, maxSize)))

				ginkgo.By("Try a worker disk size that's very big")
				veryBigData := "34567865467898765789"
				out, err = clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", fmt.Sprintf("%sGiB", veryBigData),
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(gomega.ContainSubstring("Expected a valid machine pool root disk size value '%sGiB': "+
						"invalid disk size: '%sGi'. maximum size exceeded",
						veryBigData,
						veryBigData))

				ginkgo.By("Try a worker disk size that's negative")
				out, err = clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", "-1GiB",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(
						gomega.ContainSubstring(
							"Expected a valid machine pool root disk size value '-1GiB': " +
								"invalid disk size: '-1Gi'. positive size required"))

				ginkgo.By("Try a worker disk size that's a string")
				invalidStr := "invalid"
				out, err = clusterService.CreateDryRun(
					clusterName, "--worker-disk-size", invalidStr,
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(
						gomega.ContainSubstring(
							"Expected a valid machine pool root disk size value '%s': invalid disk size "+
								"format: '%s'. accepted units are Giga or Tera in the form of "+
								"g, G, GB, GiB, Gi, t, T, TB, TiB, Ti",
							invalidStr,
							invalidStr))
			})

		ginkgo.It("to validate to create cluster with availability zones - [id:52692]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-52692"

				ginkgo.By("Create cluster with the zone not available in the region")
				out, err := clusterService.CreateDryRun(
					clusterName, "--availability-zones", "us-east-2e", "--region", "us-east-2",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"Expected a valid availability zone, 'us-east-2e' doesn't belong to region 'us-east-2' availability zones"))

				ginkgo.By("Create cluster with zones not match region")
				out, err = clusterService.CreateDryRun(
					clusterName, "--availability-zones", "us-west-2b", "--region", "us-east-2",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"Expected a valid availability zone, 'us-west-2b' doesn't belong to region 'us-east-2' availability zones"))

				ginkgo.By("Create cluster with dup zones set")
				out, err = clusterService.CreateDryRun(
					clusterName,
					"--availability-zones", "us-west-2b,us-west-2b,us-west-2b",
					"--region", "us-west-2",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"Found duplicate Availability Zone: us-west-2b"))

				ginkgo.By("Create cluster with both zone and subnet set")
				out, err = clusterService.CreateDryRun(
					clusterName,
					"--availability-zones", "us-west-2b",
					"--subnet-ids", "subnet-039f2a2a2d2d83e7f",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(gomega.ContainSubstring(
						"Setting availability zones is not supported for BYO VPC. " +
							"ROSA autodetects availability zones from subnet IDs provided"))
			})

		ginkgo.It("Validate --worker-mp-labels option for ROSA cluster creation - [id:71329]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				var (
					clusterName        = "cluster-71329"
					operatorPrefix     = "cluster-op-prefix"
					invalidKey         = "p*=test"
					emptyKey           = "=test"
					emptyWorkerMpLabel = ""
					longKey            = strings.Repeat("abcd1234", 16) + "=test"
					longValue          = "test=" + strings.Repeat("abcd1234", 16)
					duplicateKey       = "test=test1,test=test2"
					replacingFlags     = map[string]string{
						"-c":                     clusterName,
						"--cluster-name":         clusterName,
						"--domain-prefix":        clusterName,
						"--operator-role-prefix": operatorPrefix,
					}
				)

				ginkgo.By("Prepare creation command")
				var command string
				var rosalCommand config.Command
				profile.NamePrefix = helper.GenerateRandomName("ci71329", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).To(gomega.BeNil())

				command = "rosa create cluster --cluster-name " + profile.ClusterConfig.Name + " " + strings.Join(flags, " ")
				rosalCommand = config.GenerateCommand(command)

				ginkgo.By("Create ROSA cluster with the --worker-mp-labels flag and invalid key")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", invalidKey, "-y")
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				index := strings.Index(invalidKey, "=")
				key := invalidKey[:index]
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							"Invalid label key '%s': name part must consist of alphanumeric characters, '-', '_' "+
								"or '.', and must start and end with an alphanumeric character",
							key))

				ginkgo.By("Create ROSA cluster with the --worker-mp-labels flag and empty key")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", emptyKey, "-y")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							"Invalid label key '': name part must be non-empty; name part must consist of alphanumeric " +
								"characters, '-', '_' or '.', and must start and end with an alphanumeric character"))

				ginkgo.By("Create ROSA cluster with the --worker-mp-labels flag without any value")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", emptyWorkerMpLabel, "-y")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("Expected key=value format for labels"))

				ginkgo.By("Create ROSA cluster with the --worker-mp-labels flag and >63 character label key")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", longKey, "-y")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				index = strings.Index(longKey, "=")
				longLabelKey := longKey[:index]
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							"Invalid label key '%s': name part must be no more than 63 characters", longLabelKey))

				ginkgo.By("Create ROSA cluster with the --worker-mp-labels flag and >63 character label value")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", longValue, "-y")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				index = strings.Index(longValue, "=")
				longLabelValue := longValue[index+1:]
				key = longValue[:index]
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring("Invalid label value '%s': at key: '%s': must be no more than 63 characters",
							longLabelValue,
							key))

				ginkgo.By("Create ROSA cluster with the --worker-mp-labels flag and duplicated key")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--worker-mp-labels", duplicateKey, "-y")
				index = strings.Index(duplicateKey, "=")
				key = duplicateKey[:index]
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("Duplicated label key '%s' used", key))
			})

		ginkgo.It("to validate to create the cluster with version not in the channel group - [id:74399]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-74399"

				ginkgo.By("Create cluster with version not in channel group")
				errorOutput, err := clusterService.CreateDryRun(
					clusterName, "--version=4.15.100",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(errorOutput.String()).
					To(
						gomega.ContainSubstring("Expected a valid OpenShift version: A valid version number must be specified"))
			})

		ginkgo.It("to validate to create the cluster with setting 'fips' flag but '--etcd-encryption=false' - [id:74436]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-74436"

				ginkgo.By("Create cluster with fips flag but '--etcd-encryption=false")
				errorOutput, err := clusterService.CreateDryRun(
					clusterName, "--fips", "--etcd-encryption=false",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(errorOutput.String()).To(
					gomega.ContainSubstring("etcd encryption cannot be disabled on clusters with FIPS mode"))
			})
		ginkgo.It("validate use-local-credentials won't work with sts - [id:76481]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterName := helper.GenerateRandomName("c76481", 3)
				ginkgo.By("Create account-roles for testing")
				ocmResourceService := rosaClient.OCMResource
				accrolePrefix := helper.GenerateRandomName("ar76481", 3)
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
				defer func() {
					ginkgo.By("Delete the account-roles")
					output, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
						"--prefix", accrolePrefix,
						"-y")

					gomega.Expect(err).To(gomega.BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					gomega.Expect(textData).To(gomega.ContainSubstring("Successfully deleted"))
				}()
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).To(gomega.ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				gomega.Expect(err).To(gomega.BeNil())
				ar := arl.DigAccountRoles(accrolePrefix, false)
				ginkgo.By("Create cluster with use-local-credentials flag but with sts")
				out, err := clusterService.CreateDryRun(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"-y", "--dry-run",
					"--use-local-credentials",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("Local credentials are not supported for STS clusters"))
			})
	})

var _ = ginkgo.Describe("Create cluster with invalid options will",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
		)

		ginkgo.BeforeEach(func() {
			// Init the client
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
		})

		ginkgo.It("to validate subnet well when create cluster - [id:37177]", labels.Medium, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Setup vpc with list azs")
				testingTegion := "us-east-2"
				ginkgo.By("Prepare subnets for the coming testing")
				vpc, err := vpc_client.PrepareVPC("rosacli-37177", testingTegion, "", true, "")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				defer vpc.DeleteVPCChain(true)

				azs, err := vpc.AWSClient.ListAvaliableZonesForRegion(testingTegion, "availability-zone")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(azs).ToNot(gomega.BeEmpty())

				subnetMap, err := vpc.PreparePairSubnetByZone(azs[0])
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				ginkgo.By("Create cluster with non-existed subnets on AWS")
				clusterName := "cluster-37177"

				output, err, _ := clusterService.Create(clusterName,
					"--subnet-ids", "subnet-nonexisting",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("he subnet ID 'subnet-nonexisting' does not exist"))

				ginkgo.By("Create multi_az cluster with subnet which only support 1 zone")
				output, err, _ = clusterService.Create(clusterName,
					"--subnet-ids", subnetMap["private"].ID,
					"--multi-az",
					"--region", testingTegion,
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("The number of subnets for a 'multi-AZ' 'cluster' should be '6'," +
						" instead received: '1'"))

				// Can only test when az number is bigger than 2
				if len(azs) > 2 {
					ginkgo.By("Create single_az cluster with multiple zones set")
					subnetMap2, err := vpc.PreparePairSubnetByZone(azs[1])
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
					output, err, _ = clusterService.Create(clusterName,
						"--subnet-ids", strings.Join(
							[]string{
								subnetMap["private"].ID,
								subnetMap2["private"].ID},
							","),
						"--region", testingTegion,
					)
					gomega.Expect(err).To(gomega.HaveOccurred())
					gomega.Expect(output.String()).Should(
						gomega.ContainSubstring("Only a single availability zone can be provided" +
							" to a single-availability-zone cluster, instead received 2"))
				}

				ginkgo.By("Create multi_az cluster with 5 subnet set")
				// This test is only available for multi-az with at least 3 zones
				if len(azs) >= 3 {
					fitZoneSubnets := []string{}
					for _, az := range azs[0:3] {
						subnetMap, err := vpc.PreparePairSubnetByZone(az)
						gomega.Expect(err).ToNot(gomega.HaveOccurred())
						fitZoneSubnets = append(fitZoneSubnets,
							subnetMap["private"].ID,
							subnetMap["public"].ID)
					}
					fiveSubnetsList := fitZoneSubnets[0:5]
					output, err, _ = clusterService.Create(clusterName,
						"--subnet-ids", strings.Join(
							fiveSubnetsList,
							","),
						"--region", testingTegion,
						"--multi-az",
					)
					gomega.Expect(err).To(gomega.HaveOccurred())
					gomega.Expect(output.String()).Should(
						gomega.ContainSubstring("The number of subnets for a 'multi-AZ' 'cluster' should be '6', instead received: '%d'",
							len(fiveSubnetsList)))
				}

				ginkgo.By("Create with subnets in same zone")
				// pick the first az for testing
				additionalSubnetsNumber := 2
				sameZoneSubnets := []string{
					subnetMap["private"].ID,
					subnetMap["public"].ID,
				}
				for additionalSubnetsNumber > 0 {
					_, subnet, err := vpc.CreatePairSubnet(azs[0])
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
					gomega.Expect(len(subnet)).To(gomega.Equal(2))
					sameZoneSubnets = append(sameZoneSubnets, subnet[0].ID, subnet[1].ID)
					additionalSubnetsNumber--
				}

				output, err, _ = clusterService.Create(clusterName,
					"--subnet-ids", strings.Join(
						sameZoneSubnets, ","),
					"--region", testingTegion,
					"--multi-az",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring(
						"The number of Availability Zones for a Multi AZ cluster should be 3, instead received: 1"))
			})

		ginkgo.It("to validate the network when create cluster - [id:38857]", labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterName := "rosaci-38857"
				ginkgo.By("illegal machine/service/pod cidr when create cluster")
				illegalCIDRMap := map[string]string{
					"--machine-cidr": "10111.0.0.0/16",
					"--service-cidr": "10111.0.0.0/16",
					"--pod-cidr":     "10111.0.0.0/16",
				}
				for flag, invalidValue := range illegalCIDRMap {
					output, err, _ := clusterService.Create(clusterName,
						flag, invalidValue,
					)
					gomega.Expect(err).To(gomega.HaveOccurred())
					gomega.Expect(output.String()).Should(
						gomega.ContainSubstring(`invalid argument "%s" for "%s" flag: invalid CIDR address: %s`,
							invalidValue, flag, invalidValue))
				}
				ginkgo.By("Check the overlapped CIDR block validation")
				output, err, _ := clusterService.Create(clusterName,
					"--service-cidr", "1.0.0.0/16",
					"--pod-cidr", "1.0.0.0/16",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("Service CIDR '1.0.0.0/16' and pod CIDR '1.0.0.0/16' overlap"))

				ginkgo.By("Check the invalid machine/service/pod CIDR")
				invalidCIDRMap := map[string]string{
					"--machine-cidr": "2.0.0.0/8",
					"--service-cidr": "1.0.0.0/25",
					"--pod-cidr":     "1.0.0.0/28",
				}
				for flag, invalidValue := range invalidCIDRMap {
					output, err, _ := clusterService.Create(clusterName,
						flag, invalidValue,
					)
					gomega.Expect(err).To(gomega.HaveOccurred())
					switch flag {
					case "--machine-cidr":
						gomega.Expect(output.String()).Should(
							gomega.ContainSubstring("The allowed block size must be between a /16 netmask and /25"))
					case "--service-cidr":
						gomega.Expect(output.String()).Should(
							gomega.ContainSubstring("Service CIDR value range is too small for correct provisioning."))
					case "--pod-cidr":
						gomega.Expect(output.String()).Should(
							gomega.ContainSubstring("Pod CIDR value range is too small for correct provisioning"))
					}
					time.Sleep(3 * time.Second) // sleep 3 seconds for next round run

				}
				ginkgo.By("Check the invalid machine CIDR for multi az")
				output, err, _ = clusterService.Create(clusterName,
					"--machine-cidr", "2.0.0.0/25",
					"--multi-az",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("The allowed block size must be between a /16 netmask and /24"))

				ginkgo.By("Check illegal host prefix")
				output, err, _ = clusterService.Create(clusterName,
					"--machine-cidr", "2.0.0.0/25",
					"--host-prefix", "28",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("Subnet length should be between 23 and 26"))

				ginkgo.By("Check invalid host prefix")
				output, err, _ = clusterService.Create(clusterName,
					"--machine-cidr", "2.0.0.0/25",
					"--host-prefix", "invalid",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring(`invalid argument "invalid" for "--host-prefix" flag`))

			})

		ginkgo.It("to validate the invalid proxy when create cluster - [id:45509]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				zone := constants.CommonAWSRegion + "a"
				clusterName := "rosacli-45509"
				ginkgo.By("Create rosa cluster which has proxy without subnets set by command ")
				output, err := clusterService.CreateDryRun(
					"cl-45509",
					"--http-proxy", "http://example.com",
					"--https-proxy", "https://example.com",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(gomega.ContainSubstring(
					"No subnets found in current region that are valid for the chosen CIDR ranges"),
				)

				ginkgo.By("Prepare vpc with subnets")
				vpc, err := vpc_client.PrepareVPC(clusterName, constants.CommonAWSRegion, "", true, "")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				defer vpc.DeleteVPCChain(true)

				subnetMap, err := vpc.PreparePairSubnetByZone(zone)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				privateSubnet := subnetMap["private"].ID
				publicSubnet := subnetMap["public"].ID

				ginkgo.By("Create ccs existing cluster with invalid http_proxy set")
				output, err = clusterService.CreateDryRun(clusterName,
					"--region", constants.CommonAWSRegion,
					"--subnet-ids", strings.Join([]string{
						privateSubnet,
						publicSubnet,
					}, ","),
					"--http-proxy", "invalid",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(gomega.ContainSubstring("Invalid 'proxy.http_proxy' attribute 'invalid'"))

				ginkgo.By("Create ccs existing cluster with invalid http_proxy not started with http")
				output, err = clusterService.CreateDryRun(clusterName,
					"--region", constants.CommonAWSRegion,
					"--subnet-ids", strings.Join([]string{
						privateSubnet,
						publicSubnet,
					}, ","),
					"--http-proxy", "nohttp.prefix.com",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("Invalid 'proxy.http_proxy' attribute 'nohttp.prefix.com'"))

				ginkgo.By("Create ccs existing cluster with invalid https_proxy set")
				output, err = clusterService.CreateDryRun(clusterName,
					"--region", constants.CommonAWSRegion,
					"--subnet-ids", strings.Join([]string{
						privateSubnet,
						publicSubnet,
					}, ","),
					"--https-proxy", "invalid",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring(`ERR: parse "invalid": invalid URI for request`))

				ginkgo.By("Create wide-proxy cluster with invalid additional_trust_bundle set")
				tempDir, err := os.MkdirTemp("", "*")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				defer os.RemoveAll(tempDir)
				tempFile, err := helper.CreateFileWithContent(path.Join(tempDir, "rosacli-45509"), "invalid CA")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				output, err = clusterService.CreateDryRun(clusterName,
					"--region", constants.CommonAWSRegion,
					"--subnet-ids", strings.Join([]string{
						privateSubnet,
						publicSubnet,
					}, ","),
					"--additional-trust-bundle-file", tempFile,
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("ERR: Failed to parse additional trust bundle"))

				ginkgo.By("Create wide-proxy cluster with invalid additional_trust_bundle set path")
				output, err = clusterService.CreateDryRun(clusterName,
					"--region", constants.CommonAWSRegion,
					"--subnet-ids", strings.Join([]string{
						privateSubnet,
						publicSubnet,
					}, ","),
					"--additional-trust-bundle-file", "/not/existing",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("ERR: open /not/existing: no such file or directory"))

				ginkgo.By("Create wide-proxy cluster with only no_proxy set")
				output, err = clusterService.CreateDryRun(clusterName,
					"--region", constants.CommonAWSRegion,
					"--subnet-ids", strings.Join([]string{
						privateSubnet,
						publicSubnet,
					}, ","),
					"--no-proxy", "nohttp.prefix.com",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("ERR: Expected at least one of the following: http-proxy, https-proxy"))

				output, err = clusterService.CreateDryRun(clusterName,
					"--region", constants.CommonAWSRegion,
					"--subnet-ids", strings.Join([]string{
						privateSubnet,
						publicSubnet,
					}, ","),
					"--http-proxy", "http://example.com",
					"--https-proxy", "https://example.com",
					"--no-proxy", "*",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("ERR: expected a valid user no-proxy value"))

				ginkgo.By("Create a cluster with proxy settings but without subnet-ids")
				output, err = clusterService.CreateDryRun(clusterName,
					"--http-proxy", "http://example.com",
					"--https-proxy", "https://example.com",
					"--no-proxy", "example.com",
					"-y",
				)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).ShouldNot(
					gomega.ContainSubstring("The number of subnets for a 'single AZ' 'cluster' should be"))
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("cluster_wide_proxy is only supported if subnetIDs exist"),
				)
			})
	})

var _ = ginkgo.Describe("Classic cluster deletion validation",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient *rosacli.Client
		)

		ginkgo.BeforeEach(func() {
			// Init the client
			rosaClient = rosacli.NewClient()
		})

		ginkgo.It("to validate the ROSA cluster deletion will work via rosacli	- [id:38778]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterService := rosaClient.Cluster
				notExistID := "no-exist-cluster-id"
				ginkgo.By("Delete the cluster without indicated cluster Name or ID")
				cmd := []string{"rosa", "delete", "cluster"}
				out, err := rosaClient.Runner.RunCMD(cmd)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("\"cluster\" not set"))

				ginkgo.By("Delete a non-existed cluster")
				out, err = clusterService.DeleteCluster(notExistID, "-y")
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("There is no cluster with identifier or name"))

				ginkgo.By("Delete with unknown flag --interactive")
				out, err = clusterService.DeleteCluster(notExistID, "-y", "--interactive")
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("unknown flag: --interactive"))
			})
	})

var _ = ginkgo.Describe("Classic cluster creation negative testing",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient               *rosacli.Client
			clusterService           rosacli.ClusterService
			accountRolePrefixToClean string
			ocmResourceService       rosacli.OCMResourceService
		)
		ginkgo.BeforeEach(func() {

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource
		})
		ginkgo.AfterEach(func() {
			ginkgo.By("Delete the resources for testing")
			if accountRolePrefixToClean != "" {
				ginkgo.By("Delete the account-roles")
				rosaClient.Runner.UnsetArgs()
				_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accountRolePrefixToClean,
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
			}
		})

		ginkgo.It("to validate to create the sts cluster with the version not compatible with the role version	- [id:45176]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterService = rosaClient.Cluster
				ocmResourceService := rosaClient.OCMResource

				ginkgo.By("Porepare version for testing")
				var accRoleversion string
				versionService := rosaClient.Version
				versionList, err := versionService.ListAndReflectVersions(rosacli.VersionChannelGroupStable, false)
				gomega.Expect(err).To(gomega.BeNil())
				defaultVersion := versionList.DefaultVersion()
				gomega.Expect(defaultVersion).ToNot(gomega.BeNil())
				lowerVersion, err := versionList.FindNearestBackwardMinorVersion(defaultVersion.Version, 1, true)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(lowerVersion).NotTo(gomega.BeNil())

				_, _, accRoleversion, err = lowerVersion.MajorMinor()
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Create account-roles in low version 4.14")
				accrolePrefix := "testAr45176"
				path := "/a/b/"
				output, err := ocmResourceService.CreateAccountRole("--mode", "auto",
					"--prefix", accrolePrefix,
					"--path", path,
					"--version", accRoleversion,
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
				defer func() {
					ginkgo.By("Delete the account-roles")
					output, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
						"--prefix", accrolePrefix,
						"-y")

					gomega.Expect(err).To(gomega.BeNil())
					textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
					gomega.Expect(textData).To(gomega.ContainSubstring("Successfully deleted"))
				}()
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).To(gomega.ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				gomega.Expect(err).To(gomega.BeNil())
				ar := arl.DigAccountRoles(accrolePrefix, false)

				ginkgo.By("Create cluster with latest version and use the low version account-roles")
				clusterName := "cluster45176"
				operatorRolePrefix := "cluster45176-xvfa"
				out, err, _ := clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", operatorRolePrefix,
					"-y", "--dry-run",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("is not compatible with version"))
				gomega.Expect(out.String()).To(gomega.ContainSubstring("to create compatible roles and try again"))
			})
		ginkgo.It("to validate to create sts cluster with invalid role arn and operator IAM roles prefix - [id:41824]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Create account-roles for testing")
				accountRolePrefixToClean = "testAr41824"
				output, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefixToClean,
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).To(gomega.ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				gomega.Expect(err).To(gomega.BeNil())
				ar := arl.DigAccountRoles(accountRolePrefixToClean, false)

				ginkgo.By("Create cluster with operator roles prefix longer than 32 characters")
				clusterName := "test41824"
				oprPrefixExceed32Chars := "opPrefix45742opPrefix45742opPrefix45742"
				output, err, _ = clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", oprPrefixExceed32Chars,
					"-y",
				)
				gomega.Expect(err).ToNot(gomega.BeNil())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("Expected a prefix with no more than 32 characters"))

				ginkgo.By("Create cluster with operator roles prefix with invalid format")
				oprPrefixInvaliad := "%%%###@@@"
				output, err, _ = clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", oprPrefixInvaliad,
					"-y",
				)
				gomega.Expect(err).ToNot(gomega.BeNil())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("Expected valid operator roles prefix matching"))

				ginkgo.By("Create cluster with account roles with invalid format")
				invalidArn := "invalidaArnFormat"
				output, err, _ = clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", invalidArn,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", clusterName,
					"-y",
				)
				gomega.Expect(err).ToNot(gomega.BeNil())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("Expected a valid Role ARN"))
			})

		ginkgo.It("to validate creating a cluster with invalid subnets - [id:72657]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				clusterService := rosaClient.Cluster
				clusterName := "ocp-72657"

				ginkgo.By("Create cluster with invalid subnets")
				out, err := clusterService.CreateDryRun(
					clusterName, "--subnet-ids", "subnet-xxx",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("The subnet ID 'subnet-xxx' does not exist"))

			})
		ginkgo.It("to validate to create sts cluster with dulicated role arns- [id:74620]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Create account-roles for testing")
				accountRolePrefixToClean = "testAr74620"
				output, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefixToClean,
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).To(gomega.ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				gomega.Expect(err).To(gomega.BeNil())
				ar := arl.DigAccountRoles(accountRolePrefixToClean, false)

				ginkgo.By("Create cluster with operator roles prefix longer than 32 characters")
				clusterName := "test41824"
				output, err, _ = clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.SupportRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", clusterName,
					"-y",
				)
				gomega.Expect(err).ToNot(gomega.BeNil())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("ROSA IAM roles must have unique ARNs"))
			})

		ginkgo.It("to validate creating a cluster with invalid autoscaler - [id:66761]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterService := rosaClient.Cluster
				clusterName := "ocp-66761"

				ginkgo.By("Create cluster with invalid subnets")
				basicFlags := []string{"--enable-autoscaling", "--min-replicas", "3", "--max-replicas", "3"}

				errAndFlagMap := map[string][]string{
					"Error validating log-verbosity: " +
						"Number must be greater or equal " +
						"to zero": {"--autoscaler-log-verbosity", "-2"},

					"Error validating utilization-threshold: Expecting" +
						" a floating-point number between " +
						"0 and 1": {"--autoscaler-scale-down-utilization-threshold", "1.3"},

					"Error validating delay-after-add: " +
						"time: invalid duration \"e\"": {"--autoscaler-scale-down-delay-after-add", "e"},

					"Error validating delay-after-delete: " +
						"time: missing unit in duration \"3.3\"": {"--autoscaler-scale-down-delay-after-delete",
						"3.3"},
					"Error validating min-cores: Number " +
						"must be greater or equal to zero": {"--autoscaler-min-cores", "-5",
						"--autoscaler-max-cores", "0"},

					"Error validating max-cores: Number" +
						" must be greater or equal to zero": {"--autoscaler-min-cores", "0",
						"--autoscaler-max-cores", "-5"},

					"Error validating cores range: max" +
						" value must be greater or equal than min value 100": {"--autoscaler-min-cores", "100",
						"--autoscaler-max-cores", "5"},

					"Error validating max-cores: Should" +
						" provide an integer number between 0 to 2147483647": {"--autoscaler-min-cores", "5",
						"--autoscaler-max-cores", "1152000000000"},

					"Error validating memory range: max value" +
						" must be greater or equal than min value 1000": {"--autoscaler-min-memory", "1000",
						"--autoscaler-max-memory", "100"},

					"Error validating GPU range: max value " +
						"must be greater or equal than min value 15": {
						"--autoscaler-gpu-limit", "nvidia.com/gpu,0,10",
						"--autoscaler-gpu-limit", "amd.com/gpu,15,5"},
				}

				for errMsg, flag := range errAndFlagMap {

					flag = append(flag, basicFlags...)
					out, err := clusterService.CreateDryRun(
						clusterName,
						flag...,
					)
					gomega.Expect(err).NotTo(gomega.BeNil())
					gomega.Expect(err).To(gomega.HaveOccurred())
					textData := rosaClient.Parser.TextData.Input(out).Parse().Tip()
					gomega.Expect(textData).To(gomega.ContainSubstring(errMsg))
				}
			})
	})

var _ = ginkgo.Describe("HCP cluster creation negative testing",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			profilesMap    map[string]*handler.Profile
			profile        *handler.Profile
			command        string
			rosalCommand   config.Command
			clusterHandler handler.ClusterHandler
			err            error
		)
		ginkgo.BeforeEach(func() {

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster

			// Get a random profile
			profilesMap = handler.ParseProfilesByFile(path.Join(ciConfig.Test.YAMLProfilesDir, "rosa-hcp.yaml"))
			profilesNames := make([]string, 0, len(profilesMap))
			for k := range profilesMap {
				profilesNames = append(profilesNames, k)
			}
			profile = profilesMap[profilesNames[helper.RandomInt(len(profilesNames))]]
			profile.NamePrefix = constants.DefaultNamePrefix
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, profile)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Prepare creation command")
			flags, err := clusterHandler.GenerateClusterCreateFlags()
			gomega.Expect(err).To(gomega.BeNil())

			command = "rosa create cluster --cluster-name " + profile.ClusterConfig.Name + " " + strings.Join(flags, " ")
			rosalCommand = config.GenerateCommand(command)
		})

		ginkgo.AfterEach(func() {
			errs := clusterHandler.Destroy()
			gomega.Expect(len(errs)).To(gomega.Equal(0))
		})

		ginkgo.It("create HCP cluster with network type validation can work well via rosa cli - [id:73725]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				clusterName := helper.GenerateRandomName("cluster-73725", 2)
				ginkgo.By("Create HCP cluster with --no-cni and \"--network-type={OVNKubernetes, OpenshiftSDN}\" at the same time")
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--no-cni", "--network-type='{OVNKubernetes,OpenshiftSDN}'", "-y")
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							"ERR: Expected a valid network type. Valid values: [OpenShiftSDN OVNKubernetes]"))

				ginkgo.By("Create HCP cluster with invalid --no-cni value")
				rosalCommand.DeleteFlag("--network-type", true)
				rosalCommand.DeleteFlag("--no-cni", true)
				rosalCommand.AddFlags("--no-cni=ui")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							`Failed to execute root command: invalid argument "ui" for "--no-cni" flag: ` +
								`strconv.ParseBool: parsing "ui": invalid syntax`))

				ginkgo.By("Create HCP cluster with --no-cni and --network-type=OVNKubernetes at the same time")
				rosalCommand.DeleteFlag("--no-cni=ui", false)
				rosalCommand.AddFlags("--no-cni", "--network-type=OVNKubernetes")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).To(
					gomega.ContainSubstring("ERR: --no-cni and --network-type are mutually exclusive parameters"))

				ginkgo.By("Create non-HCP cluster with --no-cni flag")
				output, err = clusterService.CreateDryRun("ocp-73725", "--no-cni")
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).To(
					gomega.ContainSubstring("ERR: Disabling CNI is supported only for Hosted Control Planes"))
			})

		ginkgo.It("to validate creating a hosted cluster with invalid subnets - [id:75916]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-75916"
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				ginkgo.By("Create cluster with invalid subnets")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--subnet-ids", "subnet-xxx", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("The subnet ID 'subnet-xxx' does not exist"))
			})

		ginkgo.It("Create a hosted cluster cluster with invalid volume size [id:66372]",
			labels.Medium,
			labels.Runtime.Day1Negative,
			func() {
				minSize := constants.MinHCPDiskSize
				maxSize := constants.MaxDiskSize
				clusterName := helper.GenerateRandomName("ocp-66372", 2)
				client := rosacli.NewClient()

				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				rosalCommand.ReplaceFlagValue(replacingFlags)

				ginkgo.By("Try a worker disk size that's too small")
				rosalCommand.AddFlags(
					"--dry-run",
					"--worker-disk-size",
					fmt.Sprintf("%dGiB", minSize-1),
					"-y")

				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).NotTo(gomega.BeNil())
				stdout := client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(gomega.ContainSubstring(fmt.Sprintf(constants.DiskSizeErrRangeMsg, minSize-1, minSize, maxSize)))

				ginkgo.By("Try a worker disk size that's a little bigger")
				replacingFlags["--worker-disk-size"] = fmt.Sprintf("%dGiB", maxSize+1)
				rosalCommand.ReplaceFlagValue(replacingFlags)
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(gomega.ContainSubstring(fmt.Sprintf(constants.DiskSizeErrRangeMsg, maxSize+1, minSize, maxSize)))

				ginkgo.By("Try a worker disk size that's very big")
				veryBigData := "34567865467898765789"
				replacingFlags["--worker-disk-size"] = fmt.Sprintf("%sGiB", veryBigData)
				rosalCommand.ReplaceFlagValue(replacingFlags)
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(gomega.ContainSubstring("Expected a valid machine pool root disk size value '%sGiB': "+
						"invalid disk size: '%sGi'. maximum size exceeded",
						veryBigData,
						veryBigData))

				ginkgo.By("Try a worker disk size that's negative")
				replacingFlags["--worker-disk-size"] = "-1GiB"
				rosalCommand.ReplaceFlagValue(replacingFlags)
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(
						gomega.ContainSubstring(
							"Expected a valid machine pool root disk size value '-1GiB': " +
								"invalid disk size: '-1Gi'. positive size required"))

				ginkgo.By("Try a worker disk size that's a string")
				invalidStr := "invalid"
				replacingFlags["--worker-disk-size"] = invalidStr
				rosalCommand.ReplaceFlagValue(replacingFlags)
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				stdout = client.Parser.TextData.Input(out).Parse().Tip()
				gomega.Expect(stdout).
					To(
						gomega.ContainSubstring(
							"Expected a valid machine pool root disk size value '%s': invalid disk size "+
								"format: '%s'. accepted units are Giga or Tera in the form of "+
								"g, G, GB, GiB, Gi, t, T, TB, TiB, Ti",
							invalidStr,
							invalidStr))
			})

		ginkgo.It("to validate creating a hosted cluster with CIDR that doesn't exist - [id:70970]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-70970"
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				ginkgo.By("Create cluster with a CIDR that doesn't exist")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--machine-cidr", "192.168.1.0/23", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(
						gomega.ContainSubstring(
							"ERR: All Hosted Control Plane clusters need a pre-configured VPC. " +
								"Please check: " +
								"https://docs.openshift.com/rosa/rosa_hcp/rosa-hcp-sts-creating-a-cluster-quickly.html#rosa-hcp-creating-vpc"))
			})

		ginkgo.It("to validate create cluster with external_auth_config can work well - [id:73755]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Create non-HCP cluster with --external-auth-providers-enabled")
				clusterName := helper.GenerateRandomName("ocp-73755", 2)
				output, err := clusterService.CreateDryRun(clusterName, "--external-auth-providers-enabled")
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							"ERR: External authentication configuration is only supported for a Hosted Control Plane cluster."))

				ginkgo.By("Create HCP cluster with --external-auth-providers-enabled and cluster version lower than 4.15")
				cg := rosalCommand.GetFlagValue("--channel-group", true)
				if cg == "" {
					cg = rosacli.VersionChannelGroupStable
				}
				versionList, err := rosaClient.Version.ListAndReflectVersions(cg, rosalCommand.CheckFlagExist("--hosted-cp"))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(versionList).ToNot(gomega.BeNil())
				previousVersionsList, err := versionList.FindNearestBackwardMinorVersion("4.14", 0, true)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				foundVersion := previousVersionsList.Version
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
					"--version":       foundVersion,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				if !rosalCommand.CheckFlagExist("--external-auth-providers-enabled") {
					rosalCommand.AddFlags("--dry-run", "--external-auth-providers-enabled", "-y")
				} else {
					rosalCommand.AddFlags("--dry-run", "-y")
				}
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							"External authentication is only supported in version '4.15.9' or greater, current cluster version is '%s'",
							foundVersion))
			})

		ginkgo.It("to validate '--ec2-metadata-http-tokens' flag during creating cluster - [id:64078]",
			labels.Medium,
			labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-64078"

				ginkgo.By("Create classic cluster with invalid httpTokens")
				errorOutput, err := clusterService.CreateDryRun(
					clusterName, "--ec2-metadata-http-tokens=invalid",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(errorOutput.String()).
					To(
						gomega.ContainSubstring(
							"ERR: Expected a valid http tokens value : " +
								"ec2-metadata-http-tokens value should be one of 'required', 'optional'"))

				ginkgo.By("Create HCP cluster  with invalid httpTokens")
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--ec2-metadata-http-tokens=invalid", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).
					To(
						gomega.ContainSubstring(
							"ERR: Expected a valid http tokens value : " +
								"ec2-metadata-http-tokens value should be one of 'required', 'optional'"))
			})

		ginkgo.It("expose additional allowed principals for HCP negative - [id:74433]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Create hcp cluster using --additional-allowed-principals and invalid formatted arn")
				clusterName := "ocp-74408"
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}

				ginkgo.By("Create cluster with invalid additional allowed principals")
				rosalCommand.ReplaceFlagValue(replacingFlags)
				if rosalCommand.CheckFlagExist("--additional-allowed-principals") {
					rosalCommand.DeleteFlag("--additional-allowed-principals", true)
				}
				rosalCommand.AddFlags("--dry-run", "--additional-allowed-principals", "zzzz", "-y")
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(out.String()).
					To(
						gomega.ContainSubstring(
							"ERR: Expected valid ARNs for additional allowed principals list: Invalid ARN: arn: invalid prefix"))

				ginkgo.By("Create classic cluster with additional allowed principals")
				output, err := clusterService.CreateDryRun(clusterName,
					"--additional-allowed-principals", "zzzz",
					"-y", "--debug")
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(rosaClient.Parser.TextData.Input(output).Parse().Tip()).
					To(
						gomega.ContainSubstring(
							"ERR: Additional Allowed Principals is supported only for Hosted Control Planes"))
			})

		ginkgo.It("Updating default ingress settings is not supported for HCP clusters - [id:71174]",
			labels.Low, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Create hcp cluster using non-default ingress settings")
				clusterName := helper.GenerateRandomName("c71174", 2)
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "--default-ingress-route-selector", "10.0.0.1", "-y")
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).To(gomega.ContainSubstring(
					"Updating default ingress settings is not supported for Hosted Control Plane clusters"))
			})

		ginkgo.It("to validate create cluster with audit log forwarding - [id:73672]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Create non-HCP cluster with --audit-log-arn")
				clusterName := helper.GenerateRandomName("ocp-73672", 2)
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "-y")
				log.Logger.Debug(profile.Name)
				log.Logger.Debug(strings.Split(rosalCommand.GetFullCommand(), " "))

				ginkgo.By("Create classic cluster with  audit log arn")
				if rosalCommand.CheckFlagExist("--audit-log-arn") {
					rosalCommand.DeleteFlag("--audit-log-arn", true)
				}

				output, err := clusterService.CreateDryRun(clusterName, "--audit-log-arn", "-y")
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							"ERR: Audit log forwarding to AWS CloudWatch is only supported for Hosted Control Plane clusters"))

				ginkgo.By("Create HCP cluster with incorrect format audit log arn")
				rosalCommand.AddFlags("--audit-log-arn", "qwertyugf234543234")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).
					To(
						gomega.ContainSubstring(
							"ERR: Expected a valid value for audit log arn matching ^arn:aws"))
			})

		ginkgo.It("to validate role's managed policy when creating hcp cluster - [id:59547]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Create managed account-roles and make sure some ones are not attached the managed policies.")
				clusterService = rosaClient.Cluster
				ocmResourceService := rosaClient.OCMResource
				var arbitraryPolicyService rosacli.PolicyService
				accountRolePrefix := "test-59547"
				_, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefix,
					"--hosted-cp",
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
				defer func() {
					if accountRolePrefix != "" {
						ginkgo.By("Delete the account-roles")
						rosaClient.Runner.UnsetArgs()
						_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
							"--hosted-cp",
							"--prefix", accountRolePrefix,
							"-y")
						gomega.Expect(err).To(gomega.BeNil())
					}
				}()

				arl, _, err := ocmResourceService.ListAccountRole()
				gomega.Expect(err).To(gomega.BeNil())
				ar := arl.DigAccountRoles(accountRolePrefix, true)

				ginkgo.By("Create cluster with the account roles ")
				clusterName := helper.GenerateRandomName("ocp-59547", 2)
				replacingFlags := map[string]string{
					"-c":                 clusterName,
					"--cluster-name":     clusterName,
					"--domain-prefix":    clusterName,
					"--role-arn":         ar.InstallerRole,
					"--support-role-arn": ar.SupportRole,
					"--worker-iam-role":  ar.WorkerRole,
				}
				var accountRoles = make(map[string]string)
				arnPrefix := "arn:aws:iam::aws:policy/service-role"
				for _, r := range arl.AccountRoles(accountRolePrefix) {
					switch r.RoleType {
					case "Installer":
						accountRoles[r.RoleName] = fmt.Sprintf("%s/ROSAInstallerPolicy",
							arnPrefix)
					case "Support":
						accountRoles[r.RoleName] = fmt.Sprintf("%s/ROSASRESupportPolicy",
							arnPrefix)
					case "Worker": // nolint:goconst
						accountRoles[r.RoleName] = fmt.Sprintf("%s/ROSAWorkerInstancePolicy",
							arnPrefix)
					}
				}

				arbitraryPolicyService = rosaClient.Policy
				for r, p := range accountRoles {
					_, err := arbitraryPolicyService.DetachPolicy(r, []string{p}, "--mode", "auto")
					gomega.Expect(err).To(gomega.BeNil())
					ginkgo.By("Create cluster with the account roles")
					rosalCommand.ReplaceFlagValue(replacingFlags)

					out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
					gomega.Expect(err).To(gomega.HaveOccurred())
					gomega.Expect(out.String()).
						To(
							gomega.ContainSubstring(
								fmt.Sprintf("Failed while validating account roles: role"+
									" '%s' is missing the attached managed policy '%s'", r, p)))

					ginkgo.By("Attach the deleted managed policies")
					_, err = arbitraryPolicyService.AttachPolicy(r, []string{p}, "--mode", "auto")
					gomega.Expect(err).To(gomega.BeNil())
				}
			})

		ginkgo.It("to validate hcp creation with registry config via rosacli - [id:76396]",
			labels.Medium, labels.Runtime.Day1Negative,
			func() {
				ginkgo.By("Create non-HCP cluster with registry config")
				clusterName := helper.GenerateRandomName("ocp-76396", 2)
				registryFlags := []string{
					"--registry-config-allowed-registries",
					"--registry-config-blocked-registries",
					"--registry-config-insecure-registries",
					"--registry-config-allowed-registries-for-import",
					"--registry-config-additional-trusted-ca",
				}
				for _, flag := range registryFlags {
					if rosalCommand.CheckFlagExist(flag) {
						rosalCommand.DeleteFlag(flag, true)
					}
					output, err := clusterService.CreateDryRun(clusterName, flag, "-y")
					gomega.Expect(err).To(gomega.HaveOccurred())
					gomega.Expect(output.String()).
						To(
							gomega.ContainSubstring(
								"ERR: Setting the registry config is only supported for hosted clusters"))
				}

				ginkgo.By("create hcp with invalid value for --registry-config-allowed-registries-for-import flag")
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				rosalCommand.AddFlags("--dry-run", "-y")
				log.Logger.Debug(profile.Name)
				log.Logger.Debug(strings.Split(rosalCommand.GetFullCommand(), " "))

				rosalCommand.AddFlags("--registry-config-allowed-registries-for-import", "test.com:invalid")
				output, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				log.Logger.Info(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).To(
					gomega.ContainSubstring("ERR: Expected valid allowed registries for import values"))

				ginkgo.By("create hcp with --registry-config-blocked-registries and " +
					"--registry-config-allowed-registries at same time")
				rosalCommand.DeleteFlag("--registry-config-allowed-registries-for-import", true)
				rosalCommand.AddFlags("--registry-config-allowed-registries", "test.com",
					"--registry-config-blocked-registries", "test.blocked.com")
				output, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(output.String()).To(gomega.ContainSubstring(
					"ERR: Allowed registries and blocked registries are mutually exclusive fields"))
			})
	})
var _ = ginkgo.Describe("HCP cluster creation subnets validation",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService

			customProfile      *handler.Profile
			clusterID          string
			ocmResourceService rosacli.OCMResourceService
			testingClusterName string
			clusterHandler     handler.ClusterHandler
			rosalCommand       config.Command
			err                error
			command            string
		)
		ginkgo.BeforeEach(func() {
			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource

			ginkgo.By("Prepare custom profile")
			customProfile = &handler.Profile{
				ClusterConfig: &handler.ClusterConfig{
					HCP:                   true,
					MultiAZ:               true,
					STS:                   true,
					OIDCConfig:            "managed",
					NetworkingSet:         true,
					BYOVPC:                true,
					Zones:                 "",
					Autoscale:             false,
					PrivateLink:           false,
					DefaultIngressPrivate: false,
				},
				AccountRoleConfig: &handler.AccountRoleConfig{
					Path:               "",
					PermissionBoundary: "",
				},
				Version:      "latest",
				ChannelGroup: "stable",
				Region:       constants.CommonAWSRegion,
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, customProfile)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Init the cluster id and testing cluster name")
			ginkgo.By("Prepare creation command")
			flags, err := clusterHandler.GenerateClusterCreateFlags()
			gomega.Expect(err).To(gomega.BeNil())

			command = "rosa create cluster --cluster-name " + customProfile.ClusterConfig.Name + " " + strings.Join(flags, " ")
			rosalCommand = config.GenerateCommand(command)
		})

		ginkgo.AfterEach(func() {
			defer func() {
				ginkgo.By("Clean resources")
				clusterHandler.Destroy()
			}()

			if clusterID != "" {
				ginkgo.By("Delete cluster by id")
				rosaClient.Runner.UnsetArgs()
				_, err := clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Delete operator-roles")
				_, err = ocmResourceService.DeleteOperatorRoles(
					"-c", clusterID,
					"--mode", "auto",
					"-y",
				)
				gomega.Expect(err).To(gomega.BeNil())
			} else if testingClusterName != "" {
				// At least try to delete testing cluster
				ginkgo.By("Delete cluster by name")
				rosaClient.Runner.UnsetArgs()
				_, err := clusterService.DeleteCluster(testingClusterName, "-y")
				gomega.Expect(err).To(gomega.BeNil())
			}
		})
		ginkgo.It("HCP cluster creation subnets validation - [id:72538]",
			labels.High, labels.Runtime.Day1Negative,
			func() {
				clusterName := "ocp-72538"
				vpcName := "vpc-72538"
				replacingFlags := map[string]string{
					"-c":              clusterName,
					"--cluster-name":  clusterName,
					"--domain-prefix": clusterName,
					"--region":        constants.CommonAWSRegion,
				}
				rosalCommand.ReplaceFlagValue(replacingFlags)
				if rosalCommand.CheckFlagExist("--subnet-ids") {
					_ = rosalCommand.DeleteFlag("--subnet-ids", true)
				}

				ginkgo.By("Prepare a vpc for the testing")
				resourcesHandler := clusterHandler.GetResourcesHandler()
				vpc, err := resourcesHandler.PrepareVPC(vpcName, "", false, false)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				// defer vpc.DeleteVPCChain()

				subnetMap, err := resourcesHandler.PrepareSubnets([]string{}, true)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				availabilityZone := vpc.SubnetList[0].Zone
				additionalPrivateSubnet, err := vpc.CreatePrivateSubnet(availabilityZone, true)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				additionalPublicSubnet, err := vpc.CreatePublicSubnet(availabilityZone)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				successOutput := "Creating cluster '" + clusterName +
					"' should succeed. Run without the '--dry-run' flag to create the cluster"
				failOutput_1 := "Creating cluster '" + clusterName + "' should fail: "
				failOutput_2 := "Availability zone " + availabilityZone +
					" has more than one private subnet. Check the subnets and try again"

				ginkgo.By("Create a public cluster with 1 private subnet and 1 public subnet")
				subnets := []string{subnetMap["private"][0], subnetMap["public"][0]}
				rosalCommand.AddFlags("--dry-run", "--subnet-ids", strings.Join(subnets, ","))
				out, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(successOutput))

				ginkgo.By("Create a public cluster with 3 private subnets and 1 public subnet")
				subnets = []string{
					subnetMap["private"][0],
					subnetMap["private"][1],
					subnetMap["private"][2],
					subnetMap["public"][0],
				}
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": strings.Join(subnets, ",")})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(successOutput))

				ginkgo.By("Create a public cluster with 2 private subnets from same AZ and 1 public subnet")
				subnets = []string{subnetMap["private"][0], additionalPrivateSubnet.ID, subnetMap["public"][0]}
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": strings.Join(subnets, ",")})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(failOutput_1))
				gomega.Expect(out.String()).To(gomega.ContainSubstring(failOutput_2))

				ginkgo.By("Create a public cluster with 4 private subnets (2 subnets from same AZ) and 1 public subnet")
				subnets = []string{
					subnetMap["private"][0],
					additionalPrivateSubnet.ID,
					subnetMap["private"][1],
					subnetMap["private"][2],
					subnetMap["public"][0],
				}
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": strings.Join(subnets, ",")})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(failOutput_1))
				gomega.Expect(out.String()).To(gomega.ContainSubstring(failOutput_2))

				ginkgo.By("Create a public cluster with 1 private subnet and 2 public subnet from same AZ")
				subnets = []string{subnetMap["private"][0], subnetMap["public"][0], additionalPublicSubnet.ID}
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": strings.Join(subnets, ",")})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(successOutput))

				ginkgo.By("Create a private cluster with 1 private subnet")
				rosalCommand.AddFlags("--private")
				rosalCommand.AddFlags("--default-ingress-private")
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": subnetMap["private"][0]})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(successOutput))

				ginkgo.By("Create a private cluster with 3 private subnets")
				subnets = []string{subnetMap["private"][0], subnetMap["private"][1], subnetMap["private"][2]}
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": strings.Join(subnets, ",")})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(successOutput))

				ginkgo.By("Create a private cluster with 2 private subnets from same AZ")
				subnets = []string{subnetMap["private"][0], additionalPrivateSubnet.ID}
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": strings.Join(subnets, ",")})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(failOutput_1))
				gomega.Expect(out.String()).To(gomega.ContainSubstring(failOutput_2))

				ginkgo.By("Create a private cluster with 4 private subnets (2 subnets from same AZ)")
				subnets = []string{
					subnetMap["private"][0],
					additionalPrivateSubnet.ID,
					subnetMap["private"][1],
					subnetMap["private"][2],
				}
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": strings.Join(subnets, ",")})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(failOutput_1))
				gomega.Expect(out.String()).To(gomega.ContainSubstring(failOutput_2))

				ginkgo.By("Create a private cluster with 1 private subnet and 1 public subnet")
				subnets = []string{subnetMap["private"][0], subnetMap["public"][0]}
				rosalCommand.ReplaceFlagValue(map[string]string{"--subnet-ids": strings.Join(subnets, ",")})
				out, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(out.String()).To(gomega.ContainSubstring(
					"The following subnets have been excluded because they have an " +
						"Internet Gateway Targetded Route and the Cluster choice is private: [" + subnetMap["public"][0] + "]"))
				gomega.Expect(out.String()).To(gomega.ContainSubstring(
					"Cluster is set as private, cannot use public '%s'", subnetMap["public"][0]))
			})
	})

var _ = ginkgo.Describe("Create cluster with availability zones testing",
	labels.Feature.Machinepool,
	func() {
		defer ginkgo.GinkgoRecover()
		var (
			availabilityZones  string
			clusterID          string
			rosaClient         *rosacli.Client
			machinePoolService rosacli.MachinePoolService
		)

		ginkgo.BeforeEach(func() {
			ginkgo.By("Get the cluster")
			clusterID = config.GetClusterID()
			gomega.Expect(clusterID).ToNot(gomega.Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			machinePoolService = rosaClient.MachinePool

			ginkgo.By("Skip testing if the cluster is not a Classic cluster")
			isHostedCP, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			if isHostedCP {
				SkipNotClassic()
			}
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Clean remaining resources")
			rosaClient.CleanResources(clusterID)

		})

		ginkgo.It("User can set availability zones - [id:52691]",
			labels.Critical, labels.Runtime.Day1Post, labels.FedRAMP,
			func() {
				profile := handler.LoadProfileYamlFileByENV()
				mpID := "mp-52691"
				machineType := "m5.2xlarge" // nolint:goconst

				if profile.ClusterConfig.BYOVPC || profile.ClusterConfig.Zones == "" {
					SkipTestOnFeature("create rosa cluster with availability zones")
				}

				ginkgo.By("List machine pool and check the default one")
				availabilityZones = profile.ClusterConfig.Zones
				output, err := machinePoolService.ListMachinePool(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				mpList, err := machinePoolService.ReflectMachinePoolList(output)
				gomega.Expect(err).To(gomega.BeNil())
				mp := mpList.Machinepool(constants.DefaultClassicWorkerPool)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(helper.ReplaceCommaSpaceWithComma(mp.AvalaiblityZones)).To(gomega.Equal(availabilityZones))

				ginkgo.By("Create another machinepool")
				_, err = machinePoolService.CreateMachinePool(clusterID, mpID,
					"--replicas", "3",
					"--instance-type", machineType,
				)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				ginkgo.By("List machine pool and check availability zone")
				output, err = machinePoolService.ListMachinePool(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				mpList, err = machinePoolService.ReflectMachinePoolList(output)
				gomega.Expect(err).To(gomega.BeNil())
				mp = mpList.Machinepool(mpID)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(helper.ReplaceCommaSpaceWithComma(mp.AvalaiblityZones)).To(gomega.Equal(availabilityZones))
			})
	})
var _ = ginkgo.Describe("Create sts and hcp cluster with the IAM roles with path setting",
	labels.Feature.Cluster, func() {
		defer ginkgo.GinkgoRecover()
		var (
			clusterID      string
			rosaClient     *rosacli.Client
			profile        *handler.Profile
			err            error
			clusterService rosacli.ClusterService
			path           string
			awsClient      *aws_client.AWSClient
		)

		ginkgo.BeforeEach(func() {
			ginkgo.By("Get the cluster")
			profile = handler.LoadProfileYamlFileByENV()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			clusterID = config.GetClusterID()
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Clean remaining resources")
			rosaClient.CleanResources(clusterID)

		})

		ginkgo.It("to check the IAM roles can be used to create clsuters - [id:53570]",
			labels.Critical, labels.Runtime.Day1Post, labels.FedRAMP,
			func() {
				ginkgo.By("Skip testing if the cluster is a Classic NON-STS cluster")
				isSTS, err := clusterService.IsSTSCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				if !isSTS {
					ginkgo.Skip("Skip this case as it only supports on STS clusters")
				}

				ginkgo.By("Check the account-roles using on the cluster has path setting")
				if profile.AccountRoleConfig.Path == "" {
					ginkgo.Skip("Skip this case as it only checks the cluster which has the account-roles with path setting")
				} else {
					path = profile.AccountRoleConfig.Path
				}

				ginkgo.By("Get operator-roles arns and installer role arn")
				output, err := clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())
				operatorRolesArns := CD.OperatorIAMRoles

				installerRole := CD.STSRoleArn
				gomega.Expect(installerRole).To(gomega.ContainSubstring(path))

				ginkgo.By("Check the operator-roles has the path setting")
				for _, pArn := range operatorRolesArns {
					gomega.Expect(pArn).To(gomega.ContainSubstring(path))
				}
				if profile.ClusterConfig.STS && !profile.ClusterConfig.HCP {
					ginkgo.By("Check the operator role policies has the path setting")
					awsClient, err = aws_client.CreateAWSClient("", "")
					gomega.Expect(err).To(gomega.BeNil())
					for _, pArn := range operatorRolesArns {
						_, roleName, err := helper.ParseRoleARN(pArn)
						gomega.Expect(err).To(gomega.BeNil())
						attachedPolicy, err := awsClient.ListRoleAttachedPolicies(roleName)
						gomega.Expect(err).To(gomega.BeNil())
						gomega.Expect(*(attachedPolicy[0].PolicyArn)).To(gomega.ContainSubstring(path))
					}
				}
			})
	})

var _ = ginkgo.Describe("Create cluster with existing operator-roles prefix which roles are not using byo oidc",
	labels.Feature.Cluster, func() {
		defer ginkgo.GinkgoRecover()
		var (
			// clusterID  string
			rosaClient         *rosacli.Client
			err                error
			accountRolePrefix  string
			ocmResourceService rosacli.OCMResourceService
			clusterNameToClean string
			clusterService     rosacli.ClusterService
			clusterID          string
		)

		ginkgo.BeforeEach(func() {
			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
			clusterService = rosaClient.Cluster
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Delete the cluster")
			if clusterNameToClean != "" {
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())

				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(clusterNameToClean).ID

				rosaClient.Runner.UnsetArgs()
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
				gomega.Expect(err).To(gomega.BeNil())
			}
			ginkgo.By("Delete operator-roles")
			_, err = ocmResourceService.DeleteOperatorRoles(
				"-c", clusterID,
				"--mode", "auto",
				"-y",
			)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Delete oidc-provider")
			_, err = ocmResourceService.DeleteOIDCProvider(
				"-c", clusterID,
				"--mode", "auto",
				"-y")
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Delete account-roles")
			if accountRolePrefix != "" {
				ginkgo.By("Delete the account-roles")
				rosaClient.Runner.UnsetArgs()
				_, err := ocmResourceService.DeleteAccountRole("--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
			}

		})

		ginkgo.It("to validate to create cluster with existing operator roles prefix - [id:45742]",
			labels.Medium, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Create acount-roles")
				accountRolePrefix = helper.GenerateRandomName("ar45742", 2)
				output, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).To(gomega.ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				gomega.Expect(err).To(gomega.BeNil())
				ar := arl.DigAccountRoles(accountRolePrefix, false)

				ginkgo.By("Create one sts cluster")
				clusterNameToClean = "test-45742"
				operatorRolePreifx := "opPrefix45742"
				_, err, _ = clusterService.Create(
					clusterNameToClean, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", operatorRolePreifx,
					"-y",
				)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Create another cluster with the same operator-roles-prefix")
				clusterName := "test-45742b"
				out, err, _ := clusterService.Create(
					clusterName, "--sts",
					"--mode", "auto",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", operatorRolePreifx,
					"-y",
				)
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(out.String()).To(gomega.ContainSubstring("already exists"))
				gomega.Expect(out.String()).To(gomega.ContainSubstring("provide a different prefix"))
			})
	})

var _ = ginkgo.Describe("create/delete operator-roles and oidc-provider to cluster",
	labels.Feature.Cluster, func() {
		defer ginkgo.GinkgoRecover()
		var (
			rosaClient *rosacli.Client

			accountRolePrefix  string
			ocmResourceService rosacli.OCMResourceService
			clusterNameToClean string
			clusterService     rosacli.ClusterService
			clusterID          string
			defaultDir         string
			dirToClean         string
		)

		ginkgo.BeforeEach(func() {
			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			ocmResourceService = rosaClient.OCMResource
			clusterService = rosaClient.Cluster

			ginkgo.By("Get the default dir")
			defaultDir = rosaClient.Runner.GetDir()
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Go back original by setting runner dir")
			rosaClient.Runner.SetDir(defaultDir)

			ginkgo.By("Delete cluster")
			rosaClient.Runner.UnsetArgs()
			clusterListout, err := clusterService.List()
			gomega.Expect(err).To(gomega.BeNil())
			clusterList, err := clusterService.ReflectClusterList(clusterListout)
			gomega.Expect(err).To(gomega.BeNil())

			if clusterList.IsExist(clusterID) {
				rosaClient.Runner.UnsetArgs()
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.BeNil())
			}
			ginkgo.By("Delete operator-roles")
			_, err = ocmResourceService.DeleteOperatorRoles(
				"-c", clusterID,
				"--mode", "auto",
				"-y",
			)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Delete oidc-provider")
			_, err = ocmResourceService.DeleteOIDCProvider(
				"-c", clusterID,
				"--mode", "auto",
				"-y")
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Delete the account-roles")
			rosaClient.Runner.UnsetArgs()
			_, err = ocmResourceService.DeleteAccountRole("--mode", "auto",
				"--prefix", accountRolePrefix,
				"-y")
			gomega.Expect(err).To(gomega.BeNil())
		})

		ginkgo.It("to create/delete operator-roles and oidc-provider to cluster in manual mode - [id:43053]",
			labels.Critical, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Create acount-roles")
				accountRolePrefix = helper.GenerateRandomName("ar43053", 2)
				output, err := ocmResourceService.CreateAccountRole(
					"--mode", "auto",
					"--prefix", accountRolePrefix,
					"-y")
				gomega.Expect(err).To(gomega.BeNil())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				gomega.Expect(textData).To(gomega.ContainSubstring("Created role"))

				arl, _, err := ocmResourceService.ListAccountRole()
				gomega.Expect(err).To(gomega.BeNil())
				ar := arl.DigAccountRoles(accountRolePrefix, false)

				ginkgo.By("Create a temp dir to execute the create commands")
				dirToClean, err = os.MkdirTemp("", "*")
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Create one sts cluster in manual mode")
				rosaClient.Runner.SetDir(dirToClean)
				clusterNameToClean = helper.GenerateRandomName("c43053", 2)
				// Configure with a random str, which can solve the rerun failure
				operatorRolePreifx := helper.GenerateRandomName("opPrefix43053", 2)
				_, err, _ = clusterService.Create(
					clusterNameToClean, "--sts",
					"--mode", "manual",
					"--role-arn", ar.InstallerRole,
					"--support-role-arn", ar.SupportRole,
					"--controlplane-iam-role", ar.ControlPlaneRole,
					"--worker-iam-role", ar.WorkerRole,
					"--operator-roles-prefix", operatorRolePreifx,
					"-y",
				)
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(clusterNameToClean).ID

				ginkgo.By("Create operator-roles in manual mode")
				output, err = ocmResourceService.CreateOperatorRoles(
					"-c", clusterID,
					"--mode", "manual",
					"-y",
				)
				gomega.Expect(err).To(gomega.BeNil())
				commands := helper.ExtractCommandsToCreateAWSResources(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					gomega.Expect(err).To(gomega.BeNil())
				}

				ginkgo.By("Create oidc provider in manual mode")
				output, err = ocmResourceService.CreateOIDCProvider(
					"-c", clusterID,
					"--mode", "manual",
					"-y",
				)
				gomega.Expect(err).To(gomega.BeNil())
				commands = helper.ExtractCommandsToCreateAWSResources(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					gomega.Expect(err).To(gomega.BeNil())
				}

				ginkgo.By("Check cluster status to installing")
				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterStatus(clusterID, "installing", 3, 24)
				gomega.Expect(err).To(gomega.BeNil(), "It met error or timeout when waiting cluster to installing status")

				ginkgo.By("Delete cluster and wait it deleted")
				rosaClient.Runner.UnsetArgs()
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterDeleted(clusterID, 3, 24)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Delete operator-roles in manual mode")
				output, err = ocmResourceService.DeleteOperatorRoles(
					"-c", clusterID,
					"--mode", "manual",
					"-y",
				)
				gomega.Expect(err).To(gomega.BeNil())
				commands = helper.ExtractCommandsToDeleteAWSResoueces(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					gomega.Expect(err).To(gomega.BeNil())
				}

				ginkgo.By("Delete oidc provider in manual mode")
				output, err = ocmResourceService.DeleteOIDCProvider(
					"-c", clusterID,
					"--mode", "manual",
					"-y",
				)
				gomega.Expect(err).To(gomega.BeNil())
				commands = helper.ExtractCommandsToDeleteAWSResoueces(output)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					gomega.Expect(err).To(gomega.BeNil())
				}
			})
	})
var _ = ginkgo.Describe("Reusing opeartor prefix and oidc config to create clsuter", labels.Feature.Cluster, func() {
	defer ginkgo.GinkgoRecover()
	var (
		rosaClient               *rosacli.Client
		profile                  *handler.Profile
		err                      error
		oidcConfigToClean        string
		ocmResourceService       rosacli.OCMResourceService
		originalMajorMinorVerson string
		clusterService           rosacli.ClusterService
		awsClient                *aws_client.AWSClient
		operatorPolicyArn        string
		clusterID                string
	)
	const versionTagName = "rosa_openshift_version"

	ginkgo.BeforeEach(func() {
		ginkgo.By("Init the client")
		rosaClient = rosacli.NewClient()
		ocmResourceService = rosaClient.OCMResource
		clusterService = rosaClient.Cluster
		profile = handler.LoadProfileYamlFileByENV()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		awsClient, err = aws_client.CreateAWSClient("", "")
		gomega.Expect(err).To(gomega.BeNil())

		ginkgo.By("Get the cluster")
		clusterID = config.GetClusterID()
		gomega.Expect(clusterID).ToNot(gomega.Equal(""), "ClusterID is required. Please export CLUSTER_ID")
	})

	ginkgo.AfterEach(func() {
		hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		if !hostedCluster {
			ginkgo.By("Recover the operator role policy version")
			keysToUntag := []string{versionTagName}
			err = awsClient.UntagPolicy(operatorPolicyArn, keysToUntag)
			gomega.Expect(err).To(gomega.BeNil())
			tags := map[string]string{versionTagName: originalMajorMinorVerson}
			err = awsClient.TagPolicy(operatorPolicyArn, tags)
			gomega.Expect(err).To(gomega.BeNil())
		}

		ginkgo.By("Delete resources for testing")
		if oidcConfigToClean != "" {
			output, err := ocmResourceService.DeleteOIDCConfig(
				"--oidc-config-id", oidcConfigToClean,
				"--region", profile.Region,
				"--mode", "auto",
				"-y",
			)
			gomega.Expect(err).To(gomega.BeNil())
			textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
			gomega.Expect(textData).To(gomega.ContainSubstring("Successfully deleted the OIDC provider"))
		}

	})

	ginkgo.It("to reuse operator-roles prefix and oidc config - [id:60688]",
		labels.Critical, labels.Runtime.Day2,
		func() {
			ginkgo.By("Check if it is using oidc config")
			if profile.ClusterConfig.OIDCConfig == "" {
				ginkgo.Skip("Skip this case as it is only for byo oidc cluster")
			}

			ginkgo.By("Skip if the cluster is shared vpc cluster")
			if profile.ClusterConfig.SharedVPC {
				ginkgo.Skip("Skip this case as it is not supported for byo oidc cluster")
			}

			ginkgo.By("Prepare creation command")
			var originalOidcConfigID string
			var rosalCommand config.Command

			sharedDIR := os.Getenv("SHARED_DIR")
			filePath := sharedDIR + "/create_cluster.sh"
			rosalCommand, err = config.RetrieveClusterCreationCommand(filePath)
			gomega.Expect(err).To(gomega.BeNil())

			originalOidcConfigID = rosalCommand.GetFlagValue("--oidc-config-id", true)
			rosalCommand.AddFlags("--dry-run")
			testClusterName := "cluster60688"
			rosalCommand.ReplaceFlagValue(map[string]string{
				"-c": testClusterName,
			})
			if profile.ClusterConfig.DomainPrefixEnabled {
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--domain-prefix": "dp60688",
				})
			}

			ginkgo.By("Reuse the oidc config and operator-roles")
			stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(stdout.String()).To(gomega.ContainSubstring("Creating cluster '%s' should succeed", testClusterName))

			ginkgo.By("Reuse the operator prefix to create cluster but using different oidc config")
			output, err := ocmResourceService.CreateOIDCConfig("--mode", "auto", "-y")
			gomega.Expect(err).To(gomega.BeNil())
			oidcPrivodeARNFromOutputMessage := helper.ExtractOIDCProviderARN(output.String())
			oidcPrivodeIDFromOutputMessage := helper.ExtractOIDCProviderIDFromARN(oidcPrivodeARNFromOutputMessage)
			oidcConfigToClean, err = ocmResourceService.GetOIDCIdFromList(oidcPrivodeIDFromOutputMessage)
			gomega.Expect(err).To(gomega.BeNil())

			rosalCommand.ReplaceFlagValue(map[string]string{
				"--oidc-config-id": oidcConfigToClean,
			})
			stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
			gomega.Expect(err).NotTo(gomega.BeNil())
			gomega.Expect(stdout.String()).To(gomega.ContainSubstring("does not have trusted relationship to"))

			ginkgo.By("Find the nearest backward minor version")
			output, err = clusterService.DescribeCluster(clusterID)
			gomega.Expect(err).To(gomega.BeNil())
			clusterDetail, err := clusterService.ReflectClusterDescription(output)
			gomega.Expect(err).To(gomega.BeNil())
			operatorRolesArns := clusterDetail.OperatorIAMRoles

			versionOutput, err := clusterService.GetClusterVersion(clusterID)
			gomega.Expect(err).To(gomega.BeNil())
			clusterVersion := versionOutput.RawID
			major, minor, _, err := helper.ParseVersion(clusterVersion)
			gomega.Expect(err).To(gomega.BeNil())
			originalMajorMinorVerson = fmt.Sprintf("%d.%d", major, minor)
			testingRoleVersion := fmt.Sprintf("%d.%d", major, minor-1)

			isHosted, err := clusterService.IsHostedCPCluster(clusterID)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			if !isHosted {
				ginkgo.By("Update the all operator policies tags to low version")
				_, roleName, err := helper.ParseRoleARN(operatorRolesArns[1])
				gomega.Expect(err).To(gomega.BeNil())
				policies, err := awsClient.ListAttachedRolePolicies(roleName)
				gomega.Expect(err).To(gomega.BeNil())
				operatorPolicyArn = *policies[0].PolicyArn

				keysToUntag := []string{versionTagName}
				err = awsClient.UntagPolicy(operatorPolicyArn, keysToUntag)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Expected no error, but got: %v", err))

				tags := map[string]string{versionTagName: testingRoleVersion}

				err = awsClient.TagPolicy(operatorPolicyArn, tags)
				gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Expected no error, but got: %v", err))

				ginkgo.By("Reuse operatot-role prefix and oidc config to create cluster with non-compatible version")

				rosalCommand.ReplaceFlagValue(map[string]string{
					"--oidc-config-id": originalOidcConfigID,
				})
				stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(stdout.String()).To(gomega.ContainSubstring("is not compatible with cluster version"))
			}
		})
})
var _ = ginkgo.Describe("Sts cluster creation with external id",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService

			customProfile      *handler.Profile
			clusterID          string
			ocmResourceService rosacli.OCMResourceService
			testingClusterName string
			clusterHandler     handler.ClusterHandler
		)
		ginkgo.BeforeEach(func() {
			var err error

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource

			ginkgo.By("Get AWS account id")
			rosaClient.Runner.JsonFormat()
			rosaClient.Runner.UnsetFormat()

			ginkgo.By("Prepare custom profile")
			customProfile = &handler.Profile{
				ClusterConfig: &handler.ClusterConfig{
					HCP:           false,
					MultiAZ:       false,
					STS:           true,
					OIDCConfig:    "",
					NetworkingSet: false,
					BYOVPC:        false,
				},
				AccountRoleConfig: &handler.AccountRoleConfig{
					Path:               "/aa/bb/",
					PermissionBoundary: "",
				},
				Version:      "latest",
				ChannelGroup: "candidate",
				Region:       "us-east-2",
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, customProfile)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			defer func() {
				ginkgo.By("Clean resources")
				clusterHandler.Destroy()
			}()

			ginkgo.By("Delete cluster")
			rosaClient.Runner.UnsetArgs()
			_, err := clusterService.DeleteCluster(clusterID, "-y")
			gomega.Expect(err).To(gomega.BeNil())

			rosaClient.Runner.UnsetArgs()
			err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Delete operator-roles")
			_, err = ocmResourceService.DeleteOperatorRoles(
				"-c", clusterID,
				"--mode", "auto",
				"-y",
			)
			gomega.Expect(err).To(gomega.BeNil())
		})

		ginkgo.It("Creating cluster with sts external id should succeed - [id:75603]",
			labels.Medium, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Create classic cluster in auto mode")
				testingClusterName = helper.GenerateRandomName("c75603", 2)
				testOperatorRolePrefix := helper.GenerateRandomName("opp75603", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--operator-roles-prefix": testOperatorRolePrefix,
				})

				rosalCommand.AddFlags("--mode", "auto")

				ginkgo.By("Update installer role")
				ExternalId := "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"
				installRoleArn := rosalCommand.GetFlagValue("--role-arn", true)
				_, roleName, err := helper.ParseRoleARN(installRoleArn)
				gomega.Expect(err).To(gomega.BeNil())

				awsClient, err := aws_client.CreateAWSClient("", "")
				gomega.Expect(err).To(gomega.BeNil())
				opRole, err := awsClient.IamClient.GetRole(
					context.TODO(),
					&iam.GetRoleInput{
						RoleName: &roleName,
					})
				gomega.Expect(err).To(gomega.BeNil())

				decodedPolicyDocument, err := url.QueryUnescape(*opRole.Role.AssumeRolePolicyDocument)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("update the trust relationship")
				var policyDocument map[string]interface{}

				err = json.Unmarshal([]byte(decodedPolicyDocument), &policyDocument)
				gomega.Expect(err).To(gomega.BeNil())

				newCondition := map[string]interface{}{
					"StringEquals": map[string]interface{}{
						"sts:ExternalId": ExternalId,
					},
				}

				statements := policyDocument["Statement"].([]interface{})
				for _, statement := range statements {
					stmt := statement.(map[string]interface{})
					stmt["Condition"] = newCondition
				}
				updatedPolicyDocument, err := json.Marshal(policyDocument)
				gomega.Expect(err).To(gomega.BeNil())

				_, err = awsClient.IamClient.UpdateAssumeRolePolicy(context.TODO(), &iam.UpdateAssumeRolePolicyInput{
					RoleName:       aws.String(roleName),
					PolicyDocument: aws.String(string(updatedPolicyDocument)),
				})
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Wait for the trust relationship to be updated")
				err = wait.PollUntilContextTimeout(
					context.Background(),
					20*time.Second,
					300*time.Second,
					false,
					func(context.Context) (bool, error) {
						result, err := awsClient.IamClient.GetRole(context.TODO(), &iam.GetRoleInput{
							RoleName: aws.String(roleName),
						})

						if strings.Contains(*result.Role.AssumeRolePolicyDocument, ExternalId) {
							return true, nil
						}

						return false, err
					})
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Create cluster with external id not same with the role setting one")
				notMatchExternalId := "333B9588-36A5-ECA4-BE8D-7C673B77CDCD"
				rosalCommand.AddFlags("--external-id", notMatchExternalId)
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).ToNot(gomega.BeNil())
				gomega.Expect(stdout.String()).To(gomega.ContainSubstring(
					"An error occurred while trying to create an AWS client: Failed to assume role with ARN"))

				ginkgo.By("Create cluster with external id")
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--external-id": ExternalId,
				})
				stdout, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				err = clusterService.WaitClusterStatus(clusterID, "installing", 3, 20)
				gomega.Expect(err).To(gomega.BeNil())
			})
	})
var _ = ginkgo.Describe("HCP cluster creation supplemental testing",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService

			customProfile      *handler.Profile
			clusterID          string
			ocmResourceService rosacli.OCMResourceService
			AWSAccountID       string
			testingClusterName string
			clusterHandler     handler.ClusterHandler
		)
		ginkgo.BeforeEach(func() {
			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource

			ginkgo.By("Get AWS account id")
			rosaClient.Runner.JsonFormat()
			whoamiOutput, err := ocmResourceService.Whoami()
			gomega.Expect(err).To(gomega.BeNil())
			rosaClient.Runner.UnsetFormat()
			whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
			AWSAccountID = whoamiData.AWSAccountID

			ginkgo.By("Prepare custom profile")
			customProfile = &handler.Profile{
				ClusterConfig: &handler.ClusterConfig{
					HCP:           true,
					MultiAZ:       true,
					STS:           true,
					OIDCConfig:    "managed",
					NetworkingSet: true,
					BYOVPC:        true,
					Zones:         "",
				},
				AccountRoleConfig: &handler.AccountRoleConfig{
					Path:               "",
					PermissionBoundary: "",
				},
				Version:      "latest",
				ChannelGroup: "candidate",
				Region:       constants.CommonAWSRegion,
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, customProfile)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Init the cluster id and testing cluster name")
			clusterID = ""
			testingClusterName = ""
		})

		ginkgo.AfterEach(func() {
			defer func() {
				ginkgo.By("Clean resources")
				clusterHandler.Destroy()
			}()

			if clusterID != "" {
				ginkgo.By("Delete cluster by id")
				rosaClient.Runner.UnsetArgs()
				_, err := clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Delete operator-roles")
				_, err = ocmResourceService.DeleteOperatorRoles(
					"-c", clusterID,
					"--mode", "auto",
					"-y",
				)
				gomega.Expect(err).To(gomega.BeNil())
			} else if testingClusterName != "" {
				// At least try to delete testing cluster
				ginkgo.By("Delete cluster by name")
				rosaClient.Runner.UnsetArgs()
				_, err := clusterService.DeleteCluster(testingClusterName, "-y")
				gomega.Expect(err).To(gomega.BeNil())
			}
		})

		ginkgo.It("Check the output of the STS cluster creation with new oidc flow - [id:75925]",
			labels.Medium, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Create hcp cluster in auto mode")
				testingClusterName = helper.GenerateRandomName("c75925", 2)
				testOperatorRolePrefix := helper.GenerateRandomName("opp75925", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--operator-roles-prefix": testOperatorRolePrefix,
				})

				rosalCommand.AddFlags("--mode", "auto")
				rosalCommand.AddFlags("--billing-account", AWSAccountID)
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(stdout.String()).To(gomega.ContainSubstring("Attached trust policy"))

				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				gomega.Expect(clusterID).ToNot(gomega.BeNil())
			})

		ginkgo.It("ROSA CLI cluster creation should show install/uninstall logs - [id:75534]",
			labels.Critical,
			labels.Runtime.Day1Supplemental,
			func() {
				testingClusterName = helper.GenerateRandomName("ocp-75534", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				command := "rosa create cluster --cluster-name " +
					testingClusterName + " " + strings.Join(flags, " ") + " " + "--mode auto -y"
				rosalCommand := config.GenerateCommand(command)
				fmt.Println("debug command is: ", rosalCommand.GetFullCommand())
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))

				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(stdout.String()).To(
					gomega.ContainSubstring(fmt.Sprintf("Cluster '%s' has been created", testingClusterName)))

				ginkgo.By("Check the install logs of the hypershift cluster")
				gomega.Eventually(func() (string, error) {
					output, err := clusterService.InstallLog(testingClusterName)
					return output.String(), err
				}, time.Minute*10, time.Second*30).Should(gomega.And(
					gomega.ContainSubstring("hostedclusters %s Version", testingClusterName),
					gomega.ContainSubstring("hostedclusters %s Release image is valid", testingClusterName)))

				ginkgo.By("Check the install logs of the hypershift cluster with flag --watch")
				output, err := clusterService.InstallLog(testingClusterName, "--watch")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(gomega.ContainSubstring("hostedclusters %s Version", testingClusterName))
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("hostedclusters %s Release image is valid", testingClusterName))
				gomega.Expect(output.String()).Should(gomega.ContainSubstring("Cluster '%s' is now ready", testingClusterName))

				ginkgo.By("Delete the Hypershift cluster")
				output, err = clusterService.DeleteCluster(testingClusterName, "-y")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("Cluster '%s' will start uninstalling now", testingClusterName))

				ginkgo.By("Check the uninstall logs of the hypershift cluster")
				gomega.Eventually(func() (string, error) {
					output, err := clusterService.UnInstallLog(testingClusterName)
					return output.String(), err
				}, time.Minute*20, time.Second*30).
					Should(
						gomega.ContainSubstring("hostedclusters %s Reconciliation completed successfully", testingClusterName))

				ginkgo.By("Check the uninstall log of the hosted cluster with flag --watch")
				output, err = clusterService.UnInstallLog(testingClusterName, "--watch")
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("hostedclusters %s Reconciliation completed successfully",
						testingClusterName))
				gomega.Expect(output.String()).Should(
					gomega.ContainSubstring("Cluster '%s' completed uninstallation", testingClusterName))
				testingClusterName = ""
			})

		ginkgo.It("Check single AZ hosted cluster can be created - [id:54413]",
			labels.Critical, labels.Runtime.Day1Supplemental,
			func() {
				testingClusterName = helper.GenerateRandomName("c54413", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				if rosalCommand.CheckFlagExist("--multi-az") {
					rosalCommand.DeleteFlag("--multi-az", false)
				}
				if rosalCommand.CheckFlagExist("--subnet-ids") {
					subnets := strings.Split(rosalCommand.GetFlagValue("--subnet-ids", true), ",")
					var newSubnets []string
					if customProfile.ClusterConfig.Private && customProfile.ClusterConfig.PrivateLink {
						newSubnets = append(newSubnets, subnets[0])
					} else {
						index := len(subnets) / 2
						newSubnets = append(newSubnets, subnets[0], subnets[index])
					}
					flags := map[string]string{}
					flags["--subnet-ids"] = strings.Join(newSubnets, ",")
					rosalCommand.ReplaceFlagValue(flags)
				}

				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(stdout.String()).To(
					gomega.ContainSubstring(fmt.Sprintf("Cluster '%s' has been created", testingClusterName)))

				ginkgo.By("Retrieve cluster ID")
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID

				ginkgo.By("Wait for Cluster")
				err = clusterService.WaitClusterStatus(clusterID, constants.Ready, 3, 60)
				gomega.Expect(err).To(gomega.BeNil(), "It met error or timeout when waiting cluster to ready status")
			})

		ginkgo.It("Create hosted cluster in manual mode - [id:75536]",
			labels.High, labels.Runtime.Day1Supplemental,
			func() {
				customProfile.ClusterConfig.ManualCreationMode = true
				ginkgo.By("Prepare command for testing")
				testingClusterName = helper.GenerateRandomName("c75536", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				if rosalCommand.CheckFlagExist("--mode") {
					rosalCommand.DeleteFlag("--mode", true)
				}
				rosalCommand.AddFlags("--mode", "manual")

				ginkgo.By("Create temp dir for manual execution")
				dirForManual, err := os.MkdirTemp("", "*")
				gomega.Expect(err).To(gomega.BeNil())
				rosaClient.Runner.SetDir(dirForManual)

				ginkgo.By("Run create cluster command")
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(stdout.String()).To(
					gomega.ContainSubstring(fmt.Sprintf("Cluster '%s' has been created", testingClusterName)))

				ginkgo.By("Run individual manual commands")
				commands := helper.ExtractAWSCmdsForClusterCreation(stdout)
				for _, command := range commands {
					_, err := rosaClient.Runner.RunCMD(strings.Split(command, " "))
					gomega.Expect(err).To(gomega.BeNil())
				}

				ginkgo.By("Retrieve cluster ID")
				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID

				ginkgo.By("Wait for Cluster")
				err = clusterService.WaitClusterStatus(clusterID, constants.Ready, 3, 60)
				gomega.Expect(err).To(gomega.BeNil(), "It met error or timeout when waiting cluster to ready status")
			})
	})
var _ = ginkgo.Describe("Sts cluster creation supplemental testing",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			clusterHandler handler.ClusterHandler

			customProfile      *handler.Profile
			clusterID          string
			testingClusterName string
		)
		ginkgo.BeforeEach(func() {
			var err error

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster

			ginkgo.By("Get AWS account id")
			rosaClient.Runner.JsonFormat()
			rosaClient.Runner.UnsetFormat()

			ginkgo.By("Prepare custom profile")
			customProfile = &handler.Profile{
				ClusterConfig: &handler.ClusterConfig{
					HCP:           false,
					MultiAZ:       true,
					STS:           true,
					OIDCConfig:    "",
					NetworkingSet: false,
					BYOVPC:        false,
				},
				AccountRoleConfig: &handler.AccountRoleConfig{
					Path:               "/aa/bb/",
					PermissionBoundary: "",
				},
				Version:      "latest",
				ChannelGroup: "candidate",
				Region:       "us-east-2",
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, customProfile)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("Clean resources")
			clusterHandler.Destroy()
		})

		ginkgo.It("Check the trust policy attaching during hosted-cp cluster creation - [id:75927]",
			labels.Medium, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Create hcp cluster in auto mode")
				testingClusterName = helper.GenerateRandomName("c75927", 2)
				testOperatorRolePrefix := helper.GenerateRandomName("opp75927", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--operator-roles-prefix": testOperatorRolePrefix,
				})

				rosalCommand.AddFlags("--mode", "auto")
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())
				defer func() {
					ginkgo.By("Delete cluster")
					_, err := clusterService.DeleteCluster(testingClusterName, "-y")
					gomega.Expect(err).To(gomega.BeNil())
				}()
				gomega.Expect(stdout.String()).To(gomega.ContainSubstring("Attached trust policy"))

				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				gomega.Expect(clusterID).ToNot(gomega.BeNil())
			})

		ginkgo.It("User can set availability zones to create rosa multi-az STS cluster - [id:56224]",
			labels.Critical, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Create classic sts cluster in auto mode")
				customProfile.ClusterConfig.Zones = "us-east-2a,us-east-2b,us-east-2c"
				customProfile.NamePrefix = helper.GenerateRandomName("rosa56224", 2)
				testingClusterName = helper.GenerateRandomName("cluster56224", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				flags = append(flags, "-m")
				flags = append(flags, "auto")
				_, err, _ = clusterService.Create(testingClusterName, flags[:]...)
				gomega.Expect(err).To(gomega.BeNil())
				defer func() {
					ginkgo.By("Delete cluster")
					_, err = clusterService.DeleteCluster(clusterID, "-y")
					gomega.Expect(err).To(gomega.BeNil())
				}()

				rosaClient.Runner.UnsetArgs()
				clusterListOut, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListOut)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				gomega.Expect(clusterID).ToNot(gomega.BeNil())

				ginkgo.By("Describe cluster in json format")
				rosaClient.Runner.JsonFormat()
				jsonOutput, err := clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				rosaClient.Runner.UnsetFormat()
				jsonData := rosaClient.Parser.JsonData.Input(jsonOutput).Parse()

				zones := jsonData.DigString("nodes", "availability_zones")
				gomega.Expect(zones).To(gomega.Equal("[us-east-2a us-east-2b us-east-2c]"))
			})

		ginkgo.It("rosacli makes STS cluster by default - [id:55701]",
			labels.Medium, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Check the help message of 'rosa describe upgrade -h'")
				output, err, _ := clusterService.Create("ocp55701", "--help")
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("rosa create cluster [flags]"))
				gomega.Expect(output.String()).To(gomega.ContainSubstring("--sts"))
				gomega.Expect(output.String()).To(gomega.ContainSubstring("--non-sts"))
				gomega.Expect(output.String()).To(gomega.ContainSubstring("--mint-mode"))

				ginkgo.By("Create cluster with '--sts' flag")
				testingClusterName = helper.GenerateRandomName("c55701", 2)
				output, err, _ = clusterService.Create(testingClusterName, "--sts")
				gomega.Expect(err).NotTo(gomega.BeNil())
				gomega.Expect(output.String()).To(gomega.ContainSubstring("More than one Installer role found"))
				gomega.Expect(output.String()).To(gomega.ContainSubstring("Expected a valid role ARN"))

				ginkgo.By("Create cluster with '--non-sts' flag")
				testingClusterName = helper.GenerateRandomName("c55701", 2)
				output, err, _ = clusterService.Create(testingClusterName, "--non-sts")
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				gomega.Expect(clusterID).ToNot(gomega.BeNil())

				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err := clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(CD.STSRoleArn).To(gomega.BeEmpty())

				ginkgo.By("Delete cluster which created with '--non-sts' flag")
				rosaClient.Runner.UnsetArgs()
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Create cluster with '--mint-mode' flag")
				testingClusterName = helper.GenerateRandomName("c55701", 2)
				output, err, _ = clusterService.Create(testingClusterName, "--mint-mode")
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				clusterListout, err = clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err = clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				gomega.Expect(clusterID).ToNot(gomega.BeNil())

				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err = clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(CD.STSRoleArn).To(gomega.BeEmpty())

				ginkgo.By("Delete cluster which created with '--mint-mode' flag")
				rosaClient.Runner.UnsetArgs()
				_, err = clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Create cluster without setting '--sts'/'--non-sts'/'--mint-mode' flags but with the " +
					"account-roles arns set")
				testingClusterName = helper.GenerateRandomName("c55701", 2)
				testOperatorRolePrefix := helper.GenerateRandomName("opp55701", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--operator-roles-prefix": testOperatorRolePrefix,
				})

				rosalCommand.AddFlags("--mode", "auto")
				stdout, err := rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(stdout.String()).To(gomega.ContainSubstring("Attached trust policy"))

				rosaClient.Runner.UnsetArgs()
				clusterListout, err = clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err = clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				gomega.Expect(clusterID).ToNot(gomega.BeNil())

				output, err = clusterService.DescribeCluster(clusterID)
				gomega.Expect(err).To(gomega.BeNil())
				CD, err = clusterService.ReflectClusterDescription(output)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(CD.STSRoleArn).NotTo(gomega.BeEmpty())

				ginkgo.By("Delete cluster")
				_, err = clusterService.DeleteCluster(testingClusterName, "-y")
				gomega.Expect(err).To(gomega.BeNil())
			})
	})

var _ = ginkgo.Describe("Sts cluster with BYO oidc flow creation supplemental testing",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient         *rosacli.Client
			clusterService     rosacli.ClusterService
			clusterHandler     handler.ClusterHandler
			customProfile      *handler.Profile
			clusterID          string
			ocmResourceService rosacli.OCMResourceService
			testingClusterName string
		)
		ginkgo.BeforeEach(func() {
			var err error

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource

			ginkgo.By("Prepare custom profile")
			customProfile = &handler.Profile{
				ClusterConfig: &handler.ClusterConfig{
					HCP:           false,
					MultiAZ:       false,
					STS:           true,
					OIDCConfig:    "managed",
					NetworkingSet: false,
					BYOVPC:        false,
				},
				AccountRoleConfig: &handler.AccountRoleConfig{
					Path:               "/aa/bb/",
					PermissionBoundary: "",
				},
				Version:      "latest",
				ChannelGroup: "candidate",
				Region:       "us-east-2",
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, customProfile)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			defer func() {
				ginkgo.By("Clean resources")
				clusterHandler.Destroy()
			}()

			ginkgo.By("Delete the cluster")
			if clusterID != "" {
				rosaClient.Runner.UnsetArgs()
				_, err := clusterService.DeleteCluster(clusterID, "-y")
				gomega.Expect(err).To(gomega.BeNil())
				rosaClient.Runner.UnsetArgs()
				err = clusterService.WaitClusterDeleted(clusterID, 3, 35)
				gomega.Expect(err).To(gomega.BeNil())
			}
		})

		ginkgo.It("Create STS cluster with oidc config id but no oidc provider via rosacli in auto mode - [id:76093]",
			labels.Critical, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Prepare command for custom cluster creation")
				testingClusterName = helper.GenerateRandomName("c76093", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				rosalCommand.AddFlags("--mode", "auto", "-y")

				ginkgo.By("Delete the oidc provider")
				ocmResourceService = rosaClient.OCMResource
				rosaClient.Runner.JsonFormat()
				whoamiOutput, err := ocmResourceService.Whoami()
				gomega.Expect(err).To(gomega.BeNil())
				rosaClient.Runner.UnsetFormat()
				whoamiData := ocmResourceService.ReflectAccountsInfo(whoamiOutput)
				AWSAccountID := whoamiData.AWSAccountID

				oidcConfigID := clusterHandler.GetResourcesHandler().GetOIDCConfigID()
				oidcConfigList, _, err := ocmResourceService.ListOIDCConfig()
				gomega.Expect(err).To(gomega.BeNil())
				foundOIDCConfig := oidcConfigList.OIDCConfig(oidcConfigID)
				gomega.Expect(foundOIDCConfig).ToNot(gomega.Equal(rosacli.OIDCConfig{}))
				issueURL := foundOIDCConfig.IssuerUrl
				oidcProviderARN := fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s",
					AWSAccountID, strings.TrimPrefix(issueURL, "https://"))

				awsClient, err := aws_client.CreateAWSClient("", "")
				gomega.Expect(err).To(gomega.BeNil())
				_, err = awsClient.IamClient.DeleteOpenIDConnectProvider(context.TODO(), &iam.DeleteOpenIDConnectProviderInput{
					OpenIDConnectProviderArn: aws.String(oidcProviderARN),
				})
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Create the custom cluster")
				_, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())

				rosaClient.Runner.UnsetArgs()
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				gomega.Expect(clusterID).ToNot(gomega.BeNil())

				ginkgo.By("Wait cluster to instaling status")
				err = clusterService.WaitClusterStatus(clusterID, "installing", 3, 24)
				gomega.Expect(err).To(gomega.BeNil(), "It met error or timeout when waiting cluster to installing status")
			})
	})
var _ = ginkgo.Describe("Non-STS cluster with local credentials",
	labels.Feature.Cluster,
	func() {
		defer ginkgo.GinkgoRecover()

		var (
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService

			customProfile      *handler.Profile
			clusterID          string
			ocmResourceService rosacli.OCMResourceService
			testingClusterName string
			clusterHandler     handler.ClusterHandler
		)
		ginkgo.BeforeEach(func() {
			var err error

			ginkgo.By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			ocmResourceService = rosaClient.OCMResource

			ginkgo.By("Get AWS account id")
			rosaClient.Runner.JsonFormat()
			rosaClient.Runner.UnsetFormat()

			ginkgo.By("Prepare custom profile")
			customProfile = &handler.Profile{
				ClusterConfig: &handler.ClusterConfig{
					HCP:                 false,
					MultiAZ:             false,
					STS:                 false,
					OIDCConfig:          "",
					NetworkingSet:       false,
					BYOVPC:              false,
					UseLocalCredentials: true,
				},
				AccountRoleConfig: &handler.AccountRoleConfig{
					Path:               "/aa/bb/",
					PermissionBoundary: "",
				},
				Version:      "latest",
				ChannelGroup: "candidate",
				Region:       "us-east-2",
			}
			customProfile.NamePrefix = constants.DefaultNamePrefix
			clusterHandler, err = handler.NewTempClusterHandler(rosaClient, customProfile)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			defer func() {
				ginkgo.By("Clean resources")
				clusterHandler.Destroy()
			}()

			ginkgo.By("Delete cluster")
			rosaClient.Runner.UnsetArgs()
			_, err := clusterService.DeleteCluster(clusterID, "-y")
			gomega.Expect(err).To(gomega.BeNil())

			rosaClient.Runner.UnsetArgs()
			err = clusterService.WaitClusterDeleted(clusterID, 3, 30)
			gomega.Expect(err).To(gomega.BeNil())

			ginkgo.By("Delete operator-roles")
			_, err = ocmResourceService.DeleteOperatorRoles(
				"-c", clusterID,
				"--mode", "auto",
				"-y",
			)
			gomega.Expect(err).To(gomega.BeNil())
		})

		ginkgo.It("Creating cluster with non-sts use-local-credentials should succeed - [id:65900]",
			labels.Medium, labels.Runtime.Day1Supplemental,
			func() {
				ginkgo.By("Create classic cluster in auto mode")
				testingClusterName = helper.GenerateRandomName("c65900", 2)
				testOperatorRolePrefix := helper.GenerateRandomName("opp65900", 2)
				flags, err := clusterHandler.GenerateClusterCreateFlags()
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				command := "rosa create cluster --cluster-name " + testingClusterName + " " + strings.Join(flags, " ")
				rosalCommand := config.GenerateCommand(command)
				rosalCommand.ReplaceFlagValue(map[string]string{
					"--operator-roles-prefix": testOperatorRolePrefix,
				})

				rosalCommand.AddFlags("--mode", "auto")
				_, err = rosaClient.Runner.RunCMD(strings.Split(rosalCommand.GetFullCommand(), " "))
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Wait for the cluster to be installing")
				clusterListout, err := clusterService.List()
				gomega.Expect(err).To(gomega.BeNil())
				clusterList, err := clusterService.ReflectClusterList(clusterListout)
				gomega.Expect(err).To(gomega.BeNil())
				clusterID = clusterList.ClusterByName(testingClusterName).ID
				err = clusterService.WaitClusterStatus(clusterID, "installing", 3, 20)
				gomega.Expect(err).To(gomega.BeNil())

				ginkgo.By("Check the properties of the cluster")
				jsonData, err := clusterService.GetJSONClusterDescription(clusterID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(jsonData.DigBool("properties", "use_local_credentials")).To(gomega.BeTrue())
			})
	})
