// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package userrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUnlinkUserRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Unlink user role suite")
}
