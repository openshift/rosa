// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package operatorroles

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("create operator-roles by prefix", func() {
	var ctrl *gomock.Controller
	var runtime *rosa.Runtime
	var mockClient *aws.MockClient

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		runtime = rosa.NewRuntime()
		mockClient = aws.NewMockClient(ctrl)
		runtime.AWSClient = mockClient

		interactive.SetEnabled(false)
		args = struct {
			prefix              string
			hostedCp            bool
			installerRoleArn    string
			permissionsBoundary string
			forcePolicyCreation bool
			oidcConfigId        string
			sharedVpcRoleArn    string
			channelGroup        string
			vpcEndpointRoleArn  string
		}{}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("computePolicyARN", func() {
		It("returns correct ARN for standard partition without path", func() {
			creator := aws.Creator{Partition: "aws", AccountID: "123456789012"}
			arn := computePolicyARN(creator, "my-prefix", "openshift-cloud-credential-operator", "cloud-credentials", "")
			Expect(arn).To(Equal(
				"arn:aws:iam::123456789012:policy/my-prefix-openshift-cloud-credential-operator-cloud-credentials",
			))
		})

		It("returns correct ARN for gov-cloud partition without path", func() {
			creator := aws.Creator{Partition: "aws-us-gov", AccountID: "111222333444"}
			arn := computePolicyARN(creator, "gov-prefix", "ns", "name", "")
			Expect(arn).To(Equal(
				"arn:aws-us-gov:iam::111222333444:policy/gov-prefix-ns-name",
			))
		})

		It("returns correct ARN with a custom path", func() {
			creator := aws.Creator{Partition: "aws", AccountID: "123456789012"}
			arn := computePolicyARN(creator, "my-prefix", "ns", "name", "/custom/path/")
			Expect(arn).To(Equal(
				"arn:aws:iam::123456789012:policy/custom/path/my-prefix-ns-name",
			))
		})

		It("uses DefaultPrefix when prefix is empty", func() {
			creator := aws.Creator{Partition: "aws", AccountID: "123456789012"}
			arn := computePolicyARN(creator, "", "ns", "name", "")
			Expect(arn).To(ContainSubstring(aws.DefaultPrefix))
		})
	})

	Context("validateArgumentsOperatorRolesCreationByPrefix", func() {
		It("does not exit for valid inputs", func() {
			validateArgumentsOperatorRolesCreationByPrefix(
				runtime,
				"valid-prefix",
				"https://oidc.example.com",
				"arn:aws:iam::123456789012:role/MyRole",
			)
		})
	})

	Context("convertCredRequestsOperatorRolesIntoV1OperatorIAMRole", func() {
		It("converts a single credential request into an operator IAM role list", func() {
			operator, err := cmv1.NewSTSOperator().
				Name("cloud-credentials").
				Namespace("openshift-cloud-credential-operator").
				Build()
			Expect(err).ToNot(HaveOccurred())

			credRequests := map[string]*cmv1.STSOperator{
				"cloud_credential": operator,
			}
			creator := &aws.Creator{
				Partition: "aws",
				AccountID: "123456789012",
			}

			roleList, err := convertCredRequestsOperatorRolesIntoV1OperatorIAMRole(
				credRequests, "test-prefix", creator, "",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(roleList).To(HaveLen(1))
			Expect(roleList[0].Name()).To(Equal("cloud-credentials"))
			Expect(roleList[0].Namespace()).To(Equal("openshift-cloud-credential-operator"))
			Expect(roleList[0].RoleARN()).To(Equal(aws.ComputeOperatorRoleArn(
				"test-prefix", operator, creator, "",
			)))
		})

		It("converts multiple credential requests", func() {
			op1, err := cmv1.NewSTSOperator().
				Name("cred1").
				Namespace("ns1").
				Build()
			Expect(err).ToNot(HaveOccurred())

			op2, err := cmv1.NewSTSOperator().
				Name("cred2").
				Namespace("ns2").
				Build()
			Expect(err).ToNot(HaveOccurred())

			credRequests := map[string]*cmv1.STSOperator{
				"type1": op1,
				"type2": op2,
			}
			creator := &aws.Creator{
				Partition: "aws",
				AccountID: "999888777666",
			}

			roleList, err := convertCredRequestsOperatorRolesIntoV1OperatorIAMRole(
				credRequests, "multi-prefix", creator, "/path/",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(roleList).To(HaveLen(2))

			rolesByName := map[string]*cmv1.OperatorIAMRole{}
			for _, role := range roleList {
				rolesByName[role.Name()] = role
			}
			Expect(rolesByName["cred1"].Namespace()).To(Equal("ns1"))
			Expect(rolesByName["cred1"].RoleARN()).To(Equal(aws.ComputeOperatorRoleArn(
				"multi-prefix", op1, creator, "/path/",
			)))
			Expect(rolesByName["cred2"].Namespace()).To(Equal("ns2"))
			Expect(rolesByName["cred2"].RoleARN()).To(Equal(aws.ComputeOperatorRoleArn(
				"multi-prefix", op2, creator, "/path/",
			)))
		})

		It("returns empty list when no credential requests provided", func() {
			credRequests := map[string]*cmv1.STSOperator{}
			creator := &aws.Creator{
				Partition: "aws",
				AccountID: "123456789012",
			}

			roleList, err := convertCredRequestsOperatorRolesIntoV1OperatorIAMRole(
				credRequests, "prefix", creator, "",
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(roleList).To(BeEmpty())
		})
	})
})
