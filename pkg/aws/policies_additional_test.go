package aws

import (
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	common "github.com/openshift-online/ocm-common/pkg/aws/validations"

	"github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("hasCompatibleMajorMinorVersionTags", func() {
	var client awsClient

	BeforeEach(func() {
		client = awsClient{}
	})

	It("Returns true for exact version match", func() {
		tags := []iamtypes.Tag{
			{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("4.14.0")},
		}
		result, err := client.hasCompatibleMajorMinorVersionTags(tags, "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("Returns true when tag version has higher minor than input", func() {
		tags := []iamtypes.Tag{
			{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("4.15.0")},
		}
		result, err := client.hasCompatibleMajorMinorVersionTags(tags, "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("Returns false when tag version has lower minor than input", func() {
		tags := []iamtypes.Tag{
			{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("4.13.0")},
		}
		result, err := client.hasCompatibleMajorMinorVersionTags(tags, "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Returns true when same major.minor with different patch", func() {
		tags := []iamtypes.Tag{
			{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("4.14.5")},
		}
		result, err := client.hasCompatibleMajorMinorVersionTags(tags, "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("Returns false when tags are empty", func() {
		result, err := client.hasCompatibleMajorMinorVersionTags([]iamtypes.Tag{}, "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Returns false when no OpenShiftVersion tag exists", func() {
		tags := []iamtypes.Tag{
			{Key: awsSdk.String("some-other-key"), Value: awsSdk.String("4.14.0")},
		}
		result, err := client.hasCompatibleMajorMinorVersionTags(tags, "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Returns error when tag version string is invalid", func() {
		tags := []iamtypes.Tag{
			{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("not-a-version")},
		}
		_, err := client.hasCompatibleMajorMinorVersionTags(tags, "4.14.0")
		Expect(err).To(HaveOccurred())
	})

	It("Returns error when input version string is invalid", func() {
		tags := []iamtypes.Tag{
			{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("4.14.0")},
		}
		_, err := client.hasCompatibleMajorMinorVersionTags(tags, "not-a-version")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("IsPolicyCompatible", func() {
	var (
		client     awsClient
		mockIamAPI *mocks.MockIamApiClient
		mockCtrl   *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
		client = awsClient{iamClient: mockIamAPI}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("Returns true when version is empty", func() {
		result, err := client.IsPolicyCompatible("arn:aws:iam::123456789:policy/test", "")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("Returns true when policy tags have compatible version", func() {
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListPolicyTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("4.14.0")},
				},
			}, nil)
		result, err := client.IsPolicyCompatible("arn:aws:iam::123456789:policy/test", "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("Returns true when policy tags have higher version", func() {
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListPolicyTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("4.15.0")},
				},
			}, nil)
		result, err := client.IsPolicyCompatible("arn:aws:iam::123456789:policy/test", "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("Returns false when policy tags have incompatible version", func() {
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListPolicyTagsOutput{
				Tags: []iamtypes.Tag{
					{Key: awsSdk.String(common.OpenShiftVersion), Value: awsSdk.String("4.13.0")},
				},
			}, nil)
		result, err := client.IsPolicyCompatible("arn:aws:iam::123456789:policy/test", "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Returns false when policy has no tags", func() {
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(
			&iam.ListPolicyTagsOutput{
				Tags: []iamtypes.Tag{},
			}, nil)
		result, err := client.IsPolicyCompatible("arn:aws:iam::123456789:policy/test", "4.14.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Returns error when ListPolicyTags fails", func() {
		mockIamAPI.EXPECT().ListPolicyTags(gomock.Any(), gomock.Any()).Return(
			nil, fmt.Errorf("access denied"))
		_, err := client.IsPolicyCompatible("arn:aws:iam::123456789:policy/test", "4.14.0")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("access denied"))
	})
})
