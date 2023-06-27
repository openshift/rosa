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
	"bufio"
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

		// Existing IDP contains only admin. Add new users to it
		r.Reporter.Infof("Cluster already has an HTPasswd IDP named '%s', new users will be added to it.",
			htpasswdIDP.Name())

		htpassUserList, _ := buildUserList(cmd, r).Build()
		err = r.OCMClient.AddHTPasswdUsers(htpassUserList, cluster.ID(), htpasswdIDP.ID())
		if err != nil {
			r.Reporter.Errorf(
				"Failed to add a user to the HTPasswd IDP of cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	} else {
		// HTPasswd IDP does not exist - create it

		htpassUserList := buildUserList(cmd, r)

		idpBuilder := cmv1.NewIdentityProvider().
			Type(cmv1.IdentityProviderTypeHtpasswd).
			Name(idpName).
			Htpasswd(
				cmv1.NewHTPasswdIdentityProvider().Users(htpassUserList),
			)
		htpasswdIDP = doCreateIDP(idpName, *idpBuilder, cluster, clusterKey, r)
	}

	if interactive.Enabled() {
		for shouldAddAnotherUser(r) {
			username, password := getUserDetails(cmd, r)
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

func validateUserArgs(r *rosa.Runtime) {

	//validate mutually exclusive group of flags ( users | username | from-file)

	numOfUserArgs := 0

	// comma separated list of users -  user1:password,user2:password,user3:password
	if len(args.htpasswdUsers) != 0 {
		numOfUserArgs++
	}
	//continue support for single username/password for  backcompatibility
	// so as not to break any existing automation
	if args.htpasswdUsername != "" {
		numOfUserArgs++
	}

	//path to Htpasswd file
	if args.htpasswdFile != "" {
		numOfUserArgs++
	}

	if numOfUserArgs > 1 {
		r.Reporter.Errorf("Only one of  'users', 'from-file' or 'username/password' may be specified. \n" +
			"Choose the option 'users' to add one or more users to the IDP.\n" +
			"Choose the option 'from-file' to load users from a htpassword file")
		os.Exit(1)
	}
}
func buildUserList(cmd *cobra.Command, r *rosa.Runtime) *cmv1.HTPasswdUserListBuilder {

	validateUserArgs(r)

	userList := make(map[string]string)
	hashed := false

	if len(args.htpasswdUsers) != 0 {
		//if a user list is specified then continue with the  list
		users := args.htpasswdUsers
		for _, user := range users {
			u, p, found := strings.Cut(user, ":")
			if !found {
				r.Reporter.Errorf(
					"Users should be provided in the format of a comma separate list of user:password")
				os.Exit(1)

			}
			userList[u] = p
		}
	} else if args.htpasswdFile != "" {

		usersfile := args.htpasswdFile
		err := parseHtpasswordFile(&userList, usersfile)
		if err != nil {
			r.Reporter.Errorf(
				"Failed to load Htpasswd file '%s': %v", usersfile, err)
			os.Exit(1)
		}
		//password in htpasswd are already and do not need to be hashed again in CS
		hashed = true
	} else if args.htpasswdUsername != "" && args.htpasswdPassword != "" {
		//if userlist or htpasswdfile are not provided
		//continue support for single username/password for  backcompatibility
		//so as not to break any existing automation
		userList[args.htpasswdUsername] = args.htpasswdPassword
	} else {
		r.Reporter.Infof("At least one valid user and password is required to create the IDP.")
		username, password := getUserDetails(cmd, r)
		userList[username] = password
	}

	htpasswdUsers := []*cmv1.HTPasswdUserBuilder{}
	for username, password := range userList {

		userBuilder := cmv1.NewHTPasswdUser().Username(username)
		if hashed {
			userBuilder.HashedPassword(password)
		} else {
			userBuilder.Password(password)
		}
		htpasswdUsers = append(htpasswdUsers, userBuilder)
	}

	htpassUserList := cmv1.NewHTPasswdUserList().Items(htpasswdUsers...)
	return htpassUserList
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
		notAsciiOnly, _ := regexp.MatchString(`[^\x20-\x7E]`, password)
		containsSpace := strings.Contains(password, " ")
		tooShort := len(password) < 14
		if notAsciiOnly || containsSpace || tooShort {
			return fmt.Errorf(
				"password must be at least 14 characters (ASCII-standard) without whitespaces")
		}
		hasUppercase, _ := regexp.MatchString(`[A-Z]`, password)
		hasLowercase, _ := regexp.MatchString(`[a-z]`, password)
		hasNumberOrSymbol, _ := regexp.MatchString(`[^a-zA-Z]`, password)
		if !hasUppercase || !hasLowercase || !hasNumberOrSymbol {
			return fmt.Errorf(
				"password must include uppercase letters, lowercase letters, and numbers " +
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

func parseHtpasswordFile(usersList *map[string]string, filePath string) error {

	//A standard wellformed htpasswd file has rows of colon separated usernames and passwords
	//e.g.
	//eleven:$apr1$hRY7OJWH$km1EYH.UIRjp6CzfZQz/g1
	//vecna:$apr1$Q58SO804$B/fECNWfn5xkJXJLvu0mF/

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// split "user:password" at colon
		username, password, found := strings.Cut(line, ":")
		if !found || username == "" || password == "" {
			return fmt.Errorf("Malformed line, Expected: validUsername:validPassword, Got: %s", line)
		}

		(*usersList)[username] = password
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
