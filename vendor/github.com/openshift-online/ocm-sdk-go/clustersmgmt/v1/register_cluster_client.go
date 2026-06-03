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

	"github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// RegisterClusterClient is the client of the 'register_cluster' resource.
//
// Registers clusters that were provisioned outside this service.
type RegisterClusterClient struct {
	transport http.RoundTripper
	path      string
}

// NewRegisterClusterClient creates a new client for the 'register_cluster'
// resource using the given transport to send the requests and receive the
// responses.
func NewRegisterClusterClient(transport http.RoundTripper, path string) *RegisterClusterClient {
	return &RegisterClusterClient{
		transport: transport,
		path:      path,
	}
}

// Post creates a request for the 'post' method.
//
// Adds an existing cluster to the collection.
func (c *RegisterClusterClient) Post() *RegisterClusterPostRequest {
	return &RegisterClusterPostRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// RegisterClusterPostRequest is the request for the 'post' method.
type RegisterClusterPostRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
	body      *ClusterRegistration
}

// Parameter adds a query parameter.
func (r *RegisterClusterPostRequest) Parameter(name string, value interface{}) *RegisterClusterPostRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *RegisterClusterPostRequest) Header(name string, value interface{}) *RegisterClusterPostRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Impersonate wraps requests on behalf of another user.
// Note: Services that do not support this feature may silently ignore this call.
func (r *RegisterClusterPostRequest) Impersonate(user string) *RegisterClusterPostRequest {
	helpers.AddImpersonationHeader(&r.header, user)
	return r
}

// Body sets the value of the 'body' parameter.
//
// Attributes of the cluster registration request.
func (r *RegisterClusterPostRequest) Body(value *ClusterRegistration) *RegisterClusterPostRequest {
	r.body = value
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *RegisterClusterPostRequest) Send() (result *RegisterClusterPostResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *RegisterClusterPostRequest) SendContext(ctx context.Context) (result *RegisterClusterPostResponse, err error) {
	query := helpers.CopyQuery(r.query)
	header := helpers.CopyHeader(r.header)
	buffer := &bytes.Buffer{}
	err = writeRegisterClusterPostRequest(r, buffer)
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
	result = &RegisterClusterPostResponse{}
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
	err = readRegisterClusterPostResponse(result, reader)
	if err != nil {
		return
	}
	return
}

// RegisterClusterPostResponse is the response for the 'post' method.
type RegisterClusterPostResponse struct {
	status int
	header http.Header
	err    *errors.Error
	body   *Cluster
}

// Status returns the response status code.
func (r *RegisterClusterPostResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *RegisterClusterPostResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *RegisterClusterPostResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// Body returns the value of the 'body' parameter.
//
// Created cluster record.
func (r *RegisterClusterPostResponse) Body() *Cluster {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
// Created cluster record.
func (r *RegisterClusterPostResponse) GetBody() (value *Cluster, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}
