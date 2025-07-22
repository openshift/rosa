package roles

import (
	"fmt"
	"strings"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("Validate Shared VPC Inputs", func() {
	var ctrl *gomock.Controller
	var runtime *rosa.Runtime

	var testArn = "arn:aws:iam::123456789012:role/test"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("GetHcpSharedVpcPolicyDetails", func() {
		When("getHcpSharedVpcPolicyDetails", func() {
			It("Test that returned details + name are correct", func() {
				exists, details, name, err := GetHcpSharedVpcPolicyDetails(runtime, testArn)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
				Expect(name).To(Equal("test-assume-role"))
				expectedDetails := strings.Replace(details, fmt.Sprintf("%%{%s}", name), name, -1)
				Expect(details).To(Equal(expectedDetails))
			})
		})
	})
})
