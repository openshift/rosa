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

// HTPasswdUsersServer represents the interface the manages the 'HT_passwd_users' resource.
type HTPasswdUsersServer interface {

	// Add handles a request for the 'add' method.
	//
	// Adds a new user to the _HTPasswd_ file.
	Add(ctx context.Context, request *HTPasswdUsersAddServerRequest, response *HTPasswdUsersAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of _HTPasswd_ IDP users.
	List(ctx context.Context, request *HTPasswdUsersListServerRequest, response *HTPasswdUsersListServerResponse) error

	// HtpasswdUser returns the target 'HT_passwd_user' server for the given identifier.
	//
	// Reference to the service that manages a specific _HTPasswd_ user.
	HtpasswdUser(id string) HTPasswdUserServer
}

// HTPasswdUsersAddServerRequest is the request for the 'add' method.
type HTPasswdUsersAddServerRequest struct {
	body *HTPasswdUser
}

// Body returns the value of the 'body' parameter.
//
// New user to be added
func (r *HTPasswdUsersAddServerRequest) Body() *HTPasswdUser {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
// New user to be added
func (r *HTPasswdUsersAddServerRequest) GetBody() (value *HTPasswdUser, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// HTPasswdUsersAddServerResponse is the response for the 'add' method.
type HTPasswdUsersAddServerResponse struct {
	status int
	err    *errors.Error
	body   *HTPasswdUser
}

// Body sets the value of the 'body' parameter.
//
// New user to be added
func (r *HTPasswdUsersAddServerResponse) Body(value *HTPasswdUser) *HTPasswdUsersAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *HTPasswdUsersAddServerResponse) Status(value int) *HTPasswdUsersAddServerResponse {
	r.status = value
	return r
}

// HTPasswdUsersListServerRequest is the request for the 'list' method.
type HTPasswdUsersListServerRequest struct {
	page *int
	size *int
}

// Page returns the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *HTPasswdUsersListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
// Index of the requested page, where one corresponds to the first page.
func (r *HTPasswdUsersListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
// Number of items contained in the returned page.
func (r *HTPasswdUsersListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
// Number of items contained in the returned page.
func (r *HTPasswdUsersListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// HTPasswdUsersListServerResponse is the response for the 'list' method.
type HTPasswdUsersListServerResponse struct {
	status int
	err    *errors.Error
	items  *HTPasswdUserList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
// Retrieved list of users of the IDP.
func (r *HTPasswdUsersListServerResponse) Items(value *HTPasswdUserList) *HTPasswdUsersListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *HTPasswdUsersListServerResponse) Page(value int) *HTPasswdUsersListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
// Number of items contained in the returned page.
func (r *HTPasswdUsersListServerResponse) Size(value int) *HTPasswdUsersListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
// Total number of items of the collection.
func (r *HTPasswdUsersListServerResponse) Total(value int) *HTPasswdUsersListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *HTPasswdUsersListServerResponse) Status(value int) *HTPasswdUsersListServerResponse {
	r.status = value
	return r
}

// dispatchHTPasswdUsers navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchHTPasswdUsers(w http.ResponseWriter, r *http.Request, server HTPasswdUsersServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptHTPasswdUsersAddRequest(w, r, server)
			return
		case "GET":
			adaptHTPasswdUsersListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.HtpasswdUser(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchHTPasswdUser(w, r, target, segments[1:])
	}
}

// adaptHTPasswdUsersAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptHTPasswdUsersAddRequest(w http.ResponseWriter, r *http.Request, server HTPasswdUsersServer) {
	request := &HTPasswdUsersAddServerRequest{}
	err := readHTPasswdUsersAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &HTPasswdUsersAddServerResponse{}
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
	err = writeHTPasswdUsersAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptHTPasswdUsersListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptHTPasswdUsersListRequest(w http.ResponseWriter, r *http.Request, server HTPasswdUsersServer) {
	request := &HTPasswdUsersListServerRequest{}
	err := readHTPasswdUsersListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &HTPasswdUsersListServerResponse{}
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
	err = writeHTPasswdUsersListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
