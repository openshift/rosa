package dnsdomains

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDnsDomain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DNS domain suite")
}
