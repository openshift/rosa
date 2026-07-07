package cluster

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestListCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "List Cluster Suite")
}
