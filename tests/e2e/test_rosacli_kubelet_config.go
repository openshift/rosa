package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit kubeletconfig",
	labels.Day2,
	labels.FeatureKubeletConfig,
	func() {
		defer GinkgoRecover()

		var (
			clusterID      string
			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
			kubeletService rosacli.KubeletConfigService
			isHosted       bool
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			kubeletService = rosaClient.KubeletConfig
			clusterService = rosaClient.Cluster

			By("Check cluster is hosted")
			var err error
			isHosted, err = clusterService.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			By("Clean the cluster")
			rosaClient.CleanResources(clusterID)
		})

		It("can create podPidLimit via rosacli will work well - [id:68828]",
			labels.Critical,
			func() {
				By("Run the command to create a kubeletconfig to the cluster")
				output, _ := kubeletService.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "12345")
				if isHosted {
					Expect(output.String()).To(ContainSubstring("Hosted Control Plane clusters do not support custom KubeletConfig configuration."))
					return
				}
				Expect(output.String()).To(ContainSubstring("Creating the custom KubeletConfig for cluster '%s' "+
					"will cause all non-Control Plane nodes to reboot. "+
					"This may cause outages to your applications. Do you wish to continue", clusterID))

				By("Check if cluster is hosted control plane cluster")
				isHostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				By("Run the command to ignore the warning")
				output, err = kubeletService.CreateKubeletConfig(clusterID, "-y",
					"--pod-pids-limit", "12345")

				if isHostedCluster {
					Expect(err).To(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("Hosted Control Plane clusters do not support custom KubeletConfig configuration."))
					return
				}
				Expect(err).ToNot(HaveOccurred())
				defer kubeletService.DeleteKubeletConfig(clusterID, "-y")
				Expect(output.String()).To(ContainSubstring("Successfully created custom KubeletConfig for cluster '%s'", clusterID))

				By("Describe the kubeletconfig")
				output, err = kubeletService.DescribeKubeletConfig(clusterID)
				Expect(err).ToNot(HaveOccurred())

				kubeletConfig := kubeletService.ReflectKubeletConfigDescription(output)
				Expect(kubeletConfig.PodPidsLimit).To(Equal(12345))
			})

		It("can update podPidLimit via rosacli will work well - [id:68835]",
			labels.Critical,
			func() {
				By("Edit the kubeletconfig to the cluster before it is created")
				output, _ := rosaClient.KubeletConfig.EditKubeletConfig(clusterID,
					"--pod-pids-limit", "12345")
				if isHosted {
					Expect(output.String()).To(ContainSubstring("Hosted Control Plane clusters do not support KubeletConfig configuration"))
					return
				}
				Expect(output.String()).To(ContainSubstring("No KubeletConfig for cluster '%s' has been found."+
					" You should first create it via 'rosa create kubeletconfig'",
					clusterID))

				By("Run the command to create a kubeletconfig to the cluster")
				_, err := rosaClient.KubeletConfig.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "12345",
					"-y")
				Expect(err).ToNot(HaveOccurred())
				defer kubeletService.DeleteKubeletConfig(clusterID, "-y")

				By("Run the command to edit the kubeletconfig to the cluster to check warning")
				output, _ = rosaClient.KubeletConfig.EditKubeletConfig(clusterID,
					"--pod-pids-limit", "12345")
				Expect(output.String()).To(ContainSubstring("Updating the custom KubeletConfig for cluster '%s' "+
					"will cause all non-Control Plane nodes to reboot. "+
					"This may cause outages to your applications. Do you wish to continue", clusterID))

				By("Run the command to ignore the warning")
				By("Check if cluster is hosted control plane cluster")
				isHostedCluster, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())

				output, err = rosaClient.KubeletConfig.EditKubeletConfig(clusterID, "-y",
					"--pod-pids-limit", "12344")

				if isHostedCluster {
					Expect(err).To(HaveOccurred())
					Expect(output.String()).To(ContainSubstring("Hosted Control Plane clusters do not support custom KubeletConfig configuration."))
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Successfully updated custom KubeletConfig for cluster '%s'", clusterID))

				By("Describe the kubeletconfig")
				output, err = rosaClient.KubeletConfig.DescribeKubeletConfig(clusterID)
				Expect(err).ToNot(HaveOccurred())

				kubeletConfig := rosaClient.KubeletConfig.ReflectKubeletConfigDescription(output)
				Expect(kubeletConfig.PodPidsLimit).To(Equal(12344))
			})

		It("can delete podPidLimit via rosacli will work well - [id:68836]",
			labels.Critical,
			func() {
				By("Check if cluster is hosted control plane cluster")
				if isHosted {
					Skip("Hosted control plane cluster doesn't support the kubeleconfig for now")
				}

				By("Delete the kubeletconfig from the cluster before it is created")
				output, _ := rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID, "-y")
				Expect(output.String()).To(ContainSubstring("Failed to delete custom KubeletConfig for cluster '%s'",
					clusterID))

				By("Run the command to create a kubeletconfig to the cluster")
				_, err := rosaClient.KubeletConfig.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "12345",
					"-y")
				Expect(err).ToNot(HaveOccurred())
				defer kubeletService.DeleteKubeletConfig(clusterID, "-y")

				By("Run the command to delete the kubeletconfig from the cluster to check warning")
				output, _ = rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID)
				Expect(output.String()).To(ContainSubstring("Deleting the custom KubeletConfig for cluster '%s' "+
					"will cause all non-Control Plane nodes to reboot. "+
					"This may cause outages to your applications. Do you wish to continue", clusterID))

				By("Run the command to ignore the warning")
				output, err = rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID, "-y")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Successfully deleted custom KubeletConfig for cluster '%s'", clusterID))

				By("Describe the kubeletconfig")
				output, err = rosaClient.KubeletConfig.DescribeKubeletConfig(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("No custom KubeletConfig exists for cluster '%s'", clusterID))
			})
	})
