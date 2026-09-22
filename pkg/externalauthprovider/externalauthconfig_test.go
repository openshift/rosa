// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package externalauthprovider

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExternalAuthProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ocm Suite")
}
