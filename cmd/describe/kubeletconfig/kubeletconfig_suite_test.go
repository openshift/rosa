package kubeletconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeKubeletConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Describe KubeletConfig Suite")
}
