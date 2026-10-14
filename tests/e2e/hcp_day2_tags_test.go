// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"slices"
	"sort"
	"time"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/test/vpc_client"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/tagging"
	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
	"github.com/openshift/rosa/tests/utils/handler"
)

var _ = Describe("HCP day-2 AWS tags", labels.Runtime.Day2, func() {
	// TODO(ROSAENG-74663): Extend the CI fixture to validate tag deletion from
	// EC2 worker instances once AWS introduces the updated ROSANodePoolManagementPolicy
	// with ec2:DeleteTags. Until then, removal and clear tests assert OCM state only.
	// Separately, cover cluster-tag propagation to node pools after the API enables it.
	var cli *rosacli.Client
	var runtime *rosa.Runtime
	var clusterKey string
	var supported bool
	var hosted bool

	BeforeEach(func() {
		clusterKey = config.GetClusterID()
		Expect(clusterKey).NotTo(BeEmpty(), "CLUSTER_ID is required")
		cli = rosacli.NewClient()
		description, err := cli.Cluster.GetJSONClusterDescription(clusterKey)
		Expect(err).NotTo(HaveOccurred(), "describe cluster %s", clusterKey)
		hosted = description.DigBool("hypershift", "enabled")
		version := description.DigString("version", "raw_id")
		supported, err = tagging.SupportsControlPlaneVersion(version)
		Expect(err).NotTo(HaveOccurred(), "parse control-plane version %q", version)
		runtime = rosa.NewRuntime().WithOCM()
		DeferCleanup(runtime.Cleanup)
	})

	It("adds, updates, and removes a cluster tag in OCM",
		labels.Feature.Cluster, labels.High, func() {
			if !hosted || !supported {
				Skip("requires a ROSA HCP 4.20+ fixture")
			}
			original, err := runtime.OCMClient.GetClusterUserTags(clusterKey)
			Expect(err).NotTo(HaveOccurred(), "read original cluster customer tags")
			DeferCleanup(func() {
				out, restoreErr := cli.Cluster.EditCluster(clusterKey,
					"--tags", aws.FormatUserTagsList(original))
				Expect(restoreErr).NotTo(HaveOccurred(), "restore cluster tags: %s", out.String())
			})

			const key = "ROSA Day2 Cluster"
			for _, value := range []string{"Finance Team", "Platform Team"} {
				desired := copyTagSet(original)
				desired[key] = value
				out, editErr := cli.Cluster.EditCluster(clusterKey,
					"--tags", aws.FormatUserTagsList(desired))
				Expect(editErr).NotTo(HaveOccurred(), "set cluster tag %q=%q: %s", key, value, out.String())
				Eventually(func() (map[string]string, error) {
					return runtime.OCMClient.GetClusterUserTags(clusterKey)
				}, 5*time.Minute, 15*time.Second).Should(Equal(desired),
					"OCM cluster tags should contain %q=%q", key, value)
			}

			out, err := cli.Cluster.EditCluster(clusterKey,
				"--tags", aws.FormatUserTagsList(original))
			Expect(err).NotTo(HaveOccurred(), "remove temporary cluster tag %q: %s", key, out.String())
			Eventually(func() (map[string]string, error) {
				return runtime.OCMClient.GetClusterUserTags(clusterKey)
			}, 5*time.Minute, 15*time.Second).Should(Equal(original),
				"OCM cluster tags should match original set after omitting %q", key)
		})

	It("clears all customer cluster tags in OCM",
		labels.Feature.Cluster, labels.High, func() {
			if !hosted || !supported {
				Skip("requires a ROSA HCP 4.20+ fixture")
			}
			original, err := runtime.OCMClient.GetClusterUserTags(clusterKey)
			Expect(err).NotTo(HaveOccurred(), "read original cluster customer tags before clear")
			DeferCleanup(func() {
				out, restoreErr := cli.Cluster.EditCluster(clusterKey,
					"--tags", aws.FormatUserTagsList(original))
				Expect(restoreErr).NotTo(HaveOccurred(), "restore cleared cluster tags: %s", out.String())
			})

			const key = "ROSA Day2 Cluster Clear"
			desired := copyTagSet(original)
			desired[key] = "temporary"
			out, err := cli.Cluster.EditCluster(clusterKey, "--tags", aws.FormatUserTagsList(desired))
			Expect(err).NotTo(HaveOccurred(), "seed cluster tag before clearing: %s", out.String())
			Eventually(func() (map[string]string, error) {
				return runtime.OCMClient.GetClusterUserTags(clusterKey)
			}, 5*time.Minute, 15*time.Second).Should(Equal(desired), "seeded cluster tags should persist")

			out, err = cli.Cluster.EditCluster(clusterKey, "--tags", "")
			Expect(err).NotTo(HaveOccurred(), "clear all customer cluster tags: %s", out.String())
			Eventually(func() (map[string]string, error) {
				return runtime.OCMClient.GetClusterUserTags(clusterKey)
			}, 5*time.Minute, 15*time.Second).Should(BeEmpty(), "OCM cluster customer tags should be empty")
		})

	It("adds and updates pool tags on existing workers and removes overrides in OCM",
		labels.Feature.Machinepool, labels.High, func() {
			if !hosted || !supported {
				Skip("requires a ROSA HCP 4.20+ fixture")
			}
			vpc, poolID, workerIDs := day2TagWorkers(cli, clusterKey)
			clusterTags, err := runtime.OCMClient.GetClusterUserTags(clusterKey)
			Expect(err).NotTo(HaveOccurred(), "read inherited cluster customer tags")
			poolTags, err := runtime.OCMClient.GetNodePoolUserTags(clusterKey, poolID)
			Expect(err).NotTo(HaveOccurred(), "read original pool customer tags for %s", poolID)
			original := tagging.PoolOverrides(clusterTags, poolTags)
			DeferCleanup(func() {
				out, restoreErr := cli.MachinePool.EditMachinePool(clusterKey, poolID,
					"--tags", aws.FormatUserTagsList(original))
				Expect(restoreErr).NotTo(HaveOccurred(), "restore pool overrides: %s", out.String())
			})

			const key = "ROSA Day2 Pool"
			for _, value := range []string{"Finance Team", "Platform Team"} {
				desired := copyTagSet(original)
				desired[key] = value
				out, editErr := cli.MachinePool.EditMachinePool(clusterKey, poolID,
					"--tags", aws.FormatUserTagsList(desired))
				Expect(editErr).NotTo(HaveOccurred(), "set pool tag %q=%q: %s", key, value, out.String())
				Eventually(func() (map[string]string, error) {
					current, readErr := runtime.OCMClient.GetNodePoolUserTags(clusterKey, poolID)
					return tagging.PoolOverrides(clusterTags, current), readErr
				}, 5*time.Minute, 15*time.Second).Should(Equal(desired),
					"OCM pool overrides should contain %q=%q", key, value)
				expectDay2WorkerTag(vpc, clusterKey, poolID, workerIDs, key, value)
			}

			out, err := cli.MachinePool.EditMachinePool(clusterKey, poolID,
				"--tags", aws.FormatUserTagsList(original))
			Expect(err).NotTo(HaveOccurred(), "remove temporary pool tag %q: %s", key, out.String())
			Eventually(func() (map[string]string, error) {
				current, readErr := runtime.OCMClient.GetNodePoolUserTags(clusterKey, poolID)
				return tagging.PoolOverrides(clusterTags, current), readErr
			}, 5*time.Minute, 15*time.Second).Should(Equal(original),
				"OCM pool overrides should match original set after omitting %q", key)
		})

	It("clears pool overrides in OCM while retaining inherited cluster tags",
		labels.Feature.Machinepool, labels.High, func() {
			if !hosted || !supported {
				Skip("requires a ROSA HCP 4.20+ fixture")
			}
			pools, err := cli.MachinePool.ListAndReflectNodePools(clusterKey)
			Expect(err).NotTo(HaveOccurred(), "list node pools before clearing overrides")
			Expect(pools.NodePools).NotTo(BeEmpty(), "fixture needs a node pool to clear overrides")
			poolID := pools.NodePools[0].ID
			clusterTags, err := runtime.OCMClient.GetClusterUserTags(clusterKey)
			Expect(err).NotTo(HaveOccurred(), "read inherited cluster customer tags before pool clear")
			poolTags, err := runtime.OCMClient.GetNodePoolUserTags(clusterKey, poolID)
			Expect(err).NotTo(HaveOccurred(), "read original pool customer tags before clear")
			original := tagging.PoolOverrides(clusterTags, poolTags)
			DeferCleanup(func() {
				out, restoreErr := cli.MachinePool.EditMachinePool(clusterKey, poolID,
					"--tags", aws.FormatUserTagsList(original))
				Expect(restoreErr).NotTo(HaveOccurred(), "restore cleared pool overrides: %s", out.String())
			})

			const key = "ROSA Day2 Pool Clear"
			desired := copyTagSet(original)
			desired[key] = "temporary"
			out, err := cli.MachinePool.EditMachinePool(clusterKey, poolID,
				"--tags", aws.FormatUserTagsList(desired))
			Expect(err).NotTo(HaveOccurred(), "seed pool tag before clearing: %s", out.String())
			Eventually(func() (map[string]string, error) {
				current, readErr := runtime.OCMClient.GetNodePoolUserTags(clusterKey, poolID)
				return tagging.PoolOverrides(clusterTags, current), readErr
			}, 5*time.Minute, 15*time.Second).Should(Equal(desired), "seeded pool overrides should persist")

			out, err = cli.MachinePool.EditMachinePool(clusterKey, poolID, "--tags", "")
			Expect(err).NotTo(HaveOccurred(), "clear all pool overrides: %s", out.String())
			Eventually(func() (map[string]string, error) {
				return runtime.OCMClient.GetNodePoolUserTags(clusterKey, poolID)
			}, 5*time.Minute, 15*time.Second).Should(Equal(clusterTags),
				"OCM pool tags should contain only inherited cluster tags after clearing overrides")
		})

	It("rejects cluster and machine-pool tags on classic clusters", labels.Feature.Cluster,
		labels.Feature.Machinepool, labels.Medium, func() {
			if hosted {
				Skip("requires a classic ROSA fixture")
			}
			out, editErr := cli.Cluster.EditCluster(clusterKey, "--tags", "day2:test")
			Expect(editErr).To(HaveOccurred(), "reject classic cluster tag edit: %s", out.String())
			Expect(out.String()).To(ContainSubstring("Hosted Control Plane"),
				"classic cluster error should explain HCP requirement")
			pools, listErr := cli.MachinePool.ListAndReflectMachinePools(clusterKey)
			Expect(listErr).NotTo(HaveOccurred(), "list classic machine pools")
			Expect(pools.MachinePools).NotTo(BeEmpty(), "classic fixture needs a machine pool")
			out, editErr = cli.MachinePool.EditMachinePool(clusterKey, pools.MachinePools[0].ID,
				"--tags", "day2:test")
			Expect(editErr).To(HaveOccurred(), "reject classic machine-pool tag edit: %s", out.String())
			Expect(out.String()).To(ContainSubstring("Hosted Control Plane"),
				"classic pool error should explain HCP requirement")
		})

	It("rejects cluster and node-pool tags before HCP 4.20", labels.Feature.Cluster,
		labels.Feature.Machinepool, labels.Medium, func() {
			if !hosted || supported {
				Skip("requires a ROSA HCP pre-4.20 fixture")
			}
			pools, err := cli.MachinePool.ListAndReflectNodePools(clusterKey)
			Expect(err).NotTo(HaveOccurred(), "list node pools on pre-4.20 fixture")
			Expect(pools.NodePools).NotTo(BeEmpty(), "pre-4.20 fixture needs a node pool")
			out, editErr := cli.Cluster.EditCluster(clusterKey, "--tags", "day2:test")
			Expect(editErr).To(HaveOccurred(), "reject pre-4.20 cluster tag edit: %s", out.String())
			Expect(out.String()).To(ContainSubstring("4.20.0"),
				"cluster error should state minimum supported version")
			out, editErr = cli.MachinePool.EditMachinePool(clusterKey, pools.NodePools[0].ID,
				"--tags", "day2:test")
			Expect(editErr).To(HaveOccurred(), "reject pre-4.20 node-pool tag edit: %s", out.String())
			Expect(out.String()).To(ContainSubstring("4.20.0"),
				"node-pool error should state minimum supported version")
		})

	It("rejects protected cluster and node-pool tag keys", labels.Feature.Cluster,
		labels.Feature.Machinepool, labels.Medium, func() {
			if !hosted || !supported {
				Skip("requires a ROSA HCP 4.20+ fixture")
			}
			pools, err := cli.MachinePool.ListAndReflectNodePools(clusterKey)
			Expect(err).NotTo(HaveOccurred(), "list node pools for protected-tag test")
			Expect(pools.NodePools).NotTo(BeEmpty(), "protected-tag fixture needs a node pool")
			out, editErr := cli.Cluster.EditCluster(clusterKey, "--tags", "red-hat-managed:bad")
			Expect(editErr).To(HaveOccurred(), "reject protected cluster tag: %s", out.String())
			out, editErr = cli.MachinePool.EditMachinePool(clusterKey, pools.NodePools[0].ID,
				"--tags", `"kubernetes.io/cluster/test":bad`)
			Expect(editErr).To(HaveOccurred(), "reject protected node-pool tag: %s", out.String())
		})
})

