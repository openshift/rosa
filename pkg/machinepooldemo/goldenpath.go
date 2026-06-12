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

package machinepooldemo

// Golden-path guidance shown when the user leaves the stress-test branch.
const (
	MsgGoldenPathSubnet = "This demo follows the golden path: answer No to subnet selection " +
		"so the flow continues through availability zone."
	MsgGoldenPathAutoscaling = "This demo follows the golden path: enable autoscaling (answer Yes)."
)
