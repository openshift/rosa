package machinepool

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpgradeMachinePool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade machine pool suite")
}
