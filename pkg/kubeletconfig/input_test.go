package kubeletconfig

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("KubeletConfig Input", func() {
	It("Generates the abort message", func() {
		msg := buildAbortMessage(OperationCreate, "foo")
		Expect(msg).To(Equal("Create of KubeletConfig for cluster 'foo' aborted."))
	})

	It("Generates the prompt message", func() {
		msg := buildPromptMessage(OperationCreate, "foo")
		Expect(msg).To(Equal("Creating the KubeletConfig for cluster 'foo' will cause all non-Control Plane " +
			"nodes to reboot. This may cause outages to your applications. Do you wish to continue?"))
	})
})
