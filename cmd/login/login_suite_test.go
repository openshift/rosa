package login

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLogin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Login Suite")
}
