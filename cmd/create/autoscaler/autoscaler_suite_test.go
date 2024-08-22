package autoscaler

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAutoscaler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create Austoscaler Suite")
}
