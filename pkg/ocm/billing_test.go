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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

var _ = Describe("Billing Account", func() {
	Context("Functions should return the correct output", func() {
		It("KO: GenerateBillingAccountsList should return empty array if no CloudAccount is provided", func() {
			mockCloudAccounts := append([]*v1.CloudAccount{}, &v1.CloudAccount{})
			billingAccounts := GenerateBillingAccountsList(mockCloudAccounts)
			Expect(billingAccounts).To(BeNil())
		})

		It("OK: Successfully returns a list of billing accounts", func() {
			mockCloudAccount1 := v1.NewCloudAccount().CloudAccountID("1234567").CloudProviderID("aws").
				Contracts(v1.NewContract().StartDate(time.Now()).EndDate(time.Now().Add(2)).
					Dimensions(v1.NewContractDimension().Name("control_plane").Value("4")))
			cloudAccount1, err := mockCloudAccount1.Build()
			Expect(err).NotTo(HaveOccurred())

			mockCloudAccount2 := v1.NewCloudAccount().CloudAccountID("12345679").CloudProviderID("aws").
				Contracts(v1.NewContract().StartDate(time.Now()).EndDate(time.Now().Add(2)))
			cloudAccount2, err := mockCloudAccount2.Build()
			Expect(err).NotTo(HaveOccurred())

			billingAccountNames := GenerateBillingAccountsList([]*v1.CloudAccount{
				cloudAccount1,
				cloudAccount2,
			})

			expected := []string{"1234567 [Contract enabled]", "12345679"}
			Expect(billingAccountNames).To(Equal(expected))
		})

		It("OK: Successfully generates numbers of vCPUs and clusters", func() {
			t, err := time.Parse(time.RFC3339, "2023-08-07T15:22:00Z")
			Expect(err).To(BeNil())
			mockContract, err := v1.NewContract().StartDate(t).
				EndDate(t).
				Dimensions(v1.NewContractDimension().Name("control_plane").Value("4"),
					v1.NewContractDimension().Name("four_vcpu_hour").Value("5")).Build()
			Expect(err).NotTo(HaveOccurred())

			numberOfVCPUs, numberOfClusters := GetNumsOfVCPUsAndClusters(mockContract.Dimensions())
			Expect(numberOfVCPUs).To(Equal(5))
			Expect(numberOfClusters).To(Equal(4))
		})

		It("OK: Successfully verify valid contracts", func() {
			mockCloudAccount := v1.NewCloudAccount().CloudAccountID("1234567").
				Contracts(v1.NewContract().StartDate(time.Now()).EndDate(time.Now().Add(2)).
					Dimensions(v1.NewContractDimension().Name("control_plane").Value("4")))
			cloudAccount, err := mockCloudAccount.Build()
			Expect(err).NotTo(HaveOccurred())
			result := HasValidContracts(cloudAccount)
			Expect(result).To(Equal(true))
		})
	})
})
