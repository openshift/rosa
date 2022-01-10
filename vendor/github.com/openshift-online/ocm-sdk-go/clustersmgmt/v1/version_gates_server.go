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

	"github.com/golang/glog"
	"github.com/openshift-online/ocm-sdk-go/errors"
)

// VersionGatesServer represents the interface the manages the 'version_gates' resource.
type VersionGatesServer interface {

	// Add handles a request for the 'add' method.
	//
	// Adds a new version gate
	Add(ctx context.Context, request *VersionGatesAddServerRequest, response *VersionGatesAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves a list of version gates.
	List(ctx context.Context, request *VersionGatesListServerRequest, response *VersionGatesListServerResponse) error

	// VersionGate returns the target 'version_gate' server for the given identifier.
	//
	// Reference to the resource that manages a specific version gate.
	VersionGate(id string) VersionGateServer
}

// VersionGatesAddServerRequest is the request for the 'add' method.
type VersionGatesAddServerRequest struct {
	body *VersionGate
}

// Body returns the value of the 'body' parameter.
//
// Details of the version gate
func (r *VersionGatesAddServerRequest) Body() *VersionGate {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
// Details of the version gate
func (r *VersionGatesAddServerRequest) GetBody() (value *VersionGate, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// VersionGatesAddServerResponse is the response for the 'add' method.
type VersionGatesAddServerResponse struct {
	status int
	err    *errors.Error
	body   *VersionGate
}

// Body sets the value of the 'body' parameter.
//
// Details of the version gate
func (r *VersionGatesAddServerResponse) Body(value *VersionGate) *VersionGatesAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *VersionGatesAddServerResponse) Status(value int) *VersionGatesAddServerResponse {
	r.status = value
	return r
}

// VersionGatesListServerRequest is the request for the 'list' method.
type VersionGatesListServerRequest struct {
	order  *string
	page   *int
	search *string
	size   *int
}

// Order returns the value of the 'order' parameter.
//
// Order criteria.
//
// The syntax of this parameter is similar to the syntax of the _order by_ clause of
// an SQL statement, but using the names of the attributes of the version gate instead of
// the names of the columns of a table. For example, in order to sort the version gates
// descending by identifier the value should be:
//
// ```sql
// id desc
// ```
//
// If the parameter isn't provided, or if the value is empty, then the order of the
// results is undefined.
func (r *VersionGatesListServerRequest) Order() string {
	if r != nil && r.order != nil {
		return *r.order
	}
	return ""
}

// GetOrder returns the value of the 'order' parameter and
// a flag indicating if the parameter has a value.
//
// Order criteria.
//
// The syntax of this parameter is similar to the syntax of the _order by_ clause of
// an SQL statement, but using the names of the attributes of the version gate instead of
// the names of the columns of a table. For example, in order to sort the version gates
// descending by identifier the value should be:
//
// ```sql
// id desc
// ```
//
// If the parameter isn't provided, or if the value is empty, then the order of the
// results is undefined.
func (r *VersionGatesListServerRequest) GetOrder() (value string, ok bool) {
	ok = r != nil && r.order != nil
	if ok {
		value = *r.order
	}
	return
}

// Page returns the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *VersionGatesListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
// Index of the requested page, where one corresponds to the first page.
func (r *VersionGatesListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Search returns the value of the 'search' parameter.
//
// Search criteria.
//
// The syntax of this parameter is similar to the syntax of the _where_ clause of an
// SQL statement, but using the names of the attributes of the version gate instead of
// the names of the columns of a table.
//
// If the parameter isn't provided, or if the value is empty, then all the version gates
// that the user has permission to see will be returned.
func (r *VersionGatesListServerRequest) Search() string {
	if r != nil && r.search != nil {
		return *r.search
	}
	return ""
}

// GetSearch returns the value of the 'search' parameter and
// a flag indicating if the parameter has a value.
//
// Search criteria.
//
// The syntax of this parameter is similar to the syntax of the _where_ clause of an
// SQL statement, but using the names of the attributes of the version gate instead of
// the names of the columns of a table.
//
// If the parameter isn't provided, or if the value is empty, then all the version gates
// that the user has permission to see will be returned.
func (r *VersionGatesListServerRequest) GetSearch() (value string, ok bool) {
	ok = r != nil && r.search != nil
	if ok {
		value = *r.search
	}
	return
}

// Size returns the value of the 'size' parameter.
//
// Maximum number of items that will be contained in the returned page.
//
// Default value is `100`.
func (r *VersionGatesListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
// Maximum number of items that will be contained in the returned page.
//
// Default value is `100`.
func (r *VersionGatesListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// VersionGatesListServerResponse is the response for the 'list' method.
type VersionGatesListServerResponse struct {
	status int
	err    *errors.Error
	items  *VersionGateList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
// Retrieved list of version gates.
func (r *VersionGatesListServerResponse) Items(value *VersionGateList) *VersionGatesListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *VersionGatesListServerResponse) Page(value int) *VersionGatesListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
// Maximum number of items that will be contained in the returned page.
//
// Default value is `100`.
func (r *VersionGatesListServerResponse) Size(value int) *VersionGatesListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
// Total number of items of the collection that match the search criteria,
// regardless of the size of the page.
func (r *VersionGatesListServerResponse) Total(value int) *VersionGatesListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *VersionGatesListServerResponse) Status(value int) *VersionGatesListServerResponse {
	r.status = value
	return r
}

// dispatchVersionGates navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchVersionGates(w http.ResponseWriter, r *http.Request, server VersionGatesServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptVersionGatesAddRequest(w, r, server)
			return
		case "GET":
			adaptVersionGatesListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.VersionGate(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchVersionGate(w, r, target, segments[1:])
	}
}

// adaptVersionGatesAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptVersionGatesAddRequest(w http.ResponseWriter, r *http.Request, server VersionGatesServer) {
	request := &VersionGatesAddServerRequest{}
	err := readVersionGatesAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &VersionGatesAddServerResponse{}
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
	err = writeVersionGatesAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptVersionGatesListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptVersionGatesListRequest(w http.ResponseWriter, r *http.Request, server VersionGatesServer) {
	request := &VersionGatesListServerRequest{}
	err := readVersionGatesListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &VersionGatesListServerResponse{}
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
	err = writeVersionGatesListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
