// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEditCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Edit cluster suite")
}
