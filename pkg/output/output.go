/*
Copyright (c) 2021 Red Hat, Inc.

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

// This file contains functions used to implement the '--output' command line option.

package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/ghodss/yaml"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/aws"
	"gitlab.com/c0b/go-ordered-json"
)

// When ocm-sdk-go encounters an empty resource list, it marshals it as a
// string that represents an empty JSON array with newline and spaces in between:
// '[', '\n', ' ', ' ', '\n', ']'. This byte-array allows us to compare that so
// that the output can be shown correctly.
var emptyBuffer = []byte{91, 10, 32, 32, 10, 93}

func Print(resource interface{}) error {
	var b bytes.Buffer
	switch reflect.TypeOf(resource).String() {
	case "[]*v1.CloudRegion":

		if cloudRegions, ok := resource.([]*cmv1.CloudRegion); ok {
			cmv1.MarshalCloudRegionList(cloudRegions, &b)
		}
	case "*v1.Cluster":
		if cluster, ok := resource.(*cmv1.Cluster); ok {
			cmv1.MarshalCluster(cluster, &b)
		}
	case "[]*v1.Cluster":
		if clusters, ok := resource.([]*cmv1.Cluster); ok {
			cmv1.MarshalClusterList(clusters, &b)
		}
	case "[]*v1.IdentityProvider":
		if idps, ok := resource.([]*cmv1.IdentityProvider); ok {
			cmv1.MarshalIdentityProviderList(idps, &b)
		}
	case "[]*v1.Ingress":
		if ingresses, ok := resource.([]*cmv1.Ingress); ok {
			cmv1.MarshalIngressList(ingresses, &b)
		}
	case "[]*v1.MachinePool":
		if machinePools, ok := resource.([]*cmv1.MachinePool); ok {
			cmv1.MarshalMachinePoolList(machinePools, &b)
		}
	case "[]*v1.MachineType":
		if machineTypes, ok := resource.([]*cmv1.MachineType); ok {
			cmv1.MarshalMachineTypeList(machineTypes, &b)
		}
	case "[]*v1.Version":
		if versions, ok := resource.([]*cmv1.Version); ok {
			cmv1.MarshalVersionList(versions, &b)
		}
	case "[]aws.Role":
		{
			if roles, ok := resource.([]aws.Role); ok {
				err := aws.MarshalRoles(roles, &b)
				if err != nil {
					return err
				}
			}
		}
	}
	// Verify if the resource is an empty string and ensure that the JSON
	// representation looks correct for STDOUT.
	if b.String() == string(emptyBuffer) {
		b = *bytes.NewBufferString("[]")
	}
	str, err := parseResource(b)
	if err != nil {
		return err
	}
	fmt.Print(str)
	return nil
}

func parseResource(body bytes.Buffer) (string, error) {
	switch o {
	case "json":
		var out bytes.Buffer
		prettifyJSON(&out, body.Bytes())
		return out.String(), nil
	case "yaml":
		out, err := yaml.JSONToYAML(body.Bytes())
		if err != nil {
			return "", err
		}
		return string(out), nil
	default:
		return "", fmt.Errorf("Unknown format '%s'. Valid formats are %s", o, formats)
	}
}

func prettifyJSON(stream io.Writer, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	data := ordered.NewOrderedMap()
	err := json.Unmarshal(body, data)
	if err != nil {
		return dumpBytes(stream, body)
	}
	return dumpJSON(stream, data)
}

func dumpBytes(stream io.Writer, data []byte) error {
	_, err := stream.Write(data)
	if err != nil {
		return err
	}
	_, err = stream.Write([]byte("\n"))
	return err
}

func dumpJSON(stream io.Writer, data *ordered.OrderedMap) error {
	encoder := json.NewEncoder(stream)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
