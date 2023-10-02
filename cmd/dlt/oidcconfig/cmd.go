/*
Copyright (c) 2023 Red Hat, Inc.

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

package oidcconfig

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"github.com/zgalor/weberr"

	"github.com/openshift/rosa/cmd/dlt/oidcprovider"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	interactiveOidc "github.com/openshift/rosa/pkg/interactive/oidc"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "oidc-config",
	Aliases: []string{"oidconfig, oidcconfig"},
	Short:   "Delete OIDC Config",
	Long:    "Cleans up OIDC config based on registered OIDC Config ID.",
	Example: `  # Delete OIDC config based on registered OIDC Config ID that has been supplied
	rosa delete oidc-config --oidc-config-id <oidc_config_id>`,
	Run: run,
}

const (
	//nolint
	OidcConfigIdFlag          = "oidc-config-id"
	prefixForPrivateKeySecret = "rosa-private-key-"
)

var args struct {
	oidcConfigId string
	region       string
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.oidcConfigId,
		OidcConfigIdFlag,
		"",
		"Registered ID for identification of OIDC config",
	)

	aws.AddModeFlag(Cmd)

	interactive.AddFlag(flags)
	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Get AWS region
	region, err := aws.GetRegion(arguments.GetRegion())
	if err != nil {
		r.Reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}
	args.region = region

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "OIDC Config deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid OIDC provider creation mode: %s", err)
			os.Exit(1)
		}
	}

	if (args.oidcConfigId == "" || interactive.Enabled()) && !cmd.Flags().Changed(OidcConfigIdFlag) {
		args.oidcConfigId = interactiveOidc.GetOidcConfigID(r, cmd)
	}

	oidcConfigInput := buildOidcConfigInput(r)
	oidcConfigStrategy, err := getOidcConfigStrategy(mode, oidcConfigInput)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	oidcConfigStrategy.execute(r)
	oidcprovider.Cmd.Run(oidcprovider.Cmd, []string{"", mode, oidcConfigInput.IssuerUrl})
	r.OCMClient.DeleteOidcConfig(args.oidcConfigId)
	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Registered OIDC Config ID '%s'"+
			" has been removed from OCM and can no longer be used", args.oidcConfigId)
		if mode == aws.ModeManual {
			r.Reporter.Infof("Remember to run given commands to clean up aws resources")
		}
	}
}

type OidcConfigInput struct {
	PrivateKeySecretArn string
	BucketName          string
	IssuerUrl           string
	Managed             bool
}

func buildOidcConfigInput(r *rosa.Runtime) OidcConfigInput {
	oidcConfig, err := r.OCMClient.GetOidcConfig(args.oidcConfigId)
	if err != nil {
		r.Reporter.Errorf("There was a problem retrieving the OIDC Config '%s': %v", args.oidcConfigId, err)
		os.Exit(1)
	}
	secretArn := oidcConfig.SecretArn()
	bucketName := ""
	if !oidcConfig.Managed() {
		parsedSecretArn, _ := arn.Parse(secretArn)
		if args.region != parsedSecretArn.Region {
			r.Reporter.Errorf("Secret region '%s' differs from chosen region '%s', "+
				"please run the command supplying region parameter.", parsedSecretArn.Region, args.region)
			os.Exit(1)
		}
		secretResourceName, err := aws.GetResourceIdFromSecretArn(secretArn)
		if err != nil {
			r.Reporter.Errorf("There was a problem parsing secret ARN '%s' : %v", secretArn, err)
			os.Exit(1)
		}
		// The secret when creating from ROSA options has the following format
		// rosa-private-key-<prefix>-oidc-<random-hash-length-4>-<random-aws-created-hash>
		// The bucket is expected to be <prefix>-oidc-<random-hash-length-4>
		bucketName = strings.TrimPrefix(secretResourceName, prefixForPrivateKeySecret)
		index := strings.LastIndex(bucketName, "-")
		if index != -1 {
			bucketName = bucketName[:index]
		}
	}

	issuerUrl := oidcConfig.IssuerUrl()
	hasClusterUsingOidcConfig, err := r.OCMClient.HasAClusterUsingOidcEndpointUrl(issuerUrl)
	if err != nil {
		r.Reporter.Errorf("There was a problem checking if any clusters are using OIDC config '%s' : %v", issuerUrl, err)
		os.Exit(1)
	}
	if hasClusterUsingOidcConfig {
		r.Reporter.Errorf("There are clusters using OIDC config '%s', can't delete the configuration", issuerUrl)
		os.Exit(1)
	}
	return OidcConfigInput{
		BucketName:          bucketName,
		PrivateKeySecretArn: secretArn,
		IssuerUrl:           issuerUrl,
		Managed:             oidcConfig.Managed(),
	}
}

type DeleteOidcConfigStrategy interface {
	execute(r *rosa.Runtime)
}

type deleteUnmanagedOidcConfigAutoStrategy struct {
	oidcConfig OidcConfigInput
}

func (s *deleteUnmanagedOidcConfigAutoStrategy) execute(r *rosa.Runtime) {
	bucketName := s.oidcConfig.BucketName
	privateKeySecretArn := s.oidcConfig.PrivateKeySecretArn
	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		r.Reporter.Infof("Deleting OIDC configuration '%s'", bucketName)
	}
	if spin != nil {
		spin.Start()
	}
	err := r.AWSClient.DeleteSecretInSecretsManager(privateKeySecretArn)
	if err != nil {
		r.Reporter.Errorf("There was a problem deleting private key from secrets manager: %s", err)
		os.Exit(1)
	}
	err = r.AWSClient.DeleteS3Bucket(bucketName)
	if err != nil {
		r.Reporter.Errorf("There was a problem deleting S3 bucket '%s': %s", bucketName, err)
		os.Exit(1)
	}
	if spin != nil {
		spin.Stop()
	}
	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Deleted OIDC configuration")
	}
}

type deleteUnmanagedOidcConfigManualStrategy struct {
	oidcConfig OidcConfigInput
}

func (s *deleteUnmanagedOidcConfigManualStrategy) execute(r *rosa.Runtime) {
	commands := []string{}
	bucketName := s.oidcConfig.BucketName
	privateKeySecretArn := s.oidcConfig.PrivateKeySecretArn
	deleteSecretCommand := awscb.NewSecretsManagerCommandBuilder().
		SetCommand(awscb.DeleteSecret).
		AddParam(awscb.SecretID, privateKeySecretArn).
		AddParam(awscb.Region, args.region).
		Build()
	commands = append(commands, deleteSecretCommand)
	emptyS3BucketCommand := awscb.NewS3CommandBuilder().
		SetCommand(awscb.Remove).
		AddValueNoParam(fmt.Sprintf("s3://%s", bucketName)).
		AddParamNoValue(awscb.Recursive).
		Build()
	commands = append(commands, emptyS3BucketCommand)
	deleteS3BucketCommand := awscb.NewS3CommandBuilder().
		SetCommand(awscb.RemoveBucket).
		AddValueNoParam(fmt.Sprintf("s3://%s", bucketName)).
		Build()
	commands = append(commands, deleteS3BucketCommand)
	fmt.Println(awscb.JoinCommands(commands))
}

type deleteManagedOidcConfigStrategy struct{}

func (s *deleteManagedOidcConfigStrategy) execute(r *rosa.Runtime) {
	//It is supposed to do nothing as the call to unregister from OCM does everything
}

func getOidcConfigStrategy(mode string, input OidcConfigInput) (DeleteOidcConfigStrategy, error) {
	if input.Managed {
		return &deleteManagedOidcConfigStrategy{}, nil
	}
	switch mode {
	case aws.ModeAuto:
		return &deleteUnmanagedOidcConfigAutoStrategy{oidcConfig: input}, nil
	case aws.ModeManual:
		return &deleteUnmanagedOidcConfigManualStrategy{oidcConfig: input}, nil
	default:
		return nil, weberr.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
	}
}
