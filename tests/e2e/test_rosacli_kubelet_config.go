package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Kubeletconfig on Classic cluster",
	labels.Day2,
	labels.FeatureKubeletConfig,
	labels.NonHCPCluster,
	func() {
		defer GinkgoRecover()

		var (
			clusterID      string
			rosaClient     *rosacli.Client
			kubeletService rosacli.KubeletConfigService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			kubeletService = rosaClient.KubeletConfig

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

				Expect(output.String()).To(ContainSubstring("Creating the KubeletConfig for cluster '%s' "+
					"will cause all non-Control Plane nodes to reboot. "+
					"This may cause outages to your applications. Do you wish to continue? (y/N)", clusterID))

				By("Run the command to ignore the warning")
				output, err := kubeletService.CreateKubeletConfig(clusterID, "-y",
					"--pod-pids-limit", "12345")
				Expect(err).ToNot(HaveOccurred())
				defer kubeletService.DeleteKubeletConfig(clusterID, "-y")
				Expect(output.String()).To(ContainSubstring("Successfully created KubeletConfig for cluster '%s'", clusterID))

				By("Describe the kubeletconfig")
				output, err = kubeletService.DescribeKubeletConfig(clusterID)
				Expect(err).ToNot(HaveOccurred())

				kubeletConfig := kubeletService.ReflectKubeletConfig(output)
				Expect(kubeletConfig.PodPidsLimit).To(Equal("12345"))

				By("Create a kubeletconfig with --name again")
				output, err = kubeletService.CreateKubeletConfig(clusterID, "-y",
					"--pod-pids-limit", "12345",
					"--name", "shouldnotwork",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("A KubeletConfig for cluster '%s' already exists. You should edit it via 'rosa edit kubeletconfig'", clusterID))
			})

		It("can update podPidLimit via rosacli will work well - [id:68835]",
			labels.Critical,
			func() {
				By("Edit the kubeletconfig to the cluster before it is created")
				output, _ := rosaClient.KubeletConfig.EditKubeletConfig(clusterID,
					"--pod-pids-limit", "12345")
				Expect(output.String()).To(ContainSubstring("The specified KubeletConfig does not exist for cluster '%s'."+
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
				Expect(output.String()).To(ContainSubstring("Editing the KubeletConfig for cluster '%s' "+
					"will cause all non-Control Plane nodes to reboot. "+
					"This may cause outages to your applications. Do you wish to continue", clusterID))

				By("Run the command to ignore the warning")

				output, err = rosaClient.KubeletConfig.EditKubeletConfig(clusterID, "-y",
					"--pod-pids-limit", "12344")

				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Successfully updated KubeletConfig for cluster '%s'", clusterID))

				By("Describe the kubeletconfig")
				output, err = rosaClient.KubeletConfig.DescribeKubeletConfig(clusterID)
				Expect(err).ToNot(HaveOccurred())

				kubeletConfig := rosaClient.KubeletConfig.ReflectKubeletConfig(output)
				Expect(kubeletConfig.PodPidsLimit).To(Equal("12344"))

				By("Edit with --name and not eixtsing will not work")
				output, err = rosaClient.KubeletConfig.EditKubeletConfig(clusterID, "-y",
					"--pod-pids-limit", "12344",
					"--name", "notexisting",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("Update me if bug fixed"))
			})

		It("can delete podPidLimit via rosacli will work well - [id:68836]",
			labels.Critical,
			func() {

				By("Delete the kubeletconfig from the cluster before it is created")
				output, _ := rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID, "-y")
				Expect(output.String()).To(ContainSubstring("Failed to delete KubeletConfig for cluster '%s'",
					clusterID))

				By("Run the command to create a kubeletconfig to the cluster")
				_, err := rosaClient.KubeletConfig.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "12345",
					"-y")
				Expect(err).ToNot(HaveOccurred())
				defer kubeletService.DeleteKubeletConfig(clusterID, "-y")

				By("Run the command to delete the kubeletconfig from the cluster to check warning")
				output, _ = rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID)
				Expect(output.String()).To(ContainSubstring("Deleting the KubeletConfig for cluster '%s' "+
					"will cause all non-Control Plane nodes to reboot. "+
					"This may cause outages to your applications. Do you wish to continue", clusterID))

				By("Run the command to ignore the warning")
				output, err = rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID, "-y")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).To(ContainSubstring("Successfully deleted KubeletConfig for cluster '%s'", clusterID))

				By("Describe the kubeletconfig")
				output, err = rosaClient.KubeletConfig.DescribeKubeletConfig(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("No custom KubeletConfig exists for cluster '%s'", clusterID))

				By("Create the kubeletconfig again")
				_, err = rosaClient.KubeletConfig.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "12345",
					"--name", "existing",
					"-y")
				Expect(err).ToNot(HaveOccurred())
				defer kubeletService.DeleteKubeletConfig(clusterID, "-y")

				By("Delete with --name and not eixtsing will not work")
				output, err = rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID, "-y",
					"--name", "notexisting",
				)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("Update me if bug fixed"))

			})
	})
