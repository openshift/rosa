package e2e

import (
	"fmt"
	"io"
	nets "net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/common"
	"github.com/openshift/rosa/tests/utils/common/constants"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Verify",
	labels.Day1Post,
	func() {
		defer GinkgoRecover()
		var (
			clusterID          string
			rosaClient         *rosacli.Client
			clusterService     rosacli.ClusterService
			machinePoolService rosacli.MachinePoolService
			clusterConfig      *config.ClusterConfig
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
			machinePoolService = rosaClient.MachinePool
			var err error
			clusterConfig, err = config.ParseClusterProfile()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			By("Clean the cluster")
			rosaClient.CleanResources(clusterID)
		})

		It("the creation of rosa cluster with volume size will work - [id:66359]",
			labels.Critical,
			func() {
				By("Classic cluster check")
				isHosted, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if isHosted {
					Skip("This case is only working for classic right now")
				}

				alignDiskSize := func(diskSize string) string {
					aligned := strings.Join(strings.Split(diskSize, " "), "")
					return aligned
				}

				By("Set expected worker pool size")
				expectedDiskSize := clusterConfig.WorkerDiskSize
				if expectedDiskSize == "" {
					expectedDiskSize = "300GiB" // if no worker disk size set, it will use default value
				}

				By("Check the machinepool list")
				output, err := machinePoolService.ListMachinePool(clusterID)
				Expect(err).ToNot(HaveOccurred())

				mplist, err := machinePoolService.ReflectMachinePoolList(output)
				Expect(err).ToNot(HaveOccurred())

				workPool := mplist.Machinepool(constants.DefaultClassicWorkerPool)
				Expect(workPool).ToNot(BeNil(), "worker pool is not found for the cluster")
				Expect(alignDiskSize(workPool.DiskSize)).To(Equal(expectedDiskSize))

				By("Check the default worker pool description")
				output, err = machinePoolService.DescribeMachinePool(clusterID, constants.DefaultClassicWorkerPool)
				Expect(err).ToNot(HaveOccurred())
				mpD, err := machinePoolService.ReflectMachinePoolDescription(output)
				Expect(err).ToNot(HaveOccurred())
				Expect(alignDiskSize(mpD.DiskSize)).To(Equal(expectedDiskSize))

			})

		It("the creation of ROSA cluster with default-mp-labels option will succeed - [id:57056]",
			labels.Critical,
			func() {
				By("Classic cluster check")
				isHosted, err := clusterService.IsHostedCPCluster(clusterID)
				Expect(err).ToNot(HaveOccurred())
				if isHosted {
					Skip("This case is only working for classic right now")
				}

				By("Check the cluster config")
				mpLables := strings.Split(clusterConfig.DefaultMpLabels, ",")

				By("Check the machinepool list")
				output, err := machinePoolService.ListMachinePool(clusterID)
				Expect(err).ToNot(HaveOccurred())

				mplist, err := machinePoolService.ReflectMachinePoolList(output)
				Expect(err).ToNot(HaveOccurred())

				workPool := mplist.Machinepool(constants.DefaultClassicWorkerPool)
				Expect(workPool).ToNot(BeNil(), "worker pool is not found for the cluster")
				for _, label := range mpLables {
					Expect(workPool.Labels).To(ContainSubstring(label))
				}

				By("Check the default worker pool description")
				output, err = machinePoolService.DescribeMachinePool(clusterID, constants.DefaultClassicWorkerPool)
				Expect(err).ToNot(HaveOccurred())

				mpD, err := machinePoolService.ReflectMachinePoolDescription(output)
				Expect(err).ToNot(HaveOccurred())
				for _, label := range mpLables {
					Expect(mpD.Labels).To(ContainSubstring(label))
				}

			})

		It("the windows certificates expiration - [id:64040]",
			labels.Medium,
			func() {
				//If the case fails,please open a card to ask dev update windows certificates.
				//Example card: https://issues.redhat.com/browse/SDA-8990
				By("Get ROSA windows certificates on ocm-sdk repo")
				sdkCAFileURL := "https://raw.githubusercontent.com/openshift-online/ocm-sdk-go/main/internal/system_cas_windows.go"
				resp, err := nets.Get(sdkCAFileURL)
				Expect(err).ToNot(HaveOccurred())
				defer resp.Body.Close()
				content, err := io.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				sdkContent := string(content)

				By("Check the domains certificates if it is updated")
				domains := []string{"api.openshift.com", "sso.redhat.com"}
				for _, url := range domains {
					cmd := fmt.Sprintf("openssl s_client -connect %s:443 -showcerts 2>&1  | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p'", url)
					stdout, err := rosaClient.Runner.RunCMD([]string{"bash", "-c", cmd})
					Expect(err).ToNot(HaveOccurred())
					result := strings.Trim(stdout.String(), "\n")
					ca := strings.Split(result, "-----END CERTIFICATE-----")
					Expect(sdkContent).To(ContainSubstring(ca[0]))
					Expect(sdkContent).To(ContainSubstring(ca[1]))
				}
			})

		It("the additional security groups are working well - [id:68172]",
			labels.Day1Post,
			labels.Critical,
			labels.Exclude, //Exclude it until day1 refactor support this part. It cannot be run with current day1
			func() {
				By("Run command to check help message of security groups")
				output, err, _ := clusterService.Create("", "-h")
				Expect(err).ToNot(HaveOccurred())
				Expect(output.String()).Should(ContainSubstring("--additional-compute-security-group-ids"))
				Expect(output.String()).Should(ContainSubstring("--additional-infra-security-group-ids"))
				Expect(output.String()).Should(ContainSubstring("--additional-control-plane-security-group-ids"))

				By("Describe the cluster to check the control plane and infra additional security groups")
				des, err := clusterService.DescribeClusterAndReflect(clusterID)
				Expect(err).ToNot(HaveOccurred())
				var additionalMap []interface{}
				for _, item := range des.Nodes {
					if value, ok := item["Additional Security Group IDs"]; ok {
						additionalMap = value.([]interface{})
					}
				}
				if clusterConfig.AdditionalSecurityGroups == nil {
					Expect(additionalMap).To(BeNil())
				} else {
					Expect(additionalMap).ToNot(BeNil())
					for _, addSgGroups := range additionalMap {
						if value, ok := addSgGroups.(map[string]interface{})["Control Plane"]; ok {
							Expect(value).To(Equal(common.ReplaceCommaWithCommaSpace(clusterConfig.AdditionalSecurityGroups.ControlPlaneSecurityGroups)))
						} else {
							value = addSgGroups.(map[string]interface{})["Infra"]
							Expect(value).To(Equal(common.ReplaceCommaWithCommaSpace(clusterConfig.AdditionalSecurityGroups.InfraSecurityGroups)))
						}
					}
				}

				By("Describe the worker pool and check the compute security groups")
				mp, err := machinePoolService.DescribeAndReflectMachinePool(clusterID, constants.DefaultClassicWorkerPool)
				Expect(err).ToNot(HaveOccurred())
				if clusterConfig.AdditionalSecurityGroups == nil {
					Expect(mp.SecurityGroupIDs).To(BeEmpty())
				} else {
					Expect(mp.SecurityGroupIDs).To(Equal(common.ReplaceCommaWithCommaSpace(clusterConfig.AdditionalSecurityGroups.WorkerSecurityGroups)))
				}

			})
	})
