// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
	rosaconfig "github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
	"github.com/openshift/rosa/tests/utils/helper"
)

func requireSeaplaneEnvironment() {
	if os.Getenv("SEAPLANE_URL") != "" {
		return
	}

	cliConfig, err := rosaconfig.Load()
	Expect(err).NotTo(HaveOccurred())
	if cliConfig == nil || cliConfig.SeaplaneURL == "" {
		Skip("Set SEAPLANE_URL or configure a Seaplane endpoint before running this test")
	}
}

func tagsFromJSON(value interface{}) (map[string]string, error) {
	if value == nil {
		return map[string]string{}, nil
	}

	var tags map[string]string
	switch raw := value.(type) {
	case map[string]interface{}:
		tags = make(map[string]string, len(raw))
		for key, value := range raw {
			tagValue, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("AWS tag %q has non-string value %T", key, value)
			}
			tags[key] = tagValue
		}
	case map[string]string:
		tags = raw
	default:
		return nil, fmt.Errorf("expected AWS tags object, got %T", value)
	}

	return tags, nil
}

var _ = Describe("Seaplane Cluster AWS tags", labels.Feature.Cluster, func() {
	var (
		clusterID  string
		rosaClient *rosacli.Client
		profile    *handler.Profile
	)

	BeforeEach(func() {
		requireSeaplaneEnvironment()
		clusterID = config.GetClusterID()
		Expect(clusterID).NotTo(BeEmpty(), "ClusterID is required. Please export CLUSTER_ID")

		rosaClient = rosacli.NewClient()
		profile = handler.LoadProfileYamlFileByENV()
		Expect(profile.ClusterConfig.HCP).To(BeTrue(), "TEST_PROFILE must configure a hosted control plane")
		hosted, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
		Expect(err).NotTo(HaveOccurred())
		if !hosted {
			Skip("The test profile is not hosted")
		}
	})

	AfterEach(func() {
		if rosaClient != nil {
			Expect(rosaClient.CleanResources(clusterID)).To(Succeed())
		}
	})

	It("updates Seaplane Cluster AWS tags on Day 2 - [id:ROSAENG-65296]",
		labels.High, labels.Runtime.Day2, labels.Seaplane.Validated, func() {
			readTags := func() (map[string]string, error) {
				description, err := rosaClient.Cluster.GetJSONClusterDescription(clusterID)
				if err != nil {
					return nil, err
				}
				return tagsFromJSON(description.DigObject("aws", "tags"))
			}

			By("Capture the existing Cluster tags so they can be restored")
			originalTags, err := readTags()
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				By("Restore the original Cluster tags")
				_, restoreErr := rosaClient.Cluster.EditCluster(
					clusterID, "--tags", aws.FormatUserTagsList(originalTags), "-y")
				Expect(restoreErr).NotTo(HaveOccurred())
				Eventually(readTags, 2*time.Minute, 5*time.Second).Should(Equal(originalTags))
			})

			updatedTags := map[string]string{"rosa-e2e-day2": "cluster-updated"}
			By("Update the Cluster AWS tags")
			_, err = rosaClient.Cluster.EditCluster(
				clusterID, "--tags", aws.FormatUserTagsList(updatedTags), "-y")
			Expect(err).NotTo(HaveOccurred())

			By("Verify the updated tags in Cluster JSON describe")
			Eventually(readTags, 2*time.Minute, 5*time.Second).Should(Equal(updatedTags))
		})
})

var _ = Describe("Seaplane NodePool AWS tags", labels.Feature.Machinepool, func() {
	var (
		clusterID  string
		rosaClient *rosacli.Client
		profile    *handler.Profile
	)

	BeforeEach(func() {
		requireSeaplaneEnvironment()
		clusterID = config.GetClusterID()
		Expect(clusterID).NotTo(BeEmpty(), "ClusterID is required. Please export CLUSTER_ID")

		rosaClient = rosacli.NewClient()
		profile = handler.LoadProfileYamlFileByENV()
		Expect(profile.ClusterConfig.HCP).To(BeTrue(), "TEST_PROFILE must configure a hosted control plane")
		hosted, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
		Expect(err).NotTo(HaveOccurred())
		if !hosted {
			Skip("The test profile is not hosted")
		}
	})

	AfterEach(func() {
		if rosaClient != nil {
			Expect(rosaClient.CleanResources(clusterID)).To(Succeed())
		}
	})

	It("updates Seaplane NodePool AWS tags on Day 2 - [id:ROSAENG-65296]",
		labels.High, labels.Runtime.Day2, labels.Seaplane.Validated, func() {
			By("Select the subnet and instance type from an existing NodePool")
			existingPools, err := rosaClient.MachinePool.ListAndReflectNodePools(clusterID)
			Expect(err).NotTo(HaveOccurred())
			Expect(existingPools.NodePools).NotTo(BeEmpty())
			templatePool := existingPools.NodePools[0]
			Expect(templatePool).NotTo(BeNil())
			Expect(templatePool.Subnet).NotTo(BeEmpty())
			Expect(templatePool.InstanceType).NotTo(BeEmpty())

			poolName := helper.GenerateRandomName("np-tags", 2)
			initialTags := map[string]string{"rosa-e2e-day2": "nodepool-before"}
			updatedTags := map[string]string{"rosa-e2e-day2": "nodepool-updated"}

			readTags := func() (map[string]string, error) {
				rosaClient.Runner.JsonFormat()
				defer rosaClient.Runner.UnsetFormat()

				output, err := rosaClient.MachinePool.DescribeMachinePool(clusterID, poolName)
				if err != nil {
					return nil, err
				}
				description := rosaClient.Parser.JsonData.Input(output).Parse()
				return tagsFromJSON(description.DigObject("tags"))
			}

			By("Create a temporary zero-replica NodePool with initial tags")
			_, err = rosaClient.MachinePool.CreateMachinePool(clusterID, poolName,
				"--replicas", "0",
				"--subnet", templatePool.Subnet,
				"--instance-type", templatePool.InstanceType,
				"--tags", aws.FormatUserTagsList(initialTags))
			Expect(err).NotTo(HaveOccurred())

			By("Verify the initial NodePool tags")
			Eventually(readTags, 2*time.Minute, 5*time.Second).Should(Equal(initialTags))

			By("Update the NodePool AWS tags")
			_, err = rosaClient.MachinePool.EditMachinePool(clusterID, poolName,
				"--tags", aws.FormatUserTagsList(updatedTags))
			Expect(err).NotTo(HaveOccurred())

			By("Verify the updated tags in NodePool JSON describe")
			Eventually(readTags, 2*time.Minute, 5*time.Second).Should(Equal(updatedTags))
		})
})
