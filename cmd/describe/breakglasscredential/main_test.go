// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package breakglasscredential

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDescribeBreakGlassCredential(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Describe break glass credential suite")
}
