// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package roles

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRoles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Interactive roles suite")
}