// copyTagSet keeps the fixture's original customer tags separate from each desired edit.
func copyTagSet(tags map[string]string) map[string]string {
	copy := make(map[string]string, len(tags))
	for key, value := range tags {
		copy[key] = value
	}
	return copy
}

// day2TagWorkers chooses a pool with workers so tag and identity assertions cannot pass vacuously.
func day2TagWorkers(cli *rosacli.Client, clusterKey string) (*vpc_client.VPC, string, []string) {
	pools, err := cli.MachinePool.ListAndReflectNodePools(clusterKey)
	Expect(err).NotTo(HaveOccurred(), "list node pools for cluster %s", clusterKey)
	Expect(pools.NodePools).NotTo(BeEmpty(), "cluster %s needs a node pool", clusterKey)
	profile := handler.LoadProfileYamlFileByENV()
	vpc, err := vpc_client.GenerateVPCBySubnet(pools.NodePools[0].Subnet, profile.Region)
	Expect(err).NotTo(HaveOccurred(), "create AWS VPC client for cluster %s", clusterKey)
	for _, pool := range pools.NodePools {
		instances, listErr := vpc.AWSClient.GetInstancesByNodePoolID(pool.ID, clusterKey)
		Expect(listErr).NotTo(HaveOccurred(), "list worker instances for pool %s", pool.ID)
		if len(instances) > 0 {
			return vpc, pool.ID, instanceIDs(instances)
		}
	}
	Skip("requires a node pool with worker instances")
	return nil, "", nil
}

