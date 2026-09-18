// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package externalauthprovider_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExternalAuthProviders(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ExternalAuthProviders Suite")
}
