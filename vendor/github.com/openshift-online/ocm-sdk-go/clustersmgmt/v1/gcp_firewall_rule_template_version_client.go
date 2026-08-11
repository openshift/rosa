/*
Copyright (c) 2020 Red Hat, Inc.

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

// IMPORTANT: This file has been generated automatically, refrain from modifying it manually as all
// your changes will be lost when the file is generated again.

package v1 // github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1

import (
	"net/http"
	"path"
)

// GcpFirewallRuleTemplateVersionClient is the client of the 'gcp_firewall_rule_template_version' resource.
//
// Intermediate resource for a specific firewall rule template version.
// Path: /gcp/firewall_rule_templates/{version}
type GcpFirewallRuleTemplateVersionClient struct {
	transport http.RoundTripper
	path      string
}

// NewGcpFirewallRuleTemplateVersionClient creates a new client for the 'gcp_firewall_rule_template_version'
// resource using the given transport to send the requests and receive the
// responses.
func NewGcpFirewallRuleTemplateVersionClient(transport http.RoundTripper, path string) *GcpFirewallRuleTemplateVersionClient {
	return &GcpFirewallRuleTemplateVersionClient{
		transport: transport,
		path:      path,
	}
}

// Profiles returns the target 'gcp_firewall_rule_template_profiles' resource.
//
// Navigate to the profiles for this template version.
func (c *GcpFirewallRuleTemplateVersionClient) Profiles() *GcpFirewallRuleTemplateProfilesClient {
	return NewGcpFirewallRuleTemplateProfilesClient(
		c.transport,
		path.Join(c.path, "profiles"),
	)
}
