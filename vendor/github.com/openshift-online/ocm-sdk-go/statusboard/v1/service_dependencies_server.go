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

// ServiceDependenciesServer represents the interface the manages the 'service_dependencies' resource.
type ServiceDependenciesServer interface {

	// Add handles a request for the 'add' method.
	//
	//
	Add(ctx context.Context, request *ServiceDependenciesAddServerRequest, response *ServiceDependenciesAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of service dependencies.
	List(ctx context.Context, request *ServiceDependenciesListServerRequest, response *ServiceDependenciesListServerResponse) error

	// ServiceDependency returns the target 'service_dependency' server for the given identifier.
	//
	//
	ServiceDependency(id string) ServiceDependencyServer
}

// ServiceDependenciesAddServerRequest is the request for the 'add' method.
type ServiceDependenciesAddServerRequest struct {
	body *ServiceDependency
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ServiceDependenciesAddServerRequest) Body() *ServiceDependency {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServiceDependenciesAddServerRequest) GetBody() (value *ServiceDependency, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ServiceDependenciesAddServerResponse is the response for the 'add' method.
type ServiceDependenciesAddServerResponse struct {
	status int
	err    *errors.Error
	body   *ServiceDependency
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ServiceDependenciesAddServerResponse) Body(value *ServiceDependency) *ServiceDependenciesAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ServiceDependenciesAddServerResponse) Status(value int) *ServiceDependenciesAddServerResponse {
	r.status = value
	return r
}

// ServiceDependenciesListServerRequest is the request for the 'list' method.
type ServiceDependenciesListServerRequest struct {
	orderBy *string
	page    *int
	size    *int
}

// OrderBy returns the value of the 'order_by' parameter.
//
//
func (r *ServiceDependenciesListServerRequest) OrderBy() string {
	if r != nil && r.orderBy != nil {
		return *r.orderBy
	}
	return ""
}

// GetOrderBy returns the value of the 'order_by' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServiceDependenciesListServerRequest) GetOrderBy() (value string, ok bool) {
	ok = r != nil && r.orderBy != nil
	if ok {
		value = *r.orderBy
	}
	return
}

// Page returns the value of the 'page' parameter.
//
//
func (r *ServiceDependenciesListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServiceDependenciesListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
//
func (r *ServiceDependenciesListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServiceDependenciesListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// ServiceDependenciesListServerResponse is the response for the 'list' method.
type ServiceDependenciesListServerResponse struct {
	status int
	err    *errors.Error
	items  *ServiceDependencyList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
//
func (r *ServiceDependenciesListServerResponse) Items(value *ServiceDependencyList) *ServiceDependenciesListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
//
func (r *ServiceDependenciesListServerResponse) Page(value int) *ServiceDependenciesListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
//
func (r *ServiceDependenciesListServerResponse) Size(value int) *ServiceDependenciesListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
//
func (r *ServiceDependenciesListServerResponse) Total(value int) *ServiceDependenciesListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *ServiceDependenciesListServerResponse) Status(value int) *ServiceDependenciesListServerResponse {
	r.status = value
	return r
}

// dispatchServiceDependencies navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchServiceDependencies(w http.ResponseWriter, r *http.Request, server ServiceDependenciesServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptServiceDependenciesAddRequest(w, r, server)
			return
		case "GET":
			adaptServiceDependenciesListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.ServiceDependency(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchServiceDependency(w, r, target, segments[1:])
	}
}

// adaptServiceDependenciesAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServiceDependenciesAddRequest(w http.ResponseWriter, r *http.Request, server ServiceDependenciesServer) {
	request := &ServiceDependenciesAddServerRequest{}
	err := readServiceDependenciesAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServiceDependenciesAddServerResponse{}
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
	err = writeServiceDependenciesAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptServiceDependenciesListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServiceDependenciesListRequest(w http.ResponseWriter, r *http.Request, server ServiceDependenciesServer) {
	request := &ServiceDependenciesListServerRequest{}
	err := readServiceDependenciesListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServiceDependenciesListServerResponse{}
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
	err = writeServiceDependenciesListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
