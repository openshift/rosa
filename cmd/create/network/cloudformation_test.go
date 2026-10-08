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

	DescribeTable("uses per-AZ route tables and supports single-NAT routing",
		func(n, azCondition, natCondition, natGatewayReference string) {
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  NATGateway" + n + ":\n    Condition: " + natCondition + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateRouteTable" + n + ":\n    Type: AWS::EC2::RouteTable\n    Condition: " + azCondition + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateRoute" + n + ":\n    Type: AWS::EC2::Route\n    Condition: " + azCondition +
					"\n    Properties:\n      RouteTableId: !Ref PrivateRouteTable" + n +
					"\n      DestinationCidrBlock: 0.0.0.0/0\n      NatGatewayId: " + natGatewayReference + "\n"))
			Expect(CloudFormationTemplateFile).To(ContainSubstring(
				"  PrivateSubnetRouteTableAssociation" + n + ":\n    Condition: " + azCondition +
					"\n    Type: AWS::EC2::SubnetRouteTableAssociation\n    Properties:\n      SubnetId: !Ref SubnetPrivate" +
					n + "\n      RouteTableId: !Ref PrivateRouteTable" + n + "\n"))
		},
		Entry("AZ1", "1", "One", "One", "!Ref NATGateway1"),
		Entry("AZ2", "2", "Two", "CreateNat2", "!If\n        - UseSingleNat\n        - !If\n          - One\n          - !Ref NATGateway1\n          - !Ref NATGateway2\n        - !Ref NATGateway2"),
		Entry("AZ3", "3", "Three", "CreateNat3", "!If\n        - UseSingleNat\n        - !If\n          - One\n          - !Ref NATGateway1\n          - !If\n            - Two\n            - !Ref NATGateway2\n            - !Ref NATGateway3\n        - !Ref NATGateway3"),
		Entry("AZ4", "4", "Four", "CreateNat4", "!If\n        - UseSingleNat\n        - !If\n          - One\n          - !Ref NATGateway1\n          - !If\n            - Two\n            - !Ref NATGateway2\n            - !If\n              - Three\n              - !Ref NATGateway3\n              - !Ref NATGateway4\n        - !Ref NATGateway4"),
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
