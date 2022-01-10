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

// ApplicationServer represents the interface the manages the 'application' resource.
type ApplicationServer interface {

	// Delete handles a request for the 'delete' method.
	//
	//
	Delete(ctx context.Context, request *ApplicationDeleteServerRequest, response *ApplicationDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	//
	Get(ctx context.Context, request *ApplicationGetServerRequest, response *ApplicationGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	//
	Update(ctx context.Context, request *ApplicationUpdateServerRequest, response *ApplicationUpdateServerResponse) error

	// Services returns the target 'services' resource.
	//
	//
	Services() ServicesServer
}

// ApplicationDeleteServerRequest is the request for the 'delete' method.
type ApplicationDeleteServerRequest struct {
}

// ApplicationDeleteServerResponse is the response for the 'delete' method.
type ApplicationDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *ApplicationDeleteServerResponse) Status(value int) *ApplicationDeleteServerResponse {
	r.status = value
	return r
}

// ApplicationGetServerRequest is the request for the 'get' method.
type ApplicationGetServerRequest struct {
}

// ApplicationGetServerResponse is the response for the 'get' method.
type ApplicationGetServerResponse struct {
	status int
	err    *errors.Error
	body   *Application
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ApplicationGetServerResponse) Body(value *Application) *ApplicationGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ApplicationGetServerResponse) Status(value int) *ApplicationGetServerResponse {
	r.status = value
	return r
}

// ApplicationUpdateServerRequest is the request for the 'update' method.
type ApplicationUpdateServerRequest struct {
	body *Application
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ApplicationUpdateServerRequest) Body() *Application {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ApplicationUpdateServerRequest) GetBody() (value *Application, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ApplicationUpdateServerResponse is the response for the 'update' method.
type ApplicationUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *Application
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ApplicationUpdateServerResponse) Body(value *Application) *ApplicationUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ApplicationUpdateServerResponse) Status(value int) *ApplicationUpdateServerResponse {
	r.status = value
	return r
}

// dispatchApplication navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchApplication(w http.ResponseWriter, r *http.Request, server ApplicationServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptApplicationDeleteRequest(w, r, server)
			return
		case "GET":
			adaptApplicationGetRequest(w, r, server)
			return
		case "PATCH":
			adaptApplicationUpdateRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	case "services":
		target := server.Services()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchServices(w, r, target, segments[1:])
	default:
		errors.SendNotFound(w, r)
		return
	}
}

// adaptApplicationDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationDeleteRequest(w http.ResponseWriter, r *http.Request, server ApplicationServer) {
	request := &ApplicationDeleteServerRequest{}
	err := readApplicationDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationDeleteServerResponse{}
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
	err = writeApplicationDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptApplicationGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationGetRequest(w http.ResponseWriter, r *http.Request, server ApplicationServer) {
	request := &ApplicationGetServerRequest{}
	err := readApplicationGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationGetServerResponse{}
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
	err = writeApplicationGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptApplicationUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptApplicationUpdateRequest(w http.ResponseWriter, r *http.Request, server ApplicationServer) {
	request := &ApplicationUpdateServerRequest{}
	err := readApplicationUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ApplicationUpdateServerResponse{}
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
	err = writeApplicationUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
