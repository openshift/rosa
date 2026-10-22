// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("seaplane dispatch", func() {
	var (
		origEnabled    func() bool
		origRunCluster func(*cobra.Command)
	)

	BeforeEach(func() {
		origEnabled = seaplaneEnabled
		origRunCluster = seaplaneDeleteCluster
	})

	AfterEach(func() {
		seaplaneEnabled = origEnabled
		seaplaneDeleteCluster = origRunCluster
	})

	It("routes to seaplaneDeleteCluster when seaplane is enabled", func() {
		called := false
		seaplaneEnabled = func() bool { return true }
		seaplaneDeleteCluster = func(*cobra.Command) { called = true }

		dispatch(nil, nil)

		Expect(called).To(BeTrue())
	})
})
