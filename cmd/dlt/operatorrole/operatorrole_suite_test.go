package operatorrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteOperatorRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete operator-roles suite")
}
