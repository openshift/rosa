// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package commandbuilder_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCommandbuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Commandbuilder Suite")
}
