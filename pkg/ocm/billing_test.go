/**
Copyright (c) 2023 Red Hat, Inc.

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

package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

var _ = Describe("GenerateBillingAccountsList", func() {
	Context("GenerateBillingAccountsList should return the correct output", func() {
		It("KO: GenerateBillingAccountsList should return empty array if no CloudAccount is provided", func() {
			mockCloudAccounts := append([]*v1.CloudAccount{}, &v1.CloudAccount{})
			billingAccounts := GenerateBillingAccountsList(mockCloudAccounts)
			Expect(billingAccounts).To(BeNil())
		})
	})
})
