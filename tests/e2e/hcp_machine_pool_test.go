package e2e

import (
	"os"
	"path"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("Create Machine Pool", labels.Feature.Machinepool, func() {

	var rosaClient *rosacli.Client
	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		rosaClient = rosacli.NewClient()

		By("Skip testing if the cluster is not a HCP cluster")
		hostedCluster, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())
		if !hostedCluster {
			SkipNotHosted()
		}
	})
	It("to hosted cluster with additional security group IDs will work [id:72195]",
		labels.Critical, labels.Runtime.Day2,
		func() {
			By("Prepare security groups")
			// This part is finding security groups according to old structure of day1.
			// Need to be updated after ocm-common migration
			preparedSGFile := path.Join(os.Getenv("SHARED_DIR"), "security_groups_ids")
			if _, err := os.Stat(preparedSGFile); err != nil {
				log.Logger.Warnf("Didn't find file %s", preparedSGFile)
				SkipTestOnFeature("security groups")
			}
			sgContent, err := common.ReadFileContent(preparedSGFile)
			Expect(err).ToNot(HaveOccurred())
			sgIDs := strings.Split(sgContent, " ")

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
