// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Describe upgrades suite")
}
