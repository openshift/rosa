package cluster

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete cluster suite")
}
