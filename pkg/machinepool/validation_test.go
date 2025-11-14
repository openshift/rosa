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
			Expect(err.Error()).To(Equal("Only a single kubelet config is supported for Machine Pools"))
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
			Expect(err.Error()).To(Equal("Only a single kubelet config is supported for Machine Pools"))
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
			Expect(err.Error()).To(Equal("Input for kubelet config flag is not valid"))
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
				Should(ContainSubstring("Unable to change 'capacity-reservation-id' to 'new-id'. " +
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

		It("Passes when no capacity reservation ID is provided", func() {
			err := validateCapacityReservationReplicas("", 2, nil, false, 0, 0)
			Expect(err).To(BeNil())
		})

		It("Passes when replicas are within available capacity", func() {
			capacityReservationID := "cr-12345678"
			// Mock returns 5 total, 3 available
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(int32(5), int32(3), nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 2, mockClient, false, 0, 0)
			Expect(err).To(BeNil())
		})

		It("Fails when replicas exceed available capacity", func() {
			capacityReservationID := "cr-12345678"
			// Mock returns 5 total, 1 available
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(int32(5), int32(1), nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 3, mockClient, false, 0, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("cannot set replicas to 3: capacity reservation 'cr-12345678' only has 1 available instance(s)"))
		})

		It("Passes when autoscaling min replicas are within capacity", func() {
			capacityReservationID := "cr-12345678"
			// Mock returns 10 total, 8 available
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(int32(10), int32(8), nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 0, mockClient, true, 2, 5)
			Expect(err).To(BeNil())
		})

		It("Fails when autoscaling min replicas exceed available capacity", func() {
			capacityReservationID := "cr-12345678"
			// Mock returns 10 total, 2 available
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(int32(10), int32(2), nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 0, mockClient, true, 3, 5)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("cannot set min replicas to 3: capacity reservation 'cr-12345678' only has 2 available instance(s)"))
		})

		It("Fails when autoscaling max replicas exceed available capacity", func() {
			capacityReservationID := "cr-12345678"
			// Mock returns 10 total, 4 available
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(int32(10), int32(4), nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 0, mockClient, true, 1, 5)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("cannot set max replicas to 5: capacity reservation 'cr-12345678' only has 4 available instance(s)"))
		})

		It("Handles AWS API errors gracefully", func() {
			capacityReservationID := "cr-12345678"
			awsError := fmt.Errorf("AWS API error: capacity reservation not found")

			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(int32(0), int32(0), awsError)

			err := validateCapacityReservationReplicas(capacityReservationID, 2, mockClient, false, 0, 0)
			Expect(err).To(HaveOccurred())
			// The error is wrapped with additional context
			Expect(err.Error()).To(ContainSubstring("unable to validate capacity reservation"))
			Expect(err.Error()).To(ContainSubstring(capacityReservationID))
			Expect(err.Error()).To(ContainSubstring("AWS API error"))
		})

		It("Passes when capacity reservation has zero available instances and replicas is zero", func() {
			capacityReservationID := "cr-12345678"
			// Mock returns 5 total, 0 available
			mockClient.EXPECT().GetCapacityReservationDetails(capacityReservationID).
				Return(int32(5), int32(0), nil)

			err := validateCapacityReservationReplicas(capacityReservationID, 0, mockClient, false, 0, 0)
			Expect(err).To(BeNil())
		})
	})
})
