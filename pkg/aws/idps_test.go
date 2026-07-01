package aws

import (
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws/mocks"
	"github.com/openshift/rosa/pkg/aws/tags"
)

var _ = Describe("OIDC Provider", func() {
	var (
		client     Client
		mockCtrl   *gomock.Controller
		mockIamAPI *mocks.MockIamApiClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		client = New(
			awsSdk.Config{},
			NewLoggerWrapper(logrus.New(), nil),
			mockIamAPI,
			mocks.NewMockEc2ApiClient(mockCtrl),
			mocks.NewMockOrganizationsApiClient(mockCtrl),
			mocks.NewMockS3ApiClient(mockCtrl),
			mocks.NewMockSecretsManagerApiClient(mockCtrl),
			mocks.NewMockStsApiClient(mockCtrl),
			mocks.NewMockCloudFormationApiClient(mockCtrl),
			mocks.NewMockServiceQuotasApiClient(mockCtrl),
			mocks.NewMockServiceQuotasApiClient(mockCtrl),
			&AccessKey{},
			false,
		)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("CreateOpenIDConnectProvider", func() {
		var (
			providerURL string
			thumbprint  string
			expectedARN string
		)

		BeforeEach(func() {
			providerURL = "https://oidc.example.com/some-id"
			thumbprint = "a]cfef"
			expectedARN = "arn:aws:iam::123456789012:oidc-provider/oidc.example.com/some-id"
		})

		It("Returns the ARN when created with a clusterID tag", func() {
			clusterID := "test-cluster-123"
			mockIamAPI.EXPECT().CreateOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ interface{}, input *iam.CreateOpenIDConnectProviderInput,
					_ ...interface{}) (*iam.CreateOpenIDConnectProviderOutput, error) {
					Expect(input.Url).To(Equal(&providerURL))
					Expect(input.ThumbprintList).To(Equal([]string{thumbprint}))
					Expect(input.ClientIDList).To(ConsistOf(OIDCClientIDOpenShift, OIDCClientIDSTSAWS))
					Expect(input.Tags).To(HaveLen(2))
					Expect(input.Tags).To(ContainElement(iamtypes.Tag{
						Key:   awsSdk.String(tags.RedHatManaged),
						Value: awsSdk.String(tags.True),
					}))
					Expect(input.Tags).To(ContainElement(iamtypes.Tag{
						Key:   awsSdk.String(tags.ClusterID),
						Value: awsSdk.String(clusterID),
					}))
					return &iam.CreateOpenIDConnectProviderOutput{
						OpenIDConnectProviderArn: awsSdk.String(expectedARN),
					}, nil
				})

			arn, err := client.CreateOpenIDConnectProvider(providerURL, thumbprint, clusterID)
			Expect(err).NotTo(HaveOccurred())
			Expect(arn).To(Equal(expectedARN))
		})

		It("Only includes RedHatManaged tag when clusterID is empty", func() {
			mockIamAPI.EXPECT().CreateOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ interface{}, input *iam.CreateOpenIDConnectProviderInput,
					_ ...interface{}) (*iam.CreateOpenIDConnectProviderOutput, error) {
					Expect(input.Tags).To(HaveLen(1))
					Expect(input.Tags[0]).To(Equal(iamtypes.Tag{
						Key:   awsSdk.String(tags.RedHatManaged),
						Value: awsSdk.String(tags.True),
					}))
					return &iam.CreateOpenIDConnectProviderOutput{
						OpenIDConnectProviderArn: awsSdk.String(expectedARN),
					}, nil
				})

			arn, err := client.CreateOpenIDConnectProvider(providerURL, thumbprint, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(arn).To(Equal(expectedARN))
		})

		It("Returns error when IAM API fails", func() {
			mockIamAPI.EXPECT().CreateOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("iam api error"))

			arn, err := client.CreateOpenIDConnectProvider(providerURL, thumbprint, "cluster-1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("iam api error"))
			Expect(arn).To(BeEmpty())
		})
	})

	Context("HasOpenIDConnectProvider", func() {
		var (
			issuerURL string
			partition string
			accountID string
		)

		BeforeEach(func() {
			issuerURL = "https://oidc.example.com/some-id"
			partition = "aws"
			accountID = "123456789012"
		})

		It("Returns true when the provider exists and URL matches", func() {
			expectedProviderURL := "oidc.example.com/some-id"
			mockIamAPI.EXPECT().GetOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ interface{}, input *iam.GetOpenIDConnectProviderInput,
					_ ...interface{}) (*iam.GetOpenIDConnectProviderOutput, error) {
					expectedARN := GetOIDCProviderARN(partition, accountID, expectedProviderURL)
					Expect(awsSdk.ToString(input.OpenIDConnectProviderArn)).To(Equal(expectedARN))
					return &iam.GetOpenIDConnectProviderOutput{
						Url: awsSdk.String(expectedProviderURL),
					}, nil
				})

			exists, err := client.HasOpenIDConnectProvider(issuerURL, partition, accountID)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("Returns false with no error when provider does not exist (NoSuchEntity)", func() {
			mockIamAPI.EXPECT().GetOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				Return(nil, &iamtypes.NoSuchEntityException{Message: awsSdk.String("not found")})

			exists, err := client.HasOpenIDConnectProvider(issuerURL, partition, accountID)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("Returns false with error for an invalid issuer URL", func() {
			exists, err := client.HasOpenIDConnectProvider("not-a-valid-url", partition, accountID)
			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("Returns false with error on non-NoSuchEntity IAM error", func() {
			mockIamAPI.EXPECT().GetOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("access denied"))

			exists, err := client.HasOpenIDConnectProvider(issuerURL, partition, accountID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
			Expect(exists).To(BeFalse())
		})

		It("Returns false with error when provider exists but URL is misconfigured", func() {
			mockIamAPI.EXPECT().GetOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				Return(&iam.GetOpenIDConnectProviderOutput{
					Url: awsSdk.String("different.host.com/other-id"),
				}, nil)

			exists, err := client.HasOpenIDConnectProvider(issuerURL, partition, accountID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("misconfigured"))
			Expect(exists).To(BeFalse())
		})
	})

	Context("DeleteOpenIDConnectProvider", func() {
		var oidcProviderARN string

		BeforeEach(func() {
			oidcProviderARN = "arn:aws:iam::123456789012:oidc-provider/oidc.example.com/some-id"
		})

		It("Deletes the provider successfully", func() {
			mockIamAPI.EXPECT().DeleteOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ interface{}, input *iam.DeleteOpenIDConnectProviderInput,
					_ ...interface{}) (*iam.DeleteOpenIDConnectProviderOutput, error) {
					Expect(awsSdk.ToString(input.OpenIDConnectProviderArn)).To(Equal(oidcProviderARN))
					return &iam.DeleteOpenIDConnectProviderOutput{}, nil
				})

			err := client.DeleteOpenIDConnectProvider(oidcProviderARN)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Returns a specific error when provider does not exist", func() {
			mockIamAPI.EXPECT().DeleteOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				Return(nil, &iamtypes.NoSuchEntityException{Message: awsSdk.String("not found")})

			err := client.DeleteOpenIDConnectProvider(oidcProviderARN)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				fmt.Sprintf("the OIDC provider '%s' does not exist", oidcProviderARN)))
		})

		It("Returns the raw error for other IAM failures", func() {
			mockIamAPI.EXPECT().DeleteOpenIDConnectProvider(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("service unavailable"))

			err := client.DeleteOpenIDConnectProvider(oidcProviderARN)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("service unavailable"))
		})
	})
})
