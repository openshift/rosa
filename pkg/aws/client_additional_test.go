package aws

import (
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretsmanagertypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("Client Additional", func() {
	var (
		client   Client
		mockCtrl *gomock.Controller

		mockIamAPI            *mocks.MockIamApiClient
		mockSecretsManagerAPI *mocks.MockSecretsManagerApiClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		mockSecretsManagerAPI = mocks.NewMockSecretsManagerApiClient(mockCtrl)
		client = New(
			awsSdk.Config{},
			NewLoggerWrapper(logrus.New(), nil),
			mockIamAPI,
			mocks.NewMockEc2ApiClient(mockCtrl),
			mocks.NewMockOrganizationsApiClient(mockCtrl),
			mocks.NewMockS3ApiClient(mockCtrl),
			mockSecretsManagerAPI,
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

	Context("CheckRoleExists", func() {
		var (
			roleName string
			roleARN  string
		)

		BeforeEach(func() {
			roleName = "test-role"
			roleARN = "arn:aws:iam::123456789012:role/test-role"
		})

		It("Returns true with ARN when the role exists", func() {
			mockIamAPI.EXPECT().GetRole(gomock.Any(), &iam.GetRoleInput{
				RoleName: awsSdk.String(roleName),
			}).Return(&iam.GetRoleOutput{
				Role: &iamtypes.Role{
					Arn:      awsSdk.String(roleARN),
					RoleName: awsSdk.String(roleName),
				},
			}, nil)

			exists, arn, err := client.CheckRoleExists(roleName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(arn).To(Equal(roleARN))
		})

		It("Returns false when the role does not exist", func() {
			mockIamAPI.EXPECT().GetRole(gomock.Any(), &iam.GetRoleInput{
				RoleName: awsSdk.String(roleName),
			}).Return(nil, &iamtypes.NoSuchEntityException{
				Message: awsSdk.String("not found"),
			})

			exists, arn, err := client.CheckRoleExists(roleName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(arn).To(BeEmpty())
		})

		It("Returns error for non-NoSuchEntity API failures", func() {
			mockIamAPI.EXPECT().GetRole(gomock.Any(), &iam.GetRoleInput{
				RoleName: awsSdk.String(roleName),
			}).Return(nil, fmt.Errorf("access denied"))

			exists, arn, err := client.CheckRoleExists(roleName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
			Expect(exists).To(BeFalse())
			Expect(arn).To(BeEmpty())
		})
	})

	Context("GetRoleByARN", func() {
		It("Returns the role for a valid role ARN", func() {
			roleName := "MyRole"
			roleARN := "arn:aws:iam::123456789012:role/MyRole"

			mockIamAPI.EXPECT().GetRole(gomock.Any(), &iam.GetRoleInput{
				RoleName: awsSdk.String(roleName),
			}).Return(&iam.GetRoleOutput{
				Role: &iamtypes.Role{
					Arn:      awsSdk.String(roleARN),
					RoleName: awsSdk.String(roleName),
				},
			}, nil)

			role, err := client.GetRoleByARN(roleARN)
			Expect(err).NotTo(HaveOccurred())
			Expect(*role.RoleName).To(Equal(roleName))
			Expect(*role.Arn).To(Equal(roleARN))
		})

		It("Returns error for an invalid ARN string", func() {
			role, err := client.GetRoleByARN("not-an-arn")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected 'not-an-arn' to be a valid IAM role ARN"))
			Expect(role).To(Equal(iamtypes.Role{}))
		})

		It("Returns error when ARN is for a non-role resource", func() {
			userARN := "arn:aws:iam::123456789012:user/Bob"

			role, err := client.GetRoleByARN(userARN)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected ARN"))
			Expect(err.Error()).To(ContainSubstring("to be IAM role resource"))
			Expect(role).To(Equal(iamtypes.Role{}))
		})

		It("Extracts role name correctly from ARN with path", func() {
			roleARN := "arn:aws:iam::123456789012:role/path/to/MyRole"

			mockIamAPI.EXPECT().GetRole(gomock.Any(), &iam.GetRoleInput{
				RoleName: awsSdk.String("MyRole"),
			}).Return(&iam.GetRoleOutput{
				Role: &iamtypes.Role{
					Arn:      awsSdk.String(roleARN),
					RoleName: awsSdk.String("MyRole"),
				},
			}, nil)

			role, err := client.GetRoleByARN(roleARN)
			Expect(err).NotTo(HaveOccurred())
			Expect(*role.RoleName).To(Equal("MyRole"))
		})
	})

	Context("GetRoleByName", func() {
		It("Returns the role on success", func() {
			roleName := "test-role"
			roleARN := "arn:aws:iam::123456789012:role/test-role"

			mockIamAPI.EXPECT().GetRole(gomock.Any(), &iam.GetRoleInput{
				RoleName: awsSdk.String(roleName),
			}).Return(&iam.GetRoleOutput{
				Role: &iamtypes.Role{
					Arn:      awsSdk.String(roleARN),
					RoleName: awsSdk.String(roleName),
				},
			}, nil)

			role, err := client.GetRoleByName(roleName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*role.RoleName).To(Equal(roleName))
			Expect(*role.Arn).To(Equal(roleARN))
		})

		It("Returns error on API failure", func() {
			roleName := "nonexistent-role"

			mockIamAPI.EXPECT().GetRole(gomock.Any(), &iam.GetRoleInput{
				RoleName: awsSdk.String(roleName),
			}).Return(nil, fmt.Errorf("api error"))

			role, err := client.GetRoleByName(roleName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("api error"))
			Expect(role).To(Equal(iamtypes.Role{}))
		})
	})

	Context("CreateSecretInSecretsManager", func() {
		It("Returns the secret ARN on success", func() {
			secretName := "my-secret"
			secretValue := "super-secret-value"
			secretARN := "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf"

			mockSecretsManagerAPI.EXPECT().CreateSecret(gomock.Any(), gomock.Any()).Return(
				&secretsmanager.CreateSecretOutput{
					ARN: awsSdk.String(secretARN),
				}, nil)

			resultARN, err := client.CreateSecretInSecretsManager(secretName, secretValue)
			Expect(err).NotTo(HaveOccurred())
			Expect(resultARN).To(Equal(secretARN))
		})

		It("Returns error on API failure", func() {
			mockSecretsManagerAPI.EXPECT().CreateSecret(gomock.Any(), gomock.Any()).Return(
				nil, fmt.Errorf("secret creation failed"))

			_, err := client.CreateSecretInSecretsManager("my-secret", "value")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("secret creation failed"))
		})
	})

	Context("DeleteSecretInSecretsManager", func() {
		var secretARN string

		BeforeEach(func() {
			secretARN = "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf"
		})

		It("Deletes the secret when it exists", func() {
			mockSecretsManagerAPI.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
				&secretsmanager.DescribeSecretOutput{}, nil)
			mockSecretsManagerAPI.EXPECT().DeleteSecret(gomock.Any(), gomock.Any()).Return(
				&secretsmanager.DeleteSecretOutput{}, nil)

			err := client.DeleteSecretInSecretsManager(secretARN)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Returns nil when the secret is not found", func() {
			mockSecretsManagerAPI.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
				nil, &secretsmanagertypes.ResourceNotFoundException{
					Message: awsSdk.String("not found"),
				})

			err := client.DeleteSecretInSecretsManager(secretARN)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Returns error when DeleteSecret fails", func() {
			mockSecretsManagerAPI.EXPECT().DescribeSecret(gomock.Any(), gomock.Any()).Return(
				&secretsmanager.DescribeSecretOutput{}, nil)
			mockSecretsManagerAPI.EXPECT().DeleteSecret(gomock.Any(), gomock.Any()).Return(
				nil, fmt.Errorf("delete failed"))

			err := client.DeleteSecretInSecretsManager(secretARN)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("delete failed"))
		})
	})
})
