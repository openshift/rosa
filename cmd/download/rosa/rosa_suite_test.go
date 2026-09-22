// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosa

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDownloadRosa(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Download Rosa Suite")
}
