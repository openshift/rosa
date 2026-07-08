package idp

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteIdp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete IDP Suite")
}
