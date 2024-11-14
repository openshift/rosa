package operatorroles

import (
	"fmt"
	"strings"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("Create dns domain", func() {
	var ctrl *gomock.Controller
	var runtime *rosa.Runtime

	var testPartition = "test"
	var testArn = "arn:aws:iam::123456789012:role/test"
	var testRoleName = "test"
	var testIamTags = map[string]string{tags.RedHatManaged: aws.TrueString}
	var testPath = "/path"
	var testOperator *cmv1.STSOperator
	var testVersion = "2012-10-17"
	var mockClient *aws.MockClient

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		runtime = rosa.NewRuntime()
		mockClient = aws.NewMockClient(ctrl)
		runtime.AWSClient = mockClient
		mockClient.EXPECT().GetCreator().Return(&aws.Creator{Partition: testPartition}, nil)

		var err error
		testOperator, err = cmv1.NewSTSOperator().Namespace("test").Namespace("test-namespace").Build()
		Expect(err).ToNot(HaveOccurred())

		creator, err := runtime.AWSClient.GetCreator()
		Expect(err).ToNot(HaveOccurred())
		runtime.Creator = creator
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Common Utils for create/operatorroles Test", func() {
		When("getHcpSharedVpcPolicyDetails", func() {
			It("Test that returned details + name are correct", func() {
				details, name := getHcpSharedVpcPolicyDetails(runtime, testArn, testRoleName, testIamTags, testPath)
				Expect(name).To(Equal("rosa-assume-role-test"))
				expectedDetails := strings.Replace(details, fmt.Sprintf("%%{%s}", name), name, -1)
				Expect(details).To(Equal(expectedDetails))
			})
		})
		When("getHcpSharedVpcPolicy", func() {
			It("OK: Gets policy arn back", func() {
				returnedArn := "arn:aws:iam::123123123123:policy/test"
				mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any()).Return(returnedArn, nil)
				arn, err := getHcpSharedVpcPolicy(runtime, testArn, testRoleName, testOperator, testPath, testVersion)
				Expect(err).ToNot(HaveOccurred())
				Expect(arn).To(Equal(returnedArn))
			})
			It("KO: Returns empty policy when fails", func() {
				mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any()).Return("", errors.UserErrorf("Failed"))
				arn, err := getHcpSharedVpcPolicy(runtime, testArn, testRoleName, testOperator, testPath, testVersion)
				Expect(err).To(HaveOccurred())
				Expect(arn).To(Equal(""))
			})
		})
	})
})
