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
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// GcpFirewallRuleClient is the client of the 'gcp_firewall_rule' resource.
//
// Manages a specific GCP firewall rule.
type GcpFirewallRuleClient struct {
	transport http.RoundTripper
	path      string
}

// NewGcpFirewallRuleClient creates a new client for the 'gcp_firewall_rule'
// resource using the given transport to send the requests and receive the
// responses.
func NewGcpFirewallRuleClient(transport http.RoundTripper, path string) *GcpFirewallRuleClient {
	return &GcpFirewallRuleClient{
		transport: transport,
		path:      path,
	}
}

// Delete creates a request for the 'delete' method.
//
// Deletes the firewall rule.
func (c *GcpFirewallRuleClient) Delete() *GcpFirewallRuleDeleteRequest {
	return &GcpFirewallRuleDeleteRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// Get creates a request for the 'get' method.
//
// Retrieves the details of the firewall rule.
func (c *GcpFirewallRuleClient) Get() *GcpFirewallRuleGetRequest {
	return &GcpFirewallRuleGetRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// GcpFirewallRulePollRequest is the request for the Poll method.
type GcpFirewallRulePollRequest struct {
	request    *GcpFirewallRuleGetRequest
	interval   time.Duration
	statuses   []int
	predicates []func(interface{}) bool
}

// Parameter adds a query parameter to all the requests that will be used to retrieve the object.
func (r *GcpFirewallRulePollRequest) Parameter(name string, value interface{}) *GcpFirewallRulePollRequest {
	r.request.Parameter(name, value)
	return r
}

// Header adds a request header to all the requests that will be used to retrieve the object.
func (r *GcpFirewallRulePollRequest) Header(name string, value interface{}) *GcpFirewallRulePollRequest {
	r.request.Header(name, value)
	return r
}

// Interval sets the polling interval. This parameter is mandatory and must be greater than zero.
func (r *GcpFirewallRulePollRequest) Interval(value time.Duration) *GcpFirewallRulePollRequest {
	r.interval = value
	return r
}

// Status set the expected status of the response. Multiple values can be set calling this method
// multiple times. The response will be considered successful if the status is any of those values.
func (r *GcpFirewallRulePollRequest) Status(value int) *GcpFirewallRulePollRequest {
	r.statuses = append(r.statuses, value)
	return r
}

// Predicate adds a predicate that the response should satisfy be considered successful. Multiple
// predicates can be set calling this method multiple times. The response will be considered successful
// if all the predicates are satisfied.
func (r *GcpFirewallRulePollRequest) Predicate(value func(*GcpFirewallRuleGetResponse) bool) *GcpFirewallRulePollRequest {
	r.predicates = append(r.predicates, func(response interface{}) bool {
		return value(response.(*GcpFirewallRuleGetResponse))
	})
	return r
}

// StartContext starts the polling loop. Responses will be considered successful if the status is one of
// the values specified with the Status method and if all the predicates specified with the Predicate
// method return nil.
//
// The context must have a timeout or deadline, otherwise this method will immediately return an error.
func (r *GcpFirewallRulePollRequest) StartContext(ctx context.Context) (response *GcpFirewallRulePollResponse, err error) {
	result, err := helpers.PollContext(ctx, r.interval, r.statuses, r.predicates, r.task)
	if result != nil {
		response = &GcpFirewallRulePollResponse{
			response: result.(*GcpFirewallRuleGetResponse),
		}
	}
	return
}

// task adapts the types of the request/response types so that they can be used with the generic
// polling function from the helpers package.
func (r *GcpFirewallRulePollRequest) task(ctx context.Context) (status int, result interface{}, err error) {
	response, err := r.request.SendContext(ctx)
	if response != nil {
		status = response.Status()
		result = response
	}
	return
}

// GcpFirewallRulePollResponse is the response for the Poll method.
type GcpFirewallRulePollResponse struct {
	response *GcpFirewallRuleGetResponse
}

// Status returns the response status code.
func (r *GcpFirewallRulePollResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.response.Status()
}

// Header returns header of the response.
func (r *GcpFirewallRulePollResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.response.Header()
}

// Error returns the response error.
func (r *GcpFirewallRulePollResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.response.Error()
}

// Body returns the value of the 'body' parameter.
func (r *GcpFirewallRulePollResponse) Body() *GcpFirewallRule {
	return r.response.Body()
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
func (r *GcpFirewallRulePollResponse) GetBody() (value *GcpFirewallRule, ok bool) {
	return r.response.GetBody()
}

// Poll creates a request to repeatedly retrieve the object till the response has one of a given set
// of states and satisfies a set of predicates.
func (c *GcpFirewallRuleClient) Poll() *GcpFirewallRulePollRequest {
	return &GcpFirewallRulePollRequest{
		request: c.Get(),
	}
}

// GcpFirewallRuleDeleteRequest is the request for the 'delete' method.
type GcpFirewallRuleDeleteRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
}

// Parameter adds a query parameter.
func (r *GcpFirewallRuleDeleteRequest) Parameter(name string, value interface{}) *GcpFirewallRuleDeleteRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *GcpFirewallRuleDeleteRequest) Header(name string, value interface{}) *GcpFirewallRuleDeleteRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Impersonate wraps requests on behalf of another user.
// Note: Services that do not support this feature may silently ignore this call.
func (r *GcpFirewallRuleDeleteRequest) Impersonate(user string) *GcpFirewallRuleDeleteRequest {
	helpers.AddImpersonationHeader(&r.header, user)
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *GcpFirewallRuleDeleteRequest) Send() (result *GcpFirewallRuleDeleteResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *GcpFirewallRuleDeleteRequest) SendContext(ctx context.Context) (result *GcpFirewallRuleDeleteResponse, err error) {
	query := helpers.CopyQuery(r.query)
	header := helpers.CopyHeader(r.header)
	uri := &url.URL{
		Path:     r.path,
		RawQuery: query.Encode(),
	}
	request := &http.Request{
		Method: "DELETE",
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
	result = &GcpFirewallRuleDeleteResponse{}
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
	return
}

// GcpFirewallRuleDeleteResponse is the response for the 'delete' method.
type GcpFirewallRuleDeleteResponse struct {
	status int
	header http.Header
	err    *errors.Error
}

// Status returns the response status code.
func (r *GcpFirewallRuleDeleteResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *GcpFirewallRuleDeleteResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *GcpFirewallRuleDeleteResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// GcpFirewallRuleGetRequest is the request for the 'get' method.
type GcpFirewallRuleGetRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
}

// Parameter adds a query parameter.
func (r *GcpFirewallRuleGetRequest) Parameter(name string, value interface{}) *GcpFirewallRuleGetRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *GcpFirewallRuleGetRequest) Header(name string, value interface{}) *GcpFirewallRuleGetRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Impersonate wraps requests on behalf of another user.
// Note: Services that do not support this feature may silently ignore this call.
func (r *GcpFirewallRuleGetRequest) Impersonate(user string) *GcpFirewallRuleGetRequest {
	helpers.AddImpersonationHeader(&r.header, user)
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *GcpFirewallRuleGetRequest) Send() (result *GcpFirewallRuleGetResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *GcpFirewallRuleGetRequest) SendContext(ctx context.Context) (result *GcpFirewallRuleGetResponse, err error) {
	query := helpers.CopyQuery(r.query)
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
	result = &GcpFirewallRuleGetResponse{}
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
	err = readGcpFirewallRuleGetResponse(result, reader)
	if err != nil {
		return
	}
	return
}

// GcpFirewallRuleGetResponse is the response for the 'get' method.
type GcpFirewallRuleGetResponse struct {
	status int
	header http.Header
	err    *errors.Error
	body   *GcpFirewallRule
}

// Status returns the response status code.
func (r *GcpFirewallRuleGetResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *GcpFirewallRuleGetResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *GcpFirewallRuleGetResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// Body returns the value of the 'body' parameter.
func (r *GcpFirewallRuleGetResponse) Body() *GcpFirewallRule {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
func (r *GcpFirewallRuleGetResponse) GetBody() (value *GcpFirewallRule, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}
