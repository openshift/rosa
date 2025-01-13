package e2e

import (
	"strings"

	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Kubeletconfig on Classic cluster",
	labels.Feature.KubeletConfig,
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

			By("Skip testing if the cluster is a HCP cluster")
			isHosted, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if isHosted {
				SkipNotClassic()
			}
		})

		AfterEach(func() {
			By("Clean the cluster")
			rosaClient.CleanResources(clusterID)
		})

		It("can create podPidLimit via rosacli will work well - [id:68828]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Run the command to create a kubeletconfig to the cluster")
				output, _ := kubeletService.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "12345")

				// nolint:goconst
				Expect(output.String()).To(ContainSubstring("Creating the KubeletConfig for cluster '%s' "+
					"will cause all non-Control Plane nodes to reboot. "+
					"This may cause outages to your applications. Do you wish to continue",
					clusterID))

				By("Run the command to ignore the warning")
				output, err := kubeletService.CreateKubeletConfig(clusterID, "-y",
					"--pod-pids-limit", "12345")
				Expect(err).ToNot(HaveOccurred())
				defer kubeletService.DeleteKubeletConfig(clusterID, "-y")
				Expect(output.String()).
					To(ContainSubstring(
						"Successfully created KubeletConfig for cluster '%s'",
						clusterID))

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
				Expect(output.String()).
					To(ContainSubstring(
						"A KubeletConfig for cluster '%s' already exists. You should edit it via 'rosa edit kubeletconfig'",
						clusterID))

			})

		It("can update podPidLimit via rosacli will work well - [id:68835]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Edit the kubeletconfig to the cluster before it is created")
				output, _ := rosaClient.KubeletConfig.EditKubeletConfig(clusterID,
					"--pod-pids-limit", "12345")
				Expect(output.String()).
					To(ContainSubstring(
						"The specified KubeletConfig does not exist for cluster '%s'."+
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
				Expect(output.String()).
					To(ContainSubstring(
						"Successfully updated KubeletConfig for cluster '%s'",
						clusterID))

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
				Expect(output.String()).
					Should(ContainSubstring(
						"ERR: The specified KubeletConfig does not exist for cluster '%s'",
						clusterID))
			})

		It("can delete podPidLimit via rosacli will work well - [id:68836]",
			labels.Critical, labels.Runtime.Day2,
			func() {
				By("Delete the kubeletconfig from the cluster before it is created")
				output, _ := rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID, "-y")
				Expect(output.String()).
					To(ContainSubstring(
						"Failed to delete KubeletConfig for cluster '%s'",
						clusterID))

				By("Run the command to create a kubeletconfig to the cluster")
				_, err := rosaClient.KubeletConfig.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "12345",
					"-y")
				Expect(err).ToNot(HaveOccurred())
				defer kubeletService.DeleteKubeletConfig(clusterID, "-y")

				By("Run the command to delete the kubeletconfig from the cluster to check warning")
				output, _ = rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID)
				Expect(output.String()).
					To(ContainSubstring(
						"Deleting the KubeletConfig for cluster '%s' "+
							"will cause all non-Control Plane nodes to reboot. "+
							"This may cause outages to your applications. Do you wish to continue",
						clusterID))

				By("Run the command to ignore the warning")
				output, err = rosaClient.KubeletConfig.DeleteKubeletConfig(clusterID, "-y")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).
					To(ContainSubstring(
						"Successfully deleted KubeletConfig for cluster '%s'",
						clusterID))

				By("Describe the kubeletconfig")
				output, err = rosaClient.KubeletConfig.DescribeKubeletConfig(clusterID)
				Expect(err).To(HaveOccurred())
				Expect(output.String()).
					Should(ContainSubstring(
						"The KubeletConfig specified does not exist for cluster '%s'",
						clusterID))

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
				Expect(output.String()).
					Should(ContainSubstring(
						"The KubeletConfig with name 'notexisting' does not exist on cluster '%s'",
						clusterID))

			})
	})
