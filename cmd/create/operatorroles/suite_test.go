package operatorroles

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDnsDomain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator role suite")
}
