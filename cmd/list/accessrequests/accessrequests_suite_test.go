// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package accessrequests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestListAccessRequests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create List Access Requests Suite")
}
