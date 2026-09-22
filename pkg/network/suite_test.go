// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNetwork(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network Suite")
}
