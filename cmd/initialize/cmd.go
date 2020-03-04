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

package initialize

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gitlab.cee.redhat.com/service/moactl/pkg/aws"
)

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Applies templates to support Managed OpenShift on AWS clusters",
	Long:  "Applies templates to support Managed OpenShift on AWS clusters",
	Run:   run,
}

func run(cmd *cobra.Command, argv []string) {
	client, err := aws.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] Error creating AWS client: %s\n", err)
		os.Exit(1)
	}

	ok, err := client.ValidateCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] Error validating AWS credentials: %s\n", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "[ERR] AWS credentials are invalid\n")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "[SUCCESS] AWS credentials are valid!\n")

	ok, err = client.EnsureAdminUser()
	if !ok || err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] Error ensuring admin user: %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "[SUCCESS] Admin user is valid!\n")

	ok, err = client.ValidateSCP()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] Error validating SCP policies: %s\n", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "[ERR] SCP policies are invalid\n")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "[SUCCESS] SCP policies are valid!\n")

	ok, err = client.EnsurePermissions()
	if !ok || err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] Error ensuring account permissions: %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "[SUCCESS] Account permissions are valid!\n")
}
