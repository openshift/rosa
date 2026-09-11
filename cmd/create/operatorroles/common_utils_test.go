package operatorroles

import (
	"fmt"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("Create dns domain", func() {
	var ctrl *gomock.Controller
	var runtime *rosa.Runtime

	var testPartition = "test"
	var testArn = "arn:aws:iam::123456789012:role/test"
	var testVersion = "2012-10-17"
	var mockClient *aws.MockClient

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		runtime = rosa.NewRuntime()
		mockClient = aws.NewMockClient(ctrl)
		runtime.AWSClient = mockClient
		mockClient.EXPECT().GetCreator().Return(&aws.Creator{Partition: testPartition}, nil)

		mockClient.EXPECT().IsPolicyExists(gomock.Any()).Return(nil, nil).AnyTimes()

		creator, err := runtime.AWSClient.GetCreator()
		Expect(err).ToNot(HaveOccurred())
		runtime.Creator = creator
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Common Utils for create/operatorroles Test", func() {
		When("getHcpSharedVpcPolicy", func() {
			It("OK: Gets policy arn back", func() {
				returnedArn := "arn:aws:iam::123123123123:policy/test"
				mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any()).Return(returnedArn, nil)
				arn, err := getHcpSharedVpcPolicy(runtime, testArn, testVersion)
				Expect(err).ToNot(HaveOccurred())
				Expect(arn).To(Equal(returnedArn))
			})
			It("KO: Returns empty policy when fails", func() {
				mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any()).Return("", errors.UserErrorf("Failed"))
				arn, err := getHcpSharedVpcPolicy(runtime, testArn, testVersion)
				Expect(err).To(HaveOccurred())
				Expect(arn).To(Equal(""))
			})
		})
	})
})

var _ = Describe("validateIngressOperatorPolicyOverride", func() {
	var ctrl *gomock.Controller
	var runtime *rosa.Runtime
	var mockClient *aws.MockClient

	const (
		testPolicyArn    = "arn:aws:iam::123456789012:policy/test-policy"
		testSharedVpcArn = "arn:aws:iam::999888777666:role/shared-vpc-role"
		testInstallerPfx = "my-prefix"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		runtime = rosa.NewRuntime()
		mockClient = aws.NewMockClient(ctrl)
		runtime.AWSClient = mockClient
		mockClient.EXPECT().GetCreator().Return(&aws.Creator{Partition: "aws"}, nil)
		creator, err := runtime.AWSClient.GetCreator()
		Expect(err).ToNot(HaveOccurred())
		runtime.Creator = creator
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	When("the policy does not exist", func() {
		It("returns nil without further checks", func() {
			mockClient.EXPECT().IsPolicyExists(testPolicyArn).Return(nil, fmt.Errorf("NoSuchEntity"))
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("the policy exists", func() {
		BeforeEach(func() {
			mockClient.EXPECT().IsPolicyExists(testPolicyArn).Return(nil, nil)
		})

		It("returns error when GetDefaultPolicyDocument fails", func() {
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).
				Return("", fmt.Errorf("access denied"))
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
		})

		It("returns error when policy document is invalid JSON", func() {
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).
				Return("not-json", nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).To(HaveOccurred())
		})

		It("returns nil when policy has matching shared VPC role ARN", func() {
			doc := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "sts:AssumeRole",
					"Resource": "%s"
				}]
			}`, testSharedVpcArn)
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error when policy has different shared VPC role ARN", func() {
			differentArn := "arn:aws:iam::111111111111:role/other-role"
			doc := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "sts:AssumeRole",
					"Resource": "%s"
				}]
			}`, differentArn)
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unexpected shared VPC role ARN"))
		})

		It("returns nil when policy has no sts:AssumeRole statement", func() {
			doc := `{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::my-bucket/*"
				}]
			}`
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns nil when policy has Deny effect with sts:AssumeRole", func() {
			doc := fmt.Sprintf(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Deny",
					"Action": "sts:AssumeRole",
					"Resource": "%s"
				}]
			}`, "arn:aws:iam::111111111111:role/other-role")
			mockClient.EXPECT().GetDefaultPolicyDocument(testPolicyArn).Return(doc, nil)
			err := validateIngressOperatorPolicyOverride(runtime, testPolicyArn, testSharedVpcArn, testInstallerPfx)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
