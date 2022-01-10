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

// ApplicationsServer represents the interface the manages the 'applications' resource.
type ApplicationsServer interface {

	// Add handles a request for the 'add' method.
	//
	//
	Add(ctx context.Context, request *ApplicationsAddServerRequest, response *ApplicationsAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of applications.
	List(ctx context.Context, request *ApplicationsListServerRequest, response *ApplicationsListServerResponse) error

	// Application returns the target 'application' server for the given identifier.
	//
	//
	Application(id string) ApplicationServer
}

// ApplicationsAddServerRequest is the request for the 'add' method.
type ApplicationsAddServerRequest struct {
	body *Application
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ApplicationsAddServerRequest) Body() *Application {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationsAddServerRequest) GetBody() (value *Application, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ApplicationsAddServerResponse is the response for the 'add' method.
type ApplicationsAddServerResponse struct {
	status int
	err    *errors.Error
	body   *Application
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ApplicationsAddServerResponse) Body(value *Application) *ApplicationsAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ApplicationsAddServerResponse) Status(value int) *ApplicationsAddServerResponse {
	r.status = value
	return r
}

// ApplicationsListServerRequest is the request for the 'list' method.
type ApplicationsListServerRequest struct {
	fullname *string
	orderBy  *string
	page     *int
	size     *int
}

// Fullname returns the value of the 'fullname' parameter.
//
//
func (r *ApplicationsListServerRequest) Fullname() string {
	if r != nil && r.fullname != nil {
		return *r.fullname
	}
	return ""
}

// GetFullname returns the value of the 'fullname' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationsListServerRequest) GetFullname() (value string, ok bool) {
	ok = r != nil && r.fullname != nil
	if ok {
		value = *r.fullname
	}
	return
}

// OrderBy returns the value of the 'order_by' parameter.
//
//
func (r *ApplicationsListServerRequest) OrderBy() string {
	if r != nil && r.orderBy != nil {
		return *r.orderBy
	}
	return ""
}

// GetOrderBy returns the value of the 'order_by' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationsListServerRequest) GetOrderBy() (value string, ok bool) {
	ok = r != nil && r.orderBy != nil
	if ok {
		value = *r.orderBy
	}
	return
}

// Page returns the value of the 'page' parameter.
//
//
func (r *ApplicationsListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationsListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
//
func (r *ApplicationsListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationsListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// ApplicationsListServerResponse is the response for the 'list' method.
type ApplicationsListServerResponse struct {
	status int
	err    *errors.Error
	items  *ApplicationList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
//
func (r *ApplicationsListServerResponse) Items(value *ApplicationList) *ApplicationsListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
//
func (r *ApplicationsListServerResponse) Page(value int) *ApplicationsListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
//
func (r *ApplicationsListServerResponse) Size(value int) *ApplicationsListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
//
func (r *ApplicationsListServerResponse) Total(value int) *ApplicationsListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *ApplicationsListServerResponse) Status(value int) *ApplicationsListServerResponse {
	r.status = value
	return r
}

// dispatchApplications navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchApplications(w http.ResponseWriter, r *http.Request, server ApplicationsServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptApplicationsAddRequest(w, r, server)
			return
		case "GET":
			adaptApplicationsListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.Application(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchApplication(w, r, target, segments[1:])
	}
}

// adaptApplicationsAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationsAddRequest(w http.ResponseWriter, r *http.Request, server ApplicationsServer) {
	request := &ApplicationsAddServerRequest{}
	err := readApplicationsAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationsAddServerResponse{}
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
	err = writeApplicationsAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptApplicationsListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationsListRequest(w http.ResponseWriter, r *http.Request, server ApplicationsServer) {
	request := &ApplicationsListServerRequest{}
	err := readApplicationsListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationsListServerResponse{}
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
	err = writeApplicationsListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
