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

// HTPasswdUserServer represents the interface the manages the 'HT_passwd_user' resource.
type HTPasswdUserServer interface {

	// Delete handles a request for the 'delete' method.
	//
	// Deletes the user.
	Delete(ctx context.Context, request *HTPasswdUserDeleteServerRequest, response *HTPasswdUserDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	// Retrieves the details of the user.
	Get(ctx context.Context, request *HTPasswdUserGetServerRequest, response *HTPasswdUserGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	// Updates the user's password. The username is not editable
	Update(ctx context.Context, request *HTPasswdUserUpdateServerRequest, response *HTPasswdUserUpdateServerResponse) error
}

// HTPasswdUserDeleteServerRequest is the request for the 'delete' method.
type HTPasswdUserDeleteServerRequest struct {
}

// HTPasswdUserDeleteServerResponse is the response for the 'delete' method.
type HTPasswdUserDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *HTPasswdUserDeleteServerResponse) Status(value int) *HTPasswdUserDeleteServerResponse {
	r.status = value
	return r
}

// HTPasswdUserGetServerRequest is the request for the 'get' method.
type HTPasswdUserGetServerRequest struct {
}

// HTPasswdUserGetServerResponse is the response for the 'get' method.
type HTPasswdUserGetServerResponse struct {
	status int
	err    *errors.Error
	body   *HTPasswdUser
}

// Body sets the value of the 'body' parameter.
//
//
func (r *HTPasswdUserGetServerResponse) Body(value *HTPasswdUser) *HTPasswdUserGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *HTPasswdUserGetServerResponse) Status(value int) *HTPasswdUserGetServerResponse {
	r.status = value
	return r
}

// HTPasswdUserUpdateServerRequest is the request for the 'update' method.
type HTPasswdUserUpdateServerRequest struct {
	body *HTPasswdUser
}

// Body returns the value of the 'body' parameter.
//
//
func (r *HTPasswdUserUpdateServerRequest) Body() *HTPasswdUser {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *HTPasswdUserUpdateServerRequest) GetBody() (value *HTPasswdUser, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// HTPasswdUserUpdateServerResponse is the response for the 'update' method.
type HTPasswdUserUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *HTPasswdUser
}

// Body sets the value of the 'body' parameter.
//
//
func (r *HTPasswdUserUpdateServerResponse) Body(value *HTPasswdUser) *HTPasswdUserUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *HTPasswdUserUpdateServerResponse) Status(value int) *HTPasswdUserUpdateServerResponse {
	r.status = value
	return r
}

// dispatchHTPasswdUser navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchHTPasswdUser(w http.ResponseWriter, r *http.Request, server HTPasswdUserServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptHTPasswdUserDeleteRequest(w, r, server)
			return
		case "GET":
			adaptHTPasswdUserGetRequest(w, r, server)
			return
		case "PATCH":
			adaptHTPasswdUserUpdateRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		errors.SendNotFound(w, r)
		return
	}
}

// adaptHTPasswdUserDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptHTPasswdUserDeleteRequest(w http.ResponseWriter, r *http.Request, server HTPasswdUserServer) {
	request := &HTPasswdUserDeleteServerRequest{}
	err := readHTPasswdUserDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &HTPasswdUserDeleteServerResponse{}
	response.status = 204
	err = server.Delete(r.Context(), request, response)
	if err != nil {
		glog.Errorf(
			"Can't process request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	err = writeHTPasswdUserDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptHTPasswdUserGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptHTPasswdUserGetRequest(w http.ResponseWriter, r *http.Request, server HTPasswdUserServer) {
	request := &HTPasswdUserGetServerRequest{}
	err := readHTPasswdUserGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &HTPasswdUserGetServerResponse{}
	response.status = 200
	err = server.Get(r.Context(), request, response)
	if err != nil {
		glog.Errorf(
			"Can't process request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	err = writeHTPasswdUserGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptHTPasswdUserUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptHTPasswdUserUpdateRequest(w http.ResponseWriter, r *http.Request, server HTPasswdUserServer) {
	request := &HTPasswdUserUpdateServerRequest{}
	err := readHTPasswdUserUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &HTPasswdUserUpdateServerResponse{}
	response.status = 200
	err = server.Update(r.Context(), request, response)
	if err != nil {
		glog.Errorf(
			"Can't process request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	err = writeHTPasswdUserUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
