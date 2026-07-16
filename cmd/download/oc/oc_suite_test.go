package oc

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDownloadOC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Download OC Suite")
}
