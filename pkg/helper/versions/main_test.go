package versions

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVersionHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Version Helpers")
}
