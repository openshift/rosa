// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpgradeMachinePool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade machine pool suite")
}
