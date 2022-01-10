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

// StatusServer represents the interface the manages the 'status' resource.
type StatusServer interface {

	// Delete handles a request for the 'delete' method.
	//
	//
	Delete(ctx context.Context, request *StatusDeleteServerRequest, response *StatusDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	//
	Get(ctx context.Context, request *StatusGetServerRequest, response *StatusGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	//
	Update(ctx context.Context, request *StatusUpdateServerRequest, response *StatusUpdateServerResponse) error
}

// StatusDeleteServerRequest is the request for the 'delete' method.
type StatusDeleteServerRequest struct {
}

// StatusDeleteServerResponse is the response for the 'delete' method.
type StatusDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *StatusDeleteServerResponse) Status(value int) *StatusDeleteServerResponse {
	r.status = value
	return r
}

// StatusGetServerRequest is the request for the 'get' method.
type StatusGetServerRequest struct {
}

// StatusGetServerResponse is the response for the 'get' method.
type StatusGetServerResponse struct {
	status int
	err    *errors.Error
	body   *Status
}

// Body sets the value of the 'body' parameter.
//
//
func (r *StatusGetServerResponse) Body(value *Status) *StatusGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *StatusGetServerResponse) Status(value int) *StatusGetServerResponse {
	r.status = value
	return r
}

// StatusUpdateServerRequest is the request for the 'update' method.
type StatusUpdateServerRequest struct {
	body *Status
}

// Body returns the value of the 'body' parameter.
//
//
func (r *StatusUpdateServerRequest) Body() *Status {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusUpdateServerRequest) GetBody() (value *Status, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// StatusUpdateServerResponse is the response for the 'update' method.
type StatusUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *Status
}

// Body sets the value of the 'body' parameter.
//
//
func (r *StatusUpdateServerResponse) Body(value *Status) *StatusUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *StatusUpdateServerResponse) Status(value int) *StatusUpdateServerResponse {
	r.status = value
	return r
}

// dispatchStatus navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchStatus(w http.ResponseWriter, r *http.Request, server StatusServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptStatusDeleteRequest(w, r, server)
			return
		case "GET":
			adaptStatusGetRequest(w, r, server)
			return
		case "PATCH":
			adaptStatusUpdateRequest(w, r, server)
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

// adaptStatusDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusDeleteRequest(w http.ResponseWriter, r *http.Request, server StatusServer) {
	request := &StatusDeleteServerRequest{}
	err := readStatusDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusDeleteServerResponse{}
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
	err = writeStatusDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptStatusGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusGetRequest(w http.ResponseWriter, r *http.Request, server StatusServer) {
	request := &StatusGetServerRequest{}
	err := readStatusGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusGetServerResponse{}
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
	err = writeStatusGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptStatusUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusUpdateRequest(w http.ResponseWriter, r *http.Request, server StatusServer) {
	request := &StatusUpdateServerRequest{}
	err := readStatusUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusUpdateServerResponse{}
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
	err = writeStatusUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
