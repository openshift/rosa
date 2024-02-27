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
	"github.com/openshift/rosa/pkg/output"
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
	output.AddFlag(Cmd)
	return Cmd
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()
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
		r.Reporter.Errorf("Options '--payload', '--header', '--signature', and '--generate' are mutually exclusive")
		os.Exit(1)
	}

	if args.generate {
		// Get new tokens:
		accessToken, refreshToken, err = r.OCMClient.GetConnectionTokens(15 * time.Minute)
		if err != nil {
			r.Reporter.Errorf("Can't get new tokens: %v", err)
			os.Exit(1)
		}
	} else {
		// Get the tokens:
		accessToken, refreshToken, err = r.OCMClient.GetConnectionTokens()
		if err != nil {
			r.Reporter.Errorf("Can't get token: %v", err)
			os.Exit(1)
		}
	}

	// Select the token according to the options:
	selectedToken := accessToken
	if args.refresh {
		selectedToken = refreshToken
	}

	// Parse the token:
	parser := new(jwt.Parser)
	fmt.Println(refreshToken)
	_, parts, err := parser.ParseUnverified(selectedToken, jwt.MapClaims{})
	if err != nil {
		r.Reporter.Errorf("Can't parse token: %v", err)
		os.Exit(1)
	}
	encoding := base64.RawURLEncoding
	header, err := encoding.DecodeString(parts[0])
	if err != nil {
		r.Reporter.Errorf("Can't decode header: %v", err)
		os.Exit(1)
	}
	payload, err := encoding.DecodeString(parts[1])
	if err != nil {
		r.Reporter.Errorf("Can't decode payload: %v", err)
		os.Exit(1)
	}
	signature, err := encoding.DecodeString(parts[2])
	if err != nil {
		r.Reporter.Errorf("Can't decode signature: %v", err)
		os.Exit(1)
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
		r.Reporter.Errorf("Can't load config file: %v", err)
		os.Exit(1)
	}

	// Save the configuration:
	cfg.AccessToken = accessToken
	cfg.RefreshToken = refreshToken
	err = config.Save(cfg)
	if err != nil {
		r.Reporter.Errorf("Can't save config file: %v", err)
		os.Exit(1)
	}
}
