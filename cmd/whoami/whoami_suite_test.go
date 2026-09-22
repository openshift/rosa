// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package whoami

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWhoami(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Whoami Suite")
}
