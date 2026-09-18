// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package oidcprovider

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCreateOidcProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create OidcProvider Suite")
}
