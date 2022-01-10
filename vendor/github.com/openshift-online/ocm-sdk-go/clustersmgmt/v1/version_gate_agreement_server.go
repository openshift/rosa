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

// VersionGateAgreementServer represents the interface the manages the 'version_gate_agreement' resource.
type VersionGateAgreementServer interface {

	// Delete handles a request for the 'delete' method.
	//
	// Deletes the version gate agreement.
	Delete(ctx context.Context, request *VersionGateAgreementDeleteServerRequest, response *VersionGateAgreementDeleteServerResponse) error

	// Get handles a request for the 'get' method.
	//
	// Retrieves the details of the version gate agreement.
	Get(ctx context.Context, request *VersionGateAgreementGetServerRequest, response *VersionGateAgreementGetServerResponse) error
}

// VersionGateAgreementDeleteServerRequest is the request for the 'delete' method.
type VersionGateAgreementDeleteServerRequest struct {
}

// VersionGateAgreementDeleteServerResponse is the response for the 'delete' method.
type VersionGateAgreementDeleteServerResponse struct {
	status int
	err    *errors.Error
}

// Status sets the status code.
func (r *VersionGateAgreementDeleteServerResponse) Status(value int) *VersionGateAgreementDeleteServerResponse {
	r.status = value
	return r
}

// VersionGateAgreementGetServerRequest is the request for the 'get' method.
type VersionGateAgreementGetServerRequest struct {
}

// VersionGateAgreementGetServerResponse is the response for the 'get' method.
type VersionGateAgreementGetServerResponse struct {
	status int
	err    *errors.Error
	body   *VersionGateAgreement
}

// Body sets the value of the 'body' parameter.
//
//
func (r *VersionGateAgreementGetServerResponse) Body(value *VersionGateAgreement) *VersionGateAgreementGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *VersionGateAgreementGetServerResponse) Status(value int) *VersionGateAgreementGetServerResponse {
	r.status = value
	return r
}

// dispatchVersionGateAgreement navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchVersionGateAgreement(w http.ResponseWriter, r *http.Request, server VersionGateAgreementServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "DELETE":
			adaptVersionGateAgreementDeleteRequest(w, r, server)
			return
		case "GET":
			adaptVersionGateAgreementGetRequest(w, r, server)
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

// adaptVersionGateAgreementDeleteRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptVersionGateAgreementDeleteRequest(w http.ResponseWriter, r *http.Request, server VersionGateAgreementServer) {
	request := &VersionGateAgreementDeleteServerRequest{}
	err := readVersionGateAgreementDeleteRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &VersionGateAgreementDeleteServerResponse{}
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
	err = writeVersionGateAgreementDeleteResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptVersionGateAgreementGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptVersionGateAgreementGetRequest(w http.ResponseWriter, r *http.Request, server VersionGateAgreementServer) {
	request := &VersionGateAgreementGetServerRequest{}
	err := readVersionGateAgreementGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &VersionGateAgreementGetServerResponse{}
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
	err = writeVersionGateAgreementGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
