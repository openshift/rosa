package idp_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestIdp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Idp Suite")
}
