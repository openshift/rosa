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
	"os"
	"strings"

	"github.com/dgrijalva/jwt-go"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/logout"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

// #nosec G101
const uiTokenPage = "https://console.redhat.com/openshift/token/rosa"

var reAttempt bool

var args struct {
	tokenURL     string
	clientID     string
	clientSecret string
	scopes       []string
	env          string
	token        string
	insecure     bool
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
	Example: "  # Login to the OpenShift API with an existing token generated from " +
		`https://console.redhat.com/openshift/token/rosa
  rosa login --token=$OFFLINE_ACCESS_TOKEN`,
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
		"OpenID scope. If this option is used it will replace completely the default "+
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
		"Access or refresh token generated from https://console.redhat.com/openshift/token/rosa.",
	)
	flags.BoolVar(
		&args.insecure,
		"insecure",
		false,
		"Enables insecure communication with the server. This disables verification of TLS "+
			"certificates and host names.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check mandatory options:
	if args.env == "" {
		reporter.Errorf("Option '--env' is mandatory")
		os.Exit(1)
	}

	// Load the configuration file:
	cfg, err := ocm.Load()
	if err != nil {
		reporter.Errorf("Failed to load config file: %v", err)
		os.Exit(1)
	}
	if cfg == nil {
		cfg = new(ocm.Config)
	}

	token := args.token
	haveReqs := token != ""

	// Verify environment variables:
	if !haveReqs && !reAttempt {
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
			reporter.Errorf("Failed to verify configuration: %v", err)
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
			reporter.Errorf("Failed to parse token: %v", err)
			os.Exit(1)
		}
		haveReqs = token != ""
	}

	if !haveReqs {
		reporter.Errorf("Failed to login to OCM. See 'rosa login --help' for information.")
		os.Exit(1)
	}

	// Apply the default OpenID details if not explicitly provided by the user:
	tokenURL := sdk.DefaultTokenURL
	if args.tokenURL != "" {
		tokenURL = args.tokenURL
	}
	clientID := sdk.DefaultClientID
	if args.clientID != "" {
		clientID = args.clientID
	}

	// If the value of the `--env` is any of the aliases then replace it with the corresponding
	// real URL:
	gatewayURL, ok := ocm.URLAliases[args.env]
	if !ok {
		gatewayURL = args.env
	}

	// Update the configuration with the values given in the command line:
	cfg.TokenURL = tokenURL
	cfg.ClientID = clientID
	cfg.ClientSecret = args.clientSecret
	cfg.Scopes = args.scopes
	cfg.URL = gatewayURL
	cfg.Insecure = args.insecure

	if token != "" {
		// If a token has been provided parse it:
		parser := new(jwt.Parser)
		jwtToken, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
		if err != nil {
			reporter.Errorf("Failed to parse token '%s': %v", token, err)
			os.Exit(1)
		}

		// Put the token in the place of the configuration that corresponds to its type:
		typ, err := tokenType(jwtToken)
		if err != nil {
			reporter.Errorf("Failed to extract type from 'typ' claim of token '%s': %v", token, err)
			os.Exit(1)
		}
		switch typ {
		case "Bearer":
			cfg.AccessToken = token
			cfg.RefreshToken = ""
		case "Refresh", "Offline":
			cfg.AccessToken = ""
			cfg.RefreshToken = token
		case "":
			reporter.Errorf("Don't know how to handle empty type in token '%s'", token)
			os.Exit(1)
		default:
			reporter.Errorf("Don't know how to handle token type '%s' in token '%s'", typ, token)
			os.Exit(1)
		}
	}

	// Create a connection and get the token to verify that the crendentials are correct:
	ocmClient, err := ocm.NewClient().
		Config(cfg).
		Logger(logger).
		Build()
	if err != nil {
		if strings.Contains(err.Error(), "token needs to be updated") && !reAttempt {
			reattemptLogin(cmd, argv)
		} else {
			reporter.Errorf("Failed to create OCM connection: %v", err)
			os.Exit(1)
		}
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()
	accessToken, refreshToken, err := ocmClient.GetConnectionTokens()
	if err != nil {
		reporter.Errorf("Failed to get token. Your session might be expired: %v", err)
		reporter.Infof("Get a new offline access token at %s", uiTokenPage)
		os.Exit(1)
	}
	reAttempt = false
	// Save the configuration:
	cfg.AccessToken = accessToken
	cfg.RefreshToken = refreshToken
	err = ocm.Save(cfg)
	if err != nil {
		reporter.Errorf("Failed to save config file: %v", err)
		os.Exit(1)
	}

	username, err := cfg.GetData("username")
	if err != nil {
		reporter.Errorf("Failed to get username: %v", err)
		os.Exit(1)
	}

	reporter.Infof("Logged in as '%s' on '%s'", username, cfg.URL)
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
	cfg, err := ocm.Load()
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

		reporter.Infof("Logged in as '%s' on '%s'", username, cfg.URL)
		return nil
	}

	run(cmd, argv)
	return nil
}
