package machinepool

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestListMachinePool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "List machine pool suite")
}
