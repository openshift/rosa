// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package decision

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCreateDecision(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create Decision Suite")
}
