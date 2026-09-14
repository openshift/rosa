// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package roles

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpgradeRoles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade roles Suite")
}
