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
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// GcpFirewallRulesClient is the client of the 'gcp_firewall_rules' resource.
//
// Manages the collection of GCP firewall rules.
type GcpFirewallRulesClient struct {
	transport http.RoundTripper
	path      string
}

// NewGcpFirewallRulesClient creates a new client for the 'gcp_firewall_rules'
// resource using the given transport to send the requests and receive the
// responses.
func NewGcpFirewallRulesClient(transport http.RoundTripper, path string) *GcpFirewallRulesClient {
	return &GcpFirewallRulesClient{
		transport: transport,
		path:      path,
	}
}

// Add creates a request for the 'add' method.
//
// Creates a new GCP firewall rule.
func (c *GcpFirewallRulesClient) Add() *GcpFirewallRulesAddRequest {
	return &GcpFirewallRulesAddRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// List creates a request for the 'list' method.
//
// Lists the GCP firewall rules.
func (c *GcpFirewallRulesClient) List() *GcpFirewallRulesListRequest {
	return &GcpFirewallRulesListRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// GcpFirewallRule returns the target 'gcp_firewall_rule' resource for the given identifier.
//
// Reference to the resource that manages a specific GCP firewall rule.
func (c *GcpFirewallRulesClient) GcpFirewallRule(id string) *GcpFirewallRuleClient {
	return NewGcpFirewallRuleClient(
		c.transport,
		path.Join(c.path, id),
	)
}

// GcpFirewallRulesAddRequest is the request for the 'add' method.
type GcpFirewallRulesAddRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
	body      *GcpFirewallRule
}

// Parameter adds a query parameter.
func (r *GcpFirewallRulesAddRequest) Parameter(name string, value interface{}) *GcpFirewallRulesAddRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *GcpFirewallRulesAddRequest) Header(name string, value interface{}) *GcpFirewallRulesAddRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Impersonate wraps requests on behalf of another user.
// Note: Services that do not support this feature may silently ignore this call.
func (r *GcpFirewallRulesAddRequest) Impersonate(user string) *GcpFirewallRulesAddRequest {
	helpers.AddImpersonationHeader(&r.header, user)
	return r
}

