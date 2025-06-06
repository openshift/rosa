package network

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/network"
	opts "github.com/openshift/rosa/pkg/options/network"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("Network", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Network Stack", func() {
		It("should create a stack successfully", func() {
			template := "example.yaml"
			serviceMock := network.NewMockNetworkService(ctrl)
			serviceMock.EXPECT().CreateStack(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
			err := serviceMock.CreateStack(&template, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("Validation functions", func() {
	var (
		ctrl     *gomock.Controller
		mockArgs *opts.NetworkUserOptions
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockArgs = &opts.NetworkUserOptions{}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("parseParams", func() {
		It("should correctly parse the tags and parameters", func() {
			mockArgs.Params = []string{"Tags=key1=value1, key2=value2", "Name=test-stack", "Region=us-east-1"}
			params, tags, err := network.ParseParams(mockArgs.Params)
			Expect(err).ToNot(HaveOccurred())
			Expect(params).To(Equal(map[string]string{"Name": "test-stack", "Region": "us-east-1"}))
			Expect(tags).To(Equal(map[string]string{"key1": "value1", "key2": "value2"}))
		})

		It("should return an error when parsing invalid tags and parameters", func() {
			mockArgs.Params = []string{"InvalidTag", "Name=test-stack", "Region=us-east-1"}
			params, tags, err := network.ParseParams(mockArgs.Params)
			Expect(err).To(HaveOccurred())
			Expect(params).To(BeEmpty())
			Expect(tags).To(BeEmpty())
		})

		It("should not return an error when parsing empty tags and parameters", func() {
			mockArgs.Params = []string{}
			params, tags, err := network.ParseParams(mockArgs.Params)
			Expect(err).ToNot(HaveOccurred())
			Expect(params).To(BeEmpty())
			Expect(tags).To(BeEmpty())
		})

		It("should return an error when parsing duplicate keys in tags and parameters", func() {
			mockArgs.Params = []string{"Tags=key1=value1,key1=value2", "Name=test-stack", "Region=us-east-1"}
			params, tags, err := network.ParseParams(mockArgs.Params)
			Expect(err).To(HaveOccurred())
			Expect(params).To(BeEmpty())
			Expect(tags).To(BeEmpty())
		})
	})

	Context("selectTemplate", func() {
		It("input template to directory path", func() {
			templateFile := "test-template.yaml"
			templateDir := "cmd/create/network/templates"
			templateSelected := network.SelectTemplate(templateDir, templateFile)
			Expect(templateSelected).To(Equal("cmd/create/network/templates/test-template.yaml/cloudformation.yaml"))
		})
	})

	Context("extractTemplateCommand", func() {
		It("should use default template when no arguments are provided", func() {
			argv := []string{}
			r := rosa.NewRuntime()
			templateCommand := "rosa-quickstart-default-vpc"
			templateFile := ""
			options := &opts.NetworkUserOptions{}
			options.TemplateDir = "cmd/create/network/templates"

			extractTemplateCommand(r, argv, options, &templateCommand, &templateFile)

			Expect(templateCommand).To(Equal("rosa-quickstart-default-vpc"))
			Expect(templateFile).To(Equal(CloudFormationTemplateFile))
		})

		It("should extract the first non-`--param` argument as the template command", func() {
			argv := []string{"my-template", "--param", "key=value"}
			r := rosa.NewRuntime()
			templateCommand := ""
			templateFile := ""
			options := &opts.NetworkUserOptions{}
			options.TemplateDir = "cmd/create/other/templates"

			extractTemplateCommand(r, argv, options, &templateCommand, &templateFile)

			Expect(templateCommand).To(Equal("my-template"))
			Expect(templateFile).To(Equal("cmd/create/other/templates/my-template/cloudformation.yaml"))
		})

		It("should extract the error telling the user to use `--template-dir", func() {
			argv := []string{"my-template", "--param", "key=value"}
			r := rosa.NewRuntime()
			templateCommand := ""
			templateFile := ""
			options := &opts.NetworkUserOptions{}
			options.TemplateDir = "cmd/create/network/templates"

			err := extractTemplateCommand(r, argv, options, &templateCommand, &templateFile)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("when using a custom template please use `--template-dir`" +
				" to specify the template directory"))
		})

		It("should use the default template when the extracted template command is 'rosa-quickstart-default-vpc'", func() {
			argv := []string{"rosa-quickstart-default-vpc"}
			r := rosa.NewRuntime()
			templateCommand := ""
			templateFile := ""
			options := &opts.NetworkUserOptions{}
			options.TemplateDir = "cmd/create/network/templates"

			extractTemplateCommand(r, argv, options, &templateCommand, &templateFile)

			Expect(templateCommand).To(Equal("rosa-quickstart-default-vpc"))
			Expect(templateFile).To(Equal(CloudFormationTemplateFile))
		})
	})
})
