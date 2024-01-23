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

package login

import (
	"fmt"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/logout"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/fedramp"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

// #nosec G101
var uiTokenPage string = "https://console.redhat.com/openshift/token/rosa"

const oauthClientId = "ocm-cli"

var reAttempt bool

var args struct {
	tokenURL     string
	clientID     string
	clientSecret string
	scopes       []string
	env          string
	token        string
	insecure     bool
	useAuthCode  bool
}

var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your Red Hat account",
	Long: fmt.Sprintf("Log in to your Red Hat account, saving the credentials to the configuration file.\n"+
		"The supported mechanism is by using a token, which can be obtained at: %s\n\n"+
		"The application looks for the token in the following order, stopping when it finds it:\n"+
		"\t1. Command-line flags\n"+
		"\t2. Environment variable (ROSA_TOKEN)\n"+
		"\t3. Environment variable (OCM_TOKEN)\n"+
		"\t4. Configuration file\n"+
		"\t5. Command-line prompt\n", uiTokenPage),
	Example: fmt.Sprintf(`  # Login to the OpenShift API with an existing token generated from %s
  rosa login --token=$OFFLINE_ACCESS_TOKEN`, uiTokenPage),
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.tokenURL,
		"token-url",
		"",
		fmt.Sprintf(
			"OpenID token URL. The default value is '%s'.",
			sdk.DefaultTokenURL,
		),
	)
	flags.StringVar(
		&args.clientID,
		"client-id",
		"",
		fmt.Sprintf(
			"OpenID client identifier. The default value is '%s'.",
			sdk.DefaultClientID,
		),
	)
	flags.StringVar(
		&args.clientSecret,
		"client-secret",
		"",
		"OpenID client secret.",
	)
	flags.StringSliceVar(
		&args.scopes,
		"scope",
		sdk.DefaultScopes,
		"OpenID scope. If this option is used it will completely replace the default "+
			"scopes. Can be repeated multiple times to specify multiple scopes.",
	)
	flags.StringVar(
		&args.env,
		"env",
		sdk.DefaultURL,
		"Environment of the API gateway. The value can be the complete URL or an alias. "+
			"The valid aliases are 'production', 'staging' and 'integration'.",
	)
	flags.MarkHidden("env")
	flags.StringVarP(
		&args.token,
		"token",
		"t",
		"",
		fmt.Sprintf("Access or refresh token generated from %s.", uiTokenPage),
	)
	flags.BoolVar(
		&args.insecure,
		"insecure",
		false,
		"Enables insecure communication with the server. This disables verification of TLS "+
			"certificates and host names.",
	)
	flags.BoolVar(
		&args.useAuthCode,
		"use-auth-code",
		false,
		"Enables OAuth Authorization Code login using PKCE. If this option is provided, "+
			"the user will be taken to Red Hat SSO for authentication. In order to use a different account"+
			"log out from sso.redhat.com after using the 'ocm logout' command.",
	)
	flags.MarkHidden("use-auth-code")
	arguments.AddRegionFlag(flags)
	fedramp.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime()

	// Check mandatory options:
	env := args.env
	if env == "" {
		r.Reporter.Errorf("Option '--env' is mandatory")
		os.Exit(1)
	}

	if args.useAuthCode {
		fmt.Println("You will now be redirected to Red Hat SSO login")
		token, err := authentication.VerifyLogin(oauthClientId)
		if err != nil {
			r.Reporter.Errorf("An error occurred while retrieving the token : %v", err)
			os.Exit(1)
		}
		args.token = token
		fmt.Println("Token received successfully")
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		r.Reporter.Errorf("Failed to load config file: %v", err)
		os.Exit(1)
	}
	if cfg == nil {
		cfg = new(config.Config)
	}

	token := args.token

	// Determine if we should be using the FedRAMP environment:
	if fedramp.HasFlag(cmd) ||
		(cfg.FedRAMP && token == "") ||
		fedramp.IsGovRegion(arguments.GetRegion()) ||
		config.IsEncryptedToken(token) {
		fedramp.Enable()
		// Always default to prod
		if env == sdk.DefaultURL {
			env = "production"
		}
		if fedramp.HasAdminFlag(cmd) {
			uiTokenPage = fedramp.AdminLoginURLs[env]
		} else {
			uiTokenPage = fedramp.LoginURLs[env]
		}
	} else {
		fedramp.Disable()
	}

	haveReqs := token != ""

	// Verify environment variables:
	if !haveReqs && !reAttempt && !fedramp.Enabled() {
		token = os.Getenv("ROSA_TOKEN")
		if token == "" {
			token = os.Getenv("OCM_TOKEN")
		}
		haveReqs = token != ""
	}

	// Verify configuration file:
	if !haveReqs {
		armed, err := cfg.Armed()
		if err != nil {
			r.Reporter.Errorf("Failed to verify configuration: %v", err)
			os.Exit(1)
		}
		haveReqs = armed
	}

	// Prompt the user for token:
	if !haveReqs {
		fmt.Println("To login to your Red Hat account, get an offline access token at", uiTokenPage)
		token, err = interactive.GetPassword(interactive.Input{
			Question: "Copy the token and paste it here",
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Failed to parse token: %v", err)
			os.Exit(1)
		}
		haveReqs = token != ""
	}

	if !haveReqs {
		r.Reporter.Errorf("Failed to login to OCM. See 'rosa login --help' for information.")
		os.Exit(1)
	}

	// Red Hat SSO does not issue encrypted refresh tokens, but AWS Cognito does. If the token
	// is encrypted we can safely assume that the user is trying to use the FedRAMP environment.
	if config.IsEncryptedToken(token) {
		fedramp.Enable()
	}

	// Apply the default configuration details if not explicitly provided by the user:
	gatewayURL, ok := ocm.URLAliases[env]
	if !ok {
		gatewayURL = env
	}
	tokenURL := sdk.DefaultTokenURL
	if args.tokenURL != "" {
		tokenURL = args.tokenURL
	}
	clientID := sdk.DefaultClientID
	if args.clientID != "" {
		clientID = args.clientID
	}
	// Override configuration details for FedRAMP:
	if fedramp.Enabled() {
		if fedramp.HasAdminFlag(cmd) {
			gatewayURL, ok = fedramp.AdminURLAliases[env]
			if !ok {
				gatewayURL = env
			}
			tokenURL, ok = fedramp.AdminTokenURLs[env]
			if !ok {
				tokenURL = args.tokenURL
			}
			clientID, ok = fedramp.AdminClientIDs[env]
			if !ok {
				clientID = args.clientID
			}
		} else {
			gatewayURL, ok = fedramp.URLAliases[env]
			if !ok {
				gatewayURL = env
			}
			tokenURL, ok = fedramp.TokenURLs[env]
			if !ok {
				tokenURL = args.tokenURL
			}
			clientID, ok = fedramp.ClientIDs[env]
			if !ok {
				clientID = args.clientID
			}
		}
	}

	// Update the configuration with the values given in the command line:
	cfg.TokenURL = tokenURL
	cfg.ClientID = clientID
	cfg.ClientSecret = args.clientSecret
	cfg.Scopes = args.scopes
	cfg.URL = gatewayURL
	cfg.Insecure = args.insecure
	cfg.FedRAMP = fedramp.Enabled()

	if token != "" {
		if config.IsEncryptedToken(token) {
			cfg.AccessToken = ""
			cfg.RefreshToken = token
		} else {
			// If a token has been provided parse it:
			jwtToken, err := config.ParseToken(token)
			if err != nil {
				r.Reporter.Errorf("Failed to parse token: %v", err)
				os.Exit(1)
			}

			// Put the token in the place of the configuration that corresponds to its type:
			typ, err := tokenType(jwtToken)
			if err != nil {
				r.Reporter.Errorf("Failed to extract type from 'typ' claim of token: %v", err)
				os.Exit(1)
			}
			switch typ {
			case "Bearer", "":
				cfg.AccessToken = token
				cfg.RefreshToken = ""
			case "Refresh", "Offline":
				cfg.AccessToken = ""
				cfg.RefreshToken = token
			default:
				r.Reporter.Errorf("Don't know how to handle token type '%s' in token", typ)
				os.Exit(1)
			}
		}
	}

	// Create a connection and get the token to verify that the crendentials are correct:
	r.OCMClient, err = ocm.NewClient().
		Config(cfg).
		Logger(r.Logger).
		Build()
	if err != nil {
		if strings.Contains(err.Error(), "token needs to be updated") && !reAttempt {
			reattemptLogin(cmd, argv)
			return
		} else {
			r.Reporter.Errorf("Failed to create OCM connection: %v", err)
			os.Exit(1)
		}
	}
	defer r.Cleanup()

	accessToken, refreshToken, err := r.OCMClient.GetConnectionTokens()
	if err != nil {
		r.Reporter.Errorf("Failed to get token. Your session might be expired: %v", err)
		r.Reporter.Infof("Get a new offline access token at %s", uiTokenPage)
		os.Exit(1)
	}
	reAttempt = false
	// Save the configuration:
	cfg.AccessToken = accessToken
	cfg.RefreshToken = refreshToken
	err = config.Save(cfg)
	if err != nil {
		r.Reporter.Errorf("Failed to save config file: %v", err)
		os.Exit(1)
	}

	username, err := cfg.GetData("username")
	if err != nil {
		r.Reporter.Errorf("Failed to get username: %v", err)
		os.Exit(1)
	}

	r.Reporter.Infof("Logged in as '%s' on '%s'", username, cfg.URL)
	r.OCMClient.LogEvent("ROSALoginSuccess", map[string]string{
		ocm.Response: ocm.Success,
		ocm.Username: username,
		ocm.URL:      cfg.URL,
	})
}

