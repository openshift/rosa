package userrole_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUserRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User role suite")
}
