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

func readApplicationDeleteRequest(request *ApplicationDeleteServerRequest, r *http.Request) error {
	return nil
}
func writeApplicationDeleteRequest(request *ApplicationDeleteRequest, writer io.Writer) error {
	return nil
}
func readApplicationDeleteResponse(response *ApplicationDeleteResponse, reader io.Reader) error {
	return nil
}
func writeApplicationDeleteResponse(response *ApplicationDeleteServerResponse, w http.ResponseWriter) error {
	return nil
}
func readApplicationGetRequest(request *ApplicationGetServerRequest, r *http.Request) error {
	return nil
}
func writeApplicationGetRequest(request *ApplicationGetRequest, writer io.Writer) error {
	return nil
}
func readApplicationGetResponse(response *ApplicationGetResponse, reader io.Reader) error {
	var err error
	response.body, err = UnmarshalApplication(reader)
	return err
}
func writeApplicationGetResponse(response *ApplicationGetServerResponse, w http.ResponseWriter) error {
	return MarshalApplication(response.body, w)
}
func readApplicationUpdateRequest(request *ApplicationUpdateServerRequest, r *http.Request) error {
	var err error
	request.body, err = UnmarshalApplication(r.Body)
	return err
}
func writeApplicationUpdateRequest(request *ApplicationUpdateRequest, writer io.Writer) error {
	return MarshalApplication(request.body, writer)
}
func readApplicationUpdateResponse(response *ApplicationUpdateResponse, reader io.Reader) error {
	var err error
	response.body, err = UnmarshalApplication(reader)
	return err
}
func writeApplicationUpdateResponse(response *ApplicationUpdateServerResponse, w http.ResponseWriter) error {
	return MarshalApplication(response.body, w)
}
