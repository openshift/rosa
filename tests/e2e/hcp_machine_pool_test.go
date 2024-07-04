package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	ph "github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Create Machine Pool", labels.Feature.Machinepool, func() {

	var (
		rosaClient *rosacli.Client
		profile    *ph.Profile
	)
	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()

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
})
