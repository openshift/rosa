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

// ApplicationDependencyServer represents the interface the manages the 'application_dependency' resource.
type ApplicationDependencyServer interface {

	// Delete handles a request for the 'delete' method.
	//
	//
	Delete(ctx context.Context, request *ApplicationDependencyDeleteServerRequest, response *ApplicationDependencyDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	//
	Get(ctx context.Context, request *ApplicationDependencyGetServerRequest, response *ApplicationDependencyGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	//
	Update(ctx context.Context, request *ApplicationDependencyUpdateServerRequest, response *ApplicationDependencyUpdateServerResponse) error
}

// ApplicationDependencyDeleteServerRequest is the request for the 'delete' method.
type ApplicationDependencyDeleteServerRequest struct {
}

// ApplicationDependencyDeleteServerResponse is the response for the 'delete' method.
type ApplicationDependencyDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *ApplicationDependencyDeleteServerResponse) Status(value int) *ApplicationDependencyDeleteServerResponse {
	r.status = value
	return r
}

// ApplicationDependencyGetServerRequest is the request for the 'get' method.
type ApplicationDependencyGetServerRequest struct {
}

// ApplicationDependencyGetServerResponse is the response for the 'get' method.
type ApplicationDependencyGetServerResponse struct {
	status int
	err    *errors.Error
	body   *ApplicationDependency
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ApplicationDependencyGetServerResponse) Body(value *ApplicationDependency) *ApplicationDependencyGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ApplicationDependencyGetServerResponse) Status(value int) *ApplicationDependencyGetServerResponse {
	r.status = value
	return r
}

// ApplicationDependencyUpdateServerRequest is the request for the 'update' method.
type ApplicationDependencyUpdateServerRequest struct {
	body *ApplicationDependency
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ApplicationDependencyUpdateServerRequest) Body() *ApplicationDependency {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationDependencyUpdateServerRequest) GetBody() (value *ApplicationDependency, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ApplicationDependencyUpdateServerResponse is the response for the 'update' method.
type ApplicationDependencyUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *ApplicationDependency
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ApplicationDependencyUpdateServerResponse) Body(value *ApplicationDependency) *ApplicationDependencyUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ApplicationDependencyUpdateServerResponse) Status(value int) *ApplicationDependencyUpdateServerResponse {
	r.status = value
	return r
}

// dispatchApplicationDependency navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchApplicationDependency(w http.ResponseWriter, r *http.Request, server ApplicationDependencyServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptApplicationDependencyDeleteRequest(w, r, server)
			return
		case "GET":
			adaptApplicationDependencyGetRequest(w, r, server)
			return
		case "PATCH":
			adaptApplicationDependencyUpdateRequest(w, r, server)
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

// adaptApplicationDependencyDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationDependencyDeleteRequest(w http.ResponseWriter, r *http.Request, server ApplicationDependencyServer) {
	request := &ApplicationDependencyDeleteServerRequest{}
	err := readApplicationDependencyDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationDependencyDeleteServerResponse{}
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
	err = writeApplicationDependencyDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptApplicationDependencyGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationDependencyGetRequest(w http.ResponseWriter, r *http.Request, server ApplicationDependencyServer) {
	request := &ApplicationDependencyGetServerRequest{}
	err := readApplicationDependencyGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationDependencyGetServerResponse{}
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
	err = writeApplicationDependencyGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptApplicationDependencyUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationDependencyUpdateRequest(w http.ResponseWriter, r *http.Request, server ApplicationDependencyServer) {
	request := &ApplicationDependencyUpdateServerRequest{}
	err := readApplicationDependencyUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationDependencyUpdateServerResponse{}
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
	err = writeApplicationDependencyUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
