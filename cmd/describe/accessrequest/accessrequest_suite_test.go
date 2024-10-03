package accessrequest

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeAccessRequest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Describe AccessRequest Suite")
}
