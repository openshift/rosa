package version

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRosaVersion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ROSA Version CMD Suite")
}
