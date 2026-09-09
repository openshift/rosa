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

// This file contains information about the tool.

package info

// DefaultVersion is the CLI version. For release builds it is overridden via
// -ldflags with the value derived from the git tag. The fallback here is kept
// current by an automated post-release workflow.
var DefaultVersion = "1.2.65"

// Build contains the short Git SHA of the CLI at the point it was built. Set via `-ldflags` at build time.
var Build = "local"

const DefaultUserAgent = "ROSACLI"
