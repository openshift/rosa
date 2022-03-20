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
	"net/http"

	"github.com/openshift-online/ocm-sdk-go/errors"
)

// Server represents the interface the manages the 'root' resource.
type Server interface {

	// ApplicationDependencies returns the target 'application_dependencies' resource.
	//
	//
	ApplicationDependencies() ApplicationDependenciesServer

	// Applications returns the target 'applications' resource.
	//
	//
	Applications() ApplicationsServer

	// PeerDependencies returns the target 'peer_dependencies' resource.
	//
	//
	PeerDependencies() PeerDependenciesServer

	// Products returns the target 'products' resource.
	//
	//
	Products() ProductsServer

	// Services returns the target 'services' resource.
	//
	//
	Services() ServicesServer

	// StatusUpdates returns the target 'statuses' resource.
	//
	//
	StatusUpdates() StatusesServer

	// Statuses returns the target 'statuses' resource.
	//
	//
	Statuses() StatusesServer
}

// Dispatch navigates the servers tree rooted at the given server
// till it finds one that matches the given set of path segments, and then invokes
// the corresponding server.
func Dispatch(w http.ResponseWriter, r *http.Request, server Server, segments []string) {
	if len(segments) == 0 {
		switch r.Method {
		default:
			errors.SendMethodNotAllowed(w, r)
			return
		}
	}
	switch segments[0] {
	case "application_dependencies":
		target := server.ApplicationDependencies()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchApplicationDependencies(w, r, target, segments[1:])
	case "applications":
		target := server.Applications()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchApplications(w, r, target, segments[1:])
	case "peer_dependencies":
		target := server.PeerDependencies()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchPeerDependencies(w, r, target, segments[1:])
	case "products":
		target := server.Products()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchProducts(w, r, target, segments[1:])
	case "services":
		target := server.Services()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchServices(w, r, target, segments[1:])
	case "status_updates":
		target := server.StatusUpdates()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchStatuses(w, r, target, segments[1:])
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
