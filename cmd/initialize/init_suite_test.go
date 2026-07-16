package initialize

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInitialize(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Initialize Suite")
}
