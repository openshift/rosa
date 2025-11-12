package machinepool

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/machinepool"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Create machine pool", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("createMachinePoolBasedOnClusterType", func() {
		When("cluster is classic", func() {
			It("should create machine pool", func() {
				serviceMock := machinepool.NewMockMachinePoolService(ctrl)
				serviceMock.EXPECT().CreateMachinePoolBasedOnClusterType(gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

				mockClassicClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
					c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
					c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
					c.State(cmv1.ClusterStateReady)
					c.Hypershift(cmv1.NewHypershift().Enabled(false))
				})
				err := serviceMock.CreateMachinePoolBasedOnClusterType(rosa.NewRuntime(), NewCreateMachinePoolCommand(),
					"82339823", mockClassicClusterReady, mockClassicClusterReady.Autoscaler(),
					NewCreateMachinepoolUserOptions())
				Expect(err).ToNot(HaveOccurred())

			})
		})
	})
})

var _ = Describe("Validation functions", func() {
	var (
		ctrl                    *gomock.Controller
		mockCmd                 *cobra.Command
		mockArgs                *mpOpts.CreateMachinepoolUserOptions
		mockClassicClusterReady *cmv1.Cluster
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockCmd = NewCreateMachinePoolCommand()
		mockArgs = &mpOpts.CreateMachinepoolUserOptions{}
		mockClassicClusterReady = test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(false))
		})
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("validateClusterState", func() {
		It("should return nil when the cluster state is ready", func() {
			err := machinepool.ValidateClusterState(mockClassicClusterReady, "test-cluster")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error when the cluster state is not ready", func() {
			mockClassicClusterInstalling := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
				c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
				c.State(cmv1.ClusterStateInstalling)
				c.Hypershift(cmv1.NewHypershift().Enabled(false))
			})
			err := machinepool.ValidateClusterState(mockClassicClusterInstalling, "test-cluster")
			Expect(err).To(MatchError("cluster 'test-cluster' is not yet ready"))
		})
	})

	Context("validateLabels", func() {
		It("should return nil when the labels flag has not changed", func() {
			err := machinepool.ValidateLabels(mockCmd, mockArgs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return nil when the labels flag is set with valid labels", func() {
			mockArgs.Labels = "key1=value1,key2=value2"
			mockCmd.Flags().Set("labels", "key1=value1,key2=value2")
			err := machinepool.ValidateLabels(mockCmd, mockArgs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error when the labels flag is set with invalid labels", func() {
			mockArgs.Labels = "key1=value1,key2"
			mockCmd.Flags().Set("labels", "key1=value1,key2")
			err := machinepool.ValidateLabels(mockCmd, mockArgs)
			Expect(err).To(HaveOccurred())
		})
	})
})
