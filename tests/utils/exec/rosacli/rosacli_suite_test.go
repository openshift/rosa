// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosacli

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRosacli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rosacli Suite")
}
