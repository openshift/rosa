package ocmrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUnlinkOcmRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Unlink OCM role suite")
}
