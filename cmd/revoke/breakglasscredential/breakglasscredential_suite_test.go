// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package breakglasscredential_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBreakGlassCredential(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Breakglasscredential Suite")
}
