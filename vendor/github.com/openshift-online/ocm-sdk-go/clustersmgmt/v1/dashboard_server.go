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

// DashboardServer represents the interface the manages the 'dashboard' resource.
type DashboardServer interface {

	// Get handles a request for the 'get' method.
	//
	// Retrieves the details of the dashboard.
	Get(ctx context.Context, request *DashboardGetServerRequest, response *DashboardGetServerResponse) error
}

// DashboardGetServerRequest is the request for the 'get' method.
type DashboardGetServerRequest struct {
}

// DashboardGetServerResponse is the response for the 'get' method.
type DashboardGetServerResponse struct {
	status int
	err    *errors.Error
	body   *Dashboard
}

// Body sets the value of the 'body' parameter.
//
//
func (r *DashboardGetServerResponse) Body(value *Dashboard) *DashboardGetServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *DashboardGetServerResponse) Status(value int) *DashboardGetServerResponse {
	r.status = value
	return r
}

// dispatchDashboard navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchDashboard(w http.ResponseWriter, r *http.Request, server DashboardServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "GET":
			adaptDashboardGetRequest(w, r, server)
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

// adaptDashboardGetRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptDashboardGetRequest(w http.ResponseWriter, r *http.Request, server DashboardServer) {
	request := &DashboardGetServerRequest{}
	err := readDashboardGetRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &DashboardGetServerResponse{}
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
	err = writeDashboardGetResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
