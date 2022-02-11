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

// OperatorIAMRolesServer represents the interface the manages the 'operator_IAM_roles' resource.
type OperatorIAMRolesServer interface {

	// Add handles a request for the 'add' method.
	//
	// Adds a new operator role to the cluster.
	Add(ctx context.Context, request *OperatorIAMRolesAddServerRequest, response *OperatorIAMRolesAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of operator roles.
	List(ctx context.Context, request *OperatorIAMRolesListServerRequest, response *OperatorIAMRolesListServerResponse) error

	// OperatorIAMRole returns the target 'operator_IAM_role' server for the given identifier.
	//
	// Returns a reference to the service that manages a specific operator role.
	OperatorIAMRole(id string) OperatorIAMRoleServer
}

// OperatorIAMRolesAddServerRequest is the request for the 'add' method.
type OperatorIAMRolesAddServerRequest struct {
	body *OperatorIAMRole
}

// Body returns the value of the 'body' parameter.
//
// Description of the operator role.
func (r *OperatorIAMRolesAddServerRequest) Body() *OperatorIAMRole {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
// Description of the operator role.
func (r *OperatorIAMRolesAddServerRequest) GetBody() (value *OperatorIAMRole, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// OperatorIAMRolesAddServerResponse is the response for the 'add' method.
type OperatorIAMRolesAddServerResponse struct {
	status int
	err    *errors.Error
	body   *OperatorIAMRole
}

// Body sets the value of the 'body' parameter.
//
// Description of the operator role.
func (r *OperatorIAMRolesAddServerResponse) Body(value *OperatorIAMRole) *OperatorIAMRolesAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *OperatorIAMRolesAddServerResponse) Status(value int) *OperatorIAMRolesAddServerResponse {
	r.status = value
	return r
}

// OperatorIAMRolesListServerRequest is the request for the 'list' method.
type OperatorIAMRolesListServerRequest struct {
	page *int
	size *int
}

// Page returns the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *OperatorIAMRolesListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
// Index of the requested page, where one corresponds to the first page.
func (r *OperatorIAMRolesListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
// Number of items that will be contained in the returned page.
func (r *OperatorIAMRolesListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
// Number of items that will be contained in the returned page.
func (r *OperatorIAMRolesListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// OperatorIAMRolesListServerResponse is the response for the 'list' method.
type OperatorIAMRolesListServerResponse struct {
	status int
	err    *errors.Error
	items  *OperatorIAMRoleList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
// Retrieved list of operator roles.
func (r *OperatorIAMRolesListServerResponse) Items(value *OperatorIAMRoleList) *OperatorIAMRolesListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *OperatorIAMRolesListServerResponse) Page(value int) *OperatorIAMRolesListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
// Number of items that will be contained in the returned page.
func (r *OperatorIAMRolesListServerResponse) Size(value int) *OperatorIAMRolesListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
// Total number of items of the collection that match the search criteria,
// regardless of the size of the page.
func (r *OperatorIAMRolesListServerResponse) Total(value int) *OperatorIAMRolesListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *OperatorIAMRolesListServerResponse) Status(value int) *OperatorIAMRolesListServerResponse {
	r.status = value
	return r
}

// dispatchOperatorIAMRoles navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchOperatorIAMRoles(w http.ResponseWriter, r *http.Request, server OperatorIAMRolesServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptOperatorIAMRolesAddRequest(w, r, server)
			return
		case "GET":
			adaptOperatorIAMRolesListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.OperatorIAMRole(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchOperatorIAMRole(w, r, target, segments[1:])
	}
}

// adaptOperatorIAMRolesAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptOperatorIAMRolesAddRequest(w http.ResponseWriter, r *http.Request, server OperatorIAMRolesServer) {
	request := &OperatorIAMRolesAddServerRequest{}
	err := readOperatorIAMRolesAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &OperatorIAMRolesAddServerResponse{}
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
	err = writeOperatorIAMRolesAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptOperatorIAMRolesListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptOperatorIAMRolesListRequest(w http.ResponseWriter, r *http.Request, server OperatorIAMRolesServer) {
	request := &OperatorIAMRolesListServerRequest{}
	err := readOperatorIAMRolesListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &OperatorIAMRolesListServerResponse{}
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
	err = writeOperatorIAMRolesListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
