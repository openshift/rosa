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

// PeerDependenciesServer represents the interface the manages the 'peer_dependencies' resource.
type PeerDependenciesServer interface {

	// Add handles a request for the 'add' method.
	//
	//
	Add(ctx context.Context, request *PeerDependenciesAddServerRequest, response *PeerDependenciesAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of peer dependencies.
	List(ctx context.Context, request *PeerDependenciesListServerRequest, response *PeerDependenciesListServerResponse) error

	// PeerDependency returns the target 'peer_dependency' server for the given identifier.
	//
	//
	PeerDependency(id string) PeerDependencyServer
}

// PeerDependenciesAddServerRequest is the request for the 'add' method.
type PeerDependenciesAddServerRequest struct {
	body *PeerDependency
}

// Body returns the value of the 'body' parameter.
//
//
func (r *PeerDependenciesAddServerRequest) Body() *PeerDependency {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *PeerDependenciesAddServerRequest) GetBody() (value *PeerDependency, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// PeerDependenciesAddServerResponse is the response for the 'add' method.
type PeerDependenciesAddServerResponse struct {
	status int
	err    *errors.Error
	body   *PeerDependency
}

// Body sets the value of the 'body' parameter.
//
//
func (r *PeerDependenciesAddServerResponse) Body(value *PeerDependency) *PeerDependenciesAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *PeerDependenciesAddServerResponse) Status(value int) *PeerDependenciesAddServerResponse {
	r.status = value
	return r
}

// PeerDependenciesListServerRequest is the request for the 'list' method.
type PeerDependenciesListServerRequest struct {
	orderBy *string
	page    *int
	size    *int
}

// OrderBy returns the value of the 'order_by' parameter.
//
//
func (r *PeerDependenciesListServerRequest) OrderBy() string {
	if r != nil && r.orderBy != nil {
		return *r.orderBy
	}
	return ""
}

// GetOrderBy returns the value of the 'order_by' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *PeerDependenciesListServerRequest) GetOrderBy() (value string, ok bool) {
	ok = r != nil && r.orderBy != nil
	if ok {
		value = *r.orderBy
	}
	return
}

// Page returns the value of the 'page' parameter.
//
//
func (r *PeerDependenciesListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *PeerDependenciesListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
//
func (r *PeerDependenciesListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *PeerDependenciesListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// PeerDependenciesListServerResponse is the response for the 'list' method.
type PeerDependenciesListServerResponse struct {
	status int
	err    *errors.Error
	items  *PeerDependencyList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
//
func (r *PeerDependenciesListServerResponse) Items(value *PeerDependencyList) *PeerDependenciesListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
//
func (r *PeerDependenciesListServerResponse) Page(value int) *PeerDependenciesListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
//
func (r *PeerDependenciesListServerResponse) Size(value int) *PeerDependenciesListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
//
func (r *PeerDependenciesListServerResponse) Total(value int) *PeerDependenciesListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *PeerDependenciesListServerResponse) Status(value int) *PeerDependenciesListServerResponse {
	r.status = value
	return r
}

// dispatchPeerDependencies navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchPeerDependencies(w http.ResponseWriter, r *http.Request, server PeerDependenciesServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptPeerDependenciesAddRequest(w, r, server)
			return
		case "GET":
			adaptPeerDependenciesListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.PeerDependency(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchPeerDependency(w, r, target, segments[1:])
	}
}

// adaptPeerDependenciesAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptPeerDependenciesAddRequest(w http.ResponseWriter, r *http.Request, server PeerDependenciesServer) {
	request := &PeerDependenciesAddServerRequest{}
	err := readPeerDependenciesAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &PeerDependenciesAddServerResponse{}
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
	err = writePeerDependenciesAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptPeerDependenciesListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptPeerDependenciesListRequest(w http.ResponseWriter, r *http.Request, server PeerDependenciesServer) {
	request := &PeerDependenciesListServerRequest{}
	err := readPeerDependenciesListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &PeerDependenciesListServerResponse{}
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
	err = writePeerDependenciesListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
