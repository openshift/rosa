package userrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCreateUserRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create user-role suite")
}
