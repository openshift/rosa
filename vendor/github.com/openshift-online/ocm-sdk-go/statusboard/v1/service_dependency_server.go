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

// ServiceDependencyServer represents the interface the manages the 'service_dependency' resource.
type ServiceDependencyServer interface {

	// Delete handles a request for the 'delete' method.
	//
	//
	Delete(ctx context.Context, request *ServiceDependencyDeleteServerRequest, response *ServiceDependencyDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	//
	Get(ctx context.Context, request *ServiceDependencyGetServerRequest, response *ServiceDependencyGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	//
	Update(ctx context.Context, request *ServiceDependencyUpdateServerRequest, response *ServiceDependencyUpdateServerResponse) error
}

// ServiceDependencyDeleteServerRequest is the request for the 'delete' method.
type ServiceDependencyDeleteServerRequest struct {
}

// ServiceDependencyDeleteServerResponse is the response for the 'delete' method.
type ServiceDependencyDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *ServiceDependencyDeleteServerResponse) Status(value int) *ServiceDependencyDeleteServerResponse {
	r.status = value
	return r
}

// ServiceDependencyGetServerRequest is the request for the 'get' method.
type ServiceDependencyGetServerRequest struct {
}

// ServiceDependencyGetServerResponse is the response for the 'get' method.
type ServiceDependencyGetServerResponse struct {
	status int
	err    *errors.Error
	body   *ServiceDependency
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ServiceDependencyGetServerResponse) Body(value *ServiceDependency) *ServiceDependencyGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ServiceDependencyGetServerResponse) Status(value int) *ServiceDependencyGetServerResponse {
	r.status = value
	return r
}

// ServiceDependencyUpdateServerRequest is the request for the 'update' method.
type ServiceDependencyUpdateServerRequest struct {
	body *ServiceDependency
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ServiceDependencyUpdateServerRequest) Body() *ServiceDependency {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServiceDependencyUpdateServerRequest) GetBody() (value *ServiceDependency, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ServiceDependencyUpdateServerResponse is the response for the 'update' method.
type ServiceDependencyUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *ServiceDependency
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ServiceDependencyUpdateServerResponse) Body(value *ServiceDependency) *ServiceDependencyUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ServiceDependencyUpdateServerResponse) Status(value int) *ServiceDependencyUpdateServerResponse {
	r.status = value
	return r
}

// dispatchServiceDependency navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchServiceDependency(w http.ResponseWriter, r *http.Request, server ServiceDependencyServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptServiceDependencyDeleteRequest(w, r, server)
			return
		case "GET":
			adaptServiceDependencyGetRequest(w, r, server)
			return
		case "PATCH":
			adaptServiceDependencyUpdateRequest(w, r, server)
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

// adaptServiceDependencyDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServiceDependencyDeleteRequest(w http.ResponseWriter, r *http.Request, server ServiceDependencyServer) {
	request := &ServiceDependencyDeleteServerRequest{}
	err := readServiceDependencyDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServiceDependencyDeleteServerResponse{}
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
	err = writeServiceDependencyDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptServiceDependencyGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServiceDependencyGetRequest(w http.ResponseWriter, r *http.Request, server ServiceDependencyServer) {
	request := &ServiceDependencyGetServerRequest{}
	err := readServiceDependencyGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServiceDependencyGetServerResponse{}
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
	err = writeServiceDependencyGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptServiceDependencyUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServiceDependencyUpdateRequest(w http.ResponseWriter, r *http.Request, server ServiceDependencyServer) {
	request := &ServiceDependencyUpdateServerRequest{}
	err := readServiceDependencyUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServiceDependencyUpdateServerResponse{}
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
	err = writeServiceDependencyUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
