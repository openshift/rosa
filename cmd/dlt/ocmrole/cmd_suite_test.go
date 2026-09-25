// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package ocmrole

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteOcmRole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete OCM role suite")
}
