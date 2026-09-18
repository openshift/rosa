package userrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteUserRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete user-role suite")
}