// instanceIDs returns sorted EC2 instance IDs for rollout detection.
func instanceIDs(instances []ec2types.Instance) []string {
	ids := make([]string, 0, len(instances))
	for _, instance := range instances {
		Expect(instance.InstanceId).NotTo(BeNil(), "EC2 worker response must contain an instance ID")
		ids = append(ids, *instance.InstanceId)
	}
	sort.Strings(ids)
	return ids
}

// expectDay2WorkerTag waits for EC2 tag reconciliation while checking that worker IDs stay fixed.
func expectDay2WorkerTag(vpc *vpc_client.VPC, clusterKey, poolID string,
	workerIDs []string, key, expected string) {
	Eventually(func() error {
		instances, err := vpc.AWSClient.GetInstancesByNodePoolID(poolID, clusterKey)
		if err != nil {
			return err
		}
		if !slices.Equal(instanceIDs(instances), workerIDs) {
			return fmt.Errorf("worker instance identities changed during tag edit")
		}
		for _, instance := range instances {
			actual := ""
			found := false
			for _, tag := range instance.Tags {
				if tag.Key != nil && *tag.Key == key {
					found = true
					if tag.Value != nil {
						actual = *tag.Value
					}
				}
			}
			if (expected == "" && found) || (expected != "" && (!found || actual != expected)) {
				return fmt.Errorf("instance %s tag %s is %q, expected %q",
					*instance.InstanceId, key, actual, expected)
			}
		}
		return nil
	}, 5*time.Minute, 15*time.Second).Should(Succeed(),
		"pool %s workers should retain their IDs and have tag %q=%q within five minutes", poolID, key, expected)
}
