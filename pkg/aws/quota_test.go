package aws

import (
	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	servicequotastypes "github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Quota", func() {

	Context("GetServiceQuota", func() {
		It("Returns matching quota when found", func() {
			quotaCode := "L-12345678"
			quotas := []servicequotastypes.ServiceQuota{
				{
					QuotaCode: awsSdk.String(quotaCode),
					QuotaName: awsSdk.String("Test Quota"),
					Value:     awsSdk.Float64(100.0),
				},
			}

			result, err := GetServiceQuota(quotas, quotaCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(*result.QuotaCode).To(Equal(quotaCode))
			Expect(*result.Value).To(Equal(100.0))
		})

		It("Returns error when quota is not found", func() {
			quotas := []servicequotastypes.ServiceQuota{
				{
					QuotaCode: awsSdk.String("L-AAAAAAAA"),
					QuotaName: awsSdk.String("Other Quota"),
					Value:     awsSdk.Float64(50.0),
				},
			}

			_, err := GetServiceQuota(quotas, "L-NOTFOUND")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to find quota with service code"))
		})

		It("Returns error for an empty slice", func() {
			_, err := GetServiceQuota([]servicequotastypes.ServiceQuota{}, "L-12345678")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to find quota with service code"))
		})

		It("Finds the correct quota among multiple entries", func() {
			targetCode := "L-BBBBBBBB"
			quotas := []servicequotastypes.ServiceQuota{
				{
					QuotaCode: awsSdk.String("L-AAAAAAAA"),
					QuotaName: awsSdk.String("First Quota"),
					Value:     awsSdk.Float64(10.0),
				},
				{
					QuotaCode: awsSdk.String(targetCode),
					QuotaName: awsSdk.String("Target Quota"),
					Value:     awsSdk.Float64(200.0),
				},
				{
					QuotaCode: awsSdk.String("L-CCCCCCCC"),
					QuotaName: awsSdk.String("Third Quota"),
					Value:     awsSdk.Float64(300.0),
				},
			}

			result, err := GetServiceQuota(quotas, targetCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(*result.QuotaCode).To(Equal(targetCode))
			Expect(*result.QuotaName).To(Equal("Target Quota"))
			Expect(*result.Value).To(Equal(200.0))
		})
	})
})
