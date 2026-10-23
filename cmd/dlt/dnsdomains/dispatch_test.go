// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package dnsdomains

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("DNS domain dispatch", func() {
	DescribeTable("selects the configured API", func(enabled bool) {
		originalEnabled, originalV2, originalV1 := seaplaneEnabled, runPlatformAPI, runOCM
		DeferCleanup(func() { seaplaneEnabled, runPlatformAPI, runOCM = originalEnabled, originalV2, originalV1 })
		seaplaneEnabled = func() bool { return enabled }
		cmd := &cobra.Command{}
		argv := []string{"domain.example.com"}
		selected := ""
		runPlatformAPI = func(got *cobra.Command, args []string) {
			Expect(got).To(BeIdenticalTo(cmd))
			Expect(args).To(Equal(argv))
			selected = "v2"
		}
		runOCM = func(got *cobra.Command, args []string) {
			Expect(got).To(BeIdenticalTo(cmd))
			Expect(args).To(Equal(argv))
			selected = "v1"
		}
		dispatch(cmd, argv)
		if enabled {
			Expect(selected).To(Equal("v2"))
		} else {
			Expect(selected).To(Equal("v1"))
		}
	}, Entry("Platform API", true), Entry("OCM", false))
})
