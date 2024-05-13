package kubeletconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEditKubeletConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Edit KubeletConfig Suite")
}
