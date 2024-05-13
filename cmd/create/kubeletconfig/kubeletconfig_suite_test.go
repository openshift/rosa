package kubeletconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCreateKubeletConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create KubeletConfig Suite")
}
