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

// AwsOidcThumbprintClient is the client of the 'aws_oidc_thumbprint' resource.
//
// Thumbprint of the cluster's OpenID Connect identity provider
type AwsOidcThumbprintClient struct {
	transport http.RoundTripper
	path      string
}

// NewAwsOidcThumbprintClient creates a new client for the 'aws_oidc_thumbprint'
// resource using the given transport to send the requests and receive the
// responses.
func NewAwsOidcThumbprintClient(transport http.RoundTripper, path string) *AwsOidcThumbprintClient {
	return &AwsOidcThumbprintClient{
		transport: transport,
		path:      path,
	}
}

// Get creates a request for the 'get' method.
func (c *AwsOidcThumbprintClient) Get() *AwsOidcThumbprintGetRequest {
	return &AwsOidcThumbprintGetRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// AwsOidcThumbprintPollRequest is the request for the Poll method.
type AwsOidcThumbprintPollRequest struct {
	request    *AwsOidcThumbprintGetRequest
	interval   time.Duration
	statuses   []int
	predicates []func(interface{}) bool
}

// Parameter adds a query parameter to all the requests that will be used to retrieve the object.
func (r *AwsOidcThumbprintPollRequest) Parameter(name string, value interface{}) *AwsOidcThumbprintPollRequest {
	r.request.Parameter(name, value)
	return r
}

// Header adds a request header to all the requests that will be used to retrieve the object.
func (r *AwsOidcThumbprintPollRequest) Header(name string, value interface{}) *AwsOidcThumbprintPollRequest {
	r.request.Header(name, value)
	return r
}

// ClusterId sets the value of the 'cluster_id' parameter for all the requests that
// will be used to retrieve the object.
func (r *AwsOidcThumbprintPollRequest) ClusterId(value string) *AwsOidcThumbprintPollRequest {
	r.request.ClusterId(value)
	return r
}

// OidcConfigId sets the value of the 'oidc_config_id' parameter for all the requests that
// will be used to retrieve the object.
func (r *AwsOidcThumbprintPollRequest) OidcConfigId(value string) *AwsOidcThumbprintPollRequest {
	r.request.OidcConfigId(value)
	return r
}

// Interval sets the polling interval. This parameter is mandatory and must be greater than zero.
func (r *AwsOidcThumbprintPollRequest) Interval(value time.Duration) *AwsOidcThumbprintPollRequest {
	r.interval = value
	return r
}

// Status set the expected status of the response. Multiple values can be set calling this method
// multiple times. The response will be considered successful if the status is any of those values.
func (r *AwsOidcThumbprintPollRequest) Status(value int) *AwsOidcThumbprintPollRequest {
	r.statuses = append(r.statuses, value)
	return r
}

// Predicate adds a predicate that the response should satisfy be considered successful. Multiple
// predicates can be set calling this method multiple times. The response will be considered successful
// if all the predicates are satisfied.
func (r *AwsOidcThumbprintPollRequest) Predicate(value func(*AwsOidcThumbprintGetResponse) bool) *AwsOidcThumbprintPollRequest {
	r.predicates = append(r.predicates, func(response interface{}) bool {
		return value(response.(*AwsOidcThumbprintGetResponse))
	})
	return r
}

// StartContext starts the polling loop. Responses will be considered successful if the status is one of
// the values specified with the Status method and if all the predicates specified with the Predicate
// method return nil.
//
// The context must have a timeout or deadline, otherwise this method will immediately return an error.
func (r *AwsOidcThumbprintPollRequest) StartContext(ctx context.Context) (response *AwsOidcThumbprintPollResponse, err error) {
	result, err := helpers.PollContext(ctx, r.interval, r.statuses, r.predicates, r.task)
	if result != nil {
		response = &AwsOidcThumbprintPollResponse{
			response: result.(*AwsOidcThumbprintGetResponse),
		}
	}
	return
}

// task adapts the types of the request/response types so that they can be used with the generic
// polling function from the helpers package.
func (r *AwsOidcThumbprintPollRequest) task(ctx context.Context) (status int, result interface{}, err error) {
	response, err := r.request.SendContext(ctx)
	if response != nil {
		status = response.Status()
		result = response
	}
	return
}

// AwsOidcThumbprintPollResponse is the response for the Poll method.
type AwsOidcThumbprintPollResponse struct {
	response *AwsOidcThumbprintGetResponse
}

// Status returns the response status code.
func (r *AwsOidcThumbprintPollResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.response.Status()
}

// Header returns header of the response.
func (r *AwsOidcThumbprintPollResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.response.Header()
}

// Error returns the response error.
func (r *AwsOidcThumbprintPollResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.response.Error()
}

// Body returns the value of the 'body' parameter.
func (r *AwsOidcThumbprintPollResponse) Body() *AwsOidcThumbprint {
	return r.response.Body()
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
func (r *AwsOidcThumbprintPollResponse) GetBody() (value *AwsOidcThumbprint, ok bool) {
	return r.response.GetBody()
}

// Poll creates a request to repeatedly retrieve the object till the response has one of a given set
// of states and satisfies a set of predicates.
func (c *AwsOidcThumbprintClient) Poll() *AwsOidcThumbprintPollRequest {
	return &AwsOidcThumbprintPollRequest{
		request: c.Get(),
	}
}

// AwsOidcThumbprintGetRequest is the request for the 'get' method.
type AwsOidcThumbprintGetRequest struct {
	transport    http.RoundTripper
	path         string
	query        url.Values
	header       http.Header
	clusterId    *string
	oidcConfigId *string
}

// Parameter adds a query parameter.
func (r *AwsOidcThumbprintGetRequest) Parameter(name string, value interface{}) *AwsOidcThumbprintGetRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *AwsOidcThumbprintGetRequest) Header(name string, value interface{}) *AwsOidcThumbprintGetRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Impersonate wraps requests on behalf of another user.
// Note: Services that do not support this feature may silently ignore this call.
func (r *AwsOidcThumbprintGetRequest) Impersonate(user string) *AwsOidcThumbprintGetRequest {
	helpers.AddImpersonationHeader(&r.header, user)
	return r
}

// ClusterId sets the value of the 'cluster_id' parameter.
func (r *AwsOidcThumbprintGetRequest) ClusterId(value string) *AwsOidcThumbprintGetRequest {
	r.clusterId = &value
	return r
}

// OidcConfigId sets the value of the 'oidc_config_id' parameter.
func (r *AwsOidcThumbprintGetRequest) OidcConfigId(value string) *AwsOidcThumbprintGetRequest {
	r.oidcConfigId = &value
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *AwsOidcThumbprintGetRequest) Send() (result *AwsOidcThumbprintGetResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *AwsOidcThumbprintGetRequest) SendContext(ctx context.Context) (result *AwsOidcThumbprintGetResponse, err error) {
	query := helpers.CopyQuery(r.query)
	if r.clusterId != nil {
		helpers.AddValue(&query, "cluster_id", *r.clusterId)
	}
	if r.oidcConfigId != nil {
		helpers.AddValue(&query, "oidc_config_id", *r.oidcConfigId)
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
	result = &AwsOidcThumbprintGetResponse{}
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
	err = readAwsOidcThumbprintGetResponse(result, reader)
	if err != nil {
		return
	}
	return
}

// AwsOidcThumbprintGetResponse is the response for the 'get' method.
type AwsOidcThumbprintGetResponse struct {
	status int
	header http.Header
	err    *errors.Error
	body   *AwsOidcThumbprint
}

// Status returns the response status code.
func (r *AwsOidcThumbprintGetResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *AwsOidcThumbprintGetResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *AwsOidcThumbprintGetResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// Body returns the value of the 'body' parameter.
func (r *AwsOidcThumbprintGetResponse) Body() *AwsOidcThumbprint {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
func (r *AwsOidcThumbprintGetResponse) GetBody() (value *AwsOidcThumbprint, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}
