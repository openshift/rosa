package test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test helpers suite")
}
