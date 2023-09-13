package machinepool

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeMachinePool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Describe machine pool suite")
}