// Body sets the value of the 'body' parameter.
//
// Description of the GCP firewall rule.
func (r *GcpFirewallRulesAddRequest) Body(value *GcpFirewallRule) *GcpFirewallRulesAddRequest {
	r.body = value
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *GcpFirewallRulesAddRequest) Send() (result *GcpFirewallRulesAddResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *GcpFirewallRulesAddRequest) SendContext(ctx context.Context) (result *GcpFirewallRulesAddResponse, err error) {
	query := helpers.CopyQuery(r.query)
	header := helpers.CopyHeader(r.header)
	buffer := &bytes.Buffer{}
	err = writeGcpFirewallRulesAddRequest(r, buffer)
	if err != nil {
		return
	}
	uri := &url.URL{
		Path:     r.path,
		RawQuery: query.Encode(),
	}
	request := &http.Request{
		Method: "POST",
		URL:    uri,
		Header: header,
		Body:   io.NopCloser(buffer),
	}
	if ctx != nil {
		request = request.WithContext(ctx)
	}
	response, err := r.transport.RoundTrip(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	result = &GcpFirewallRulesAddResponse{}
	result.status = response.StatusCode
	result.header = response.Header
	reader := bufio.NewReader(response.Body)
	_, err = reader.Peek(1)
	if err == io.EOF {
		err = nil
		return
	}
	if result.status >= 400 {
		result.err, err = errors.UnmarshalErrorStatus(reader, result.status)
		if err != nil {
			return
		}
		err = result.err
		return
	}
	err = readGcpFirewallRulesAddResponse(result, reader)
	if err != nil {
		return
	}
	return
}

// GcpFirewallRulesAddResponse is the response for the 'add' method.
type GcpFirewallRulesAddResponse struct {
	status int
	header http.Header
	err    *errors.Error
	body   *GcpFirewallRule
}

// Status returns the response status code.
func (r *GcpFirewallRulesAddResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *GcpFirewallRulesAddResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *GcpFirewallRulesAddResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// Body returns the value of the 'body' parameter.
//
// Description of the GCP firewall rule.
func (r *GcpFirewallRulesAddResponse) Body() *GcpFirewallRule {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
// Description of the GCP firewall rule.
func (r *GcpFirewallRulesAddResponse) GetBody() (value *GcpFirewallRule, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// GcpFirewallRulesListRequest is the request for the 'list' method.
type GcpFirewallRulesListRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
	page      *int
	size      *int
}

// Parameter adds a query parameter.
func (r *GcpFirewallRulesListRequest) Parameter(name string, value interface{}) *GcpFirewallRulesListRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *GcpFirewallRulesListRequest) Header(name string, value interface{}) *GcpFirewallRulesListRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Impersonate wraps requests on behalf of another user.
// Note: Services that do not support this feature may silently ignore this call.
func (r *GcpFirewallRulesListRequest) Impersonate(user string) *GcpFirewallRulesListRequest {
	helpers.AddImpersonationHeader(&r.header, user)
	return r
}

// Page sets the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *GcpFirewallRulesListRequest) Page(value int) *GcpFirewallRulesListRequest {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
// Maximum number of items that will be contained in the returned page.
func (r *GcpFirewallRulesListRequest) Size(value int) *GcpFirewallRulesListRequest {
	r.size = &value
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *GcpFirewallRulesListRequest) Send() (result *GcpFirewallRulesListResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *GcpFirewallRulesListRequest) SendContext(ctx context.Context) (result *GcpFirewallRulesListResponse, err error) {
	query := helpers.CopyQuery(r.query)
	if r.page != nil {
		helpers.AddValue(&query, "page", *r.page)
	}
	if r.size != nil {
		helpers.AddValue(&query, "size", *r.size)
	}
	header := helpers.CopyHeader(r.header)
	uri := &url.URL{
		Path:     r.path,
		RawQuery: query.Encode(),
	}
	request := &http.Request{
		Method: "GET",
		URL:    uri,
		Header: header,
	}
	if ctx != nil {
		request = request.WithContext(ctx)
	}
	response, err := r.transport.RoundTrip(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	result = &GcpFirewallRulesListResponse{}
	result.status = response.StatusCode
	result.header = response.Header
	reader := bufio.NewReader(response.Body)
	_, err = reader.Peek(1)
	if err == io.EOF {
		err = nil
		return
	}
	if result.status >= 400 {
		result.err, err = errors.UnmarshalErrorStatus(reader, result.status)
		if err != nil {
			return
		}
		err = result.err
		return
	}
	err = readGcpFirewallRulesListResponse(result, reader)
	if err != nil {
		return
	}
	return
}

// GcpFirewallRulesListResponse is the response for the 'list' method.
type GcpFirewallRulesListResponse struct {
	status int
	header http.Header
	err    *errors.Error
	items  *GcpFirewallRuleList
	page   *int
	size   *int
	total  *int
}

// Status returns the response status code.
func (r *GcpFirewallRulesListResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *GcpFirewallRulesListResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *GcpFirewallRulesListResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// Items returns the value of the 'items' parameter.
//
// Retrieved list of GCP firewall rules.
func (r *GcpFirewallRulesListResponse) Items() *GcpFirewallRuleList {
	if r == nil {
		return nil
	}
	return r.items
}

// GetItems returns the value of the 'items' parameter and
// a flag indicating if the parameter has a value.
//
// Retrieved list of GCP firewall rules.
func (r *GcpFirewallRulesListResponse) GetItems() (value *GcpFirewallRuleList, ok bool) {
	ok = r != nil && r.items != nil
	if ok {
		value = r.items
	}
	return
}

// Page returns the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *GcpFirewallRulesListResponse) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
// Index of the requested page, where one corresponds to the first page.
func (r *GcpFirewallRulesListResponse) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
// Maximum number of items that will be contained in the returned page.
func (r *GcpFirewallRulesListResponse) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
// Maximum number of items that will be contained in the returned page.
func (r *GcpFirewallRulesListResponse) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// Total returns the value of the 'total' parameter.
//
// Total number of items of the collection.
func (r *GcpFirewallRulesListResponse) Total() int {
	if r != nil && r.total != nil {
		return *r.total
	}
	return 0
}

// GetTotal returns the value of the 'total' parameter and
// a flag indicating if the parameter has a value.
//
// Total number of items of the collection.
func (r *GcpFirewallRulesListResponse) GetTotal() (value int, ok bool) {
	ok = r != nil && r.total != nil
	if ok {
		value = *r.total
	}
	return
}
