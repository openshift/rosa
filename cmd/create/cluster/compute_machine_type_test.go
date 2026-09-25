/*
Copyright (c) 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
)

var _ = Describe("resolveComputeMachineTypeDefault", func() {
	It("leaves HCP unset so CS can apply rosaHcpDefaults", func() {
		Expect(resolveComputeMachineTypeDefault(true, "m5.xlarge")).To(BeEmpty())
	})

	It("keeps the flavour default for classic clusters", func() {
		Expect(resolveComputeMachineTypeDefault(false, "m5.xlarge")).To(Equal("m5.xlarge"))
	})
})

var _ = Describe("interactiveComputeMachineTypeDefault", func() {
	It("prefers m7i.xlarge for HCP when available", func() {
		Expect(interactiveComputeMachineTypeDefault(true, "m5.xlarge", []string{"m5.xlarge", mpOpts.DefaultInstanceType})).
			To(Equal(mpOpts.DefaultInstanceType))
	})

	It("falls back to the flavour default when m7i is unavailable", func() {
		Expect(interactiveComputeMachineTypeDefault(true, "m5.xlarge", []string{"m5.xlarge", "m5.2xlarge"})).
			To(Equal("m5.xlarge"))
	})

	It("uses the flavour default for classic clusters", func() {
		Expect(interactiveComputeMachineTypeDefault(false, "m5.xlarge", []string{mpOpts.DefaultInstanceType, "m5.xlarge"})).
			To(Equal("m5.xlarge"))
	})
})
