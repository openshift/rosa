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

// GcpFirewallRuleTemplateProfilesClient is the client of the 'gcp_firewall_rule_template_profiles' resource.
//
// Collection of profiles for a specific firewall rule template version.
// Path: /gcp/firewall_rule_templates/{version}/profiles
type GcpFirewallRuleTemplateProfilesClient struct {
	transport http.RoundTripper
	path      string
}

// NewGcpFirewallRuleTemplateProfilesClient creates a new client for the 'gcp_firewall_rule_template_profiles'
// resource using the given transport to send the requests and receive the
// responses.
func NewGcpFirewallRuleTemplateProfilesClient(transport http.RoundTripper, path string) *GcpFirewallRuleTemplateProfilesClient {
	return &GcpFirewallRuleTemplateProfilesClient{
		transport: transport,
		path:      path,
	}
}

// GcpFirewallRuleTemplate returns the target 'gcp_firewall_rule_template' resource for the given identifier.
//
// Reference to a specific profile.
func (c *GcpFirewallRuleTemplateProfilesClient) GcpFirewallRuleTemplate(id string) *GcpFirewallRuleTemplateClient {
	return NewGcpFirewallRuleTemplateClient(
		c.transport,
		path.Join(c.path, id),
	)
}
