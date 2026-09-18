// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package logforwarder

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLogForwarder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Edit Log Forwarder suite")
}
