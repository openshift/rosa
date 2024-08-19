package e2e

import (
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	ph "github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Create Machine Pool", labels.Feature.Machinepool, func() {

	var (
		rosaClient         *rosacli.Client
		machinePoolService rosacli.MachinePoolService
		profile            *ph.Profile
	)
	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()
		machinePoolService = rosaClient.MachinePool

		By("Skip testing if the cluster is not a HCP cluster")
		hostedCluster, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())

		profile = ph.LoadProfileYamlFileByENV()

		if !hostedCluster {
			SkipNotHosted()
		}
	})
	It("to hosted cluster with additional security group IDs will work [id:72195]",
		labels.Critical, labels.Runtime.Day2,
		func() {
			By("Load the vpc client of the machinepool")
			mps, err := rosaClient.MachinePool.ListAndReflectNodePools(clusterID)
			Expect(err).ToNot(HaveOccurred())

			subnetID := mps.NodePools[0].Subnet
			vpcClient, err := vpc_client.GenerateVPCBySubnet(subnetID, profile.Region)
			Expect(err).ToNot(HaveOccurred())

			By("Prepare security groups")
			sgIDs, err := vpcClient.CreateAdditionalSecurityGroups(3, "72195", "testing for case 72195")
			Expect(err).ToNot(HaveOccurred())

			By("Create machinepool with security groups set")
			mpName := "mp-72195"
			_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--additional-security-group-ids", strings.Join(sgIDs, ","),
				"--replicas", "1",
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)

			By("Check the machinepool detail by describe")
			mpDescription, err := rosaClient.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
			Expect(err).ToNot(HaveOccurred())

			Expect(mpDescription.AdditionalSecurityGroupIDs).To(Equal(strings.Join(sgIDs, ", ")))

			By("Create another machinepool without security groups and describe it")
			mpName = "mp-72195-nsg"
			_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, mpName,
				"--replicas", "1",
				"-y",
			)
			Expect(err).ToNot(HaveOccurred())
			defer rosaClient.MachinePool.DeleteMachinePool(clusterID, mpName)
			By("Check the machinepool detail by describe")
			mpDescription, err = rosaClient.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
			Expect(err).ToNot(HaveOccurred())
			Expect(mpDescription.AdditionalSecurityGroupIDs).To(BeEmpty())
		})

	It("machinepool AWS preflight tag validation[id:73638]",
		labels.Medium, labels.Runtime.Day2,
		func() {

			By("Check the help message of machinepool creation")
			mpID := common.GenerateRandomName("mp-73638", 2)
			out, err := machinePoolService.CreateMachinePool(clusterID, mpID, "-h")
			Expect(err).ToNot(HaveOccurred(), out.String())
			Expect(out.String()).Should(ContainSubstring("--tags strings"))

			By("Create a machinepool with tags set")
			tags := []string{
				"test:testvalue",
				"test2:testValue/openshift",
			}
			out, err = machinePoolService.CreateMachinePool(clusterID, mpID,
				"--replicas", "3",
				"--tags", strings.Join(tags, ","),
			)
			Expect(err).ToNot(HaveOccurred(), out.String())

			By("Describe the machinepool")
			description, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, mpID)
			Expect(err).ToNot(HaveOccurred(), out.String())

			for _, tag := range tags {
				Expect(description.Tags).Should(ContainSubstring(strings.Replace(tag, ":", "=", -1)))
			}

			By("Create machinepool with too many tags")
			maxTags := 25
			var tooManyTags []string
			for i := 0; i < maxTags+1; i++ {
				t := strconv.Itoa(i)
				key := "foo" + t
				kvp := key + ":testValue"
				tooManyTags = append(tooManyTags, kvp)
			}

			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", strings.Join(tooManyTags, ","),
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("Invalid Node Pool AWS tags: Resource has too many AWS tags"))

			By("Create machinepool with a tag too long")
			maxKeyTagLength := 128
			maxValueTagLength := 256

			tooLongKeyTag := strings.Repeat("z", maxKeyTagLength+1)
			tag := tooLongKeyTag + ":testValue"

			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: expected a valid user tag key 'zzz"))

			tooLongValueTag := strings.Repeat("z", maxValueTagLength+1)
			tag = "testKey:" + tooLongValueTag

			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: expected a valid user tag value 'zzz"))

			By("Create machinepool using aws as a prefix")
			tag = "aws:testKey:" + "testValue"
			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: invalid tag format for tag '[aws testKey testValue]'"))

			By("Create machinepool with an invalid tag")
			tag = "#" + ":testValue"
			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: expected a valid user tag key '#'"))

			tag = "testKey:" + "#"
			out, err = machinePoolService.CreateMachinePool(clusterID, "invalid-73469",
				"--replicas", "3",
				"--tags", tag,
			)
			Expect(err).To(HaveOccurred())
			Expect(out.String()).Should(ContainSubstring("ERR: expected a valid user tag value '#'"))
		})
})
