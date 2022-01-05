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

// ApplicationDependenciesServer represents the interface the manages the 'application_dependencies' resource.
type ApplicationDependenciesServer interface {

	// Add handles a request for the 'add' method.
	//
	//
	Add(ctx context.Context, request *ApplicationDependenciesAddServerRequest, response *ApplicationDependenciesAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of application dependencies.
	List(ctx context.Context, request *ApplicationDependenciesListServerRequest, response *ApplicationDependenciesListServerResponse) error

	// ApplicationDependency returns the target 'application_dependency' server for the given identifier.
	//
	//
	ApplicationDependency(id string) ApplicationDependencyServer
}

// ApplicationDependenciesAddServerRequest is the request for the 'add' method.
type ApplicationDependenciesAddServerRequest struct {
	body *ApplicationDependency
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ApplicationDependenciesAddServerRequest) Body() *ApplicationDependency {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationDependenciesAddServerRequest) GetBody() (value *ApplicationDependency, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ApplicationDependenciesAddServerResponse is the response for the 'add' method.
type ApplicationDependenciesAddServerResponse struct {
	status int
	err    *errors.Error
	body   *ApplicationDependency
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ApplicationDependenciesAddServerResponse) Body(value *ApplicationDependency) *ApplicationDependenciesAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ApplicationDependenciesAddServerResponse) Status(value int) *ApplicationDependenciesAddServerResponse {
	r.status = value
	return r
}

// ApplicationDependenciesListServerRequest is the request for the 'list' method.
type ApplicationDependenciesListServerRequest struct {
	orderBy *string
	page    *int
	size    *int
}

// OrderBy returns the value of the 'order_by' parameter.
//
//
func (r *ApplicationDependenciesListServerRequest) OrderBy() string {
	if r != nil && r.orderBy != nil {
		return *r.orderBy
	}
	return ""
}

// GetOrderBy returns the value of the 'order_by' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationDependenciesListServerRequest) GetOrderBy() (value string, ok bool) {
	ok = r != nil && r.orderBy != nil
	if ok {
		value = *r.orderBy
	}
	return
}

// Page returns the value of the 'page' parameter.
//
//
func (r *ApplicationDependenciesListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationDependenciesListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
//
func (r *ApplicationDependenciesListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationDependenciesListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// ApplicationDependenciesListServerResponse is the response for the 'list' method.
type ApplicationDependenciesListServerResponse struct {
	status int
	err    *errors.Error
	items  *ApplicationDependencyList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
//
func (r *ApplicationDependenciesListServerResponse) Items(value *ApplicationDependencyList) *ApplicationDependenciesListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
//
func (r *ApplicationDependenciesListServerResponse) Page(value int) *ApplicationDependenciesListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
//
func (r *ApplicationDependenciesListServerResponse) Size(value int) *ApplicationDependenciesListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
//
func (r *ApplicationDependenciesListServerResponse) Total(value int) *ApplicationDependenciesListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *ApplicationDependenciesListServerResponse) Status(value int) *ApplicationDependenciesListServerResponse {
	r.status = value
	return r
}

// dispatchApplicationDependencies navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchApplicationDependencies(w http.ResponseWriter, r *http.Request, server ApplicationDependenciesServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptApplicationDependenciesAddRequest(w, r, server)
			return
		case "GET":
			adaptApplicationDependenciesListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.ApplicationDependency(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchApplicationDependency(w, r, target, segments[1:])
	}
}

// adaptApplicationDependenciesAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationDependenciesAddRequest(w http.ResponseWriter, r *http.Request, server ApplicationDependenciesServer) {
	request := &ApplicationDependenciesAddServerRequest{}
	err := readApplicationDependenciesAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationDependenciesAddServerResponse{}
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
	err = writeApplicationDependenciesAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptApplicationDependenciesListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationDependenciesListRequest(w http.ResponseWriter, r *http.Request, server ApplicationDependenciesServer) {
	request := &ApplicationDependenciesListServerRequest{}
	err := readApplicationDependenciesListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationDependenciesListServerResponse{}
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
	err = writeApplicationDependenciesListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
