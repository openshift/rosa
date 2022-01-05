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

// VersionGateAgreementsServer represents the interface the manages the 'version_gate_agreements' resource.
type VersionGateAgreementsServer interface {

	// Add handles a request for the 'add' method.
	//
	// Adds a new agreed version gate to the cluster.
	Add(ctx context.Context, request *VersionGateAgreementsAddServerRequest, response *VersionGateAgreementsAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of reasons.
	List(ctx context.Context, request *VersionGateAgreementsListServerRequest, response *VersionGateAgreementsListServerResponse) error

	// VersionGateAgreement returns the target 'version_gate_agreement' server for the given identifier.
	//
	// Reference to the service that manages a specific version gate agreement.
	VersionGateAgreement(id string) VersionGateAgreementServer
}

// VersionGateAgreementsAddServerRequest is the request for the 'add' method.
type VersionGateAgreementsAddServerRequest struct {
	body *VersionGateAgreement
}

// Body returns the value of the 'body' parameter.
//
// Details of the version gate agreement.
func (r *VersionGateAgreementsAddServerRequest) Body() *VersionGateAgreement {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
// Details of the version gate agreement.
func (r *VersionGateAgreementsAddServerRequest) GetBody() (value *VersionGateAgreement, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// VersionGateAgreementsAddServerResponse is the response for the 'add' method.
type VersionGateAgreementsAddServerResponse struct {
	status int
	err    *errors.Error
	body   *VersionGateAgreement
}

// Body sets the value of the 'body' parameter.
//
// Details of the version gate agreement.
func (r *VersionGateAgreementsAddServerResponse) Body(value *VersionGateAgreement) *VersionGateAgreementsAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *VersionGateAgreementsAddServerResponse) Status(value int) *VersionGateAgreementsAddServerResponse {
	r.status = value
	return r
}

// VersionGateAgreementsListServerRequest is the request for the 'list' method.
type VersionGateAgreementsListServerRequest struct {
	page *int
	size *int
}

// Page returns the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *VersionGateAgreementsListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
// Index of the requested page, where one corresponds to the first page.
func (r *VersionGateAgreementsListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
// Number of items contained in the returned page.
func (r *VersionGateAgreementsListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
// Number of items contained in the returned page.
func (r *VersionGateAgreementsListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// VersionGateAgreementsListServerResponse is the response for the 'list' method.
type VersionGateAgreementsListServerResponse struct {
	status int
	err    *errors.Error
	items  *VersionGateAgreementList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
// Retrieved list of version gate agreement.
func (r *VersionGateAgreementsListServerResponse) Items(value *VersionGateAgreementList) *VersionGateAgreementsListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *VersionGateAgreementsListServerResponse) Page(value int) *VersionGateAgreementsListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
// Number of items contained in the returned page.
func (r *VersionGateAgreementsListServerResponse) Size(value int) *VersionGateAgreementsListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
// Total number of items of the collection.
func (r *VersionGateAgreementsListServerResponse) Total(value int) *VersionGateAgreementsListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *VersionGateAgreementsListServerResponse) Status(value int) *VersionGateAgreementsListServerResponse {
	r.status = value
	return r
}

// dispatchVersionGateAgreements navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchVersionGateAgreements(w http.ResponseWriter, r *http.Request, server VersionGateAgreementsServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptVersionGateAgreementsAddRequest(w, r, server)
			return
		case "GET":
			adaptVersionGateAgreementsListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.VersionGateAgreement(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchVersionGateAgreement(w, r, target, segments[1:])
	}
}

// adaptVersionGateAgreementsAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptVersionGateAgreementsAddRequest(w http.ResponseWriter, r *http.Request, server VersionGateAgreementsServer) {
	request := &VersionGateAgreementsAddServerRequest{}
	err := readVersionGateAgreementsAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &VersionGateAgreementsAddServerResponse{}
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
	err = writeVersionGateAgreementsAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptVersionGateAgreementsListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptVersionGateAgreementsListRequest(w http.ResponseWriter, r *http.Request, server VersionGateAgreementsServer) {
	request := &VersionGateAgreementsListServerRequest{}
	err := readVersionGateAgreementsListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &VersionGateAgreementsListServerResponse{}
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
	err = writeVersionGateAgreementsListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
