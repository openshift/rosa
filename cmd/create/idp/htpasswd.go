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
	"github.com/openshift/rosa/pkg/rosa"
)

const ClusterAdminUsername = "cluster-admin"

func createHTPasswdIDP(cmd *cobra.Command,
	cluster *cmv1.Cluster,
	clusterKey string,
	idpName string,
	r *rosa.Runtime) {
	var err error

	validateUserArgs(r)

	//get users
	userList, isHashedPassword := getUserList(cmd, r)

	//build HTPasswdUserList
	htpasswdUsers := []*cmv1.HTPasswdUserBuilder{}
	for username, password := range userList {

		userBuilder := cmv1.NewHTPasswdUser().Username(username)
		if isHashedPassword {
			userBuilder.HashedPassword(password)
		} else {
			userBuilder.Password(password)
		}
		htpasswdUsers = append(htpasswdUsers, userBuilder)
	}

	htpassUserList := cmv1.NewHTPasswdUserList().Items(htpasswdUsers...)

	idpBuilder := cmv1.NewIdentityProvider().
		Type(cmv1.IdentityProviderTypeHtpasswd).
		Name(idpName).
		Htpasswd(
			cmv1.NewHTPasswdIdentityProvider().Users(htpassUserList),
		)
	htpasswdIDP := doCreateIDP(idpName, *idpBuilder, cluster, clusterKey, r)

	if interactive.Enabled() {
		for shouldAddAnotherUser(r) {
			username, password := GetUserDetails(cmd, r, "username", "password", "", "")
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

func getUserList(cmd *cobra.Command, r *rosa.Runtime) (userList map[string]string, hashed bool) {

	userList = make(map[string]string)
	hashed = false

	//if none of the user args are set, interactively prompt starting with htpasswd-file arg
	htpasswdFile := args.htpasswdFile
	if htpasswdFile == "" && len(args.htpasswdUsers) == 0 && args.htpasswdUsername == "" {
		var err error
		htpasswdFile, err = interactive.GetString(interactive.Input{
			Question: "Configure users from HTPasswd file",
			Help:     cmd.Flags().Lookup("from-file").Usage,
			Default:  htpasswdFile,
			Required: false,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid --from-file value: %s", err)
			os.Exit(1)
		}
	}

	//if htpasswdFile provided, process users in the file and return
	if htpasswdFile != "" {
		err := parseHtpasswordFile(&userList, htpasswdFile)
		if err != nil {
			r.Reporter.Errorf(
				"Failed to load Htpasswd file '%s': %v", htpasswdFile, err)
			os.Exit(1)
		}
		//password in htpasswd are already and do not need to be hashed again in CS
		hashed = true
		return
	}

	// htpasswdFile is not set, check for other user args

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
		return
	}

	if args.htpasswdUsername != "" && args.htpasswdPassword != "" {
		//if userlist or htpasswdfile are not provided
		//continue support for single username/password for backcompatibility
		//so as not to break any existing automation
		userList[args.htpasswdUsername] = args.htpasswdPassword
		return
	}

	// none of the userinfo args are set, prompt interactively for users
	r.Reporter.Infof("At least one valid user and password is required to create the IDP.")
	username, password := GetUserDetails(cmd, r, "username", "password", "", "")
	userList[username] = password

	return
}

func GetUserDetails(cmd *cobra.Command, r *rosa.Runtime,
	usernameKey, passwordKey, defaultUsername, defaultPassword string) (string, string) {
	username, err := interactive.GetString(interactive.Input{
		Question: "Username",
		Help:     cmd.Flags().Lookup(usernameKey).Usage,
		Default:  defaultUsername,
		Required: true,
		Validators: []interactive.Validator{
			UsernameValidator,
		},
	})
	if err != nil {
		exitHTPasswdCreate("Expected a valid username: %s", r.ClusterKey, err, r)
	}
	password, err := interactive.GetPassword(interactive.Input{
		Question: "Password",
		Help:     cmd.Flags().Lookup(passwordKey).Usage,
		Default:  defaultPassword,
		Required: true,
		Validators: []interactive.Validator{
			PasswordValidator,
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

func UsernameValidator(val interface{}) error {
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

func PasswordValidator(val interface{}) error {
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
