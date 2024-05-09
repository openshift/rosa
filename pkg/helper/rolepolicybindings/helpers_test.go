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

package rolepolicybindings

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
)

func TestIdp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RolePolicyBinding")
}

var (
	roleName1     = "sample-role-1"
	roleName2     = "sample-role-2"
	policyArn1    = "sample-policy-arn-1"
	policyName1   = "sample-policy-name-1"
	policyArn2    = "sample-policy-arn-2"
	policyName2   = "sample-policy-name-2"
	errDesc       = "sample-err-description"
	failedBinding = cmv1.NewRolePolicyBinding().
			Name(roleName1).
			Status(cmv1.NewRolePolicyBindingStatus().
				Value(RolePolicyBindingFailedStatus).
				Description(errDesc))
	policyBuilder1 = cmv1.NewRolePolicy().Arn(policyArn1).Name(policyName1).Type(aws.Inline)
	policyBuilder2 = cmv1.NewRolePolicy().Arn(policyArn2).Name(policyName2).Type("customer")
	binding1       = cmv1.NewRolePolicyBinding().
			Name(roleName1).
			Policies(policyBuilder1)
	actualBinding2 = cmv1.NewRolePolicyBinding().
			Name(roleName2).
			Policies(policyBuilder2)
	desiredBinding2 = cmv1.NewRolePolicyBinding().
			Name(roleName2).
			Policies(policyBuilder1, policyBuilder2)
)

var _ = Describe("Policy Service", func() {
	Context("Attach Policy", Ordered, func() {
		It("Test ValidateRolePolicyBindings", func() {
			bindingList, err := cmv1.NewRolePolicyBindingList().Items(failedBinding).Build()
			Expect(err).ShouldNot(HaveOccurred())
			err = CheckRolePolicyBindingStatus(bindingList)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Failed to get attach policies of role %s: %s",
				roleName1, errDesc)))
		})
		It("Test CheckMissingRolePolicyBindings -- No missing rolepolicy binding", func() {
			desiredBindings, err := cmv1.NewRolePolicyBindingList().Items(binding1).Build()
			Expect(err).ShouldNot(HaveOccurred())
			actualBindings, err := cmv1.NewRolePolicyBindingList().Items(binding1).Build()
			Expect(err).ShouldNot(HaveOccurred())
			output, isBindingMissed := CheckMissingRolePolicyBindings(desiredBindings, actualBindings)
			Expect(isBindingMissed).To(Equal(false))
			Expect(output).To(Equal(""))
		})
		It("Test CheckMissingRolePolicyBindings -- Find missing rolepolicy binding", func() {
			desiredBindings, err := cmv1.NewRolePolicyBindingList().Items(binding1, desiredBinding2).Build()
			Expect(err).ShouldNot(HaveOccurred())
			actualBindings, err := cmv1.NewRolePolicyBindingList().Items(binding1, actualBinding2).Build()
			Expect(err).ShouldNot(HaveOccurred())
			output, isBindingMissed := CheckMissingRolePolicyBindings(desiredBindings, actualBindings)
			Expect(isBindingMissed).To(Equal(true))
			Expect(output).To(Equal(fmt.Sprintf("Policy '%s' missed in role '%s'\n"+
				"Run the following commands to attach the missing policies:\n"+
				"rosa attach policy --role-name %s --policy-arns %s --mode auto\n",
				policyArn1, roleName2, roleName2, policyArn1)))
		})
		It("Test TransformToRolePolicyDetails", func() {
			bindingList, err := cmv1.NewRolePolicyBindingList().Items(binding1, actualBinding2).Build()
			Expect(err).ShouldNot(HaveOccurred())
			rolePoliyDetails := TransformToRolePolicyDetails(bindingList)
			Expect(rolePoliyDetails[roleName1]).NotTo(BeNil())
			Expect(rolePoliyDetails[roleName1]).To(HaveLen(1))
			Expect(rolePoliyDetails[roleName1][0]).To(BeEquivalentTo(aws.PolicyDetail{
				PolicyName: policyName1,
				PolicyArn:  policyArn1,
				PolicyType: aws.Inline,
			}))
			Expect(rolePoliyDetails[roleName2]).NotTo(BeNil())
			Expect(rolePoliyDetails[roleName2]).To(HaveLen(1))
			Expect(rolePoliyDetails[roleName2][0]).To(BeEquivalentTo(aws.PolicyDetail{
				PolicyName: policyName2,
				PolicyArn:  policyArn2,
				PolicyType: aws.Attached,
			}))
		})
	})
})
