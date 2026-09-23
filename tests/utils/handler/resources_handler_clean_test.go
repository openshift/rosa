// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"fmt"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
)

var _ = Describe("isAWSAuthorizationError", func() {
	It("returns true for UnauthorizedOperation", func() {
		err := &smithy.GenericAPIError{Code: "UnauthorizedOperation", Message: "not authorized"}
		Expect(isAWSAuthorizationError(err)).To(BeTrue())
	})

	It("returns true for AccessDenied", func() {
		err := &smithy.GenericAPIError{Code: "AccessDenied", Message: "access denied"}
		Expect(isAWSAuthorizationError(err)).To(BeTrue())
	})

	It("returns false for other API errors", func() {
		err := &smithy.GenericAPIError{Code: "DependencyViolation", Message: "has dependent object"}
		Expect(isAWSAuthorizationError(err)).To(BeFalse())
	})

	It("returns false for nil error", func() {
		Expect(isAWSAuthorizationError(nil)).To(BeFalse())
	})

	It("returns true for wrapped UnauthorizedOperation", func() {
		inner := &smithy.GenericAPIError{Code: "UnauthorizedOperation", Message: "ec2:DeleteSecurityGroup"}
		err := fmt.Errorf("delete security group sg-123: %w", inner)
		Expect(isAWSAuthorizationError(err)).To(BeTrue())
	})

	It("returns false for non-API errors containing auth keywords", func() {
		err := fmt.Errorf("UnauthorizedOperation: plain string, not smithy")
		Expect(isAWSAuthorizationError(err)).To(BeFalse())
	})
})

var _ = Describe("hostedZoneIsHCPInternal", func() {
	It("returns true for hypershift.local suffix", func() {
		Expect(hostedZoneIsHCPInternal("mycluster.hypershift.local")).To(BeTrue())
	})

	It("returns false for ingress zone name", func() {
		Expect(hostedZoneIsHCPInternal("rosa.mycluster.example.com")).To(BeFalse())
	})

	It("returns false for empty string", func() {
		Expect(hostedZoneIsHCPInternal("")).To(BeFalse())
	})

	It("returns false when hypershift.local appears mid-string", func() {
		Expect(hostedZoneIsHCPInternal("hypershift.local.example.com")).To(BeFalse())
	})
})

