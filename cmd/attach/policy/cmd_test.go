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
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/policy"
	. "github.com/openshift/rosa/pkg/test"
)

func TestAttachPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa attach policy")
}

var _ = Describe("rosa attach policy", func() {
	Context("Create Command", func() {
		It("Returns Command", func() {

			cmd := NewAttachPolicyCommand()
			Expect(cmd).NotTo(BeNil())

			Expect(cmd.Use).To(Equal(use))
			Expect(cmd.Example).To(Equal(example))
			Expect(cmd.Short).To(Equal(short))
			Expect(cmd.Long).To(Equal(long))
			Expect(cmd.Args).NotTo(BeNil())
			Expect(cmd.Run).NotTo(BeNil())

			flag := cmd.Flags().Lookup("role-name")
			Expect(flag).NotTo(BeNil())

			flag = cmd.Flags().Lookup("policy-arns")
			Expect(flag).NotTo(BeNil())
		})
	})

	Context("Execute command", func() {

		const (
			roleName   = "sample-role"
			policyArn1 = "arn:aws:iam::111111111111:policy/Sample-Policy-1"
			policyArn2 = "arn:aws:iam::111111111111:policy/Sample-Policy-2"
			policyArn3 = "arn:aws:iam::111111111111:policy/Sample-Policy-3"

			roleNotFoundMsg   = "roleNotFoundMsg"
			policyNotFoundMsg = "policyNotFoundMsg"
		)

		var (
			t          *TestingRuntime
			c          *cobra.Command
			mockClient *mock.MockClient
			options    *RosaAttachPolicyOptions
			quota      *servicequotas.GetServiceQuotaOutput
			role       *iamtypes.Role
		)

		BeforeEach(func() {
			c = NewAttachPolicyCommand()
			options = &RosaAttachPolicyOptions{
				roleName:   roleName,
				policyArns: policyArn1 + "," + policyArn2,
			}
			c.Flags().Set("mode", "auto")
			quotaValue := 2.0
			role = &iamtypes.Role{
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(tags.RedHatManaged),
						Value: aws.String("true"),
					},
				},
			}
			quota = &servicequotas.GetServiceQuotaOutput{
				Quota: &types.ServiceQuota{
					Value: &quotaValue,
				},
			}

			t = NewTestRuntime()
			mockCtrl := gomock.NewController(GinkgoT())
			mockClient = mock.NewMockClient(mockCtrl)
			t.RosaRuntime.AWSClient = mockClient

			account, _ := amsv1.NewAccount().Organization(amsv1.NewOrganization().
				ID("123").ExternalID("456")).Build()
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, FormatResource(account)))

		})

		It("Returns an error if the role does not exist", func() {
			mockClient.EXPECT().GetRoleByName(roleName).Return(iamtypes.Role{}, fmt.Errorf(roleNotFoundMsg))
			runner := AttachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Failed to find the role '%s': %s", roleName, roleNotFoundMsg)))
		})

		It("Returns an error if the role does not has tag 'red-hat-managed'", func() {
			mockClient.EXPECT().GetRoleByName(roleName).Return(iamtypes.Role{}, nil)
			runner := AttachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Cannot attach/detach policies to non-ROSA roles"))
		})

		It("Returns an error if one policy does not exist", func() {
			mockClient.EXPECT().GetRoleByName(roleName).Return(*role, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn1).Return(nil, fmt.Errorf(policyNotFoundMsg))
			runner := AttachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Failed to find the policy '%s': %s", policyArn1, policyNotFoundMsg)))
		})

		It("Returns an error if exceeds policy quota per role", func() {
			mockClient.EXPECT().GetRoleByName(roleName).Return(*role, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn1).Return(nil, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn2).Return(nil, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn3).Return(nil, nil)
			mockClient.EXPECT().GetAttachedPolicy(aws.String(roleName)).Return([]mock.PolicyDetail{}, nil)
			mockClient.EXPECT().GetIAMServiceQuota(policy.QuotaCode).Return(quota, nil)
			options.policyArns = options.policyArns + "," + policyArn3
			runner := AttachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Failed to attach policies due to quota limitations"+
				" (total limit: %d, expected: %d)", 2, 3)))
		})

		It("Attach policy to role", func() {
			mockClient.EXPECT().GetRoleByName(roleName).Return(*role, nil)
			mockClient.EXPECT().GetIAMServiceQuota(policy.QuotaCode).Return(quota, nil)
			mockClient.EXPECT().GetAttachedPolicy(aws.String(roleName)).Return([]mock.PolicyDetail{}, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn1).Return(nil, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn2).Return(nil, nil)
			mockClient.EXPECT().AttachRolePolicy(t.RosaRuntime.Reporter, roleName, policyArn1).Return(nil)
			mockClient.EXPECT().AttachRolePolicy(t.RosaRuntime.Reporter, roleName, policyArn2).Return(nil)
			runner := AttachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
