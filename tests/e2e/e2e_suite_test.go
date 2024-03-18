package e2e

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var ctx context.Context
var clusterID string

func TestROSACLIProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e tests suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
})
