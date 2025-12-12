package logforwarder

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateLogForwarderOptions", func() {
	var (
		logForwarderOptions *CreateLogForwarderOptions
		userOptions         *CreateLogForwarderUserOptions
	)

	BeforeEach(func() {
		logForwarderOptions = NewCreateLogForwarderOptions()
		userOptions = NewCreateLogForwarderUserOptions()
	})

	Context("Bind", func() {
		It("should bind user options correctly", func() {
			userOptions.logFwdConfig = "test-config.yml"

			err := logForwarderOptions.Bind(userOptions)

			Expect(err).ToNot(HaveOccurred())
			Expect(logForwarderOptions.args.logFwdConfig).To(Equal("test-config.yml"))
		})

		It("should handle empty config file path", func() {
			userOptions.logFwdConfig = ""

			err := logForwarderOptions.Bind(userOptions)

			Expect(err).ToNot(HaveOccurred())
			Expect(logForwarderOptions.args.logFwdConfig).To(Equal(""))
		})
	})
})
