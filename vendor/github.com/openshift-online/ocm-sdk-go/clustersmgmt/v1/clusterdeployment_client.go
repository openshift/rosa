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
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// ClusterdeploymentClient is the client of the 'clusterdeployment' resource.
//
// Manages a specific clusterdeployment.
type ClusterdeploymentClient struct {
	transport http.RoundTripper
	path      string
}

// NewClusterdeploymentClient creates a new client for the 'clusterdeployment'
// resource using the given transport to send the requests and receive the
// responses.
func NewClusterdeploymentClient(transport http.RoundTripper, path string) *ClusterdeploymentClient {
	return &ClusterdeploymentClient{
		transport: transport,
		path:      path,
	}
}

// Delete creates a request for the 'delete' method.
//
// Deletes the clusterdeployment.
func (c *ClusterdeploymentClient) Delete() *ClusterdeploymentDeleteRequest {
	return &ClusterdeploymentDeleteRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// Get creates a request for the 'get' method.
//
// Retrieves the details of the clusterdeployment.
func (c *ClusterdeploymentClient) Get() *ClusterdeploymentGetRequest {
	return &ClusterdeploymentGetRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// ClusterdeploymentPollRequest is the request for the Poll method.
type ClusterdeploymentPollRequest struct {
	request    *ClusterdeploymentGetRequest
	interval   time.Duration
	statuses   []int
	predicates []func(interface{}) bool
}

// Parameter adds a query parameter to all the requests that will be used to retrieve the object.
func (r *ClusterdeploymentPollRequest) Parameter(name string, value interface{}) *ClusterdeploymentPollRequest {
	r.request.Parameter(name, value)
	return r
}

// Header adds a request header to all the requests that will be used to retrieve the object.
func (r *ClusterdeploymentPollRequest) Header(name string, value interface{}) *ClusterdeploymentPollRequest {
	r.request.Header(name, value)
	return r
}

// Interval sets the polling interval. This parameter is mandatory and must be greater than zero.
func (r *ClusterdeploymentPollRequest) Interval(value time.Duration) *ClusterdeploymentPollRequest {
	r.interval = value
	return r
}

// Status set the expected status of the response. Multiple values can be set calling this method
// multiple times. The response will be considered successful if the status is any of those values.
func (r *ClusterdeploymentPollRequest) Status(value int) *ClusterdeploymentPollRequest {
	r.statuses = append(r.statuses, value)
	return r
}

// Predicate adds a predicate that the response should satisfy be considered successful. Multiple
// predicates can be set calling this method multiple times. The response will be considered successful
// if all the predicates are satisfied.
func (r *ClusterdeploymentPollRequest) Predicate(value func(*ClusterdeploymentGetResponse) bool) *ClusterdeploymentPollRequest {
	r.predicates = append(r.predicates, func(response interface{}) bool {
		return value(response.(*ClusterdeploymentGetResponse))
	})
	return r
}

// StartContext starts the polling loop. Responses will be considered successful if the status is one of
// the values specified with the Status method and if all the predicates specified with the Predicate
// method return nil.
//
// The context must have a timeout or deadline, otherwise this method will immediately return an error.
func (r *ClusterdeploymentPollRequest) StartContext(ctx context.Context) (response *ClusterdeploymentPollResponse, err error) {
	result, err := helpers.PollContext(ctx, r.interval, r.statuses, r.predicates, r.task)
	if result != nil {
		response = &ClusterdeploymentPollResponse{
			response: result.(*ClusterdeploymentGetResponse),
		}
	}
	return
}

// task adapts the types of the request/response types so that they can be used with the generic
// polling function from the helpers package.
func (r *ClusterdeploymentPollRequest) task(ctx context.Context) (status int, result interface{}, err error) {
	response, err := r.request.SendContext(ctx)
	if response != nil {
		status = response.Status()
		result = response
	}
	return
}

// ClusterdeploymentPollResponse is the response for the Poll method.
type ClusterdeploymentPollResponse struct {
	response *ClusterdeploymentGetResponse
}

// Status returns the response status code.
func (r *ClusterdeploymentPollResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.response.Status()
}

// Header returns header of the response.
func (r *ClusterdeploymentPollResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.response.Header()
}

// Error returns the response error.
func (r *ClusterdeploymentPollResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.response.Error()
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ClusterdeploymentPollResponse) Body() *ClusterDeployment {
	return r.response.Body()
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ClusterdeploymentPollResponse) GetBody() (value *ClusterDeployment, ok bool) {
	return r.response.GetBody()
}

// Poll creates a request to repeatedly retrieve the object till the response has one of a given set
// of states and satisfies a set of predicates.
func (c *ClusterdeploymentClient) Poll() *ClusterdeploymentPollRequest {
	return &ClusterdeploymentPollRequest{
		request: c.Get(),
	}
}

// ClusterdeploymentDeleteRequest is the request for the 'delete' method.
type ClusterdeploymentDeleteRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
}

// Parameter adds a query parameter.
func (r *ClusterdeploymentDeleteRequest) Parameter(name string, value interface{}) *ClusterdeploymentDeleteRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *ClusterdeploymentDeleteRequest) Header(name string, value interface{}) *ClusterdeploymentDeleteRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *ClusterdeploymentDeleteRequest) Send() (result *ClusterdeploymentDeleteResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *ClusterdeploymentDeleteRequest) SendContext(ctx context.Context) (result *ClusterdeploymentDeleteResponse, err error) {
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
	result = &ClusterdeploymentDeleteResponse{}
	result.status = response.StatusCode
	result.header = response.Header
	if result.status >= 400 {
		result.err, err = errors.UnmarshalError(response.Body)
		if err != nil {
			return
		}
		err = result.err
		return
	}
	return
}

// ClusterdeploymentDeleteResponse is the response for the 'delete' method.
type ClusterdeploymentDeleteResponse struct {
	status int
	header http.Header
	err    *errors.Error
}

// Status returns the response status code.
func (r *ClusterdeploymentDeleteResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *ClusterdeploymentDeleteResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *ClusterdeploymentDeleteResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// ClusterdeploymentGetRequest is the request for the 'get' method.
type ClusterdeploymentGetRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
}

// Parameter adds a query parameter.
func (r *ClusterdeploymentGetRequest) Parameter(name string, value interface{}) *ClusterdeploymentGetRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *ClusterdeploymentGetRequest) Header(name string, value interface{}) *ClusterdeploymentGetRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *ClusterdeploymentGetRequest) Send() (result *ClusterdeploymentGetResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *ClusterdeploymentGetRequest) SendContext(ctx context.Context) (result *ClusterdeploymentGetResponse, err error) {
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
	result = &ClusterdeploymentGetResponse{}
	result.status = response.StatusCode
	result.header = response.Header
	if result.status >= 400 {
		result.err, err = errors.UnmarshalError(response.Body)
		if err != nil {
			return
		}
		err = result.err
		return
	}
	err = readClusterdeploymentGetResponse(result, response.Body)
	if err != nil {
		return
	}
	return
}

// ClusterdeploymentGetResponse is the response for the 'get' method.
type ClusterdeploymentGetResponse struct {
	status int
	header http.Header
	err    *errors.Error
	body   *ClusterDeployment
}

// Status returns the response status code.
func (r *ClusterdeploymentGetResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *ClusterdeploymentGetResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *ClusterdeploymentGetResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ClusterdeploymentGetResponse) Body() *ClusterDeployment {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ClusterdeploymentGetResponse) GetBody() (value *ClusterDeployment, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}
