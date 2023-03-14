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

// HostedOidcConfigsClient is the client of the 'hosted_oidc_configs' resource.
//
// Manages the collection hosted oidc configurations.
type HostedOidcConfigsClient struct {
	transport http.RoundTripper
	path      string
}

// NewHostedOidcConfigsClient creates a new client for the 'hosted_oidc_configs'
// resource using the given transport to send the requests and receive the
// responses.
func NewHostedOidcConfigsClient(transport http.RoundTripper, path string) *HostedOidcConfigsClient {
	return &HostedOidcConfigsClient{
		transport: transport,
		path:      path,
	}
}

// Add creates a request for the 'add' method.
//
// Creates a hosting under Red Hat's S3 bucket for byo oidc configuration
func (c *HostedOidcConfigsClient) Add() *HostedOidcConfigsAddRequest {
	return &HostedOidcConfigsAddRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// List creates a request for the 'list' method.
//
// Retrieves the list of hosted oidc configs.
func (c *HostedOidcConfigsClient) List() *HostedOidcConfigsListRequest {
	return &HostedOidcConfigsListRequest{
		transport: c.transport,
		path:      c.path,
	}
}

// HostedOidcConfig returns the target 'hosted_oidc_config' resource for the given identifier.
//
// Reference to the service that manages an specific identity provider.
func (c *HostedOidcConfigsClient) HostedOidcConfig(id string) *HostedOidcConfigClient {
	return NewHostedOidcConfigClient(
		c.transport,
		path.Join(c.path, id),
	)
}

// HostedOidcConfigsAddRequest is the request for the 'add' method.
type HostedOidcConfigsAddRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
	body      *HostedOidcConfig
}

// Parameter adds a query parameter.
func (r *HostedOidcConfigsAddRequest) Parameter(name string, value interface{}) *HostedOidcConfigsAddRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *HostedOidcConfigsAddRequest) Header(name string, value interface{}) *HostedOidcConfigsAddRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Impersonate wraps requests on behalf of another user.
// Note: Services that do not support this feature may silently ignore this call.
func (r *HostedOidcConfigsAddRequest) Impersonate(user string) *HostedOidcConfigsAddRequest {
	helpers.AddImpersonationHeader(&r.header, user)
	return r
}

// Body sets the value of the 'body' parameter.
func (r *HostedOidcConfigsAddRequest) Body(value *HostedOidcConfig) *HostedOidcConfigsAddRequest {
	r.body = value
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *HostedOidcConfigsAddRequest) Send() (result *HostedOidcConfigsAddResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *HostedOidcConfigsAddRequest) SendContext(ctx context.Context) (result *HostedOidcConfigsAddResponse, err error) {
	query := helpers.CopyQuery(r.query)
	header := helpers.CopyHeader(r.header)
	buffer := &bytes.Buffer{}
	err = writeHostedOidcConfigsAddRequest(r, buffer)
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
	result = &HostedOidcConfigsAddResponse{}
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
	err = readHostedOidcConfigsAddResponse(result, reader)
	if err != nil {
		return
	}
	return
}

// HostedOidcConfigsAddResponse is the response for the 'add' method.
type HostedOidcConfigsAddResponse struct {
	status int
	header http.Header
	err    *errors.Error
	body   *HostedOidcConfig
}

// Status returns the response status code.
func (r *HostedOidcConfigsAddResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *HostedOidcConfigsAddResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *HostedOidcConfigsAddResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// Body returns the value of the 'body' parameter.
func (r *HostedOidcConfigsAddResponse) Body() *HostedOidcConfig {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
func (r *HostedOidcConfigsAddResponse) GetBody() (value *HostedOidcConfig, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// HostedOidcConfigsListRequest is the request for the 'list' method.
type HostedOidcConfigsListRequest struct {
	transport http.RoundTripper
	path      string
	query     url.Values
	header    http.Header
	page      *int
	size      *int
}

// Parameter adds a query parameter.
func (r *HostedOidcConfigsListRequest) Parameter(name string, value interface{}) *HostedOidcConfigsListRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *HostedOidcConfigsListRequest) Header(name string, value interface{}) *HostedOidcConfigsListRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Impersonate wraps requests on behalf of another user.
// Note: Services that do not support this feature may silently ignore this call.
func (r *HostedOidcConfigsListRequest) Impersonate(user string) *HostedOidcConfigsListRequest {
	helpers.AddImpersonationHeader(&r.header, user)
	return r
}

// Page sets the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *HostedOidcConfigsListRequest) Page(value int) *HostedOidcConfigsListRequest {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
// Number of items contained in the returned page.
func (r *HostedOidcConfigsListRequest) Size(value int) *HostedOidcConfigsListRequest {
	r.size = &value
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *HostedOidcConfigsListRequest) Send() (result *HostedOidcConfigsListResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *HostedOidcConfigsListRequest) SendContext(ctx context.Context) (result *HostedOidcConfigsListResponse, err error) {
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
	result = &HostedOidcConfigsListResponse{}
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
	err = readHostedOidcConfigsListResponse(result, reader)
	if err != nil {
		return
	}
	return
}

// HostedOidcConfigsListResponse is the response for the 'list' method.
type HostedOidcConfigsListResponse struct {
	status int
	header http.Header
	err    *errors.Error
	items  *HostedOidcConfigList
	page   *int
	size   *int
	total  *int
}

// Status returns the response status code.
func (r *HostedOidcConfigsListResponse) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}

// Header returns header of the response.
func (r *HostedOidcConfigsListResponse) Header() http.Header {
	if r == nil {
		return nil
	}
	return r.header
}

// Error returns the response error.
func (r *HostedOidcConfigsListResponse) Error() *errors.Error {
	if r == nil {
		return nil
	}
	return r.err
}

// Items returns the value of the 'items' parameter.
//
// Retrieved list of identity providers.
func (r *HostedOidcConfigsListResponse) Items() *HostedOidcConfigList {
	if r == nil {
		return nil
	}
	return r.items
}

// GetItems returns the value of the 'items' parameter and
// a flag indicating if the parameter has a value.
//
// Retrieved list of identity providers.
func (r *HostedOidcConfigsListResponse) GetItems() (value *HostedOidcConfigList, ok bool) {
	ok = r != nil && r.items != nil
	if ok {
		value = r.items
	}
	return
}

// Page returns the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *HostedOidcConfigsListResponse) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
// Index of the requested page, where one corresponds to the first page.
func (r *HostedOidcConfigsListResponse) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
// Number of items contained in the returned page.
func (r *HostedOidcConfigsListResponse) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
// Number of items contained in the returned page.
func (r *HostedOidcConfigsListResponse) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// Total returns the value of the 'total' parameter.
//
// Total number of items of the collection.
func (r *HostedOidcConfigsListResponse) Total() int {
	if r != nil && r.total != nil {
		return *r.total
	}
	return 0
}

// GetTotal returns the value of the 'total' parameter and
// a flag indicating if the parameter has a value.
//
// Total number of items of the collection.
func (r *HostedOidcConfigsListResponse) GetTotal() (value int, ok bool) {
	ok = r != nil && r.total != nil
	if ok {
		value = *r.total
	}
	return
}
