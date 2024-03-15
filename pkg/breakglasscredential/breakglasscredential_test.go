package breakglasscredential

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBreakGlassCredential(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ocm Suite")
}
