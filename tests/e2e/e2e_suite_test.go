package e2e

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var ctx context.Context
var clusterID string

func TestROSACLIProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ROSA CLI e2e tests suite")
}

var _ = BeforeSuite(func() {
	clusterID = os.Getenv("CLUSTER_ID")
	ctx = context.Background()
})
