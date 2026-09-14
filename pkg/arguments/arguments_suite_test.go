// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package arguments

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArguments(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Arguments Suite")
}
