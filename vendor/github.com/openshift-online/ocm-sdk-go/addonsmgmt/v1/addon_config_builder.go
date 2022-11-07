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

package v1 // github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1

// AddonConfigBuilder contains the data and logic needed to build 'addon_config' objects.
//
// Representation of an addon config.
// The attributes under it are to be used by the addon once its installed in the cluster.
type AddonConfigBuilder struct {
	bitmap_              uint32
	environmentVariables []*AddonEnvironmentVariableBuilder
	secretPropagations   []*AddonSecretPropagationBuilder
}

// NewAddonConfig creates a new builder of 'addon_config' objects.
func NewAddonConfig() *AddonConfigBuilder {
	return &AddonConfigBuilder{}
}

// Empty returns true if the builder is empty, i.e. no attribute has a value.
func (b *AddonConfigBuilder) Empty() bool {
	return b == nil || b.bitmap_ == 0
}

// EnvironmentVariables sets the value of the 'environment_variables' attribute to the given values.
//
//
func (b *AddonConfigBuilder) EnvironmentVariables(values ...*AddonEnvironmentVariableBuilder) *AddonConfigBuilder {
	b.environmentVariables = make([]*AddonEnvironmentVariableBuilder, len(values))
	copy(b.environmentVariables, values)
	b.bitmap_ |= 1
	return b
}

// SecretPropagations sets the value of the 'secret_propagations' attribute to the given values.
//
//
func (b *AddonConfigBuilder) SecretPropagations(values ...*AddonSecretPropagationBuilder) *AddonConfigBuilder {
	b.secretPropagations = make([]*AddonSecretPropagationBuilder, len(values))
	copy(b.secretPropagations, values)
	b.bitmap_ |= 2
	return b
}

// Copy copies the attributes of the given object into this builder, discarding any previous values.
func (b *AddonConfigBuilder) Copy(object *AddonConfig) *AddonConfigBuilder {
	if object == nil {
		return b
	}
	b.bitmap_ = object.bitmap_
	if object.environmentVariables != nil {
		b.environmentVariables = make([]*AddonEnvironmentVariableBuilder, len(object.environmentVariables))
		for i, v := range object.environmentVariables {
			b.environmentVariables[i] = NewAddonEnvironmentVariable().Copy(v)
		}
	} else {
		b.environmentVariables = nil
	}
	if object.secretPropagations != nil {
		b.secretPropagations = make([]*AddonSecretPropagationBuilder, len(object.secretPropagations))
		for i, v := range object.secretPropagations {
			b.secretPropagations[i] = NewAddonSecretPropagation().Copy(v)
		}
	} else {
		b.secretPropagations = nil
	}
	return b
}

// Build creates a 'addon_config' object using the configuration stored in the builder.
func (b *AddonConfigBuilder) Build() (object *AddonConfig, err error) {
	object = new(AddonConfig)
	object.bitmap_ = b.bitmap_
	if b.environmentVariables != nil {
		object.environmentVariables = make([]*AddonEnvironmentVariable, len(b.environmentVariables))
		for i, v := range b.environmentVariables {
			object.environmentVariables[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	if b.secretPropagations != nil {
		object.secretPropagations = make([]*AddonSecretPropagation, len(b.secretPropagations))
		for i, v := range b.secretPropagations {
			object.secretPropagations[i], err = v.Build()
			if err != nil {
				return
			}
		}
	}
	return
}
