package e2e

import (
	"context"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Network verifier",
	labels.Feature.VerifyResources,
	func() {
		defer GinkgoRecover()

		var (
			clusterID      string
			rosaClient     *rosacli.Client
			networkService rosacli.NetworkVerifierService
			clusterService rosacli.ClusterService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			networkService = rosaClient.NetworkVerifier
			clusterService = rosaClient.Cluster
		})

		// Verify network via the rosa cli
		It("can verify network - [id:64917]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Get cluster description")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				By("Check if non BYO VPC cluster")
				isBYOVPC, err := clusterService.IsBYOVPCCluster(clusterID)
				Expect(err).To(BeNil())
				if !isBYOVPC {
					Skip("It does't support the verification for non byo vpc cluster - cannot run this test")
				}

				By("Run network verifier vith clusterID")
				output, err = networkService.CreateNetworkVerifierWithCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).
					To(ContainSubstring(
						"Run the following command to wait for verification to all subnets to complete:\n" +
							"rosa verify network --watch --status-only"))

				By("Get the cluster subnets")
				var subnetsNetworkInfo string
				for _, networkLine := range clusterDetail.Network {
					if value, containsKey := networkLine["Subnets"]; containsKey {
						subnetsNetworkInfo = value
						break
					}
				}
				subnets := strings.Replace(subnetsNetworkInfo, " ", "", -1)
				region := clusterDetail.Region
				installerRoleArn := clusterDetail.STSRoleArn

				By("Check the network verifier status")
				err = wait.PollUntilContextTimeout(
					context.Background(),
					20*time.Second,
					200*time.Second,
					false,
					func(context.Context) (bool, error) {
						output, err = networkService.GetNetworkVerifierStatus(
							"--region", region,
							"--subnet-ids", subnets,
						)
						if strings.Contains(output.String(), "pending") {
							return false, err
						}
						return true, err
					})
				common.AssertWaitPollNoErr(err, "Network verification result are not ready after 200")

				By("Check the network verifier with tags attributes")
				output, err = networkService.CreateNetworkVerifierWithCluster(clusterID,
					"--tags", "t1:v1")
				Expect(err).ToNot(HaveOccurred())

				By("Check the network verifier status")
				err = wait.PollUntilContextTimeout(
					context.Background(),
					20*time.Second,
					200*time.Second,
					false,
					func(context.Context) (bool, error) {
						output, err = networkService.GetNetworkVerifierStatus(
							"--region", region,
							"--subnet-ids", subnets,
						)
						if strings.Contains(output.String(), "pending") {
							return false, err
						}
						return true, err
					})
				common.AssertWaitPollNoErr(err, "Network verification result are not ready after 200")

				By("Run network verifier vith subnet id")
				if installerRoleArn == "" {
					Skip("It does't support the verification with subnets for non STS cluster - cannot run this test")
				}
				output, err = networkService.CreateNetworkVerifierWithSubnets(
					"--region", region,
					"--subnet-ids", subnets,
					"--role-arn", installerRoleArn,
					"--tags", "t2:v2",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).
					To(ContainSubstring(
						"Run the following command to wait for verification to all subnets to complete:\n" +
							"rosa verify network --watch --status-only"))
				Expect(output.String()).To(ContainSubstring("pending"))

				By("Check the network verifier with hosted-cp attributes")
				output, err = networkService.CreateNetworkVerifierWithSubnets(
					"--region", region,
					"--subnet-ids", subnets,
					"--role-arn", installerRoleArn,
					"--hosted-cp",
				)
				Expect(err).ToNot(HaveOccurred())
			})

		It("validation should work well - [id:68751]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Get cluster description")
				output, err := clusterService.DescribeCluster(clusterID)
				Expect(err).To(BeNil())
				clusterDetail, err := clusterService.ReflectClusterDescription(output)
				Expect(err).To(BeNil())

				By("Get the cluster subnets")
				var subnetsNetworkInfo string
				for _, networkLine := range clusterDetail.Network {
					if value, containsKey := networkLine["Subnets"]; containsKey {
						subnetsNetworkInfo = value
						break
					}
				}
				subnets := strings.Replace(subnetsNetworkInfo, " ", "", -1)
				region := clusterDetail.Region
				isBYOVPC, err := clusterService.IsBYOVPCCluster(clusterID)
				Expect(err).To(BeNil())

				if !isBYOVPC {
					By("Run network verifier with non BYO VPC cluster")
					output, err = networkService.CreateNetworkVerifierWithCluster(clusterID)
					Expect(err).To(HaveOccurred())
					Expect(output.String()).
						To(ContainSubstring(
							"ERR: Running the network verifier is only supported for BYO VPC clusters"))
					return
				}
				By("Run network verifier without clusterID")
				output, err = networkService.CreateNetworkVerifierWithCluster("non-existing")
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					To(ContainSubstring(
						"ERR: Failed to get cluster 'non-existing': There is no cluster with identifier or name 'non-existing'"))

				By("Run network verifier with --hosted-cp")
				output, err = networkService.CreateNetworkVerifierWithCluster(clusterID, "--hosted-cp")
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					To(ContainSubstring("ERR: '--hosted-cp' flag is not required when running the network verifier with cluster"))

				By("Check the network for cluster with invalid tags")
				output, err = networkService.CreateNetworkVerifierWithCluster(clusterID,
					"--tags", "t1=v1")
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					To(ContainSubstring(
						"ERR: invalid tag format for tag '[t1=v1]'. Expected tag format: 'key value'"))

				By("Run the network verified without role")
				output, err = networkService.CreateNetworkVerifierWithSubnets(
					"--region", region,
					"--subnet-ids", subnets,
					"--hosted-cp",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("ERR: role-arn is required"))
			})
	})
