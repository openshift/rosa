package aws

import (
	"context"
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	servicequotastypes "github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws/mocks"
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

	Context("GetIAMServiceQuota", func() {
		var (
			client          Client
			mockCtrl        *gomock.Controller
			mockIamQuotaAPI *mocks.MockServiceQuotasApiClient
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockIamQuotaAPI = mocks.NewMockServiceQuotasApiClient(mockCtrl)
			client = New(
				awsSdk.Config{},
				NewLoggerWrapper(logrus.New(), nil),
				mocks.NewMockIamApiClient(mockCtrl),
				mocks.NewMockEc2ApiClient(mockCtrl),
				mocks.NewMockOrganizationsApiClient(mockCtrl),
				mocks.NewMockS3ApiClient(mockCtrl),
				mocks.NewMockSecretsManagerApiClient(mockCtrl),
				mocks.NewMockStsApiClient(mockCtrl),
				mocks.NewMockCloudFormationApiClient(mockCtrl),
				mocks.NewMockServiceQuotasApiClient(mockCtrl),
				mockIamQuotaAPI,
				&AccessKey{},
				false,
			)
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("Returns quota on success", func() {
			quotaCode := "L-F4A5425F"
			expectedOutput := &servicequotas.GetServiceQuotaOutput{
				Quota: &servicequotastypes.ServiceQuota{
					QuotaCode: awsSdk.String(quotaCode),
					QuotaName: awsSdk.String("Roles per account"),
					Value:     awsSdk.Float64(1000.0),
				},
			}

			mockIamQuotaAPI.EXPECT().GetServiceQuota(
				context.Background(),
				&servicequotas.GetServiceQuotaInput{
					ServiceCode: awsSdk.String(IAMServiceCode),
					QuotaCode:   awsSdk.String(quotaCode),
				},
			).Return(expectedOutput, nil)

			result, err := client.GetIAMServiceQuota(quotaCode)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expectedOutput))
			Expect(*result.Quota.QuotaCode).To(Equal(quotaCode))
			Expect(*result.Quota.Value).To(Equal(1000.0))
		})

		It("Propagates API error", func() {
			quotaCode := "L-F4A5425F"

			mockIamQuotaAPI.EXPECT().GetServiceQuota(
				context.Background(),
				&servicequotas.GetServiceQuotaInput{
					ServiceCode: awsSdk.String(IAMServiceCode),
					QuotaCode:   awsSdk.String(quotaCode),
				},
			).Return(nil, fmt.Errorf("access denied"))

			result, err := client.GetIAMServiceQuota(quotaCode)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
			Expect(result).To(BeNil())
		})
	})
})