var _ = Describe("waitForVPCEndpointENIsClearedWithTimeout", func() {
	var (
		ctrl      *gomock.Controller
		mockEc2   *aws_client.MockEC2ClientAPI
		awsclient *aws_client.AWSClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockEc2 = aws_client.NewMockEC2ClientAPI(ctrl)
		awsclient = &aws_client.AWSClient{Ec2Client: mockEc2}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("returns immediately when there are no vpc endpoints or lingering ENIs", func() {
		mockEc2.EXPECT().DescribeVpcEndpoints(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeVpcEndpointsOutput{}, nil)
		mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeNetworkInterfacesOutput{}, nil)

		err := waitForVPCEndpointENIsClearedWithTimeout(awsclient, "vpc-123", time.Millisecond, 50*time.Millisecond)
		Expect(err).ToNot(HaveOccurred())
	})

	It("deletes vpc endpoints and waits for their ENIs to clear", func() {
		endpointID := "vpce-123"
		eniID := "eni-123"
		mockEc2.EXPECT().DescribeVpcEndpoints(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeVpcEndpointsOutput{
				VpcEndpoints: []types.VpcEndpoint{{VpcEndpointId: &endpointID}},
			}, nil)
		mockEc2.EXPECT().DeleteVpcEndpoints(gomock.Any(), gomock.Any()).
			Return(&ec2.DeleteVpcEndpointsOutput{}, nil)

		gomock.InOrder(
			mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
				Return(&ec2.DescribeNetworkInterfacesOutput{
					NetworkInterfaces: []types.NetworkInterface{
						{NetworkInterfaceId: &eniID, InterfaceType: types.NetworkInterfaceTypeVpcEndpoint},
					},
				}, nil),
			mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
				Return(&ec2.DescribeNetworkInterfacesOutput{}, nil),
		)

		err := waitForVPCEndpointENIsClearedWithTimeout(awsclient, "vpc-123", 10*time.Millisecond, time.Second)
		Expect(err).ToNot(HaveOccurred())
	})

	It("times out if the endpoint ENI never clears", func() {
		eniID := "eni-999"
		mockEc2.EXPECT().DescribeVpcEndpoints(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeVpcEndpointsOutput{}, nil)
		mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{NetworkInterfaceId: &eniID, InterfaceType: types.NetworkInterfaceTypeVpcEndpoint},
				},
			}, nil).AnyTimes()

		err := waitForVPCEndpointENIsClearedWithTimeout(awsclient, "vpc-123", 5*time.Millisecond, 30*time.Millisecond)
		Expect(err).To(HaveOccurred())
	})

	It("ignores ENIs that are not vpc endpoint interfaces", func() {
		eniID := "eni-normal"
		mockEc2.EXPECT().DescribeVpcEndpoints(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeVpcEndpointsOutput{}, nil)
		mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{NetworkInterfaceId: &eniID, InterfaceType: types.NetworkInterfaceTypeInterface},
				},
			}, nil)

		err := waitForVPCEndpointENIsClearedWithTimeout(awsclient, "vpc-123", time.Millisecond, 50*time.Millisecond)
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("deleteForeignOwnedSecurityGroupsWithClient", func() {
	var (
		ctrl      *gomock.Controller
		mockEc2   *aws_client.MockEC2ClientAPI
		awsclient *aws_client.AWSClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockEc2 = aws_client.NewMockEC2ClientAPI(ctrl)
		awsclient = &aws_client.AWSClient{Ec2Client: mockEc2}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("deletes every security group it owns", func() {
		sgID1, sgID2 := "sg-owned-1", "sg-owned-2"
		mockEc2.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{GroupId: &sgID1, GroupName: &sgID1, Description: &sgID1},
					{GroupId: &sgID2, GroupName: &sgID2, Description: &sgID2},
				},
			}, nil)
		mockEc2.EXPECT().DescribeSecurityGroupRules(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupRulesOutput{}, nil).Times(2)
		mockEc2.EXPECT().DeleteSecurityGroup(gomock.Any(), gomock.Any()).
			Return(&ec2.DeleteSecurityGroupOutput{}, nil)
		mockEc2.EXPECT().DeleteSecurityGroup(gomock.Any(), gomock.Any()).
			Return(&ec2.DeleteSecurityGroupOutput{}, nil)

		err := deleteForeignOwnedSecurityGroupsWithClient(awsclient, "vpc-123")
		Expect(err).ToNot(HaveOccurred())
	})

	It("tolerates a foreign-owned security group and keeps deleting the rest", func() {
		ownedID, foreignID := "sg-owned", "sg-foreign"
		mockEc2.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{GroupId: &foreignID, GroupName: &foreignID, Description: &foreignID},
					{GroupId: &ownedID, GroupName: &ownedID, Description: &ownedID},
				},
			}, nil)
		mockEc2.EXPECT().DescribeSecurityGroupRules(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupRulesOutput{}, nil).Times(2)
		mockEc2.EXPECT().DeleteSecurityGroup(gomock.Any(), gomock.Any()).
			Return(nil, &smithy.GenericAPIError{
				Code:    "UnauthorizedOperation",
				Message: "A subnet in this vpc is shared but the provided object is not owned by you",
			})
		mockEc2.EXPECT().DeleteSecurityGroup(gomock.Any(), gomock.Any()).
			Return(&ec2.DeleteSecurityGroupOutput{}, nil)

		err := deleteForeignOwnedSecurityGroupsWithClient(awsclient, "vpc-123")
		Expect(err).ToNot(HaveOccurred())
	})

	It("propagates a failure to list security groups", func() {
		mockEc2.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("boom"))

		err := deleteForeignOwnedSecurityGroupsWithClient(awsclient, "vpc-123")
		Expect(err).To(HaveOccurred())
	})
})
