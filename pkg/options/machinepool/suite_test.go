package machinepool

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCreateMachinePoolOptions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create machine pool options suite")
}
