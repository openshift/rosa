/*
Copyright (c) 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package policy

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	r         *rosa.Runtime
	awsClient *mock.MockClient
	ocmClient *ocm.Client
	policySvc PolicyService

	quota *servicequotas.GetServiceQuotaOutput
	role  *iamtypes.Role

	roleName, policyArn1, policyArn2 string
	policyArns                       []string
)

func TestDescribeUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Policy Service suite")
}

var _ = Describe("Policy Service", func() {
	Context("Attach Policy", Ordered, func() {
		BeforeAll(func() {
			r = rosa.NewRuntime()
			roleName = "sample-role"
			policyArn1 = "arn:aws:iam::111111111111:policy/Sample-Policy-1"
			policyArn2 = "arn:aws:iam::111111111111:policy/Sample-Policy-2"
			policyArns = []string{policyArn1, policyArn2}
			role = &iamtypes.Role{
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(tags.RedHatManaged),
						Value: aws.String("true"),
					},
				},
			}
			quotaValue := 2.0
			quota = &servicequotas.GetServiceQuotaOutput{
				Quota: &types.ServiceQuota{
					Value: &quotaValue,
				},
			}

			mockCtrl := gomock.NewController(GinkgoT())
			awsClient = mock.NewMockClient(mockCtrl)

			logger, err := logging.NewGoLoggerBuilder().
				Debug(true).
				Build()
			Expect(err).To(BeNil())
			// Set up the connection with the fake config
			connection, err := sdk.NewConnectionBuilder().
				Logger(logger).
				Tokens("").
				URL("http://fake.api").
				Build()
			Expect(err).To(BeNil())
			ocmClient = ocm.NewClientWithConnection(connection)

			policySvc = NewPolicyService(ocmClient, awsClient)
		})
		It("Test validateRoleAndPolicies -- valid rolename and policyarn", func() {
			awsClient.EXPECT().GetRoleByName(roleName).Return(*role, nil)
			err := validateRoleAndPolicies(awsClient, "", policyArns)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Invalid role name '%s', expected a valid role name matching %s",
				"", mock.RoleNameRE.String())))
			err = validateRoleAndPolicies(awsClient, roleName, []string{"invalid-policy-arn"})
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Invalid policy arn '%s', expected a valid policy arn matching %s",
				"invalid-policy-arn", mock.PolicyArnRE.String())))
		})
		It("Test ValidateAttachOptions", func() {
			awsClient.EXPECT().GetRoleByName(roleName).Return(*role, nil)
			awsClient.EXPECT().GetIAMServiceQuota(QuotaCode).Return(quota, nil)
			awsClient.EXPECT().GetAttachedPolicy(aws.String(roleName)).Return([]mock.PolicyDetail{}, nil)
			awsClient.EXPECT().IsPolicyExists(policyArn1).Return(nil, nil)
			awsClient.EXPECT().IsPolicyExists(policyArn2).Return(nil, nil)
			err := policySvc.ValidateAttachOptions(roleName, policyArns)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Test AutoAttachArbitraryPolicy", func() {
			awsClient.EXPECT().AttachRolePolicy(r.Reporter, roleName, policyArn1).Return(nil)
			awsClient.EXPECT().AttachRolePolicy(r.Reporter, roleName, policyArn2).Return(nil)
			err := policySvc.AutoAttachArbitraryPolicy(r.Reporter, roleName, policyArns,
				"sample-account-id", "sample-org-id")
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Test ManualAttachArbitraryPolicy", func() {
			output := policySvc.ManualAttachArbitraryPolicy(roleName, policyArns,
				"sample-account-id", "sample-org-id")
			Expect(output).To(Equal(fmt.Sprintf(
				"aws iam attach-role-policy --role-name %s --policy-arn %s\n"+
					"aws iam attach-role-policy --role-name %s --policy-arn %s\n",
				roleName, policyArn1, roleName, policyArn2)))
		})
		It("Test ValidateDetachOptions", func() {
			awsClient.EXPECT().GetRoleByName(roleName).Return(*role, nil)
			awsClient.EXPECT().IsPolicyExists(policyArn1).Return(nil, nil)
			awsClient.EXPECT().IsPolicyExists(policyArn2).Return(nil, nil)
			err := policySvc.ValidateDetachOptions(roleName, policyArns)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Test AutoDetachArbitraryPolicy", func() {
			awsClient.EXPECT().DetachRolePolicy(policyArn1, roleName).Return(nil)
			awsClient.EXPECT().DetachRolePolicy(policyArn2, roleName).Return(&iamtypes.NoSuchEntityException{})
			output, err := policySvc.AutoDetachArbitraryPolicy(roleName, policyArns,
				"sample-account-id", "sample-org-id")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).To(Equal(fmt.Sprintf("Detached policy '%s' from role '%s'\n"+
				"The policy '%s' is currently not attached to role '%s'",
				policyArn1, roleName, policyArn2, roleName)))
		})
		It("Test ManualDetachArbitraryPolicy", func() {
			awsClient.EXPECT().ListAttachedRolePolicies(roleName).Return([]string{policyArn1}, nil)
			output, warn, err := policySvc.ManualDetachArbitraryPolicy(roleName, policyArns,
				"sample-account-id", "sample-org-id")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(warn).To(Equal(fmt.Sprintf(
				"The policy '%s' is currently not attached to role '%s'\n",
				policyArn2, roleName)))
			Expect(output).To(Equal(fmt.Sprintf(
				"aws iam detach-role-policy --role-name %s --policy-arn %s\n",
				roleName, policyArn1)))
		})
	})
})
