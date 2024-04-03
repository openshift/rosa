package breakglasscredential

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	usernameFlag   = "username"
	expirationFlag = "expiration"
)

type BreakGlassCredentialArgs struct {
	username           string
	expirationDuration time.Duration
}

func AddBreakGlassCredentialFlags(cmd *cobra.Command) *BreakGlassCredentialArgs {
	args := &BreakGlassCredentialArgs{}

	cmd.Flags().StringVar(
		&args.username,
		usernameFlag,
		"",
		"Username for the break glass credential.",
	)

	cmd.Flags().DurationVar(
		&args.expirationDuration,
		expirationFlag,
		0,
		fmt.Sprintf("Expire the break glass credential after a relative duration like 2h, 8h. "+
			"The expiration duration needs to be at least 10 minutes from now and to be at maximum 24 hours."),
	)
	return args
}

func GetBreakGlassCredentialOptions(cmd *pflag.FlagSet, args *BreakGlassCredentialArgs) (
	*BreakGlassCredentialArgs, error) {
	var err error
	result := &BreakGlassCredentialArgs{}

	result.username = args.username
	result.expirationDuration = args.expirationDuration

	if !IsBreakGlassCredentialSetViaCLI(cmd) {
		if !interactive.Enabled() {
			return nil, nil
		}
	}

	if interactive.Enabled() && !cmd.Changed(usernameFlag) {
		result.username, err = interactive.GetString(interactive.Input{
			Question: "Username",
			Default:  result.username,
			Help:     cmd.Lookup(usernameFlag).Usage,
		})
		if err != nil {
			return nil, err
		}
	}

	if interactive.Enabled() && !cmd.Changed(expirationFlag) {
		inputString, err := interactive.GetString(interactive.Input{
			Question: "Expiration duration",
			Default:  result.expirationDuration,
			Help:     cmd.Lookup(expirationFlag).Usage,
		})
		if err != nil {
			return nil, err
		}
		if inputString != "" {
			duration, err := time.ParseDuration(inputString)
			if err != nil {
				return nil, err
			}
			result.expirationDuration = duration
		}
	}

	return result, nil
}

func CreateBreakGlass(cluster *cmv1.Cluster,
	clusterKey string,
	args *BreakGlassCredentialArgs, r *rosa.Runtime) (*cmv1.BreakGlassCredential, error) {

	breakGlassConfig, err := CreateBreakGlassConfig(args)
	if err != nil {
		return &cmv1.BreakGlassCredential{}, fmt.Errorf("failed to create a break glass credential for cluster '%s': %s",
			clusterKey, err)
	}

	credential, err := r.OCMClient.CreateBreakGlassCredential(cluster.ID(), breakGlassConfig)

	if err != nil {
		return &cmv1.BreakGlassCredential{}, fmt.Errorf("failed to create a break glass credential for cluster '%s': %s",
			clusterKey, err)
	}
	return credential, nil

}

func CreateBreakGlassConfig(args *BreakGlassCredentialArgs) (*cmv1.BreakGlassCredential, error) {
	breakGlassBuilder := cmv1.NewBreakGlassCredential()

	if args != nil {
		if args.username != "" {
			breakGlassBuilder.Username(args.username)
		}

		if args.expirationDuration != 0 {
			expirationTimeStamp := time.Now().Add(args.expirationDuration).Round(time.Second)
			breakGlassBuilder.ExpirationTimestamp(expirationTimeStamp)
		}
	}

	return breakGlassBuilder.Build()
}

func FormatBreakGlassCredentialOutput(breakGlassCredential *cmv1.BreakGlassCredential) (map[string]interface{}, error) {

	var b bytes.Buffer
	err := cmv1.MarshalBreakGlassCredential(breakGlassCredential, &b)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]interface{})
	err = json.Unmarshal(b.Bytes(), &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func IsBreakGlassCredentialSetViaCLI(cmd *pflag.FlagSet) bool {
	for _, parameter := range []string{usernameFlag, expirationFlag} {

		if cmd.Changed(parameter) {
			return true
		}
	}

	return false
}
