package token

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/config"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	writer io.Writer = os.Stdout
	args   struct {
		header    bool
		payload   bool
		signature bool
		refresh   bool
		generate  bool
	}
)

var Cmd = NewTokenCommand()

func NewTokenCommand() *cobra.Command {
	Cmd := &cobra.Command{
		Use:   "token",
		Short: "Generates a token",
		Long:  "Uses the stored credentials to generate a token.",
		Args:  cobra.NoArgs,
		Run:   run,
	}
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.payload,
		"payload",
		false,
		"Print the JSON payload.",
	)
	flags.BoolVar(
		&args.header,
		"header",
		false,
		"Print the JSON header.",
	)
	flags.BoolVar(
		&args.signature,
		"signature",
		false,
		"Print the signature.",
	)
	flags.BoolVar(
		&args.refresh,
		"refresh",
		false,
		"Print the refresh token instead of the access token.",
	)
	flags.BoolVar(
		&args.generate,
		"generate",
		false,
		"Generate a new token.",
	)
	return Cmd
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	err := CreateToken(r)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func CreateToken(r *rosa.Runtime) error {
	var accessToken string
	var refreshToken string
	var err error
	// Check the options:
	count := 0
	if args.header {
		count++
	}
	if args.payload {
		count++
	}
	if args.signature {
		count++
	}
	if args.generate {
		count++
	}

	if count > 1 {
		return fmt.Errorf("options '--payload', '--header', '--signature', and '--generate' are mutually exclusive")
	}

	accessToken, refreshToken, err = getAccessTokens(r, args.generate)
	if err != nil {
		return fmt.Errorf("can't get token: %v", err)
	}

	// Select the token according to the options:
	selectedToken := accessToken
	if args.refresh {
		selectedToken = refreshToken
	}

	// Parse the token:
	parser := new(jwt.Parser)
	_, parts, err := parser.ParseUnverified(selectedToken, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("can't parse token: %v", err)
	}
	encoding := base64.RawURLEncoding
	header, err := encoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("can't decode header: %v", err)
	}
	payload, err := encoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("can't decode payload: %v", err)
	}
	signature, err := encoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("can't decode signature: %v", err)
	}

	// Print the data:
	if args.header {
		fmt.Fprintf(writer, "%s\n", header)
	} else if args.payload {
		fmt.Fprintf(writer, "%s\n", payload)
	} else if args.signature {
		fmt.Fprintf(writer, "%s\n", signature)
	} else {
		fmt.Fprintf(writer, "%s\n", selectedToken)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("can't load config file: %v", err)
	}

	// Save the configuration:
	cfg.AccessToken = accessToken
	cfg.RefreshToken = refreshToken
	if err = config.Save(cfg); err != nil {
		return fmt.Errorf("can't save config file: %v", err)
	}
	return nil
}

func getAccessTokens(r *rosa.Runtime, generate bool) (string, string, error) {
	var accessToken, refreshToken string
	var err error
	if generate {
		// Get new tokens:
		accessToken, refreshToken, err = r.OCMClient.GetConnectionTokens(15 * time.Minute)
		if err != nil {
			return "", "", fmt.Errorf("can't get new tokens: %v", err)
		}
	} else {
		// Get the tokens:
		accessToken, refreshToken, err = r.OCMClient.GetConnectionTokens()
		if err != nil {
			return "", "", fmt.Errorf("can't get token: %v", err)
		}
	}
	return accessToken, refreshToken, nil
}
