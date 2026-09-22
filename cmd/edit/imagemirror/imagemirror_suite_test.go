// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package imagemirror

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImageMirror(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Edit ImageMirror suite")
}
