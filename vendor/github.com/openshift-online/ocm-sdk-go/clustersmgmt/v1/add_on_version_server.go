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

// AddOnVersionServer represents the interface the manages the 'add_on_version' resource.
type AddOnVersionServer interface {

	// Delete handles a request for the 'delete' method.
	//
	// Deletes the add-on version.
	Delete(ctx context.Context, request *AddOnVersionDeleteServerRequest, response *AddOnVersionDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	// Retrieves the details of the add-on version.
	Get(ctx context.Context, request *AddOnVersionGetServerRequest, response *AddOnVersionGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	// Updates the add-on version.
	Update(ctx context.Context, request *AddOnVersionUpdateServerRequest, response *AddOnVersionUpdateServerResponse) error
}

// AddOnVersionDeleteServerRequest is the request for the 'delete' method.
type AddOnVersionDeleteServerRequest struct {
}

// AddOnVersionDeleteServerResponse is the response for the 'delete' method.
type AddOnVersionDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *AddOnVersionDeleteServerResponse) Status(value int) *AddOnVersionDeleteServerResponse {
	r.status = value
	return r
}

// AddOnVersionGetServerRequest is the request for the 'get' method.
type AddOnVersionGetServerRequest struct {
}

// AddOnVersionGetServerResponse is the response for the 'get' method.
type AddOnVersionGetServerResponse struct {
	status int
	err    *errors.Error
	body   *AddOnVersion
}

// Body sets the value of the 'body' parameter.
//
//
func (r *AddOnVersionGetServerResponse) Body(value *AddOnVersion) *AddOnVersionGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *AddOnVersionGetServerResponse) Status(value int) *AddOnVersionGetServerResponse {
	r.status = value
	return r
}

// AddOnVersionUpdateServerRequest is the request for the 'update' method.
type AddOnVersionUpdateServerRequest struct {
	body *AddOnVersion
}

// Body returns the value of the 'body' parameter.
//
//
func (r *AddOnVersionUpdateServerRequest) Body() *AddOnVersion {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *AddOnVersionUpdateServerRequest) GetBody() (value *AddOnVersion, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// AddOnVersionUpdateServerResponse is the response for the 'update' method.
type AddOnVersionUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *AddOnVersion
}

// Body sets the value of the 'body' parameter.
//
//
func (r *AddOnVersionUpdateServerResponse) Body(value *AddOnVersion) *AddOnVersionUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *AddOnVersionUpdateServerResponse) Status(value int) *AddOnVersionUpdateServerResponse {
	r.status = value
	return r
}

// dispatchAddOnVersion navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchAddOnVersion(w http.ResponseWriter, r *http.Request, server AddOnVersionServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptAddOnVersionDeleteRequest(w, r, server)
			return
		case "GET":
			adaptAddOnVersionGetRequest(w, r, server)
			return
		case "PATCH":
			adaptAddOnVersionUpdateRequest(w, r, server)
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

// adaptAddOnVersionDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptAddOnVersionDeleteRequest(w http.ResponseWriter, r *http.Request, server AddOnVersionServer) {
	request := &AddOnVersionDeleteServerRequest{}
	err := readAddOnVersionDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &AddOnVersionDeleteServerResponse{}
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
	err = writeAddOnVersionDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptAddOnVersionGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptAddOnVersionGetRequest(w http.ResponseWriter, r *http.Request, server AddOnVersionServer) {
	request := &AddOnVersionGetServerRequest{}
	err := readAddOnVersionGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &AddOnVersionGetServerResponse{}
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
	err = writeAddOnVersionGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptAddOnVersionUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptAddOnVersionUpdateRequest(w http.ResponseWriter, r *http.Request, server AddOnVersionServer) {
	request := &AddOnVersionUpdateServerRequest{}
	err := readAddOnVersionUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &AddOnVersionUpdateServerResponse{}
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
	err = writeAddOnVersionUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
