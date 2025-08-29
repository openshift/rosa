package imagemirror

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImageMirror(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete ImageMirror suite")
}
