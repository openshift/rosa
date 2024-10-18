package network

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNetworkOptions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network options suite")
}
