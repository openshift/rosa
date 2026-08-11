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
	api_v1 "github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1"
)

// GcpFirewallRuleTemplate represents the values of the 'gcp_firewall_rule_template' type.
//
// GCP firewall rule template definition.
// Templates define the set of firewall rules that will be created for a cluster.
// Each template has a version and one or more profiles (e.g. "public", "private").
type GcpFirewallRuleTemplate = api_v1.GcpFirewallRuleTemplate

// GcpFirewallRuleTemplateListKind is the name of the type used to represent list of objects of
// type 'gcp_firewall_rule_template'.
const GcpFirewallRuleTemplateListKind = api_v1.GcpFirewallRuleTemplateListKind

// GcpFirewallRuleTemplateListLinkKind is the name of the type used to represent links to list
// of objects of type 'gcp_firewall_rule_template'.
const GcpFirewallRuleTemplateListLinkKind = api_v1.GcpFirewallRuleTemplateListLinkKind

// GcpFirewallRuleTemplateNilKind is the name of the type used to nil lists of objects of
// type 'gcp_firewall_rule_template'.
const GcpFirewallRuleTemplateListNilKind = api_v1.GcpFirewallRuleTemplateListNilKind

type GcpFirewallRuleTemplateList = api_v1.GcpFirewallRuleTemplateList
