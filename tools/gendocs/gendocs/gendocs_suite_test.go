package gendocs_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGendocs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gendocs Suite")
}
