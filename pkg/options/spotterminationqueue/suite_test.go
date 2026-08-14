package spotterminationqueue

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSpotTerminationQueueOptions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spot termination queue options suite")
}
