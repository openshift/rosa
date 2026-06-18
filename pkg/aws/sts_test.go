package aws

import (
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("STS", func() {
	var (
		client     Client
		mockCtrl   *gomock.Controller
		mockSTSApi *mocks.MockStsApiClient
		mockIamAPI *mocks.MockIamApiClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		mockSTSApi = mocks.NewMockStsApiClient(mockCtrl)
		client = New(
			awsSdk.Config{},
			NewLoggerWrapper(logrus.New(), nil),
			mockIamAPI,
			mocks.NewMockEc2ApiClient(mockCtrl),
			mocks.NewMockOrganizationsApiClient(mockCtrl),
			mocks.NewMockS3ApiClient(mockCtrl),
			mocks.NewMockSecretsManagerApiClient(mockCtrl),
			mockSTSApi,
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

	Context("ValidateRoleARNAccountIDMatchCallerAccountID", func() {
		It("returns nil when role ARN account matches caller account", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(
				&sts.GetCallerIdentityOutput{
					Arn:     awsSdk.String("arn:aws:iam::123456789012:user/TestUser"),
					Account: awsSdk.String("123456789012"),
				}, nil,
			)

			err := client.ValidateRoleARNAccountIDMatchCallerAccountID(
				"arn:aws:iam::123456789012:role/MyRole",
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error when role ARN account differs from caller account", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(
				&sts.GetCallerIdentityOutput{
					Arn:     awsSdk.String("arn:aws:iam::123456789012:user/TestUser"),
					Account: awsSdk.String("123456789012"),
				}, nil,
			)

			err := client.ValidateRoleARNAccountIDMatchCallerAccountID(
				"arn:aws:iam::999999999999:role/OtherRole",
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("doesn't match the user's account ID"))
			Expect(err.Error()).To(ContainSubstring("999999999999"))
			Expect(err.Error()).To(ContainSubstring("123456789012"))
		})

		It("returns error for an invalid ARN string", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(
				&sts.GetCallerIdentityOutput{
					Arn:     awsSdk.String("arn:aws:iam::123456789012:user/TestUser"),
					Account: awsSdk.String("123456789012"),
				}, nil,
			)

			err := client.ValidateRoleARNAccountIDMatchCallerAccountID("not-a-valid-arn")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid prefix"))
		})

		It("returns error when GetCreator (GetCallerIdentity) fails", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(
				nil, fmt.Errorf("STS service unavailable"),
			)

			err := client.ValidateRoleARNAccountIDMatchCallerAccountID(
				"arn:aws:iam::123456789012:role/MyRole",
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get AWS creator"))
			Expect(err.Error()).To(ContainSubstring("STS service unavailable"))
		})

		It("works with GovCloud ARNs when accounts match", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(
				&sts.GetCallerIdentityOutput{
					Arn:     awsSdk.String("arn:aws-us-gov:iam::111222333444:user/GovUser"),
					Account: awsSdk.String("111222333444"),
				}, nil,
			)

			err := client.ValidateRoleARNAccountIDMatchCallerAccountID(
				"arn:aws-us-gov:iam::111222333444:role/GovRole",
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns mismatch error with GovCloud ARNs when accounts differ", func() {
			mockSTSApi.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(
				&sts.GetCallerIdentityOutput{
					Arn:     awsSdk.String("arn:aws-us-gov:iam::111222333444:user/GovUser"),
					Account: awsSdk.String("111222333444"),
				}, nil,
			)

			err := client.ValidateRoleARNAccountIDMatchCallerAccountID(
				"arn:aws-us-gov:iam::555666777888:role/OtherGovRole",
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("doesn't match the user's account ID"))
		})
	})

	Context("HasPermissionsBoundary", func() {
		It("returns true when role has a permissions boundary", func() {
			mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
				&iam.GetRoleOutput{
					Role: &iamtypes.Role{
						RoleName: awsSdk.String("test-role"),
						PermissionsBoundary: &iamtypes.AttachedPermissionsBoundary{
							PermissionsBoundaryArn: awsSdk.String("arn:aws:iam::123456789012:policy/boundary"),
						},
					},
				}, nil,
			)

			hasBoundary, err := client.HasPermissionsBoundary("test-role")
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBoundary).To(BeTrue())
		})

		It("returns false when role has no permissions boundary", func() {
			mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
				&iam.GetRoleOutput{
					Role: &iamtypes.Role{
						RoleName:            awsSdk.String("test-role"),
						PermissionsBoundary: nil,
					},
				}, nil,
			)

			hasBoundary, err := client.HasPermissionsBoundary("test-role")
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBoundary).To(BeFalse())
		})

		It("returns error when GetRole API call fails", func() {
			mockIamAPI.EXPECT().GetRole(gomock.Any(), gomock.Any()).Return(
				nil, fmt.Errorf("NoSuchEntity: role not found"),
			)

			hasBoundary, err := client.HasPermissionsBoundary("nonexistent-role")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("NoSuchEntity"))
			Expect(hasBoundary).To(BeFalse())
		})
	})

	Context("SortRolesByLinkedRole", func() {
		It("puts linked roles before unlinked roles", func() {
			roles := []Role{
				{RoleName: "unlinked-1", Linked: RoleNo},
				{RoleName: "linked-1", Linked: RoleYes},
				{RoleName: "unlinked-2", Linked: RoleNo},
				{RoleName: "linked-2", Linked: RoleYes},
			}

			SortRolesByLinkedRole(roles)

			Expect(roles[0].Linked).To(Equal(RoleYes))
			Expect(roles[1].Linked).To(Equal(RoleYes))
			Expect(roles[2].Linked).To(Equal(RoleNo))
			Expect(roles[3].Linked).To(Equal(RoleNo))
		})

		It("preserves relative order within linked roles (stable sort)", func() {
			roles := []Role{
				{RoleName: "linked-A", Linked: RoleYes},
				{RoleName: "linked-B", Linked: RoleYes},
				{RoleName: "linked-C", Linked: RoleYes},
			}

			SortRolesByLinkedRole(roles)

			Expect(roles[0].RoleName).To(Equal("linked-A"))
			Expect(roles[1].RoleName).To(Equal("linked-B"))
			Expect(roles[2].RoleName).To(Equal("linked-C"))
		})

		It("preserves relative order within unlinked roles (stable sort)", func() {
			roles := []Role{
				{RoleName: "unlinked-X", Linked: RoleNo},
				{RoleName: "unlinked-Y", Linked: RoleNo},
				{RoleName: "unlinked-Z", Linked: RoleNo},
			}

			SortRolesByLinkedRole(roles)

			Expect(roles[0].RoleName).To(Equal("unlinked-X"))
			Expect(roles[1].RoleName).To(Equal("unlinked-Y"))
			Expect(roles[2].RoleName).To(Equal("unlinked-Z"))
		})

		It("does not panic on an empty slice", func() {
			roles := []Role{}
			Expect(func() { SortRolesByLinkedRole(roles) }).NotTo(Panic())
			Expect(roles).To(BeEmpty())
		})

		It("handles a single linked role", func() {
			roles := []Role{
				{RoleName: "only-role", Linked: RoleYes},
			}

			SortRolesByLinkedRole(roles)

			Expect(roles).To(HaveLen(1))
			Expect(roles[0].RoleName).To(Equal("only-role"))
		})

		It("handles a single unlinked role", func() {
			roles := []Role{
				{RoleName: "only-role", Linked: RoleNo},
			}

			SortRolesByLinkedRole(roles)

			Expect(roles).To(HaveLen(1))
			Expect(roles[0].RoleName).To(Equal("only-role"))
		})

		It("preserves stability when linked and unlinked are interleaved", func() {
			roles := []Role{
				{RoleName: "unlinked-1", Linked: RoleNo},
				{RoleName: "linked-1", Linked: RoleYes},
				{RoleName: "unlinked-2", Linked: RoleNo},
				{RoleName: "linked-2", Linked: RoleYes},
				{RoleName: "unlinked-3", Linked: RoleNo},
			}

			SortRolesByLinkedRole(roles)

			Expect(roles[0].RoleName).To(Equal("linked-1"))
			Expect(roles[1].RoleName).To(Equal("linked-2"))
			Expect(roles[2].RoleName).To(Equal("unlinked-1"))
			Expect(roles[3].RoleName).To(Equal("unlinked-2"))
			Expect(roles[4].RoleName).To(Equal("unlinked-3"))
		})
	})
})
