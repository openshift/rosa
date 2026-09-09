package userrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLinkUserRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Link user role suite")
}
