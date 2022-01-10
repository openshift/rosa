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

// PeerDependencyServer represents the interface the manages the 'peer_dependency' resource.
type PeerDependencyServer interface {

	// Delete handles a request for the 'delete' method.
	//
	//
	Delete(ctx context.Context, request *PeerDependencyDeleteServerRequest, response *PeerDependencyDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	//
	Get(ctx context.Context, request *PeerDependencyGetServerRequest, response *PeerDependencyGetServerResponse) error

	// Update handles a request for the 'update' method.
	//
	//
	Update(ctx context.Context, request *PeerDependencyUpdateServerRequest, response *PeerDependencyUpdateServerResponse) error
}

// PeerDependencyDeleteServerRequest is the request for the 'delete' method.
type PeerDependencyDeleteServerRequest struct {
}

// PeerDependencyDeleteServerResponse is the response for the 'delete' method.
type PeerDependencyDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *PeerDependencyDeleteServerResponse) Status(value int) *PeerDependencyDeleteServerResponse {
	r.status = value
	return r
}

// PeerDependencyGetServerRequest is the request for the 'get' method.
type PeerDependencyGetServerRequest struct {
}

// PeerDependencyGetServerResponse is the response for the 'get' method.
type PeerDependencyGetServerResponse struct {
	status int
	err    *errors.Error
	body   *Service
}

// Body sets the value of the 'body' parameter.
//
//
func (r *PeerDependencyGetServerResponse) Body(value *Service) *PeerDependencyGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *PeerDependencyGetServerResponse) Status(value int) *PeerDependencyGetServerResponse {
	r.status = value
	return r
}

// PeerDependencyUpdateServerRequest is the request for the 'update' method.
type PeerDependencyUpdateServerRequest struct {
	body *PeerDependency
}

// Body returns the value of the 'body' parameter.
//
//
func (r *PeerDependencyUpdateServerRequest) Body() *PeerDependency {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *PeerDependencyUpdateServerRequest) GetBody() (value *PeerDependency, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// PeerDependencyUpdateServerResponse is the response for the 'update' method.
type PeerDependencyUpdateServerResponse struct {
	status int
	err    *errors.Error
	body   *PeerDependency
}

// Body sets the value of the 'body' parameter.
//
//
func (r *PeerDependencyUpdateServerResponse) Body(value *PeerDependency) *PeerDependencyUpdateServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *PeerDependencyUpdateServerResponse) Status(value int) *PeerDependencyUpdateServerResponse {
	r.status = value
	return r
}

// dispatchPeerDependency navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchPeerDependency(w http.ResponseWriter, r *http.Request, server PeerDependencyServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptPeerDependencyDeleteRequest(w, r, server)
			return
		case "GET":
			adaptPeerDependencyGetRequest(w, r, server)
			return
		case "PATCH":
			adaptPeerDependencyUpdateRequest(w, r, server)
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

// adaptPeerDependencyDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptPeerDependencyDeleteRequest(w http.ResponseWriter, r *http.Request, server PeerDependencyServer) {
	request := &PeerDependencyDeleteServerRequest{}
	err := readPeerDependencyDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &PeerDependencyDeleteServerResponse{}
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
	err = writePeerDependencyDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptPeerDependencyGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptPeerDependencyGetRequest(w http.ResponseWriter, r *http.Request, server PeerDependencyServer) {
	request := &PeerDependencyGetServerRequest{}
	err := readPeerDependencyGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &PeerDependencyGetServerResponse{}
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
	err = writePeerDependencyGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptPeerDependencyUpdateRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptPeerDependencyUpdateRequest(w http.ResponseWriter, r *http.Request, server PeerDependencyServer) {
	request := &PeerDependencyUpdateServerRequest{}
	err := readPeerDependencyUpdateRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &PeerDependencyUpdateServerResponse{}
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
	err = writePeerDependencyUpdateResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
