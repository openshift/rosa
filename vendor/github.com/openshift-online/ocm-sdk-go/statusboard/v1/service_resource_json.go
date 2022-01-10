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
	"io"
	"net/http"
)

func readServiceDeleteRequest(request *ServiceDeleteServerRequest, r *http.Request) error {
	return nil
}
func writeServiceDeleteRequest(request *ServiceDeleteRequest, writer io.Writer) error {
	return nil
}
func readServiceDeleteResponse(response *ServiceDeleteResponse, reader io.Reader) error {
	return nil
}
func writeServiceDeleteResponse(response *ServiceDeleteServerResponse, w http.ResponseWriter) error {
	return nil
}
func readServiceGetRequest(request *ServiceGetServerRequest, r *http.Request) error {
	return nil
}
func writeServiceGetRequest(request *ServiceGetRequest, writer io.Writer) error {
	return nil
}
func readServiceGetResponse(response *ServiceGetResponse, reader io.Reader) error {
	var err error
	response.body, err = UnmarshalService(reader)
	return err
}
func writeServiceGetResponse(response *ServiceGetServerResponse, w http.ResponseWriter) error {
	return MarshalService(response.body, w)
}
func readServiceUpdateRequest(request *ServiceUpdateServerRequest, r *http.Request) error {
	var err error
	request.body, err = UnmarshalService(r.Body)
	return err
}
func writeServiceUpdateRequest(request *ServiceUpdateRequest, writer io.Writer) error {
	return MarshalService(request.body, writer)
}
func readServiceUpdateResponse(response *ServiceUpdateResponse, reader io.Reader) error {
	var err error
	response.body, err = UnmarshalService(reader)
	return err
}
func writeServiceUpdateResponse(response *ServiceUpdateServerResponse, w http.ResponseWriter) error {
	return MarshalService(response.body, w)
}
