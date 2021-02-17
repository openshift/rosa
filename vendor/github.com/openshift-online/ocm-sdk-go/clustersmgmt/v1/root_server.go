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
	"net/http"

	"github.com/openshift-online/ocm-sdk-go/errors"
)

// Server represents the interface the manages the 'root' resource.
type Server interface {

	// AWSInfrastructureAccessRoles returns the target 'AWS_infrastructure_access_roles' resource.
	//
	// Reference to the resource that manages the collection of AWS
	// infrastructure access roles.
	AWSInfrastructureAccessRoles() AWSInfrastructureAccessRolesServer

	// Addons returns the target 'add_ons' resource.
	//
	// Reference to the resource that manages the collection of add-ons.
	Addons() AddOnsServer

	// CloudProviders returns the target 'cloud_providers' resource.
	//
	// Reference to the resource that manages the collection of cloud providers.
	CloudProviders() CloudProvidersServer

	// Clusters returns the target 'clusters' resource.
	//
	// Reference to the resource that manages the collection of clusters.
	Clusters() ClustersServer

	// Dashboards returns the target 'dashboards' resource.
	//
	// Reference to the resource that manages the collection of dashboards.
	Dashboards() DashboardsServer

	// Flavours returns the target 'flavours' resource.
	//
	// Reference to the service that manages the collection of flavours.
	Flavours() FlavoursServer

	// MachineTypes returns the target 'machine_types' resource.
	//
	// Reference to the resource that manage the collection of machine types.
	MachineTypes() MachineTypesServer

	// Products returns the target 'products' resource.
	//
	// Reference to the resource that manages the collection of products.
	Products() ProductsServer

	// ProvisionShards returns the target 'provision_shards' resource.
	//
	// Reference to the resource that manages the collection of provision shards.
	ProvisionShards() ProvisionShardsServer

	// Versions returns the target 'versions' resource.
	//
	// Reference to the resource that manage the collection of versions.
	Versions() VersionsServer
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
	case "aws_infrastructure_access_roles":
		target := server.AWSInfrastructureAccessRoles()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchAWSInfrastructureAccessRoles(w, r, target, segments[1:])
	case "addons":
		target := server.Addons()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchAddOns(w, r, target, segments[1:])
	case "cloud_providers":
		target := server.CloudProviders()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchCloudProviders(w, r, target, segments[1:])
	case "clusters":
		target := server.Clusters()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchClusters(w, r, target, segments[1:])
	case "dashboards":
		target := server.Dashboards()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchDashboards(w, r, target, segments[1:])
	case "flavours":
		target := server.Flavours()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchFlavours(w, r, target, segments[1:])
	case "machine_types":
		target := server.MachineTypes()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchMachineTypes(w, r, target, segments[1:])
	case "products":
		target := server.Products()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchProducts(w, r, target, segments[1:])
	case "provision_shards":
		target := server.ProvisionShards()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchProvisionShards(w, r, target, segments[1:])
	case "versions":
		target := server.Versions()
		if target == nil {
			errors.SendNotFound(w, r)
			return
		}
		dispatchVersions(w, r, target, segments[1:])
	default:
		errors.SendNotFound(w, r)
		return
	}
}
