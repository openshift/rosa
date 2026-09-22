// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package oidcconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOidcConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create OidcConfig Suite")
}
