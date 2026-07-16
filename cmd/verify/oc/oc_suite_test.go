package oc

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVerifyOC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Verify OC Suite")
}
