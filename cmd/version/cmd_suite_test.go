// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRosaVersion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ROSA Version CMD Suite")
}
