package dnsdomains

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDNSDomain(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "Delete DNS domain suite") }
