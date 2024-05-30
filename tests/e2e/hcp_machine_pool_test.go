package e2e

import (
	"os"
	"path"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("Create Machine Pool", func() {

	var client *rosacli.Client

	BeforeEach(func() {
		Expect(clusterID).ToNot(BeEmpty(), "Cluster ID is empty, please export the env variable CLUSTER_ID")
		client = rosacli.NewClient()
		clusterService := client.Cluster
		hostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
		Expect(err).ToNot(HaveOccurred())

		if !hostedCluster {
			Skip("This case applies only on hosted cluster")
		}
	})
	It("to hosted cluster with additional security group IDs will work [id:72195]", func() {
		By("Prepare security groups")
		// This part is finding security groups according to old structure of day1.
		// Need to be updated after ocm-common migration
		preparedSGFile := path.Join(os.Getenv("SHARED_DIR"), "security_groups_ids")
		if _, err := os.Stat(preparedSGFile); err != nil {
			log.Logger.Warnf("Didn't find file %s", preparedSGFile)
			Skip("the security groups are not prepared, skip this testing")
		}
		sgContent, err := common.ReadFileContent(preparedSGFile)
		Expect(err).ToNot(HaveOccurred())
		sgIDs := strings.Split(sgContent, " ")

		By("Create machinepool with security groups set")
		mpName := "mp-72195"
		_, err = client.MachinePool.CreateMachinePool(clusterID, mpName,
			"--additional-security-group-ids", strings.Join(sgIDs, ","),
			"--replicas", "1",
			"-y",
		)
		Expect(err).ToNot(HaveOccurred())
		defer client.MachinePool.DeleteMachinePool(clusterID, mpName)

		By("Check the machinepool detail by describe")
		mpDescription, err := client.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
		Expect(err).ToNot(HaveOccurred())

		Expect(mpDescription.AdditionalSecurityGroupIDs).To(Equal(strings.Join(sgIDs, ", ")))

		By("Create another machinepool without security groups and describe it")
		mpName = "mp-72195-nsg"
		_, err = client.MachinePool.CreateMachinePool(clusterID, mpName,
			"--replicas", "1",
			"-y",
		)
		Expect(err).ToNot(HaveOccurred())
		defer client.MachinePool.DeleteMachinePool(clusterID, mpName)
		By("Check the machinepool detail by describe")
		mpDescription, err = client.MachinePool.DescribeAndReflectNodePool(clusterID, mpName)
		Expect(err).ToNot(HaveOccurred())
		Expect(mpDescription.AdditionalSecurityGroupIDs).To(BeEmpty())
	})
})
