/*
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

package iamserviceaccount

import (
	iamServiceAccountOptions "github.com/openshift/rosa/pkg/options/iamserviceaccount"
	"github.com/openshift/rosa/pkg/reporter"
)

type CreateIamServiceAccountOptions struct {
	reporter reporter.Logger

	args *iamServiceAccountOptions.CreateIamServiceAccountUserOptions
}

func NewCreateIamServiceAccountUserOptions() *iamServiceAccountOptions.CreateIamServiceAccountUserOptions {
	return &iamServiceAccountOptions.CreateIamServiceAccountUserOptions{
		Namespace: "default",
		Path:      "/",
	}
}

func NewCreateIamServiceAccountOptions() *CreateIamServiceAccountOptions {
	return &CreateIamServiceAccountOptions{
		reporter: reporter.CreateReporter(),
		args:     &iamServiceAccountOptions.CreateIamServiceAccountUserOptions{},
	}
}

func (c *CreateIamServiceAccountOptions) IamServiceAccount() *iamServiceAccountOptions.CreateIamServiceAccountUserOptions {
	return c.args
}
