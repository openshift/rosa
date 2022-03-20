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

// StatusUpdateServer represents the interface the manages the 'status_update' resource.
type StatusUpdateServer interface {

	// Delete handles a request for the 'delete' method.
	//
	//
	Delete(ctx context.Context, request *StatusUpdateDeleteServerRequest, response *StatusUpdateDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	//
	Get(ctx context.Context, request *StatusUpdateGetServerRequest, response *StatusUpdateGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	//
	Update(ctx context.Context, request *StatusUpdateUpdateServerRequest, response *StatusUpdateUpdateServerResponse) error
}

// StatusUpdateDeleteServerRequest is the request for the 'delete' method.
type StatusUpdateDeleteServerRequest struct {
}

// StatusUpdateDeleteServerResponse is the response for the 'delete' method.
type StatusUpdateDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *StatusUpdateDeleteServerResponse) Status(value int) *StatusUpdateDeleteServerResponse {
	r.status = value
	return r
}

// StatusUpdateGetServerRequest is the request for the 'get' method.
type StatusUpdateGetServerRequest struct {
}

// StatusUpdateGetServerResponse is the response for the 'get' method.
type StatusUpdateGetServerResponse struct {
	status int
	err    *errors.Error
	body   *Status
}

// Body sets the value of the 'body' parameter.
//
//
func (r *StatusUpdateGetServerResponse) Body(value *Status) *StatusUpdateGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *StatusUpdateGetServerResponse) Status(value int) *StatusUpdateGetServerResponse {
	r.status = value
	return r
}

// StatusUpdateUpdateServerRequest is the request for the 'update' method.
type StatusUpdateUpdateServerRequest struct {
	body *Status
}

// Body returns the value of the 'body' parameter.
//
//
func (r *StatusUpdateUpdateServerRequest) Body() *Status {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *StatusUpdateUpdateServerRequest) GetBody() (value *Status, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// StatusUpdateUpdateServerResponse is the response for the 'update' method.
type StatusUpdateUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *Status
}

// Body sets the value of the 'body' parameter.
//
//
func (r *StatusUpdateUpdateServerResponse) Body(value *Status) *StatusUpdateUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *StatusUpdateUpdateServerResponse) Status(value int) *StatusUpdateUpdateServerResponse {
	r.status = value
	return r
}

// dispatchStatusUpdate navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchStatusUpdate(w http.ResponseWriter, r *http.Request, server StatusUpdateServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptStatusUpdateDeleteRequest(w, r, server)
			return
		case "GET":
			adaptStatusUpdateGetRequest(w, r, server)
			return
		case "PATCH":
			adaptStatusUpdateUpdateRequest(w, r, server)
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

// adaptStatusUpdateDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusUpdateDeleteRequest(w http.ResponseWriter, r *http.Request, server StatusUpdateServer) {
	request := &StatusUpdateDeleteServerRequest{}
	err := readStatusUpdateDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusUpdateDeleteServerResponse{}
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
	err = writeStatusUpdateDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptStatusUpdateGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusUpdateGetRequest(w http.ResponseWriter, r *http.Request, server StatusUpdateServer) {
	request := &StatusUpdateGetServerRequest{}
	err := readStatusUpdateGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusUpdateGetServerResponse{}
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
	err = writeStatusUpdateGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptStatusUpdateUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptStatusUpdateUpdateRequest(w http.ResponseWriter, r *http.Request, server StatusUpdateServer) {
	request := &StatusUpdateUpdateServerRequest{}
	err := readStatusUpdateUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &StatusUpdateUpdateServerResponse{}
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
	err = writeStatusUpdateUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
