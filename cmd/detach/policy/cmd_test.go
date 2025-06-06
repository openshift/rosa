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
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/cobra"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	. "github.com/openshift/rosa/pkg/test"
)

func TestDetachPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa detach policy")
}

var _ = Describe("rosa detach policy", func() {
	Context("Create Command", func() {
		It("Returns Command", func() {

			cmd := NewDetachPolicyCommand()
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

			roleNotFoundMsg   = "roleNotFoundMsg"
			policyNotFoundMsg = "policyNotFoundMsg"
		)

		var (
			t          *TestingRuntime
			c          *cobra.Command
			mockClient *mock.MockClient
			options    *RosaDetachPolicyOptions
			role       *iamtypes.Role
		)

		BeforeEach(func() {
			c = NewDetachPolicyCommand()
			options = &RosaDetachPolicyOptions{
				roleName:   roleName,
				policyArns: policyArn1 + "," + policyArn2,
			}
			c.Flags().Set("mode", "auto")
			role = &iamtypes.Role{
				Tags: []iamtypes.Tag{
					{
						Key:   aws.String(tags.RedHatManaged),
						Value: aws.String("true"),
					},
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
			runner := DetachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Failed to find the role '%s': %s", roleName, roleNotFoundMsg)))
		})

		It("Returns an error if the role does not has tag 'red-hat-managed'", func() {
			mockClient.EXPECT().GetRoleByName(roleName).Return(iamtypes.Role{}, nil)
			runner := DetachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Cannot attach/detach policies to non-ROSA roles"))
		})

		It("Returns an error if one policy does not exist", func() {
			mockClient.EXPECT().GetRoleByName(roleName).Return(*role, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn1).Return(nil, fmt.Errorf(policyNotFoundMsg))
			runner := DetachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"Failed to find the policy '%s': %s", policyArn1, policyNotFoundMsg)))
		})

		It("Detach policy from role", func() {
			mockClient.EXPECT().GetRoleByName(roleName).Return(*role, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn1).Return(nil, nil)
			mockClient.EXPECT().IsPolicyExists(policyArn2).Return(nil, nil)
			mockClient.EXPECT().DetachRolePolicy(policyArn1, roleName).Return(nil)
			mockClient.EXPECT().DetachRolePolicy(policyArn2, roleName).Return(nil)
			runner := DetachPolicyRunner(options)
			err := runner(context.Background(), t.RosaRuntime, c, nil)
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
