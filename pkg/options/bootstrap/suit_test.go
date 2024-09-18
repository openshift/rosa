package Bootstrap

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBootstrapOptions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bootstrap options suite")
}
