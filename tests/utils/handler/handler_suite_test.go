// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handler Suite")
}
