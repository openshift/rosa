// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"fmt"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"

	"github.com/openshift/rosa/tests/utils/log"
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

	It("logs diagnostic details for blocking VPC endpoint ENIs", func() {
		var buf bytes.Buffer
		originalOut := log.Logger.Out
		log.Logger.SetOutput(&buf)
		defer log.Logger.SetOutput(originalOut)

		mockEc2.EXPECT().DescribeVpcEndpoints(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeVpcEndpointsOutput{}, nil)
		mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: aws.String("eni-diag"),
						InterfaceType:      types.NetworkInterfaceTypeVpcEndpoint,
						Status:             types.NetworkInterfaceStatusInUse,
						Description:        aws.String("VPC Endpoint Interface vpce-abc"),
						OwnerId:            aws.String("111122223333"),
						SubnetId:           aws.String("subnet-abc"),
						Attachment: &types.NetworkInterfaceAttachment{
							InstanceId:      aws.String("i-worker"),
							InstanceOwnerId: aws.String("444455556666"),
							Status:          types.AttachmentStatusAttached,
						},
					},
				},
			}, nil).AnyTimes()

		err := waitForVPCEndpointENIsClearedWithTimeout(awsclient, "vpc-123",
			5*time.Millisecond, 30*time.Millisecond)
		Expect(err).To(HaveOccurred(), "should time out while VPC endpoint ENIs remain")

		logs := buf.String()
		Expect(logs).To(ContainSubstring(`eni_id="eni-diag"`),
			"log should contain the blocking ENI ID")
		Expect(logs).To(ContainSubstring(`interface_type="vpc_endpoint"`),
			"log should contain the interface type")
		Expect(logs).To(ContainSubstring(`status="in-use"`),
			"log should contain the ENI status")
		Expect(logs).To(ContainSubstring(`description="VPC Endpoint Interface vpce-abc"`),
			"log should contain the ENI description")
		Expect(logs).To(ContainSubstring(`subnet_id="subnet-abc"`),
			"log should contain the subnet ID")
		Expect(logs).To(ContainSubstring(`instance_id="i-worker"`),
			"log should contain the instance ID")
		Expect(logs).To(ContainSubstring(`attachment_status="attached"`),
			"log should contain the attachment status")
		// Account IDs should be redacted by the SensitiveHook
		Expect(logs).NotTo(ContainSubstring("111122223333"),
			"owner account ID should be redacted from logs")
		Expect(logs).NotTo(ContainSubstring("444455556666"),
			"instance owner account ID should be redacted from logs")
	})

	It("redacts account IDs embedded in ENI description", func() {
		var buf bytes.Buffer
		originalOut := log.Logger.Out
		log.Logger.SetOutput(&buf)
		defer log.Logger.SetOutput(originalOut)

		mockEc2.EXPECT().DescribeVpcEndpoints(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeVpcEndpointsOutput{}, nil)
		mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: aws.String("eni-desc-redact"),
						InterfaceType:      types.NetworkInterfaceTypeVpcEndpoint,
						Description:        aws.String("Interface for NAT Gateway 999888777666"),
					},
				},
			}, nil).AnyTimes()

		err := waitForVPCEndpointENIsClearedWithTimeout(awsclient, "vpc-123",
			5*time.Millisecond, 30*time.Millisecond)
		Expect(err).To(HaveOccurred(), "should time out while VPC endpoint ENIs remain")

		logs := buf.String()
		Expect(logs).To(ContainSubstring(`eni_id="eni-desc-redact"`),
			"log should contain the blocking ENI ID")
		Expect(logs).NotTo(ContainSubstring("999888777666"),
			"account ID in description should be redacted")
		Expect(logs).To(ContainSubstring("Interface for NAT Gateway"),
			"non-sensitive parts of description should be preserved")
	})

	It("does not log diagnostics when no VPC endpoint ENIs are present", func() {
		var buf bytes.Buffer
		originalOut := log.Logger.Out
		log.Logger.SetOutput(&buf)
		defer log.Logger.SetOutput(originalOut)

		mockEc2.EXPECT().DescribeVpcEndpoints(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeVpcEndpointsOutput{}, nil)
		mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeNetworkInterfacesOutput{}, nil)

		err := waitForVPCEndpointENIsClearedWithTimeout(awsclient, "vpc-123",
			time.Millisecond, 50*time.Millisecond)
		Expect(err).ToNot(HaveOccurred(), "should complete without error when no blocking ENIs exist")
		Expect(buf.String()).NotTo(ContainSubstring("blocked by network interface"),
			"should not log diagnostic messages when no VPC endpoint ENIs are present")
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

	It("logs blocking ENIs when security group deletion fails with a non-auth error", func() {
		var buf bytes.Buffer
		originalOut := log.Logger.Out
		log.Logger.SetOutput(&buf)
		defer log.Logger.SetOutput(originalOut)

		sgID := "sg-blocked"
		mockEc2.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{GroupId: &sgID, GroupName: &sgID, Description: &sgID},
				},
			}, nil)
		mockEc2.EXPECT().DescribeSecurityGroupRules(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupRulesOutput{}, nil)
		mockEc2.EXPECT().DeleteSecurityGroup(gomock.Any(), gomock.Any()).
			Return(nil, &smithy.GenericAPIError{
				Code:    "DependencyViolation",
				Message: "resource sg-blocked has a dependent object",
			})
		mockEc2.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeNetworkInterfacesOutput{
				NetworkInterfaces: []types.NetworkInterface{
					{
						NetworkInterfaceId: aws.String("eni-blocking"),
						InterfaceType:      types.NetworkInterfaceTypeInterface,
						Status:             types.NetworkInterfaceStatusInUse,
						Description:        aws.String("ELB app/my-lb"),
						RequesterManaged:   aws.Bool(true),
						RequesterId:        aws.String("amazon-elb"),
						SubnetId:           aws.String("subnet-123"),
						Attachment: &types.NetworkInterfaceAttachment{
							InstanceOwnerId: aws.String("amazon-aws"),
							Status:          types.AttachmentStatusAttached,
						},
					},
				},
			}, nil)

		err := deleteForeignOwnedSecurityGroupsWithClient(awsclient, "vpc-456")
		Expect(err).ToNot(HaveOccurred(),
			"should delete security groups without returning an error")

		logs := buf.String()
		Expect(logs).To(ContainSubstring(`security_group_id="sg-blocked"`),
			"log should contain the blocked security group ID")
		Expect(logs).To(ContainSubstring(`eni_id="eni-blocking"`),
			"log should contain the blocking ENI ID")
		Expect(logs).To(ContainSubstring(`interface_type="interface"`),
			"log should contain the interface type")
		Expect(logs).To(ContainSubstring(`description="ELB app/my-lb"`),
			"log should contain the ENI description")
		Expect(logs).To(ContainSubstring(`requester_managed=true`),
			"log should contain the requester managed flag")
		// Service aliases like "amazon-elb" should NOT be redacted
		Expect(logs).To(ContainSubstring(`requester_id="amazon-elb"`),
			"service alias requester_id should not be redacted")
		Expect(logs).To(ContainSubstring(`instance_owner_id="amazon-aws"`),
			"service alias instance_owner_id should not be redacted")
	})

	It("does not log ENI diagnostics for auth errors", func() {
		var buf bytes.Buffer
		originalOut := log.Logger.Out
		log.Logger.SetOutput(&buf)
		defer log.Logger.SetOutput(originalOut)

		sgID := "sg-foreign"
		mockEc2.EXPECT().DescribeSecurityGroups(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{GroupId: &sgID, GroupName: &sgID, Description: &sgID},
				},
			}, nil)
		mockEc2.EXPECT().DescribeSecurityGroupRules(gomock.Any(), gomock.Any()).
			Return(&ec2.DescribeSecurityGroupRulesOutput{}, nil)
		mockEc2.EXPECT().DeleteSecurityGroup(gomock.Any(), gomock.Any()).
			Return(nil, &smithy.GenericAPIError{
				Code:    "UnauthorizedOperation",
				Message: "not authorized",
			})

		err := deleteForeignOwnedSecurityGroupsWithClient(awsclient, "vpc-123")
		Expect(err).ToNot(HaveOccurred(),
			"auth errors on security group deletion should not propagate")
		Expect(buf.String()).NotTo(ContainSubstring("blocked by network interface"),
			"should not log ENI diagnostics for authorization failures")
	})
})
