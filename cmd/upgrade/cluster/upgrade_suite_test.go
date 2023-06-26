package cluster_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpgradeCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade cluster Suite")
}
