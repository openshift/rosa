package logforwarding

import (
	"testing"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"
)

func TestHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Helper Suite")
}
