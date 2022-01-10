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

// ServiceServer represents the interface the manages the 'service' resource.
type ServiceServer interface {

	// Delete handles a request for the 'delete' method.
	//
	//
	Delete(ctx context.Context, request *ServiceDeleteServerRequest, response *ServiceDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	//
	Get(ctx context.Context, request *ServiceGetServerRequest, response *ServiceGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	//
	Update(ctx context.Context, request *ServiceUpdateServerRequest, response *ServiceUpdateServerResponse) error

	// Statuses returns the target 'statuses' resource.
	//
	//
	Statuses() StatusesServer
}

// ServiceDeleteServerRequest is the request for the 'delete' method.
type ServiceDeleteServerRequest struct {
}

// ServiceDeleteServerResponse is the response for the 'delete' method.
type ServiceDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *ServiceDeleteServerResponse) Status(value int) *ServiceDeleteServerResponse {
	r.status = value
	return r
}

// ServiceGetServerRequest is the request for the 'get' method.
type ServiceGetServerRequest struct {
}

// ServiceGetServerResponse is the response for the 'get' method.
type ServiceGetServerResponse struct {
	status int
	err    *errors.Error
	body   *Service
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ServiceGetServerResponse) Body(value *Service) *ServiceGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ServiceGetServerResponse) Status(value int) *ServiceGetServerResponse {
	r.status = value
	return r
}

// ServiceUpdateServerRequest is the request for the 'update' method.
type ServiceUpdateServerRequest struct {
	body *Service
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ServiceUpdateServerRequest) Body() *Service {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ServiceUpdateServerRequest) GetBody() (value *Service, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ServiceUpdateServerResponse is the response for the 'update' method.
type ServiceUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *Service
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ServiceUpdateServerResponse) Body(value *Service) *ServiceUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ServiceUpdateServerResponse) Status(value int) *ServiceUpdateServerResponse {
	r.status = value
	return r
}

// dispatchService navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchService(w http.ResponseWriter, r *http.Request, server ServiceServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptServiceDeleteRequest(w, r, server)
			return
		case "GET":
			adaptServiceGetRequest(w, r, server)
			return
		case "PATCH":
			adaptServiceUpdateRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	case "statuses":
		target := server.Statuses()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchStatuses(w, r, target, segments[1:])
	default:
		errors.SendNotFound(w, r)
		return
	}
}

// adaptServiceDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServiceDeleteRequest(w http.ResponseWriter, r *http.Request, server ServiceServer) {
	request := &ServiceDeleteServerRequest{}
	err := readServiceDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServiceDeleteServerResponse{}
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
	err = writeServiceDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptServiceGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServiceGetRequest(w http.ResponseWriter, r *http.Request, server ServiceServer) {
	request := &ServiceGetServerRequest{}
	err := readServiceGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServiceGetServerResponse{}
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
	err = writeServiceGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptServiceUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptServiceUpdateRequest(w http.ResponseWriter, r *http.Request, server ServiceServer) {
	request := &ServiceUpdateServerRequest{}
	err := readServiceUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ServiceUpdateServerResponse{}
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
	err = writeServiceUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
