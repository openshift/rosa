package clusterregistryconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRegistryConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Config Suite")
}
