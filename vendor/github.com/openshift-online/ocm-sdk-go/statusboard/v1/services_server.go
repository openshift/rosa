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

	"github.com/golang/glog"
	"github.com/openshift-online/ocm-sdk-go/errors"
)

// ServicesServer represents the interface the manages the 'services' resource.
type ServicesServer interface {

	// Add handles a request for the 'add' method.
	//
	//
	Add(ctx context.Context, request *ServicesAddServerRequest, response *ServicesAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of services.
	List(ctx context.Context, request *ServicesListServerRequest, response *ServicesListServerResponse) error

	// Service returns the target 'service' server for the given identifier.
	//
	//
	Service(id string) ServiceServer
}

// ServicesAddServerRequest is the request for the 'add' method.
type ServicesAddServerRequest struct {
	body *Service
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ServicesAddServerRequest) Body() *Service {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServicesAddServerRequest) GetBody() (value *Service, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ServicesAddServerResponse is the response for the 'add' method.
type ServicesAddServerResponse struct {
	status int
	err    *errors.Error
	body   *Service
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ServicesAddServerResponse) Body(value *Service) *ServicesAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ServicesAddServerResponse) Status(value int) *ServicesAddServerResponse {
	r.status = value
	return r
}

// ServicesListServerRequest is the request for the 'list' method.
type ServicesListServerRequest struct {
	fullname *string
	mine     *bool
	orderBy  *string
	page     *int
	size     *int
}

// Fullname returns the value of the 'fullname' parameter.
//
//
func (r *ServicesListServerRequest) Fullname() string {
	if r != nil && r.fullname != nil {
		return *r.fullname
	}
	return ""
}

// GetFullname returns the value of the 'fullname' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServicesListServerRequest) GetFullname() (value string, ok bool) {
	ok = r != nil && r.fullname != nil
	if ok {
		value = *r.fullname
	}
	return
}

// Mine returns the value of the 'mine' parameter.
//
//
func (r *ServicesListServerRequest) Mine() bool {
	if r != nil && r.mine != nil {
		return *r.mine
	}
	return false
}

// GetMine returns the value of the 'mine' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServicesListServerRequest) GetMine() (value bool, ok bool) {
	ok = r != nil && r.mine != nil
	if ok {
		value = *r.mine
	}
	return
}

// OrderBy returns the value of the 'order_by' parameter.
//
//
func (r *ServicesListServerRequest) OrderBy() string {
	if r != nil && r.orderBy != nil {
		return *r.orderBy
	}
	return ""
}

// GetOrderBy returns the value of the 'order_by' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServicesListServerRequest) GetOrderBy() (value string, ok bool) {
	ok = r != nil && r.orderBy != nil
	if ok {
		value = *r.orderBy
	}
	return
}

// Page returns the value of the 'page' parameter.
//
//
func (r *ServicesListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServicesListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
//
func (r *ServicesListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServicesListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// ServicesListServerResponse is the response for the 'list' method.
type ServicesListServerResponse struct {
	status int
	err    *errors.Error
	items  *ServiceList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
//
func (r *ServicesListServerResponse) Items(value *ServiceList) *ServicesListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
//
func (r *ServicesListServerResponse) Page(value int) *ServicesListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
//
func (r *ServicesListServerResponse) Size(value int) *ServicesListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
//
func (r *ServicesListServerResponse) Total(value int) *ServicesListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *ServicesListServerResponse) Status(value int) *ServicesListServerResponse {
	r.status = value
	return r
}

// dispatchServices navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchServices(w http.ResponseWriter, r *http.Request, server ServicesServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptServicesAddRequest(w, r, server)
			return
		case "GET":
			adaptServicesListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.Service(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchService(w, r, target, segments[1:])
	}
}

// adaptServicesAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServicesAddRequest(w http.ResponseWriter, r *http.Request, server ServicesServer) {
	request := &ServicesAddServerRequest{}
	err := readServicesAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServicesAddServerResponse{}
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
	err = writeServicesAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptServicesListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServicesListRequest(w http.ResponseWriter, r *http.Request, server ServicesServer) {
	request := &ServicesListServerRequest{}
	err := readServicesListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServicesListServerResponse{}
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
	err = writeServicesListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
