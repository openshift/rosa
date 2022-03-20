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

package v1 // github.com/openshift-online/ocm-sdk-go/statusboard/v1

import (
	"context"
	"net/http"
	time "time"

	"github.com/golang/glog"
	"github.com/openshift-online/ocm-sdk-go/errors"
)

// StatusUpdatesServer represents the interface the manages the 'status_updates' resource.
type StatusUpdatesServer interface {

	// Add handles a request for the 'add' method.
	//
	//
	Add(ctx context.Context, request *StatusUpdatesAddServerRequest, response *StatusUpdatesAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of statuses.
	List(ctx context.Context, request *StatusUpdatesListServerRequest, response *StatusUpdatesListServerResponse) error

	// Status returns the target 'status' server for the given identifier.
	//
	//
	Status(id string) StatusServer
}

// StatusUpdatesAddServerRequest is the request for the 'add' method.
type StatusUpdatesAddServerRequest struct {
	body *Status
}

// Body returns the value of the 'body' parameter.
//
//
func (r *StatusUpdatesAddServerRequest) Body() *Status {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusUpdatesAddServerRequest) GetBody() (value *Status, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// StatusUpdatesAddServerResponse is the response for the 'add' method.
type StatusUpdatesAddServerResponse struct {
	status int
	err    *errors.Error
	body   *Status
}

// Body sets the value of the 'body' parameter.
//
//
func (r *StatusUpdatesAddServerResponse) Body(value *Status) *StatusUpdatesAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *StatusUpdatesAddServerResponse) Status(value int) *StatusUpdatesAddServerResponse {
	r.status = value
	return r
}

// StatusUpdatesListServerRequest is the request for the 'list' method.
type StatusUpdatesListServerRequest struct {
	createdAfter  *time.Time
	createdBefore *time.Time
	page          *int
	productIds    *string
	size          *int
}

// CreatedAfter returns the value of the 'created_after' parameter.
//
//
func (r *StatusUpdatesListServerRequest) CreatedAfter() time.Time {
	if r != nil && r.createdAfter != nil {
		return *r.createdAfter
	}
	return time.Time{}
}

// GetCreatedAfter returns the value of the 'created_after' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusUpdatesListServerRequest) GetCreatedAfter() (value time.Time, ok bool) {
	ok = r != nil && r.createdAfter != nil
	if ok {
		value = *r.createdAfter
	}
	return
}

// CreatedBefore returns the value of the 'created_before' parameter.
//
//
func (r *StatusUpdatesListServerRequest) CreatedBefore() time.Time {
	if r != nil && r.createdBefore != nil {
		return *r.createdBefore
	}
	return time.Time{}
}

// GetCreatedBefore returns the value of the 'created_before' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusUpdatesListServerRequest) GetCreatedBefore() (value time.Time, ok bool) {
	ok = r != nil && r.createdBefore != nil
	if ok {
		value = *r.createdBefore
	}
	return
}

// Page returns the value of the 'page' parameter.
//
//
func (r *StatusUpdatesListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusUpdatesListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// ProductIds returns the value of the 'product_ids' parameter.
//
//
func (r *StatusUpdatesListServerRequest) ProductIds() string {
	if r != nil && r.productIds != nil {
		return *r.productIds
	}
	return ""
}

// GetProductIds returns the value of the 'product_ids' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusUpdatesListServerRequest) GetProductIds() (value string, ok bool) {
	ok = r != nil && r.productIds != nil
	if ok {
		value = *r.productIds
	}
	return
}

// Size returns the value of the 'size' parameter.
//
//
func (r *StatusUpdatesListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusUpdatesListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// StatusUpdatesListServerResponse is the response for the 'list' method.
type StatusUpdatesListServerResponse struct {
	status int
	err    *errors.Error
	items  *StatusList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
//
func (r *StatusUpdatesListServerResponse) Items(value *StatusList) *StatusUpdatesListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
//
func (r *StatusUpdatesListServerResponse) Page(value int) *StatusUpdatesListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
//
func (r *StatusUpdatesListServerResponse) Size(value int) *StatusUpdatesListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
//
func (r *StatusUpdatesListServerResponse) Total(value int) *StatusUpdatesListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *StatusUpdatesListServerResponse) Status(value int) *StatusUpdatesListServerResponse {
	r.status = value
	return r
}

// dispatchStatusUpdates navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchStatusUpdates(w http.ResponseWriter, r *http.Request, server StatusUpdatesServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptStatusUpdatesAddRequest(w, r, server)
			return
		case "GET":
			adaptStatusUpdatesListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.Status(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchStatus(w, r, target, segments[1:])
	}
}

// adaptStatusUpdatesAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusUpdatesAddRequest(w http.ResponseWriter, r *http.Request, server StatusUpdatesServer) {
	request := &StatusUpdatesAddServerRequest{}
	err := readStatusUpdatesAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusUpdatesAddServerResponse{}
	response.status = 201
	err = server.Add(r.Context(), request, response)
	if err != nil {
		glog.Errorf(
			"Can't process request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	err = writeStatusUpdatesAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptStatusUpdatesListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusUpdatesListRequest(w http.ResponseWriter, r *http.Request, server StatusUpdatesServer) {
	request := &StatusUpdatesListServerRequest{}
	err := readStatusUpdatesListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusUpdatesListServerResponse{}
	response.status = 200
	err = server.List(r.Context(), request, response)
	if err != nil {
		glog.Errorf(
			"Can't process request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	err = writeStatusUpdatesListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
