package tuningconfigs

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTuningConfigs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TuningConfigs Create Suite")
}
