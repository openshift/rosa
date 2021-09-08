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

// ArchivedClustersServer represents the interface the manages the 'archived_clusters' resource.
type ArchivedClustersServer interface {

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of archived clusters.
	List(ctx context.Context, request *ArchivedClustersListServerRequest, response *ArchivedClustersListServerResponse) error

	// Cluster returns the target 'cluster' server for the given identifier.
	//
	// Returns a reference to the service that manages an specific cluster.
	Cluster(id string) ClusterServer
}

// ArchivedClustersListServerRequest is the request for the 'list' method.
type ArchivedClustersListServerRequest struct {
	page   *int
	search *string
	size   *int
}

// Page returns the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *ArchivedClustersListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
// Index of the requested page, where one corresponds to the first page.
func (r *ArchivedClustersListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Search returns the value of the 'search' parameter.
//
// Search criteria.
//
// The syntax of this parameter is similar to the syntax of the _where_ clause of a
// SQL statement, but using the names of the attributes of the cluster instead of
// the names of the columns of a table. For example, in order to retrieve all the
// clusters with a name starting with `my` in the `us-east-1` region the value
// should be:
//
// [source,sql]
// ----
// name like 'my%' and region.id = 'us-east-1'
// ----
//
// If the parameter isn't provided, or if the value is empty, then all the
// clusters that the user has permission to see will be returned.
func (r *ArchivedClustersListServerRequest) Search() string {
	if r != nil && r.search != nil {
		return *r.search
	}
	return ""
}

// GetSearch returns the value of the 'search' parameter and
// a flag indicating if the parameter has a value.
//
// Search criteria.
//
// The syntax of this parameter is similar to the syntax of the _where_ clause of a
// SQL statement, but using the names of the attributes of the cluster instead of
// the names of the columns of a table. For example, in order to retrieve all the
// clusters with a name starting with `my` in the `us-east-1` region the value
// should be:
//
// [source,sql]
// ----
// name like 'my%' and region.id = 'us-east-1'
// ----
//
// If the parameter isn't provided, or if the value is empty, then all the
// clusters that the user has permission to see will be returned.
func (r *ArchivedClustersListServerRequest) GetSearch() (value string, ok bool) {
	ok = r != nil && r.search != nil
	if ok {
		value = *r.search
	}
	return
}

// Size returns the value of the 'size' parameter.
//
// Maximum number of items that will be contained in the returned page.
func (r *ArchivedClustersListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
// Maximum number of items that will be contained in the returned page.
func (r *ArchivedClustersListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// ArchivedClustersListServerResponse is the response for the 'list' method.
type ArchivedClustersListServerResponse struct {
	status int
	err    *errors.Error
	items  *ClusterList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
// Retrieved list of archived clusters.
func (r *ArchivedClustersListServerResponse) Items(value *ClusterList) *ArchivedClustersListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
// Index of the requested page, where one corresponds to the first page.
func (r *ArchivedClustersListServerResponse) Page(value int) *ArchivedClustersListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
// Maximum number of items that will be contained in the returned page.
func (r *ArchivedClustersListServerResponse) Size(value int) *ArchivedClustersListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
// Total number of items of the collection that match the search criteria,
// regardless of the size of the page.
func (r *ArchivedClustersListServerResponse) Total(value int) *ArchivedClustersListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *ArchivedClustersListServerResponse) Status(value int) *ArchivedClustersListServerResponse {
	r.status = value
	return r
}

// dispatchArchivedClusters navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchArchivedClusters(w http.ResponseWriter, r *http.Request, server ArchivedClustersServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "GET":
			adaptArchivedClustersListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.Cluster(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchCluster(w, r, target, segments[1:])
	}
}

// adaptArchivedClustersListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptArchivedClustersListRequest(w http.ResponseWriter, r *http.Request, server ArchivedClustersServer) {
	request := &ArchivedClustersListServerRequest{}
	err := readArchivedClustersListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ArchivedClustersListServerResponse{}
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
	err = writeArchivedClustersListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
