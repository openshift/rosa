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

// StatusesServer represents the interface the manages the 'statuses' resource.
type StatusesServer interface {

	// Add handles a request for the 'add' method.
	//
	//
	Add(ctx context.Context, request *StatusesAddServerRequest, response *StatusesAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of statuses.
	List(ctx context.Context, request *StatusesListServerRequest, response *StatusesListServerResponse) error

	// Status returns the target 'status' server for the given identifier.
	//
	//
	Status(id string) StatusServer
}

// StatusesAddServerRequest is the request for the 'add' method.
type StatusesAddServerRequest struct {
	body *Status
}

// Body returns the value of the 'body' parameter.
//
//
func (r *StatusesAddServerRequest) Body() *Status {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusesAddServerRequest) GetBody() (value *Status, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// StatusesAddServerResponse is the response for the 'add' method.
type StatusesAddServerResponse struct {
	status int
	err    *errors.Error
	body   *Status
}

// Body sets the value of the 'body' parameter.
//
//
func (r *StatusesAddServerResponse) Body(value *Status) *StatusesAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *StatusesAddServerResponse) Status(value int) *StatusesAddServerResponse {
	r.status = value
	return r
}

// StatusesListServerRequest is the request for the 'list' method.
type StatusesListServerRequest struct {
	createdAfter  *time.Time
	createdBefore *time.Time
	page          *int
	productIds    *string
	size          *int
}

// CreatedAfter returns the value of the 'created_after' parameter.
//
//
func (r *StatusesListServerRequest) CreatedAfter() time.Time {
	if r != nil && r.createdAfter != nil {
		return *r.createdAfter
	}
	return time.Time{}
}

// GetCreatedAfter returns the value of the 'created_after' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusesListServerRequest) GetCreatedAfter() (value time.Time, ok bool) {
	ok = r != nil && r.createdAfter != nil
	if ok {
		value = *r.createdAfter
	}
	return
}

// CreatedBefore returns the value of the 'created_before' parameter.
//
//
func (r *StatusesListServerRequest) CreatedBefore() time.Time {
	if r != nil && r.createdBefore != nil {
		return *r.createdBefore
	}
	return time.Time{}
}

// GetCreatedBefore returns the value of the 'created_before' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusesListServerRequest) GetCreatedBefore() (value time.Time, ok bool) {
	ok = r != nil && r.createdBefore != nil
	if ok {
		value = *r.createdBefore
	}
	return
}

// Page returns the value of the 'page' parameter.
//
//
func (r *StatusesListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusesListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// ProductIds returns the value of the 'product_ids' parameter.
//
//
func (r *StatusesListServerRequest) ProductIds() string {
	if r != nil && r.productIds != nil {
		return *r.productIds
	}
	return ""
}

// GetProductIds returns the value of the 'product_ids' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusesListServerRequest) GetProductIds() (value string, ok bool) {
	ok = r != nil && r.productIds != nil
	if ok {
		value = *r.productIds
	}
	return
}

// Size returns the value of the 'size' parameter.
//
//
func (r *StatusesListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusesListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// StatusesListServerResponse is the response for the 'list' method.
type StatusesListServerResponse struct {
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
func (r *StatusesListServerResponse) Items(value *StatusList) *StatusesListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
//
func (r *StatusesListServerResponse) Page(value int) *StatusesListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
//
func (r *StatusesListServerResponse) Size(value int) *StatusesListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
//
func (r *StatusesListServerResponse) Total(value int) *StatusesListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *StatusesListServerResponse) Status(value int) *StatusesListServerResponse {
	r.status = value
	return r
}

// dispatchStatuses navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchStatuses(w http.ResponseWriter, r *http.Request, server StatusesServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptStatusesAddRequest(w, r, server)
			return
		case "GET":
			adaptStatusesListRequest(w, r, server)
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

// adaptStatusesAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusesAddRequest(w http.ResponseWriter, r *http.Request, server StatusesServer) {
	request := &StatusesAddServerRequest{}
	err := readStatusesAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusesAddServerResponse{}
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
	err = writeStatusesAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptStatusesListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusesListRequest(w http.ResponseWriter, r *http.Request, server StatusesServer) {
	request := &StatusesListServerRequest{}
	err := readStatusesListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusesListServerResponse{}
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
	err = writeStatusesListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
