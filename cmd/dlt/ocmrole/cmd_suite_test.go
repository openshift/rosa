package ocmrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteOcmRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete OCM role suite")
}
