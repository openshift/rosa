// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package quota

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQuota(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Verify Quota Suite")
}