var _ = Describe("Kubeletconfig on HCP cluster",
	labels.Day2,
	labels.NonClassicCluster,
	labels.FeatureKubeletConfig,
	func() {
		var (
			clusterID      string
			rosaClient     *rosacli.Client
			kubeletService rosacli.KubeletConfigService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			kubeletService = rosaClient.KubeletConfig
		})

		AfterEach(func() {
			By("Clean the cluster")
			kubes, err := kubeletService.ListKubeletConfigsAndReflect(clusterID)
			Expect(err).NotTo(HaveOccurred())
			for _, kube := range kubes.KubeletConfigs {
				kubeletService.DeleteKubeletConfig(clusterID,
					"--name", kube.Name,
					"-y")
			}

		})
		It("can be created/updated/deleted successfully - [id:73753]",
			labels.High,
			func() {
				By("List the kubeletconfig with not existing cluster")
				out, err := kubeletService.ListKubeletConfigs(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("There are no KubeletConfigs for cluster '%s'", clusterID))

				By("Create kubeletconfig with name specified without flag --name")
				out, err = kubeletService.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "4096",
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("Successfully created KubeletConfig for cluster '%s'", clusterID))

				By("List the kubeletconfig")
				kubes, err := kubeletService.ListKubeletConfigsAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(kubes.KubeletConfigs)).To(Equal(1))
				Expect(kubes.KubeletConfigs[0].Name).To(Equal(fmt.Sprintf("kubelet-%s", kubes.KubeletConfigs[0].ID)))
				Expect(kubes.KubeletConfigs[0].PodPidsLimit).To(Equal("4096"))

				By("Create kubeletconfig with name specified with flag --name")
				name := "kubeletconfig-73753"
				_, err = kubeletService.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "12345",
					"--name", name,
				)
				Expect(err).ToNot(HaveOccurred())

				By("List the kubeletconfig")
				kubes, err = kubeletService.ListKubeletConfigsAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(kubes.KubeletConfigs)).To(Equal(2))
				Expect(kubes.KubeletConfig(name).Name).To(Equal(name))
				Expect(kubes.KubeletConfig(name).PodPidsLimit).To(Equal("12345"))

				By("Edit the kubeletconfig will succeed")
				_, err = kubeletService.EditKubeletConfig(clusterID,
					"--pod-pids-limit", "12346",
					"--name", name,
				)
				Expect(err).ToNot(HaveOccurred())

				By("Describe the kubeletconfig to check")
				out, err = kubeletService.DescribeKubeletConfig(clusterID,
					"--name", name,
				)
				Expect(err).ToNot(HaveOccurred())
				kube := kubeletService.ReflectKubeletConfig(out)
				Expect(kube.PodPidsLimit).To(Equal("12346"))

				By("Delete the kubeletconfig")
				_, err = kubeletService.DeleteKubeletConfig(clusterID,
					"--name", name,
				)
				Expect(err).ToNot(HaveOccurred())

				By("Describe the kubeletconfig again")
				out, err = kubeletService.DescribeKubeletConfig(clusterID,
					"--name", name,
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("The KubeletConfig specified does not exist for cluster '%s'", clusterID))

			})
		It("can validate when create/edit/delete/describe - [id:73754]",
			labels.Medium,
			func() {
				By("List the kubeletconfig with not existing cluster")
				out, err := kubeletService.ListKubeletConfigs("notexisting")
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("There is no cluster with identifier or name '%s'", "notexisting"))

				By("Create kubeletconfig with invalid name")
				out, err = kubeletService.CreateKubeletConfig(clusterID,
					"--name", "'%^&*('",
					"--pod-pids-limit", "4096",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("The name must be a lowercase RFC 1123 subdomain."))

				By("Create kubeletconfig with invalid pod pids limit value like 123456789")
				out, err = kubeletService.CreateKubeletConfig(clusterID,
					"--name", "valid",
					"--pod-pids-limit", "123456789",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("The maximum value for --pod-pids-limit is '16384'. You have supplied '123456789'"))

				By("Create a kubeletconfig")
				dupName := "dupname-73754"
				out, err = kubeletService.CreateKubeletConfig(clusterID,
					"--name", dupName,
					"--pod-pids-limit", "4096",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Create another kubeletconfig with same name")
				out, err = kubeletService.CreateKubeletConfig(clusterID,
					"--name", dupName,
					"--pod-pids-limit", "4096",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("'KubeletConfig with name '%s' already exists'", dupName))

				By("Edit with invalid pod pids limit value")
				out, err = kubeletService.EditKubeletConfig(clusterID,
					"--name", dupName,
					"--pod-pids-limit", "123456789",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("The maximum value for --pod-pids-limit is '16384'. You have supplied '123456789'"))

				By("edit/delete/describe kubeletconfig with not existing")
				out, err = kubeletService.EditKubeletConfig(clusterID,
					"--name", "notexisting",
					"--pod-pids-limit", "4096",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("The specified KubeletConfig does not exist for cluster '%s'", clusterID))

				out, err = kubeletService.DescribeKubeletConfig(clusterID,
					"--name", "notexisting",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("The KubeletConfig specified does not exist for cluster '%s'", clusterID))

				out, err = kubeletService.DeleteKubeletConfig(clusterID,
					"--name", "notexisting",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("The KubeletConfig with name 'notexisting' does not exist on cluster"))
			})
	})
