package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
	utilConfig "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/log"
	"github.com/openshift/rosa/tests/utils/profilehandler"
)

var _ = Describe("Cluster preparation", labels.Feature.Cluster, func() {
	It("by profile",
		labels.Runtime.Day1,
		func() {
			client := rosacli.NewClient()
			profile := profilehandler.LoadProfileYamlFileByENV()
			cluster, err := profilehandler.CreateClusterByProfile(profile, client, config.Test.GlobalENV.WaitSetupClusterReady)
			Expect(err).ToNot(HaveOccurred())
			log.Logger.Infof("Cluster prepared successfully with id %s", cluster.ID)
		})

	It("to wait for cluster ready",
		labels.Runtime.Day1Readiness,
		func() {
			clusterDetail, err := profilehandler.ParserClusterDetail()
			Expect(err).ToNot(HaveOccurred())
			client := rosacli.NewClient()
			profilehandler.WaitForClusterReady(client, clusterDetail.ClusterID, config.Test.GlobalENV.ClusterWaitingTime)
		})

	It("Create cluster with invalid volume size [id:66372]",
		labels.Medium,
		labels.Runtime.Day1Negative,
		func() {
			minSize := 128
			maxSize := 16384
			// Setup
			client := rosacli.NewClient()
			rosaCommand, err := utilConfig.RetrieveClusterCreationCommand(config.Test.CreateCommandFile)
			Expect(err).To(BeNil())
			if !rosaCommand.CheckFlagExist("--worker-disk-size") {
				rosaCommand.AddFlags("--worker-disk-size", "300GiB")
			}

			var values map[string]string
			var cmd string

			By("Try a worker disk size that's too small")
			values = map[string]string{
				"--worker-disk-size": fmt.Sprintf("%dGiB", minSize-1),
			}
			rosaCommand.ReplaceFlagValue(values)
			cmd = rosaCommand.GetFullCommand()
			out, err := client.Runner.RunCMD([]string{"/bin/sh", "-c", cmd})
			stdout := client.Parser.TextData.Input(out).Parse().Tip()
			Expect(stdout).To(ContainSubstring("Invalid root disk size: %d GiB. Must be between %d GiB and %d GiB.", minSize-1, minSize, maxSize))

			By("Try a worker disk size that's too big")
			values = map[string]string{
				"--worker-disk-size": fmt.Sprintf("%dGiB", maxSize+1),
			}
			rosaCommand.ReplaceFlagValue(values)
			cmd = rosaCommand.GetFullCommand()
			out, err = client.Runner.RunCMD([]string{"/bin/sh", "-c", cmd})
			stdout = client.Parser.TextData.Input(out).Parse().Tip()
			Expect(stdout).To(ContainSubstring("Invalid root disk size: %d GiB. Must be between %d GiB and %d GiB.", maxSize+1, minSize, maxSize))

			By("Try a worker disk size that's negative")
			values = map[string]string{
				"--worker-disk-size": "-1GiB",
			}
			rosaCommand.ReplaceFlagValue(values)
			cmd = rosaCommand.GetFullCommand()
			out, err = client.Runner.RunCMD([]string{"/bin/sh", "-c", cmd})
			stdout = client.Parser.TextData.Input(out).Parse().Tip()
			Expect(stdout).To(ContainSubstring("Expected a valid machine pool root disk size value '-1GiB': invalid disk size: '-1Gi'. positive size required"))

			By("Try a worker disk size that's a string")
			values = map[string]string{
				"--worker-disk-size": "invalid",
			}
			rosaCommand.ReplaceFlagValue(values)
			cmd = rosaCommand.GetFullCommand()
			out, err = client.Runner.RunCMD([]string{"/bin/sh", "-c", cmd})
			stdout = client.Parser.TextData.Input(out).Parse().Tip()
			Expect(stdout).To(ContainSubstring("Expected a valid machine pool root disk size value 'invalid': invalid disk size format: 'invalid'. accepted units are Giga or Tera in the form of g, G, GB, GiB, Gi, t, T, TB, TiB, Ti"))
		})
})
