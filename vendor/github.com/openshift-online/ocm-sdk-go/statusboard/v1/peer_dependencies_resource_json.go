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

	"github.com/openshift-online/ocm-sdk-go/helpers"
)

func readPeerDependenciesAddRequest(request *PeerDependenciesAddServerRequest, r *http.Request) error {
	var err error
	request.body, err = UnmarshalPeerDependency(r.Body)
	return err
}
func writePeerDependenciesAddRequest(request *PeerDependenciesAddRequest, writer io.Writer) error {
	return MarshalPeerDependency(request.body, writer)
}
func readPeerDependenciesAddResponse(response *PeerDependenciesAddResponse, reader io.Reader) error {
	var err error
	response.body, err = UnmarshalPeerDependency(reader)
	return err
}
func writePeerDependenciesAddResponse(response *PeerDependenciesAddServerResponse, w http.ResponseWriter) error {
	return MarshalPeerDependency(response.body, w)
}
func readPeerDependenciesListRequest(request *PeerDependenciesListServerRequest, r *http.Request) error {
	var err error
	query := r.URL.Query()
	request.orderBy, err = helpers.ParseString(query, "order_by")
	if err != nil {
		return err
	}
	request.page, err = helpers.ParseInteger(query, "page")
	if err != nil {
		return err
	}
	if request.page == nil {
		request.page = helpers.NewInteger(1)
	}
	request.size, err = helpers.ParseInteger(query, "size")
	if err != nil {
		return err
	}
	if request.size == nil {
		request.size = helpers.NewInteger(100)
	}
	return nil
}
func writePeerDependenciesListRequest(request *PeerDependenciesListRequest, writer io.Writer) error {
	return nil
}
func readPeerDependenciesListResponse(response *PeerDependenciesListResponse, reader io.Reader) error {
	iterator, err := helpers.NewIterator(reader)
	if err != nil {
		return err
	}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "page":
			value := iterator.ReadInt()
			response.page = &value
		case "size":
			value := iterator.ReadInt()
			response.size = &value
		case "total":
			value := iterator.ReadInt()
			response.total = &value
		case "items":
			items := readPeerDependencyList(iterator)
			response.items = &PeerDependencyList{
				items: items,
			}
		default:
			iterator.ReadAny()
		}
	}
	return iterator.Error
}
func writePeerDependenciesListResponse(response *PeerDependenciesListServerResponse, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.status)
	stream := helpers.NewStream(w)
	stream.WriteObjectStart()
	stream.WriteObjectField("kind")
	count := 1
	stream.WriteString(PeerDependencyListKind)
	if response.items != nil && response.items.href != "" {
		stream.WriteMore()
		stream.WriteObjectField("href")
		stream.WriteString(response.items.href)
		count++
	}
	if response.page != nil {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("page")
		stream.WriteInt(*response.page)
		count++
	}
	if response.size != nil {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("size")
		stream.WriteInt(*response.size)
		count++
	}
	if response.total != nil {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("total")
		stream.WriteInt(*response.total)
		count++
	}
	if response.items != nil {
		if response.items.items != nil {
			if count > 0 {
				stream.WriteMore()
			}
			stream.WriteObjectField("items")
			writePeerDependencyList(response.items.items, stream)
			count++
		}
	}
	stream.WriteObjectEnd()
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}
