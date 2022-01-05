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

// ProductsServer represents the interface the manages the 'products' resource.
type ProductsServer interface {

	// Add handles a request for the 'add' method.
	//
	//
	Add(ctx context.Context, request *ProductsAddServerRequest, response *ProductsAddServerResponse) error

	// List handles a request for the 'list' method.
	//
	// Retrieves the list of products.
	List(ctx context.Context, request *ProductsListServerRequest, response *ProductsListServerResponse) error

	// Product returns the target 'product' server for the given identifier.
	//
	//
	Product(id string) ProductServer
}

// ProductsAddServerRequest is the request for the 'add' method.
type ProductsAddServerRequest struct {
	body *Product
}

// Body returns the value of the 'body' parameter.
//
//
func (r *ProductsAddServerRequest) Body() *Product {
	if r == nil {
		return nil
	}
	return r.body
}

// GetBody returns the value of the 'body' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ProductsAddServerRequest) GetBody() (value *Product, ok bool) {
	ok = r != nil && r.body != nil
	if ok {
		value = r.body
	}
	return
}

// ProductsAddServerResponse is the response for the 'add' method.
type ProductsAddServerResponse struct {
	status int
	err    *errors.Error
	body   *Product
}

// Body sets the value of the 'body' parameter.
//
//
func (r *ProductsAddServerResponse) Body(value *Product) *ProductsAddServerResponse {
	r.body = value
	return r
}

// Status sets the status code.
func (r *ProductsAddServerResponse) Status(value int) *ProductsAddServerResponse {
	r.status = value
	return r
}

// ProductsListServerRequest is the request for the 'list' method.
type ProductsListServerRequest struct {
	fullname *string
	orderBy  *string
	page     *int
	size     *int
}

// Fullname returns the value of the 'fullname' parameter.
//
//
func (r *ProductsListServerRequest) Fullname() string {
	if r != nil && r.fullname != nil {
		return *r.fullname
	}
	return ""
}

// GetFullname returns the value of the 'fullname' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ProductsListServerRequest) GetFullname() (value string, ok bool) {
	ok = r != nil && r.fullname != nil
	if ok {
		value = *r.fullname
	}
	return
}

// OrderBy returns the value of the 'order_by' parameter.
//
//
func (r *ProductsListServerRequest) OrderBy() string {
	if r != nil && r.orderBy != nil {
		return *r.orderBy
	}
	return ""
}

// GetOrderBy returns the value of the 'order_by' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ProductsListServerRequest) GetOrderBy() (value string, ok bool) {
	ok = r != nil && r.orderBy != nil
	if ok {
		value = *r.orderBy
	}
	return
}

// Page returns the value of the 'page' parameter.
//
//
func (r *ProductsListServerRequest) Page() int {
	if r != nil && r.page != nil {
		return *r.page
	}
	return 0
}

// GetPage returns the value of the 'page' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ProductsListServerRequest) GetPage() (value int, ok bool) {
	ok = r != nil && r.page != nil
	if ok {
		value = *r.page
	}
	return
}

// Size returns the value of the 'size' parameter.
//
//
func (r *ProductsListServerRequest) Size() int {
	if r != nil && r.size != nil {
		return *r.size
	}
	return 0
}

// GetSize returns the value of the 'size' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *ProductsListServerRequest) GetSize() (value int, ok bool) {
	ok = r != nil && r.size != nil
	if ok {
		value = *r.size
	}
	return
}

// ProductsListServerResponse is the response for the 'list' method.
type ProductsListServerResponse struct {
	status int
	err    *errors.Error
	items  *ProductList
	page   *int
	size   *int
	total  *int
}

// Items sets the value of the 'items' parameter.
//
//
func (r *ProductsListServerResponse) Items(value *ProductList) *ProductsListServerResponse {
	r.items = value
	return r
}

// Page sets the value of the 'page' parameter.
//
//
func (r *ProductsListServerResponse) Page(value int) *ProductsListServerResponse {
	r.page = &value
	return r
}

// Size sets the value of the 'size' parameter.
//
//
func (r *ProductsListServerResponse) Size(value int) *ProductsListServerResponse {
	r.size = &value
	return r
}

// Total sets the value of the 'total' parameter.
//
//
func (r *ProductsListServerResponse) Total(value int) *ProductsListServerResponse {
	r.total = &value
	return r
}

// Status sets the status code.
func (r *ProductsListServerResponse) Status(value int) *ProductsListServerResponse {
	r.status = value
	return r
}

// dispatchProducts navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func dispatchProducts(w http.ResponseWriter, r *http.Request, server ProductsServer, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		case "POST":
			adaptProductsAddRequest(w, r, server)
			return
		case "GET":
			adaptProductsListRequest(w, r, server)
			return
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	default:
		target := server.Product(segments[0])
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchProduct(w, r, target, segments[1:])
	}
}

// adaptProductsAddRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptProductsAddRequest(w http.ResponseWriter, r *http.Request, server ProductsServer) {
	request := &ProductsAddServerRequest{}
	err := readProductsAddRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ProductsAddServerResponse{}
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
	err = writeProductsAddResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}

// adaptProductsListRequest translates the given HTTP request into a call to
// the corresponding method of the given server. Then it translates the
// results returned by that method into an HTTP response.
func adaptProductsListRequest(w http.ResponseWriter, r *http.Request, server ProductsServer) {
	request := &ProductsListServerRequest{}
	err := readProductsListRequest(request, r)
	if err != nil {
		glog.Errorf(
			"Can't read request for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		errors.SendInternalServerError(w, r)
		return
	}
	response := &ProductsListServerResponse{}
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
	err = writeProductsListResponse(response, w)
	if err != nil {
		glog.Errorf(
			"Can't write response for method '%s' and path '%s': %v",
			r.Method, r.URL.Path, err,
		)
		return
	}
}
