package e2e

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/utils/config"
)

var ctx context.Context
var clusterID string

func TestROSACLIProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ROSA CLI e2e tests suite")
}

var _ = BeforeSuite(func() {
	clusterID = config.GetClusterID()
	ctx = context.Background()
})
