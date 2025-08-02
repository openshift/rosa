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
