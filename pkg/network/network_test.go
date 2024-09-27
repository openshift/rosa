package network

import (
	"os"

	gomock "go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network", func() {
	var (
		mockCtrl     *gomock.Controller
		networkSvc NetworkService
		params       map[string]string
		tags         map[string]string
		templateFile string
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		networkSvc = NewNetworkService()
		params = map[string]string{
			"Name":   "test-stack",
			"Region": "us-west-2",
		}
		tags = map[string]string{
			"Environment": "test",
		}
		templateFile = "test-template.yaml"

		// Mock reading template file
		os.WriteFile(templateFile, []byte("AWSTemplateFormatVersion: '2010-09-09'"), 0644)
	})

	AfterEach(func() {
		mockCtrl.Finish()
		os.Remove(templateFile)
	})

	It("should return an error if the template file does not exist", func() {
		err := networkSvc.CreateStack("nonexistent-template.yaml", params, tags)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unable to read template file"))
	})
})
