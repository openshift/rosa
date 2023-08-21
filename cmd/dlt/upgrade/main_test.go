package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete upgrade suite")
}
