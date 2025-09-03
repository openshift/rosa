package imagemirrors

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImageMirrors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "List ImageMirrors suite")
}
