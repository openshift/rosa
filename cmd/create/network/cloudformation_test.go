// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CloudFormationTemplateFile", func() {
	It("matches the standalone template file used for source builds", func() {
		fileContent, err := os.ReadFile("templates/rosa-quickstart-default-vpc/cloudformation.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSpace(CloudFormationTemplateFile)).To(Equal(strings.TrimSpace(string(fileContent))))
	})

	DescribeTable("routes each private subnet through the NAT Gateway in its own Availability Zone",
		func(n string, condition string) {
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  NATGateway" + n + ":\n    Condition: " + condition + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateRouteTable" + n + ":\n    Type: AWS::EC2::RouteTable\n    Condition: " + condition + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateRoute" + n + ":\n    Type: AWS::EC2::Route\n    Condition: " + condition +
					"\n    Properties:\n      RouteTableId: !Ref PrivateRouteTable" + n +
					"\n      DestinationCidrBlock: 0.0.0.0/0\n      NatGatewayId: !Ref NATGateway" + n + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateSubnetRouteTableAssociation" + n + ":\n    Condition: " + condition +
					"\n    Type: AWS::EC2::SubnetRouteTableAssociation\n    Properties:\n      SubnetId: !Ref SubnetPrivate" +
					n + "\n      RouteTableId: !Ref PrivateRouteTable" + n + "\n"))
		},
		Entry("AZ1", "1", "One"),
		Entry("AZ2", "2", "Two"),
		Entry("AZ3", "3", "Three"),
		Entry("AZ4", "4", "Four"),
	)

	It("guards every private route table reference so unconfigured AZs are never referenced", func() {
		Expect(CloudFormationTemplateFile).To(ContainSubstring(
			"      RouteTableIds:\n        - !Ref PublicRouteTable\n" +
				"        - !If [One, !Ref PrivateRouteTable1, !Ref \"AWS::NoValue\"]\n" +
				"        - !If [Two, !Ref PrivateRouteTable2, !Ref \"AWS::NoValue\"]\n" +
				"        - !If [Three, !Ref PrivateRouteTable3, !Ref \"AWS::NoValue\"]\n" +
				"        - !If [Four, !Ref PrivateRouteTable4, !Ref \"AWS::NoValue\"]\n"))

		Expect(CloudFormationTemplateFile).To(ContainSubstring(
			`Value: !Join [",", [!If [One, !Ref PrivateRouteTable1, !Ref "AWS::NoValue"], ` +
				`!If [Two, !Ref PrivateRouteTable2, !Ref "AWS::NoValue"], ` +
				`!If [Three, !Ref PrivateRouteTable3, !Ref "AWS::NoValue"], ` +
				`!If [Four, !Ref PrivateRouteTable4, !Ref "AWS::NoValue"]]]`))
	})
})
