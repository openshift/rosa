package instancetypes_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInstancetypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instancetypes Suite")
}
