package machinepool

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEditMachinePool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Edit machinepool suite")
}
