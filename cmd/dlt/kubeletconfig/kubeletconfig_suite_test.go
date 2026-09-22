// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package kubeletconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeleteKubeletConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete KubeletConfig Suite")
}
