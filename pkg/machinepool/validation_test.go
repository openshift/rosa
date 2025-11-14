package machinepool

import (
	"fmt"

	"go.uber.org/mock/gomock"

	"github.com/AlecAivazis/survey/v2/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
)

var _ = Describe("MachinePool validation", func() {
	Context("KubeletConfigs", func() {

		It("Fails if customer requests more than 1 kubelet config via []string", func() {
			kubeletConfigs := []string{"foo", "bar"}
			err := ValidateKubeletConfig(kubeletConfigs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("only a single kubelet config is supported for Machine Pools"))
		})

		It("Fails if customer requests more than 1 kubelet config via []core.OptionAnswer", func() {
			kubeletConfigs := []core.OptionAnswer{
				{
					Value: "foo",
					Index: 0,
				},
				{
					Value: "bar",
					Index: 1,
				},
			}
			err := ValidateKubeletConfig(kubeletConfigs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("only a single kubelet config is supported for Machine Pools"))
		})

		It("Passes if a customer selects only a single kubelet config via []core.OptionAnswer", func() {
			kubeletConfigs := []core.OptionAnswer{
				{
					Value: "foo",
					Index: 0,
				},
			}
			err := ValidateKubeletConfig(kubeletConfigs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes if a customer selects only a single kubelet config via []string", func() {
			kubeletConfigs := []string{"foo"}
			err := ValidateKubeletConfig(kubeletConfigs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes with empty selection via []string", func() {
			err := ValidateKubeletConfig([]string{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes with empty selection via []core.OptionAnswer", func() {
			err := ValidateKubeletConfig([]core.OptionAnswer{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Fails if the input is not a []string or []core.OptionAnswer", func() {
			err := ValidateKubeletConfig("foo")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("input for kubelet config flag is not valid"))
		})
	})
	Context("Validate edit machinepool options", func() {
		It("Fails with autoscaling + replicas set", func() {
			Expect(validateEditInput("machine", true, 1,
				2, 1, true, true, true,
				true, "test")).ToNot(Succeed())
		})
		It("Fails with max, min, and replicas < 0", func() {
			Expect(validateEditInput("machine", true, -1,
				1, 0, false, true, true,
				true, "test")).ToNot(Succeed())
			Expect(validateEditInput("machine", true, 1,
				-1, 0, false, true, true,
				true, "test")).ToNot(Succeed())
			Expect(validateEditInput("machine", false, 0,
				0, -1, true, true, false,
				false, "test")).ToNot(Succeed())
		})
		It("Fails with max < min replicas", func() {
			Expect(validateEditInput("machine", true, 5,
				4, 0, false, true, true,
				true, "test")).ToNot(Succeed())
		})
		It("Fails when no autoscaling, but min/max are set", func() {
			Expect(validateEditInput("machine", false, 1,
				0, 1, true, false, true,
				false, "test")).ToNot(Succeed())
			Expect(validateEditInput("machine", false, 0,
				1, 1, true, false, false,
				true, "test")).ToNot(Succeed())
			Expect(validateEditInput("machine", false, 1,
				1, 1, true, false, true,
				true, "test")).ToNot(Succeed())
		})
		It("Passes (autoscaling)", func() {
			Expect(validateEditInput("machine", true, 1,
				2, 0, false, true, true,
				true, "test")).To(Succeed())
		})
		It("Passes (not autoscaling)", func() {
			Expect(validateEditInput("machine", false, 0,
				0, 2, true, false, false,
				false, "test")).To(Succeed())
		})
	})
	Context("Validate capacity-reservation-id", func() {
		It("Passes when no capacity reservation ID is set", func() {
			Expect(validateCapacityReservationId("new-id", "aws-nodepool-1",
				"")).To(Succeed())
		})
		It("Fails when capacity reservation ID is set", func() {
			Expect(validateCapacityReservationId("new-id", "aws-nodepool-1", "old-id").Error()).
				Should(ContainSubstring("unable to change 'capacity-reservation-id' to 'new-id'. " +
					"AWS NodePool 'aws-nodepool-1' already has a Capacity Reservation ID: 'old-id'"))
		})
	})

	Context("Validate capacity reservation replicas", func() {
		var (
			mockCtrl   *gomock.Controller
			mockClient *aws.MockClient
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = aws.NewMockClient(mockCtrl)
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("Passes when replicas are within available capacity", func() {
			capacityReservationID := "cr-12345678"
			availableInstances := int32(3)
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(availableInstances, nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 2, mockClient)
			Expect(err).To(BeNil())
		})

		It("Fails when replicas exceed available capacity", func() {
			capacityReservationID := "cr-12345678"
			availableInstances := int32(1)
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(availableInstances, nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 3, mockClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("cannot set replicas to 3: capacity reservation 'cr-12345678' only has 1 available instance(s)"))
		})

		It("Handles AWS API errors gracefully", func() {
			capacityReservationID := "cr-12345678"
			awsError := fmt.Errorf("AWS API error: capacity reservation not found")

			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(int32(0), awsError)

			err := validateCapacityReservationReplicas(capacityReservationID, 2, mockClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unable to validate capacity reservation"))
			Expect(err.Error()).To(ContainSubstring(capacityReservationID))
			Expect(err.Error()).To(ContainSubstring("AWS API error"))
		})

		It("Passes when capacity reservation has zero available instances and replicas is zero", func() {
			capacityReservationID := "cr-12345678"
			availableInstances := int32(0)
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(availableInstances, nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 0, mockClient)
			Expect(err).To(BeNil())
		})
	})
})
