package logforwarders

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeLogForwarders(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Describe log forwarders suite")
}
