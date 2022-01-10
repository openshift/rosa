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

// AddOnVersionsServer represents the interface the manages the 'add_on_versions' resource.
type AddOnVersionsServer interface {

	// Add handles a request for the 'add' method.
	//
	// Create a new add-on version and add it to the collection of add-ons.
	Add(ctx context.Context, request *AddOnVersionsAddServerRequest, response *AddOnVersionsAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of add-on versions.
	List(ctx context.Context, request *AddOnVersionsListServerRequest, response *AddOnVersionsListServerResponse) error

	// Version returns the target 'add_on_version' server for the given identifier.
	//
	// Returns a reference to the service that manages a specific add-on version.
	Version(id string) AddOnVersionServer
}

// AddOnVersionsAddServerRequest is the request for the 'add' method.
type AddOnVersionsAddServerRequest struct {
	body *AddOnVersion
}

// Body returns the value of the 'body' parameter.
//
// Description of the add-on version.
func (r *AddOnVersionsAddServerRequest) Body() *AddOnVersion {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
// Description of the add-on version.
func (r *AddOnVersionsAddServerRequest) GetBody() (value *AddOnVersion, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// AddOnVersionsAddServerResponse is the response for the 'add' method.
type AddOnVersionsAddServerResponse struct {
	status int
	err    *errors.Error
	body   *AddOnVersion
}

// Body sets the value of the 'body' parameter.
//
// Description of the add-on version.
func (r *AddOnVersionsAddServerResponse) Body(value *AddOnVersion) *AddOnVersionsAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *AddOnVersionsAddServerResponse) Status(value int) *AddOnVersionsAddServerResponse {
	r.status = value
	return r
}

// AddOnVersionsListServerRequest is the request for the 'list' method.
type AddOnVersionsListServerRequest struct {
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
// a SQL statement, but using the names of the attributes of the add-on instead of
// the names of the columns of a table. For example, in order to sort the add-on
// versions descending by id the value should be:
//
// ```sql
// id desc
// ```
//
// If the parameter isn't provided, or if the value is empty, then the order of the
// results is undefined.
func (r *AddOnVersionsListServerRequest) Order() string {
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
// a SQL statement, but using the names of the attributes of the add-on instead of
// the names of the columns of a table. For example, in order to sort the add-on
// versions descending by id the value should be:
//
// ```sql
// id desc
// ```
//
// If the parameter isn't provided, or if the value is empty, then the order of the
// results is undefined.
func (r *AddOnVersionsListServerRequest) GetOrder() (value string, ok bool) {
	ok = r != nil && r.order != nil
	if ok {
		value = *r.order
	}
	return
}

// Page returns the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *AddOnVersionsListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
// Index of the requested page, where one corresponds to the first page.
func (r *AddOnVersionsListServerRequest) GetPage() (value int, ok bool) {
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
// SQL statement, but using the names of the attributes of the add-on version instead
// of the names of the columns of a table. For example, in order to retrieve all the
// add-on versions with an id starting with `0.1` the value should be:
//
// ```sql
// id like '0.1.%'
// ```
//
// If the parameter isn't provided, or if the value is empty, then all the add-on
// versions that the user has permission to see will be returned.
func (r *AddOnVersionsListServerRequest) Search() string {
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
// SQL statement, but using the names of the attributes of the add-on version instead
// of the names of the columns of a table. For example, in order to retrieve all the
// add-on versions with an id starting with `0.1` the value should be:
//
// ```sql
// id like '0.1.%'
// ```
//
// If the parameter isn't provided, or if the value is empty, then all the add-on
// versions that the user has permission to see will be returned.
func (r *AddOnVersionsListServerRequest) GetSearch() (value string, ok bool) {
	ok = r != nil && r.search != nil
	if ok {
		value = *r.search
	}
	return
}

// Size returns the value of the 'size' parameter.
//
// Maximum number of items that will be contained in the returned page.
func (r *AddOnVersionsListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
// Maximum number of items that will be contained in the returned page.
func (r *AddOnVersionsListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// AddOnVersionsListServerResponse is the response for the 'list' method.
type AddOnVersionsListServerResponse struct {
	status int
	err    *errors.Error
	items  *AddOnVersionList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
// Retrieved list of add-on versions.
func (r *AddOnVersionsListServerResponse) Items(value *AddOnVersionList) *AddOnVersionsListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *AddOnVersionsListServerResponse) Page(value int) *AddOnVersionsListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
// Maximum number of items that will be contained in the returned page.
func (r *AddOnVersionsListServerResponse) Size(value int) *AddOnVersionsListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
// Total number of items of the collection that match the search criteria,
// regardless of the size of the page.
func (r *AddOnVersionsListServerResponse) Total(value int) *AddOnVersionsListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *AddOnVersionsListServerResponse) Status(value int) *AddOnVersionsListServerResponse {
	r.status = value
	return r
}

// dispatchAddOnVersions navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchAddOnVersions(w http.ResponseWriter, r *http.Request, server AddOnVersionsServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptAddOnVersionsAddRequest(w, r, server)
			return
		case "GET":
			adaptAddOnVersionsListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.Version(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchAddOnVersion(w, r, target, segments[1:])
	}
}

// adaptAddOnVersionsAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptAddOnVersionsAddRequest(w http.ResponseWriter, r *http.Request, server AddOnVersionsServer) {
	request := &AddOnVersionsAddServerRequest{}
	err := readAddOnVersionsAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &AddOnVersionsAddServerResponse{}
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
	err = writeAddOnVersionsAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptAddOnVersionsListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptAddOnVersionsListRequest(w http.ResponseWriter, r *http.Request, server AddOnVersionsServer) {
	request := &AddOnVersionsListServerRequest{}
	err := readAddOnVersionsListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &AddOnVersionsListServerResponse{}
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
	err = writeAddOnVersionsListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
