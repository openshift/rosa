/*
Copyright (c) 2025 Red Hat, Inc.

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

package imagemirror

import (
	"github.com/openshift/rosa/pkg/reporter"
)

type CreateImageMirrorUserOptions struct {
	Type    string
	Source  string
	Mirrors []string
}

type CreateImageMirrorOptions struct {
	reporter reporter.Logger
	args     *CreateImageMirrorUserOptions
}

func NewCreateImageMirrorUserOptions() *CreateImageMirrorUserOptions {
	return &CreateImageMirrorUserOptions{
		Type:    "digest",
		Source:  "",
		Mirrors: []string{},
	}
}

func NewCreateImageMirrorOptions() *CreateImageMirrorOptions {
	return &CreateImageMirrorOptions{
		reporter: reporter.CreateReporter(),
		args:     NewCreateImageMirrorUserOptions(),
	}
}

func (o *CreateImageMirrorOptions) Args() *CreateImageMirrorUserOptions {
	return o.args
}
