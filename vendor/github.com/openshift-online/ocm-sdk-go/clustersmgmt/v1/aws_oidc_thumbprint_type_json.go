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
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// MarshalAwsOidcThumbprint writes a value of the 'aws_oidc_thumbprint' type to the given writer.
func MarshalAwsOidcThumbprint(object *AwsOidcThumbprint, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeAwsOidcThumbprint(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// writeAwsOidcThumbprint writes a value of the 'aws_oidc_thumbprint' type to the given stream.
func writeAwsOidcThumbprint(object *AwsOidcThumbprint, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = object.bitmap_&1 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("issuer_url")
		stream.WriteString(object.issuerUrl)
		count++
	}
	present_ = object.bitmap_&2 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("thumbprint")
		stream.WriteString(object.thumbprint)
	}
	stream.WriteObjectEnd()
}

// UnmarshalAwsOidcThumbprint reads a value of the 'aws_oidc_thumbprint' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalAwsOidcThumbprint(source interface{}) (object *AwsOidcThumbprint, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readAwsOidcThumbprint(iterator)
	err = iterator.Error
	return
}

// readAwsOidcThumbprint reads a value of the 'aws_oidc_thumbprint' type from the given iterator.
func readAwsOidcThumbprint(iterator *jsoniter.Iterator) *AwsOidcThumbprint {
	object := &AwsOidcThumbprint{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "issuer_url":
			value := iterator.ReadString()
			object.issuerUrl = value
			object.bitmap_ |= 1
		case "thumbprint":
			value := iterator.ReadString()
			object.thumbprint = value
			object.bitmap_ |= 2
		default:
			iterator.ReadAny()
		}
	}
	return object
}
