package operatorroles

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpgradeOperatorRoles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade operator-roles suite")
}
