package externalauthprovider

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeExternalAuthProviders(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Describe external authentication provider suite")
}
