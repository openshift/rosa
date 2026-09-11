package operatorrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOperatorRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete operator role suite")
}
