// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestROSACLISeaplaneE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ROSA CLI Seaplane E2E tests suite")
}
