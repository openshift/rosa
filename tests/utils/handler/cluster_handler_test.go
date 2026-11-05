// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/seaplane"
	"github.com/openshift/rosa/tests/ci/config"
	ClusterConfigure "github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/constants"
)

var _ = Describe("newClusterHandler", func() {
	AfterEach(func() {
		seaplane.Reset()
	})

	It("does not persist state for temporary handlers", func() {
		tempDir := GinkgoT().TempDir()
		clusterConfig := filepath.Join(tempDir, "cluster-config")
		clusterDetail := filepath.Join(tempDir, "cluster-detail.json")
		userData := filepath.Join(tempDir, "resources.json")

		originalClusterConfig := config.Test.ClusterConfigFile
		originalClusterDetail := config.Test.ClusterDetailFile
		originalUserData := config.Test.UserDataFile
		config.Test.ClusterConfigFile = clusterConfig
		config.Test.ClusterDetailFile = clusterDetail
		config.Test.UserDataFile = userData
		DeferCleanup(func() {
			config.Test.ClusterConfigFile = originalClusterConfig
			config.Test.ClusterDetailFile = originalClusterDetail
			config.Test.UserDataFile = originalUserData
		})

		handler, err := NewTempClusterHandler(nil, &Profile{
			Region:        "us-east-1",
			ClusterConfig: &ClusterConfig{},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(handler.(*clusterHandler).saveToFile()).To(Succeed())
		for _, file := range []string{clusterConfig, clusterDetail, userData} {
			_, err := os.Stat(file)
			Expect(os.IsNotExist(err)).To(BeTrue(), "temporary handler wrote %s", file)
		}
	})

	It("does not require shared account credentials for Seaplane resources", func() {
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		tempDir := GinkgoT().TempDir()
		userData := filepath.Join(tempDir, "resources.json")
		Expect(os.WriteFile(
			userData,
			[]byte(`{"additional_principals":"arn:aws:iam::123456789012:role/test-role"}`),
			0o600,
		)).To(Succeed())
		clusterDetail := filepath.Join(tempDir, "cluster-detail.json")
		Expect(os.WriteFile(clusterDetail, []byte("{}"), 0o600)).To(Succeed())
		clusterConfig := filepath.Join(tempDir, "cluster-config.json")
		Expect(os.WriteFile(clusterConfig, []byte("{}"), 0o600)).To(Succeed())

		originalUserData := config.Test.UserDataFile
		originalClusterDetail := config.Test.ClusterDetailFile
		originalClusterConfig := config.Test.ClusterConfigFile
		config.Test.UserDataFile = userData
		config.Test.ClusterDetailFile = clusterDetail
		config.Test.ClusterConfigFile = clusterConfig
		DeferCleanup(func() {
			config.Test.UserDataFile = originalUserData
			config.Test.ClusterDetailFile = originalClusterDetail
			config.Test.ClusterConfigFile = originalClusterConfig
		})

		_, err := NewClusterHandlerFromFilesystem(nil, &Profile{
			Region:        "us-east-1",
			ClusterConfig: &ClusterConfig{AdditionalPrincipals: true},
		})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("classifyClusterState", func() {
	It("recognizes ready state", func() {
		Expect(classifyClusterState(constants.Ready)).To(Equal(clusterStateReady))
	})

	It("recognizes uninstalling as terminal", func() {
		Expect(classifyClusterState(constants.Uninstalling)).To(Equal(clusterStateTerminal))
	})

	It("recognizes error as terminal", func() {
		Expect(classifyClusterState(constants.Error)).To(Equal(clusterStateTerminal))
	})

	It("recognizes waiting state", func() {
		Expect(classifyClusterState(constants.Waiting)).To(Equal(clusterStateWaiting))
	})

	It("recognizes installing as transient", func() {
		Expect(classifyClusterState(constants.Installing)).To(Equal(clusterStateTransient))
	})

	It("recognizes pending as transient", func() {
		Expect(classifyClusterState(constants.Pending)).To(Equal(clusterStateTransient))
	})

	It("recognizes validating as transient", func() {
		Expect(classifyClusterState(constants.Validating)).To(Equal(clusterStateTransient))
	})

	It("returns unknown for unrecognized states", func() {
		Expect(classifyClusterState("hibernating")).To(Equal(clusterStateUnknown))
	})

	It("returns unknown for empty state", func() {
		Expect(classifyClusterState("")).To(Equal(clusterStateUnknown))
	})
})

var _ = Describe("Seaplane node pool setup", func() {
	It("uses configured replicas on every pool", func() {
		n, err := seaplaneNodePoolReplicas(&ClusterConfig{WorkerPoolReplicas: 3}, 3)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(3))
	})

	It("uses the ROSA single-AZ default", func() {
		n, err := seaplaneNodePoolReplicas(&ClusterConfig{}, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(2))
	})

	It("rejects unsupported autoscaling profiles", func() {
		_, err := seaplaneNodePoolReplicas(&ClusterConfig{Autoscale: true}, 1)
		Expect(err).To(MatchError("seaplane e2e setup does not support autoscaled default node pools"))
	})

	It("compacts node pool conditions for timeout diagnostics", func() {
		message := " - Synced: True AllSynced: 1/1 desires applied successfully\n" +
			" - Ready: False NotReady: * Machine workers-0: NodeHealthy: waiting"
		Expect(compactSeaplaneNodePoolConditions(message)).To(Equal(
			"- Synced: True AllSynced: 1/1 desires applied successfully " +
				"- Ready: False NotReady: * Machine workers-0: NodeHealthy: waiting"))
	})

	It("keeps Ready node pools ready when condition details are present", func() {
		phase, summary := seaplaneNodePoolStatus(
			" Ready ",
			" - Synced: True AllSynced: 1/1 desires applied successfully")
		Expect(phase).To(Equal("Ready"))
		Expect(summary).To(Equal(
			"Ready (- Synced: True AllSynced: 1/1 desires applied successfully)"))
	})
})

var _ = Describe("GenerateClusterCreateFlags seaplane", func() {
	AfterEach(func() {
		seaplane.Reset()
	})

	DescribeTable("rejects custom KMS before preparing any resources",
		func(kmsKey, etcdKMS bool) {
			seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
			GinkgoT().Setenv("CLUSTER_NAME", "seaplane-kms-test")
			ch := &clusterHandler{
				profile: &Profile{ClusterConfig: &ClusterConfig{
					KMSKey: kmsKey, EtcdKMS: etcdKMS, STS: true, BYOVPC: true, OIDCConfig: "managed",
				}},
				clusterConfig: &ClusterConfigure.ClusterConfig{},
			}
			flags, err := ch.GenerateClusterCreateFlags()
			Expect(err).To(MatchError(seaplaneCustomKMSError))
			Expect(flags).To(BeEmpty())
		},
		Entry("EBS key", true, false),
		Entry("etcd key", false, true),
		Entry("both keys", true, true),
	)

	DescribeTable("returns an error instead of parsing OCM KMS data on Seaplane",
		func(etcdKMS bool) {
			seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
			ch := &clusterHandler{}
			Expect(ch.elaborateKMSKeyForSTSCluster(etcdKMS)).To(MatchError(seaplaneCustomKMSError))
		},
		Entry("EBS key", false),
		Entry("etcd key", true),
	)

	It("skips NAME_PREFIX when CLUSTER_NAME is set", func() {
		const releaseImage = "quay.io/openshift-release-dev/ocp-release:5.0.0-rc.5-multi"
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		GinkgoT().Setenv("CLUSTER_NAME", "seaplane-e2e-12345")
		GinkgoT().Setenv("SEAPLANE_VERSION", "")
		GinkgoT().Setenv("OCP_IMAGE", releaseImage)
		ch := &clusterHandler{
			profile:          &Profile{Region: "us-east-1", ClusterConfig: &ClusterConfig{}},
			clusterConfig:    &ClusterConfigure.ClusterConfig{},
			clusterDetail:    &ClusterDetail{},
			resourcesHandler: &resourcesHandler{persist: false, resources: &Resources{}},
		}
		flags, err := ch.GenerateClusterCreateFlags()
		Expect(err).ToNot(HaveOccurred())
		Expect(ch.profile.ClusterConfig.Name).To(Equal("seaplane-e2e-12345"))
		Expect(flags).To(ContainElement("seaplane-e2e-12345"))
	})

	It("reuses generated Seaplane flags instead of preparing resources twice", func() {
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		GinkgoT().Setenv("CLUSTER_NAME", "seaplane-e2e-12345")
		GinkgoT().Setenv("SEAPLANE_VERSION", "")
		GinkgoT().Setenv("OCP_IMAGE", "quay.io/openshift-release-dev/ocp-release:5.0.0-rc.5-multi")
		ch := &clusterHandler{
			profile:          &Profile{Region: "us-east-1", ClusterConfig: &ClusterConfig{}},
			clusterConfig:    &ClusterConfigure.ClusterConfig{},
			clusterDetail:    &ClusterDetail{},
			resourcesHandler: &resourcesHandler{persist: false, resources: &Resources{}},
		}

		first, err := ch.GenerateClusterCreateFlags()
		Expect(err).NotTo(HaveOccurred())
		first[0] = "mutated"
		second, err := ch.GenerateClusterCreateFlags()
		Expect(err).NotTo(HaveOccurred())
		Expect(second[0]).To(Equal(rosaAutoConfirmFlag))
	})

	It("passes the release image configured in the profile", func() {
		const releaseImage = "quay.io/openshift-release-dev/ocp-release:5.0.0-rc.5-multi"
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		GinkgoT().Setenv("CLUSTER_NAME", "seaplane-e2e-12345")
		GinkgoT().Setenv("SEAPLANE_VERSION", "")
		GinkgoT().Setenv("OCP_IMAGE", "")
		ch := &clusterHandler{
			profile:          &Profile{Version: releaseImage, Region: "us-east-1", ClusterConfig: &ClusterConfig{}},
			clusterConfig:    &ClusterConfigure.ClusterConfig{},
			clusterDetail:    &ClusterDetail{},
			resourcesHandler: &resourcesHandler{persist: false, resources: &Resources{}},
		}
		flags, err := ch.GenerateClusterCreateFlags()
		Expect(err).ToNot(HaveOccurred())
		Expect(flags).To(ContainElements("--version", releaseImage))
	})

	It("uses OCP_IMAGE when the profile has no release image", func() {
		const releaseImage = "quay.io/openshift-release-dev/ocp-release:5.0.0-rc.5-multi"
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		GinkgoT().Setenv("CLUSTER_NAME", "seaplane-e2e-12345")
		GinkgoT().Setenv("SEAPLANE_VERSION", "")
		GinkgoT().Setenv("OCP_IMAGE", releaseImage)
		ch := &clusterHandler{
			profile:          &Profile{Version: "latest", Region: "us-east-1", ClusterConfig: &ClusterConfig{}},
			clusterConfig:    &ClusterConfigure.ClusterConfig{},
			clusterDetail:    &ClusterDetail{},
			resourcesHandler: &resourcesHandler{persist: false, resources: &Resources{}},
		}

		flags, err := ch.GenerateClusterCreateFlags()
		Expect(err).NotTo(HaveOccurred())
		Expect(flags).To(ContainElements("--version", releaseImage))
		Expect(ch.clusterConfig.Version.VersionRequirement).To(Equal(releaseImage))
	})

	It("fails before cluster creation when no release image is configured", func() {
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		GinkgoT().Setenv("CLUSTER_NAME", "seaplane-e2e-12345")
		GinkgoT().Setenv("SEAPLANE_VERSION", "")
		GinkgoT().Setenv("OCP_IMAGE", "")
		ch := &clusterHandler{
			profile:          &Profile{Version: "latest", Region: "us-east-1", ClusterConfig: &ClusterConfig{}},
			clusterConfig:    &ClusterConfigure.ClusterConfig{},
			clusterDetail:    &ClusterDetail{},
			resourcesHandler: &resourcesHandler{persist: false, resources: &Resources{}},
		}

		flags, err := ch.GenerateClusterCreateFlags()
		Expect(err).To(MatchError(ContainSubstring("set SEAPLANE_VERSION or OCP_IMAGE to a full pullspec")))
		Expect(flags).To(BeNil())
	})

	It("keeps the SEAPLANE_VERSION override in the saved cluster configuration", func() {
		const profileVersion = "quay.io/openshift-release-dev/ocp-release:5.0.0-ec.6-multi"
		const overrideVersion = "quay.io/openshift-release-dev/ocp-release:5.0.0-rc.5-multi"
		seaplane.SetFromFlag("https://example.execute-api.us-east-1.amazonaws.com/prod")
		GinkgoT().Setenv("CLUSTER_NAME", "seaplane-e2e-12345")
		GinkgoT().Setenv("SEAPLANE_VERSION", overrideVersion)
		GinkgoT().Setenv("OCP_IMAGE", "")
		ch := &clusterHandler{
			profile:          &Profile{Version: profileVersion, Region: "us-east-1", ClusterConfig: &ClusterConfig{}},
			clusterConfig:    &ClusterConfigure.ClusterConfig{},
			clusterDetail:    &ClusterDetail{},
			resourcesHandler: &resourcesHandler{persist: false, resources: &Resources{}},
		}

		flags, err := ch.GenerateClusterCreateFlags()
		Expect(err).NotTo(HaveOccurred())
		Expect(flags).To(ContainElements("--version", overrideVersion))
		Expect(ch.clusterConfig.Version.VersionRequirement).To(Equal(overrideVersion))
	})
})
