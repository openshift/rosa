package accountroles

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpgradeAccountRoles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade account-roles suite")
}
