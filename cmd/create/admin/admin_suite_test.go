// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIdp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Admin Suite")
}
