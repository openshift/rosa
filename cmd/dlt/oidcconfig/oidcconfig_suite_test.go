package oidcconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteOidcConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete OidcConfig Suite")
}