func reattemptLogin(cmd *cobra.Command, argv []string) {
	logout.Cmd.Run(cmd, argv)
	reAttempt = true
	run(cmd, argv)
}

// tokenType extracts the value of the `typ` claim. It returns the value as a string, or the empty
// string if there is no such claim.
func tokenType(jwtToken *jwt.Token) (typ string, err error) {
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("Expected map claims but got %T", claims)
		return
	}
	claim, ok := claims["typ"]
	if !ok {
		return
	}
	value, ok := claim.(string)
	if !ok {
		err = fmt.Errorf("Expected string 'typ' but got %T", claim)
		return
	}
	typ = value
	return
}

func Call(cmd *cobra.Command, argv []string, reporter *rprtr.Object) error {
	loginFlags := []string{"token-url", "client-id", "client-secret", "scope", "env", "token", "insecure"}
	hasLoginFlags := false
	// Check if the user set login flags
	for _, loginFlag := range loginFlags {
		if cmd.Flags().Changed(loginFlag) {
			hasLoginFlags = true
			break
		}
	}
	if hasLoginFlags {
		// Always force login if user sets login flags
		run(cmd, argv)
		return nil
	}

	// Verify if user is already logged in:
	isLoggedIn := false
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Failed to load config file: %v", err)
	}
	if cfg != nil {
		// Check that credentials in the config file are valid
		isLoggedIn, err = cfg.Armed()
		if err != nil {
			return fmt.Errorf("Failed to determine if user is logged in: %v", err)
		}
	}

	if isLoggedIn {
		username, err := cfg.GetData("username")
		if err != nil {
			return fmt.Errorf("Failed to get username: %v", err)
		}

		if reporter.IsTerminal() {
			reporter.Infof("Logged in as '%s' on '%s'", username, cfg.URL)
		}
		return nil
	}

	run(cmd, argv)
	return nil
}
