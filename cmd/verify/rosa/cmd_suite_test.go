package rosa

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRosaVerify(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ROSA Verify Suite")
}
