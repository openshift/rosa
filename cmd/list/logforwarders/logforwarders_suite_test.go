package logforwarders

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLogForwarders(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "List LogForwarders Suite")
}