var _ = Describe("Kubeletconfig on HCP cluster",
	labels.Feature.KubeletConfig,
	func() {
		var (
			clusterID           string
			rosaClient          *rosacli.Client
			kubeletService      rosacli.KubeletConfigService
			machinePoolService  rosacli.MachinePoolService
			meetThrottleVersion bool
		)

		BeforeEach(func() {
			By("Init the throttle version")
			throttleVersion, _ := semver.NewVersion("4.14.0-a.0")

			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			kubeletService = rosaClient.KubeletConfig
			machinePoolService = rosaClient.MachinePool

			By("Skip testing if the cluster is not a HCP cluster")
			hosted, err := rosaClient.Cluster.IsHostedCPCluster(clusterID)
			Expect(err).ToNot(HaveOccurred())
			if !hosted {
				Skip("Classic kubelet config is covered by 68828")
			}

			clusterDescription, err := rosaClient.Cluster.DescribeClusterAndReflect(clusterID)
			Expect(err).ToNot(HaveOccurred())
			cVersion, err := semver.NewVersion(clusterDescription.OpenshiftVersion)
			Expect(err).ToNot(HaveOccurred())
			meetThrottleVersion = !cVersion.LessThan(throttleVersion)
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
			labels.High, labels.Runtime.Day2,
			func() {
				By("List the kubeletconfig with not existing cluster")
				out, err := kubeletService.ListKubeletConfigs(clusterID)
				if !meetThrottleVersion {
					Expect(err).To(HaveOccurred())
					Expect(out.String()).Should(
						ContainSubstring("KubeletConfig management is only supported on clusters with OCP '4.14' onwards"))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(out.String()).
						Should(ContainSubstring(
							"There are no KubeletConfigs for cluster '%s'",
							clusterID))
				}

				By("Create kubeletconfig with name specified without flag --name")
				out, err = kubeletService.CreateKubeletConfig(clusterID,
					"--pod-pids-limit", "4096",
				)
				if !meetThrottleVersion {
					Expect(err).To(HaveOccurred())
					Expect(out.String()).Should(
						ContainSubstring("KubeletConfig management is only supported on clusters with OCP '4.14' onwards"))
					return
				}

				Expect(err).To(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("Name?"))

				By("Create kubeletconfigs with name specified with flag --name")
				name := "kubeletconfig-73753"
				kubeMap := map[string]string{
					name:                    "12345",
					"kubeletconfig-73753-2": "12346",
				}
				for kubename, podPidValue := range kubeMap {
					_, err = kubeletService.CreateKubeletConfig(clusterID,
						"--pod-pids-limit", podPidValue,
						"--name", kubename,
					)
					Expect(err).ToNot(HaveOccurred())
				}
				By("List the kubeletconfig")
				kubes, err := kubeletService.ListKubeletConfigsAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(kubes.KubeletConfigs)).To(Equal(2))
				for kubename, podPidValue := range kubeMap {
					Expect(kubes.KubeletConfig(kubename).Name).To(Equal(kubename))
					Expect(kubes.KubeletConfig(kubename).PodPidsLimit).To(Equal(podPidValue))
				}

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
				Expect(out.String()).
					Should(ContainSubstring(
						"The KubeletConfig specified does not exist for cluster '%s'",
						clusterID))

			})
		It("can validate when create/edit/delete/describe - [id:73754]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("List the kubeletconfig with not existing cluster")
				out, err := kubeletService.ListKubeletConfigs("notexisting")
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"There is no cluster with identifier or name '%s'",
						"notexisting"))

				By("Create kubeletconfig with invalid name")
				out, err = kubeletService.CreateKubeletConfig(clusterID,
					"--name", "'%^&*('",
					"--pod-pids-limit", "4096",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"The name must be a lowercase RFC 1123 subdomain"))

				By("Create kubeletconfig with invalid pod pids limit value like 123456789")
				out, err = kubeletService.CreateKubeletConfig(clusterID,
					"--name", "valid",
					"--pod-pids-limit", "123456789",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"The maximum value for --pod-pids-limit is '16384'. You have supplied '123456789'"))

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
				Expect(out.String()).
					Should(ContainSubstring(
						"KubeletConfig with name '%s' already exists", dupName))

				By("Edit with invalid pod pids limit value")
				out, err = kubeletService.EditKubeletConfig(clusterID,
					"--name", dupName,
					"--pod-pids-limit", "123456789",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"The maximum value for --pod-pids-limit is '16384'. You have supplied '123456789'"))

				By("edit/delete/describe kubeletconfig with not existing")
				out, err = kubeletService.EditKubeletConfig(clusterID,
					"--name", "notexisting",
					"--pod-pids-limit", "4096",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"The specified KubeletConfig does not exist for cluster '%s'",
						clusterID))

				out, err = kubeletService.DescribeKubeletConfig(clusterID,
					"--name", "notexisting",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"The KubeletConfig specified does not exist for cluster '%s'",
						clusterID))

				out, err = kubeletService.DeleteKubeletConfig(clusterID,
					"--name", "notexisting",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"The KubeletConfig with name 'notexisting' does not exist on cluster"))
			})

		It("can be attach to machinepool successfully - [id:73765]",
			labels.High, labels.Runtime.Day2,
			func() {
				By("Skip the test if cluster version is lower than the supported throttle version")
				if !meetThrottleVersion {
					SkipTestOnFeature("kubelet-config")
				}

				By("Prepare kubeletconfigs")
				kubeName1, kubeName2 := "kube-73765", "kube-73765-2"
				for name, pidValue := range map[string]string{
					kubeName1: "4096",
					kubeName2: "16384",
				} {
					_, err := kubeletService.CreateKubeletConfig(clusterID,
						"--name", name,
						"--pod-pids-limit", pidValue,
					)
					Expect(err).ToNot(HaveOccurred())
				}

				By("Check the help message")
				out, err := machinePoolService.CreateMachinePool(clusterID, "fake", "-h")
				Expect(err).ToNot(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("--kubelet-configs"))

				By("Create a machinepool with  --kubelet-configs set")
				mpName := "mp-73765"
				_, err = machinePoolService.CreateMachinePool(clusterID, mpName,
					"--replicas", "0",
					"--kubelet-configs", kubeName1,
				)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, mpName)

				By("Describe the machinepool")
				mpD, err := machinePoolService.DescribeAndReflectNodePool(clusterID, mpName)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpD.KubeletConfigs).Should(Equal(kubeName1))

				By("Check the machinepool edit command")
				out, err = machinePoolService.EditMachinePool(clusterID, "fake", "-h")
				Expect(err).ToNot(HaveOccurred())
				Expect(out.String()).Should(ContainSubstring("--kubelet-configs"))

				By("Edit above machinepool with another kubeletconfig")
				out, err = machinePoolService.EditMachinePool(clusterID, mpName,
					"--kubelet-configs", kubeName2,
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Describe the machinepool and check the output")
				mpD, err = machinePoolService.DescribeAndReflectNodePool(clusterID, mpName)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpD.KubeletConfigs).Should(Equal(kubeName2))

				By("Update to no kubeletconfig")
				out, err = machinePoolService.EditMachinePool(clusterID, mpName,
					"--kubelet-configs", "",
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Check the machinepool description")
				mpD, err = machinePoolService.DescribeAndReflectNodePool(clusterID, mpName)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpD.KubeletConfigs).Should(BeEmpty())

				By("Attach the kubeletconfig again")
				out, err = machinePoolService.EditMachinePool(clusterID, mpName,
					"--kubelet-configs", kubeName2,
					"-y",
				)
				Expect(err).ToNot(HaveOccurred())

				By("Describe the machinepool and check the output")
				mpD, err = machinePoolService.DescribeAndReflectNodePool(clusterID, mpName)
				Expect(err).ToNot(HaveOccurred())
				Expect(mpD.KubeletConfigs).Should(Equal(kubeName2))

			})

		It("can validate well when attach to machinepool - [id:73766]",
			labels.Medium, labels.Runtime.Day2,
			func() {
				By("Prepare kubeletconfigs")
				kubeName1, kubeName2 := "kube-73766", "kube-73766-2"
				for name, pidValue := range map[string]string{
					kubeName1: "4096",
					kubeName2: "16384",
				} {
					_, err := kubeletService.CreateKubeletConfig(clusterID,
						"--name", name,
						"--pod-pids-limit", pidValue,
					)
					Expect(err).ToNot(HaveOccurred())
				}

				By("Create machinepool with multiple kubelet config ids")
				out, err := machinePoolService.CreateMachinePool(clusterID, "multi-kubes-2",
					"--replicas", "0",
					"--kubelet-configs", strings.Join([]string{kubeName1, kubeName2}, ","),
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"Only a single kubelet config is supported for Machine Pools"))

				By("Create machinepool with not existing kubeletconfig name")
				out, err = machinePoolService.CreateMachinePool(clusterID, "unknown",
					"--replicas", "0",
					"--kubelet-configs", "notexisting",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"KubeletConfig with name '%s' does not exist for cluster", "notexisting"))

				By("Create a machinepool with existing kubeletconfig")
				mpName := "mp-73766"
				out, err = machinePoolService.CreateMachinePool(clusterID, mpName,
					"--replicas", "0",
					"--kubelet-configs", kubeName1,
				)
				Expect(err).ToNot(HaveOccurred())
				defer machinePoolService.DeleteMachinePool(clusterID, mpName)

				By("Delete the kubeletconfig")
				out, err = kubeletService.DeleteKubeletConfig(clusterID,
					"--name", kubeName1)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"cannot be updated or deleted. It is referenced in the following node pools"))

				By("Edit the kubeletconfig")
				out, err = kubeletService.EditKubeletConfig(clusterID,
					"--name", kubeName1,
					"--pod-pids-limit", "12345")
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"cannot be updated or deleted. It is referenced in the following node pools"))

				By("Edit machinepool with multiple kubelet config ids")
				out, err = machinePoolService.EditMachinePool(clusterID, mpName,
					"--kubelet-configs", strings.Join([]string{kubeName1, kubeName2}, ","),
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"Only a single kubelet config is supported for Machine Pools"))

				By("Edit machinepool with not existing kubeletconfig name")
				out, err = machinePoolService.EditMachinePool(clusterID, mpName,
					"--kubelet-configs", "nonexisting",
					"-y",
				)
				Expect(err).To(HaveOccurred())
				Expect(out.String()).
					Should(ContainSubstring(
						"KubeletConfig with name '%s' does not exist for cluster", "nonexisting"))
			})
	})
