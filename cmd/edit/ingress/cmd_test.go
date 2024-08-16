/*
Copyright (c) 2024 Red Hat, Inc.

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

package ingress

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse component routes", func() {
	DescribeTable(
		"Parses input string for component routes",
		func(input string) {
			componentRouteBuilder, err := parseComponentRoutes(input)
			Expect(err).To(BeNil())
			for key, builder := range componentRouteBuilder {
				expectedHostname := fmt.Sprintf("%s-host", key)
				expectedTlsRef := fmt.Sprintf("%s-secret", key)
				componentRoute, err := builder.Build()
				Expect(err).To(BeNil())
				Expect(componentRoute.Hostname()).To(Equal(expectedHostname))
				Expect(componentRoute.TlsSecretRef()).To(Equal(expectedTlsRef))
			}
		},
		//nolint:lll
		Entry(
			"base",
			"oauth: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret",
		),
		//nolint:lll
		Entry(
			"includes \"",
			"oauth: hostname=\"oauth-host\";tlsSecretRef=\"oauth-secret\",downloads: hostname=\"downloads-host\";tlsSecretRef=\"downloads-secret\",console: hostname=\"console-host\";tlsSecretRef=\"console-secret\"",
		),
	)
	Context("Fails to parse input string for component routes", func() {
		It("fails due to invalid component route", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"unknown: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("'unknown' is not a valid component name. Expected include [oauth, console, downloads]"))
		})
		It("fails due to wrong amount of component routes", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("the expected amount of component routes is 3, but 2 have been supplied"))
		})
		It("fails if it can split ':' in more than one key separation", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth:-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal(
				//nolint:lll
				"only the name of the component should be followed by ':' or the component should always include it's parameters separated by ':'",
			))
		})
		It("fails if it can't split the component name and it's parameters", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal(
				//nolint:lll
				"only the name of the component should be followed by ':' or the component should always include it's parameters separated by ':'",
			))
		})
		It("fails due to invalid parameter", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: unknown=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("'unknown' is not a valid parameter for a component route. Expected include [hostname, tlsSecretRef]"))
		})
		It("fails due to wrong amount of parameters", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth-host,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("only 2 parameters are expected for each component"))
		})
		It("fails if it can't split the attribute name and it's value", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal(
				//nolint:lll
				"only the name of the parameter should be followed by '=' or the paremater should always include a value separated by '='",
			))
		})
		It("fails if it can split the attribute name and it's value into more than 2 parts", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret=asd",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal(
				//nolint:lll
				"only the name of the parameter should be followed by '=' or the paremater should always include a value separated by '='",
			))
		})
	})
})
