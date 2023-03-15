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
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// MarshalHostedOidcConfig writes a value of the 'hosted_oidc_config' type to the given writer.
func MarshalHostedOidcConfig(object *HostedOidcConfig, writer io.Writer) error {
	stream := helpers.NewStream(writer)
	writeHostedOidcConfig(object, stream)
	err := stream.Flush()
	if err != nil {
		return err
	}
	return stream.Error
}

// writeHostedOidcConfig writes a value of the 'hosted_oidc_config' type to the given stream.
func writeHostedOidcConfig(object *HostedOidcConfig, stream *jsoniter.Stream) {
	count := 0
	stream.WriteObjectStart()
	var present_ bool
	present_ = object.bitmap_&1 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("href")
		stream.WriteString(object.href)
		count++
	}
	present_ = object.bitmap_&2 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("id")
		stream.WriteString(object.id)
		count++
	}
	present_ = object.bitmap_&4 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("creation_timestamp")
		stream.WriteString((object.creationTimestamp).Format(time.RFC3339))
		count++
	}
	present_ = object.bitmap_&8 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("installer_role_arn")
		stream.WriteString(object.installerRoleArn)
		count++
	}
	present_ = object.bitmap_&16 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("oidc_endpoint_url")
		stream.WriteString(object.oidcEndpointUrl)
		count++
	}
	present_ = object.bitmap_&32 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("oidc_folder_name")
		stream.WriteString(object.oidcFolderName)
		count++
	}
	present_ = object.bitmap_&64 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("oidc_private_key_secret_arn")
		stream.WriteString(object.oidcPrivateKeySecretArn)
		count++
	}
	present_ = object.bitmap_&128 != 0
	if present_ {
		if count > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField("organization_id")
		stream.WriteString(object.organizationId)
	}
	stream.WriteObjectEnd()
}

// UnmarshalHostedOidcConfig reads a value of the 'hosted_oidc_config' type from the given
// source, which can be an slice of bytes, a string or a reader.
func UnmarshalHostedOidcConfig(source interface{}) (object *HostedOidcConfig, err error) {
	iterator, err := helpers.NewIterator(source)
	if err != nil {
		return
	}
	object = readHostedOidcConfig(iterator)
	err = iterator.Error
	return
}

// readHostedOidcConfig reads a value of the 'hosted_oidc_config' type from the given iterator.
func readHostedOidcConfig(iterator *jsoniter.Iterator) *HostedOidcConfig {
	object := &HostedOidcConfig{}
	for {
		field := iterator.ReadObject()
		if field == "" {
			break
		}
		switch field {
		case "href":
			value := iterator.ReadString()
			object.href = value
			object.bitmap_ |= 1
		case "id":
			value := iterator.ReadString()
			object.id = value
			object.bitmap_ |= 2
		case "creation_timestamp":
			text := iterator.ReadString()
			value, err := time.Parse(time.RFC3339, text)
			if err != nil {
				iterator.ReportError("", err.Error())
			}
			object.creationTimestamp = value
			object.bitmap_ |= 4
		case "installer_role_arn":
			value := iterator.ReadString()
			object.installerRoleArn = value
			object.bitmap_ |= 8
		case "oidc_endpoint_url":
			value := iterator.ReadString()
			object.oidcEndpointUrl = value
			object.bitmap_ |= 16
		case "oidc_folder_name":
			value := iterator.ReadString()
			object.oidcFolderName = value
			object.bitmap_ |= 32
		case "oidc_private_key_secret_arn":
			value := iterator.ReadString()
			object.oidcPrivateKeySecretArn = value
			object.bitmap_ |= 64
		case "organization_id":
			value := iterator.ReadString()
			object.organizationId = value
			object.bitmap_ |= 128
		default:
			iterator.ReadAny()
		}
	}
	return object
}
