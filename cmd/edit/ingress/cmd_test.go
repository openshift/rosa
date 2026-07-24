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
	"bytes"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
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
	DescribeTable(
		"When parsing HCP component routes it should only allow console and downloads",
		func(input string) {
			componentRouteBuilder, err := parseComponentRoutesForAllowed(input, expectedHcpComponentRoutes)
			Expect(err).To(BeNil())
			Expect(componentRouteBuilder).To(HaveLen(2))
			Expect(componentRouteBuilder).To(HaveKey("console"))
			Expect(componentRouteBuilder).To(HaveKey("downloads"))
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
			"console: hostname=console-host;tlsSecretRef=console-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret",
		),
	)
	Context("When parsing HCP component routes it should reject invalid input", func() {
		It("When oauth is provided it should be rejected", func() {
			_, err := parseComponentRoutesForAllowed(
				//nolint:lll
				"oauth: hostname=oauth-host;tlsSecretRef=oauth-secret,console: hostname=console-host;tlsSecretRef=console-secret",
				expectedHcpComponentRoutes,
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("'oauth' is not a valid component name. Expected include [console, downloads]"))
		})
		It("When a duplicate component route is provided it should be rejected", func() {
			_, err := parseComponentRoutesForAllowed(
				//nolint:lll
				"console: hostname=console-host;tlsSecretRef=console-secret,console: hostname=console-host2;tlsSecretRef=console-secret2",
				expectedHcpComponentRoutes,
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("component route \"console\" was supplied more than once"))
		})
		It("When only one route is provided it should succeed", func() {
			result, err := parseComponentRoutesForAllowed(
				"console: hostname=console-host;tlsSecretRef=console-secret",
				expectedHcpComponentRoutes,
			)
			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(1))
			Expect(result).To(HaveKey("console"))
		})
		It("When a partial update is provided it should only include the specified route in the serialized payload", func() {
			result, err := parseComponentRoutesForAllowed(
				"console: hostname=console-host;tlsSecretRef=console-secret",
				expectedHcpComponentRoutes,
			)
			Expect(err).To(BeNil())
			ingress, err := cmv1.NewIngress().ComponentRoutes(result).Build()
			Expect(err).To(BeNil())

			var buf bytes.Buffer
			err = cmv1.MarshalIngress(ingress, &buf)
			Expect(err).To(BeNil())
			var body map[string]interface{}
			err = json.Unmarshal(buf.Bytes(), &body)
			Expect(err).To(BeNil())

			cr := body["component_routes"].(map[string]interface{})
			Expect(cr).To(HaveLen(1))
			Expect(cr).To(HaveKey("console"))
			Expect(cr).ToNot(HaveKey("downloads"))
			console := cr["console"].(map[string]interface{})
			Expect(console["hostname"]).To(Equal("console-host"))
			Expect(console["tls_secret_ref"]).To(Equal("console-secret"))
		})
		It("When empty values are provided it should serialize as a clear", func() {
			result, err := parseComponentRoutesForAllowed(
				"downloads: hostname=;tlsSecretRef=",
				expectedHcpComponentRoutes,
			)
			Expect(err).To(BeNil())
			ingress, err := cmv1.NewIngress().ComponentRoutes(result).Build()
			Expect(err).To(BeNil())

			var buf bytes.Buffer
			err = cmv1.MarshalIngress(ingress, &buf)
			Expect(err).To(BeNil())
			var body map[string]interface{}
			err = json.Unmarshal(buf.Bytes(), &body)
			Expect(err).To(BeNil())

			cr := body["component_routes"].(map[string]interface{})
			Expect(cr).To(HaveLen(1))
			Expect(cr).To(HaveKey("downloads"))
			downloads := cr["downloads"].(map[string]interface{})
			Expect(downloads["hostname"]).To(Equal(""))
			Expect(downloads["tls_secret_ref"]).To(Equal(""))
		})
	})
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
		It("When partial classic routes are provided it should succeed", func() {
			result, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret",
			)
			Expect(err).To(BeNil())
			Expect(result).To(HaveLen(2))
			Expect(result).To(HaveKey("oauth"))
			Expect(result).To(HaveKey("downloads"))
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
