/*
Copyright (c) 2022 Red Hat, Inc.

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

package idp

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const ClusterAdminUsername = "cluster-admin"

func createHTPasswdIDP(cmd *cobra.Command,
	cluster *cmv1.Cluster,
	clusterKey string,
	idpName string,
	r *rosa.Runtime) {
	var err error
	username := args.htpasswdUsername
	password := args.htpasswdPassword

	// Choose which way to create the IDP according to whether it already has an admin or not.
	htpasswdIDP, userList := FindExistingHTPasswdIDP(cluster, r)
	if htpasswdIDP != nil {
		// if existing idp has any users other than `cluster-admin`, then it was created as a proper idp
		// and not as an admin container during `rosa create admin`. A cluster may only have one
		// htpasswd idp so `create idp` should not continue.
		containsAdminOnly := HasClusterAdmin(userList) && userList.Len() == 1
		if !containsAdminOnly {
			r.Reporter.Errorf(
				"Cluster '%s' already has an HTPasswd IDP named '%s'. "+
					"Clusters may only have 1 HTPasswd IDP.", clusterKey, htpasswdIDP.Name())
			os.Exit(1)
		}

		idp, ok := htpasswdIDP.GetHtpasswd()
		if !ok {
			r.Reporter.Errorf(
				"Failed to get htpasswd idp of cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
		if idp.Username() != "" {
			r.Reporter.Errorf("Users can't be added to a single user HTPasswd IDP. Delete the IDP and recreate " +
				"it as a multi user HTPasswd IDP")
			return
		}

		// Existing IDP contains only admin. Add new user to it
		r.Reporter.Infof("Cluster already has an HTPasswd IDP named '%s', new users will be added to it.",
			htpasswdIDP.Name())
		if username == "" || password == "" {
			r.Reporter.Infof("At least one user is required to create the IDP.")
			username, password = getUserDetails(cmd, r)
		}
		err = r.OCMClient.AddHTPasswdUser(username, password, cluster.ID(), htpasswdIDP.ID())
		if err != nil {
			r.Reporter.Errorf(
				"Failed to add a user to the HTPasswd IDP of cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
		r.Reporter.Infof("User '%s' added", username)
	} else {
		// HTPasswd IDP does not exist - create it
		if username == "" || password == "" {
			r.Reporter.Infof("At least one user is required to create the IDP.")
			username, password = getUserDetails(cmd, r)
		}

		idpBuilder := cmv1.NewIdentityProvider().
			Type(cmv1.IdentityProviderTypeHtpasswd).
			Name(idpName).
			Htpasswd(
				cmv1.NewHTPasswdIdentityProvider().Users(
					cmv1.NewHTPasswdUserList().Items(CreateHTPasswdUser(username, password)),
				),
			)
		htpasswdIDP = doCreateIDP(idpName, *idpBuilder, cluster, clusterKey, r)
	}

	if interactive.Enabled() {
		for shouldAddAnotherUser(r) {
			username, password = getUserDetails(cmd, r)
			err = r.OCMClient.AddHTPasswdUser(username, password, cluster.ID(), htpasswdIDP.ID())
			if err != nil {
				r.Reporter.Errorf(
					"Failed to add a user to the HTPasswd IDP of cluster '%s': %v", clusterKey, err)
				os.Exit(1)
			}
			r.Reporter.Infof("User '%s' added", username)
		}
	}
}

func getUserDetails(cmd *cobra.Command, r *rosa.Runtime) (string, string) {
	username, err := interactive.GetString(interactive.Input{
		Question: "Username",
		Help:     cmd.Flags().Lookup("username").Usage,
		Default:  "",
		Required: true,
		Validators: []interactive.Validator{
			usernameValidator,
		},
	})
	if err != nil {
		exitHTPasswdCreate("Expected a valid username: %s", r.ClusterKey, err, r)
	}
	password, err := interactive.GetPassword(interactive.Input{
		Question: "Password",
		Help:     cmd.Flags().Lookup("password").Usage,
		Default:  "",
		Required: true,
		Validators: []interactive.Validator{
			passwordValidator,
		},
	})
	if err != nil {
		exitHTPasswdCreate("Expected a valid password: %s", r.ClusterKey, err, r)
	}
	return username, password
}

func shouldAddAnotherUser(r *rosa.Runtime) bool {
	addAnother, err := interactive.GetBool(interactive.Input{
		Question: "Add another user",
		Help:     "HTPasswd: Add more users to the IDP, to log into the cluster with.\n",
		Default:  false,
	})
	if err != nil {
		exitHTPasswdCreate("Expected a valid reply: %s", r.ClusterKey, err, r)
	}
	return addAnother
}

func CreateHTPasswdUser(username, password string) *cmv1.HTPasswdUserBuilder {
	builder := cmv1.NewHTPasswdUser()
	if username != "" {
		builder = builder.Username(username)
	}
	if password != "" {
		builder = builder.Password(password)
	}
	return builder
}

func exitHTPasswdCreate(format, clusterKey string, err error, r *rosa.Runtime) {
	r.Reporter.Errorf("Failed to create IDP for cluster '%s': %v",
		clusterKey,
		fmt.Errorf(format, err))
	os.Exit(1)
}

func usernameValidator(val interface{}) error {
	if username, ok := val.(string); ok {
		if username == ClusterAdminUsername {
			return fmt.Errorf("username '%s' is not allowed", username)
		}
		if strings.ContainsAny(username, "/:%") {
			return fmt.Errorf("invalid username '%s': "+
				"username must not contain /, :, or %%", username)
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got '%v'", val)
}

func passwordValidator(val interface{}) error {
	if password, ok := val.(string); ok {
		re := regexp.MustCompile(`[^\x20-\x7E]`)
		invalidChars := re.FindAllString(password, -1)
		notAsciiOnly := len(invalidChars) > 0
		containsSpace := strings.Contains(password, " ")
		tooShort := len(password) < 14
		pwdErrors := []string{}
		if notAsciiOnly {
			pwdErrors = append(pwdErrors, fmt.Sprintf("must not contain special characters [%s]",
				strings.Join(invalidChars, ", ")))
		}
		if containsSpace {
			pwdErrors = append(pwdErrors, "must not contain whitespace")
		}
		if tooShort {
			pwdErrors = append(pwdErrors, fmt.Sprintf("must be at least 14 characters (got %d)", len(password)))
		}
		if notAsciiOnly || containsSpace || tooShort {
			if len(pwdErrors) > 1 {
				pwdErrors[len(pwdErrors)-1] = "and " + pwdErrors[len(pwdErrors)-1]
			}

			return fmt.Errorf("Password " + strings.Join(pwdErrors, ", "))
		}
		hasUppercase, _ := regexp.MatchString(`[A-Z]`, password)
		hasLowercase, _ := regexp.MatchString(`[a-z]`, password)
		hasNumberOrSymbol, _ := regexp.MatchString(`[^a-zA-Z]`, password)
		if !hasUppercase || !hasLowercase || !hasNumberOrSymbol {
			return fmt.Errorf(
				"Password must include uppercase letters, lowercase letters, and numbers " +
					"or symbols (ASCII-standard characters only)")
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got '%v'", val)
}

func HasClusterAdmin(userList *cmv1.HTPasswdUserList) bool {
	hasAdmin := false
	userList.Each(func(user *cmv1.HTPasswdUser) bool {
		if user.Username() == ClusterAdminUsername {
			hasAdmin = true
		}
		return true
	})
	return hasAdmin
}

func FindExistingHTPasswdIDP(cluster *cmv1.Cluster, r *rosa.Runtime) (
	htpasswdIDP *cmv1.IdentityProvider, userList *cmv1.HTPasswdUserList) {
	r.Reporter.Debugf("Loading cluster's identity providers")
	idps, err := r.OCMClient.GetIdentityProviders(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get identity providers for cluster '%s': %v", r.ClusterKey, err)
		os.Exit(1)
	}

	for _, item := range idps {
		if ocm.IdentityProviderType(item) == ocm.HTPasswdIDPType {
			htpasswdIDP = item
		}
	}
	if htpasswdIDP != nil {
		userList, err = r.OCMClient.GetHTPasswdUserList(cluster.ID(), htpasswdIDP.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to get user list of the HTPasswd IDP of '%s': %v", r.ClusterKey, err)
			os.Exit(1)
		}
	}
	return
}
