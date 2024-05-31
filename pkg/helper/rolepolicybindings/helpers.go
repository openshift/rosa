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
	"slices"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
)

const (
	RolePolicyBindingFailedStatus = "failed"
)

func CheckRolePolicyBindingStatus(bindings *cmv1.RolePolicyBindingList) error {
	for _, binding := range bindings.Slice() {
		if binding.Status().Value() == RolePolicyBindingFailedStatus {
			return fmt.Errorf("Failed to get attach policies of role %s: %s",
				binding.Name(), binding.Status().Description())
		}
	}
	return nil
}

func CheckMissingRolePolicyBindings(desired, actual *cmv1.RolePolicyBindingList) (string, bool) {
	output := ""
	actualBindings := map[string][]string{}
	for _, binding := range actual.Slice() {
		roleBindings := []string{}
		if binding != nil && binding.Policies() != nil {
			for _, policy := range binding.Policies() {
				roleBindings = append(roleBindings, policy.Arn())
			}
		}
		actualBindings[binding.Name()] = roleBindings
	}
	missingBindings := map[string][]string{}
	for _, binding := range desired.Slice() {
		if binding == nil {
			continue
		}
		for _, policy := range binding.Policies() {
			if !slices.Contains(actualBindings[binding.Name()], policy.Arn()) {
				if missingBindings[binding.Name()] == nil {
					missingBindings[binding.Name()] = []string{policy.Arn()}
				} else {
					missingBindings[binding.Name()] = append(missingBindings[binding.Name()],
						policy.Arn())
				}
				output = fmt.Sprintf(output+"Policy '%s' missed in role '%s'\n", policy.Arn(), binding.Name())
			}
		}
	}
	if len(missingBindings) == 0 {
		return output, false
	}
	output = output + "Run the following commands to attach the missing policies:\n"
	for roleName, policies := range missingBindings {
		output = output + fmt.Sprintf("rosa attach policy --role-name %s --policy-arns %s --mode auto\n",
			roleName, strings.Join(policies[:], ","))
	}
	return output, true
}

func TransformToRolePolicyDetails(bindingList *cmv1.RolePolicyBindingList) map[string][]aws.PolicyDetail {
	rolePolicyDetails := map[string][]aws.PolicyDetail{}
	for _, binding := range bindingList.Slice() {
		policyDetails := []aws.PolicyDetail{}
		if binding.Policies() != nil {
			for _, policy := range binding.Policies() {
				policyType := policy.Type()
				if policyType != aws.Inline {
					policyType = aws.Attached
				}
				policyDetails = append(policyDetails, aws.PolicyDetail{
					PolicyName: policy.Name(),
					PolicyArn:  policy.Arn(),
					PolicyType: policyType,
				})
			}
		}
		rolePolicyDetails[binding.Name()] = policyDetails
	}
	return rolePolicyDetails
}
