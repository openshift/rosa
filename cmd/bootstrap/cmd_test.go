package bootstrap

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/bootstrap"
	bsOpts "github.com/openshift/rosa/pkg/options/bootstrap"
)

var _ = Describe("Bootstrap", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Bootstrap Stack", func() {
		It("should create a stack successfully", func() {
			serviceMock := bootstrap.NewMockBootstrapService(ctrl)
			serviceMock.EXPECT().CreateStack(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
			err := serviceMock.CreateStack("example.yaml", nil, nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("Validation functions", func() {
	var (
		ctrl     *gomock.Controller
		mockArgs *bsOpts.BootstrapUserOptions
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockArgs = &bsOpts.BootstrapUserOptions{}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("parseParams", func() {
		It("should correctly parse the tags and parameters", func() {
			mockArgs.Params = []string{"Tags=key1=value1, key2=value2", "Name=test-stack", "Region=us-east-1"}
			params, tags := bootstrap.ParseParams(mockArgs.Params)
			Expect(params).To(Equal(map[string]string{"Name": "test-stack", "Region": "us-east-1"}))
			Expect(tags).To(Equal(map[string]string{"key1": "value1", " key2": "value2"}))
		})
	})

	Context("selectTemplate", func() {
		It("input template to directory path", func() {
			templateFile := "test-template.yaml"
			templateSelected := bootstrap.SelectTemplate(templateFile)
			Expect(templateSelected).To(Equal("cmd/bootstrap/templates/test-template.yaml/cloudformation.yaml"))
		})
	})
})
