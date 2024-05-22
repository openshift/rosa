package addon

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeAddon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "describe addon suite")
}
