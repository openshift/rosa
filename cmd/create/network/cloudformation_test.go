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
	It("matches the standalone template files used for source builds", func() {
		for _, template := range []struct {
			path    string
			content string
		}{
			{"templates/rosa-quickstart-default-vpc/cloudformation.yaml", CloudFormationTemplateFile},
			{"templates/rosa-quickstart-hcp-vpc/cloudformation.yaml", CloudFormationHCPTemplateFile},
		} {
			fileContent, err := os.ReadFile(template.path)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.TrimSpace(template.content)).To(Equal(strings.TrimSpace(string(fileContent))), template.path)
		}
	})

	DescribeTable("routes each private subnet through its Availability Zone NAT Gateway in multi-NAT mode",
		func(n string, natCondition string, azCondition string, privateRouteCondition string) {
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  NATGateway" + n + ":\n    Condition: " + natCondition + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateRouteTable" + n + ":\n    Type: AWS::EC2::RouteTable\n    Condition: " + privateRouteCondition + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateRoute" + n + ":\n    Type: AWS::EC2::Route\n    Condition: " + privateRouteCondition +
					"\n    Properties:\n      RouteTableId: !Ref PrivateRouteTable" + n +
					"\n      DestinationCidrBlock: 0.0.0.0/0\n      NatGatewayId: !Ref NATGateway" + n + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateSubnetRouteTableAssociation" + n + ":\n    Condition: " + azCondition +
					"\n    Type: AWS::EC2::SubnetRouteTableAssociation\n    Properties:\n      SubnetId: !Ref SubnetPrivate" +
					n + "\n      RouteTableId: !If [UseSingleNat, !Ref PrivateRouteTable, !Ref PrivateRouteTable" + n + "]\n"))
		},
		Entry("AZ1", "1", "One", "One", "CreatePrivateRT1"),
		Entry("AZ2", "2", "CreateNat2", "Two", "CreatePrivateRT2"),
		Entry("AZ3", "3", "CreateNat3", "Three", "CreatePrivateRT3"),
		Entry("AZ4", "4", "CreateNat4", "Four", "CreatePrivateRT4"),
	)

	It("guards every private route table reference so unconfigured AZs are never referenced", func() {
		Expect(CloudFormationTemplateFile).To(ContainSubstring(
			"      RouteTableIds:\n        - !Ref PublicRouteTable\n" +
				"        - !If [UseSingleNat, !Ref PrivateRouteTable, !Ref \"AWS::NoValue\"]\n" +
				"        - !If [CreatePrivateRT1, !Ref PrivateRouteTable1, !Ref \"AWS::NoValue\"]\n" +
				"        - !If [CreatePrivateRT2, !Ref PrivateRouteTable2, !Ref \"AWS::NoValue\"]\n" +
				"        - !If [CreatePrivateRT3, !Ref PrivateRouteTable3, !Ref \"AWS::NoValue\"]\n" +
				"        - !If [CreatePrivateRT4, !Ref PrivateRouteTable4, !Ref \"AWS::NoValue\"]\n"))

		Expect(CloudFormationTemplateFile).To(ContainSubstring(
			`Value: !Join [",", [!If [UseSingleNat, !Ref PrivateRouteTable, !Ref "AWS::NoValue"], ` +
				`!If [CreatePrivateRT1, !Ref PrivateRouteTable1, !Ref "AWS::NoValue"], ` +
				`!If [CreatePrivateRT2, !Ref PrivateRouteTable2, !Ref "AWS::NoValue"], ` +
				`!If [CreatePrivateRT3, !Ref PrivateRouteTable3, !Ref "AWS::NoValue"], ` +
				`!If [CreatePrivateRT4, !Ref PrivateRouteTable4, !Ref "AWS::NoValue"]]]`))
	})

	It("uses one shared private route table when single-NAT mode is enabled", func() {
		Expect(CloudFormationTemplateFile).To(ContainSubstring(
			"  PrivateRouteTable:\n    Type: AWS::EC2::RouteTable\n    Condition: UseSingleNat\n"))
		Expect(CloudFormationTemplateFile).To(ContainSubstring(
			"  PrivateRoute:\n    Type: AWS::EC2::Route\n    Condition: UseSingleNat\n    Properties:\n" +
				"      RouteTableId: !Ref PrivateRouteTable\n      DestinationCidrBlock: 0.0.0.0/0\n" +
				"      NatGatewayId: !If\n        - One\n        - !Ref NATGateway1\n"))
	})
})
