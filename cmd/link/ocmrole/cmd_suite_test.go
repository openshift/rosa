// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package ocmrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLinkOcmRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Link OCM role suite")
}
