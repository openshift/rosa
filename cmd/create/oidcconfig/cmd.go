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
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/openshift-online/ocm-common/pkg/rosa/oidcconfigs"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	"github.com/zgalor/weberr"

	"github.com/openshift/rosa/cmd/create/oidcprovider"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	. "github.com/openshift/rosa/pkg/constants"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	interactiveRoles "github.com/openshift/rosa/pkg/interactive/roles"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	region           string
	rawFiles         bool
	userPrefix       string
	managed          bool
	installerRoleArn string
}

func SetCreateOidcProviderCommand(cmd rosa.CommandInterface) {
	oidcprovider.CreateOidcProvider = cmd
}

func NewCreateOidcConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "oidc-config",
		Aliases: []string{"oidcconfig"},
		Short:   "Create OIDC config compliant with OIDC protocol.",
		Long: "Create OIDC config in a S3 bucket for the " +
			"client AWS account and populates it to be compliant with OIDC protocol. " +
			"It also creates a Secret in Secrets Manager containing the private key.",
		Example: `  # Create OIDC config
		rosa create oidc-config`,
		Run:  rosa.DefaultRunner(rosa.RuntimeWithOCM(), CreateOidcConfigRunner()),
		Args: cobra.NoArgs,
	}

	flags := cmd.Flags()

	flags.BoolVar(
		&args.rawFiles,
		rawFilesFlag,
		false,
		"Creates OIDC config documents (Private RSA key, Discovery document, JSON Web Key Set) "+
			"and saves locally for the client to create the configuration.",
	)

	flags.StringVar(
		&args.userPrefix,
		userPrefixFlag,
		"",
		"Prefix for the OIDC configuration, secret and provider.",
	)

	flags.BoolVar(
		&args.managed,
		managedFlag,
		true,
		"Indicates whether it is a Red Hat managed or unmanaged (Customer hosted) OIDC Configuration.",
	)

	// normalizing installer role argument to support deprecated flag
	flags.SetNormalizeFunc(arguments.NormalizeFlags)
	flags.StringVar(
		&args.installerRoleArn,
		InstallerRoleArnFlag,
		"",
		"STS Role ARN with get secrets permission.",
	)

	interactive.AddModeFlag(cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
	arguments.AddRegionFlag(flags)
	output.AddFlag(cmd)

	return cmd
}

const (
	maxLengthUserPrefix = 15

	rawFilesFlag   = "raw-files"
	userPrefixFlag = "prefix"
	managedFlag    = "managed"
)

func checkInteractiveModeNeeded(cmd *cobra.Command) {
	modeNotChanged := !cmd.Flags().Changed("mode")
	if modeNotChanged && !cmd.Flags().Changed(rawFilesFlag) {
		interactive.Enable()
		return
	}
	oidcConfigKindNotSet := !cmd.Flags().Changed(managedFlag)
	if oidcConfigKindNotSet && !confirm.Yes() {
		interactive.Enable()
		return
	}
	modeIsAuto := cmd.Flag("mode").Value.String() == interactive.ModeAuto
	installerRoleArnNotSet := (!cmd.Flags().Changed(InstallerRoleArnFlag) || args.installerRoleArn == "") &&
		!confirm.Yes()
	if !args.managed && (modeNotChanged || (modeIsAuto && installerRoleArnNotSet)) {
		interactive.Enable()
		return
	}
}

func CreateOidcConfigRunner() rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		mode, err := interactive.GetMode()
		if err != nil {
			return err
		}

		// Get AWS region
		region, err := aws.GetRegion(arguments.GetRegion())
		if err != nil {
			return fmt.Errorf("Error getting region: %v", err)
		}
		args.region = region

		checkInteractiveModeNeeded(cmd)

		if interactive.Enabled() && !cmd.Flags().Changed(managedFlag) {
			args.managed = confirm.Prompt(true, "Would you like to create a Managed (Red Hat hosted) OIDC Configuration")
		}

		if args.rawFiles && mode != "" {
			return fmt.Errorf("--%s param is not supported alongside --mode param.", rawFilesFlag)
		}

		if args.rawFiles && args.installerRoleArn != "" {
			return fmt.Errorf("--%s param is not supported alongside --%s param", rawFilesFlag, InstallerRoleArnFlag)
		}

		if args.rawFiles && args.managed {
			return fmt.Errorf("--%s param is not supported alongside --%s param", rawFilesFlag, managedFlag)
		}

		if !args.rawFiles && interactive.Enabled() && !cmd.Flags().Changed("mode") {
			question := "OIDC Config creation mode"
			if args.managed {
				runtime.Reporter.Warnf("For a managed OIDC Config only auto mode is supported. " +
					"However, you may choose the provider creation mode")
				question = "OIDC Provider creation mode"
			}
			mode, err = interactive.GetOptionMode(cmd, mode, question)
			if err != nil {
				return fmt.Errorf("Expected a valid %s: %s", question, err)
			}
		}

		if output.HasFlag() && mode != "" && mode != interactive.ModeAuto {
			return fmt.Errorf("--output param is not supported outside auto mode.")
		}

		if args.managed && args.userPrefix != "" {
			return fmt.Errorf("--%s param is not supported for managed OIDC config", userPrefixFlag)
		}

		if args.managed && args.installerRoleArn != "" {
			return fmt.Errorf("--%s param is not supported for managed OIDC config", InstallerRoleArnFlag)
		}

		if !args.managed {
			if !args.rawFiles {
				if !output.HasFlag() && runtime.Reporter.IsTerminal() {
					runtime.Reporter.Infof("This command will create a S3 bucket populating it with documents " +
						"to be compliant with OIDC protocol. It will also create a Secret in Secrets Manager containing the private key")
				}
				if mode == interactive.ModeAuto && (interactive.Enabled() || (confirm.Yes() && args.installerRoleArn == "")) {
					args.installerRoleArn = interactiveRoles.
						GetInstallerRoleArn(
							runtime,
							cmd,
							args.installerRoleArn,
							MinorVersionForGetSecret,
							runtime.AWSClient.FindRoleARNs,
						)
				}
				if interactive.Enabled() {
					prefix, err := interactive.GetString(interactive.Input{
						Question:   "Prefix for OIDC",
						Help:       cmd.Flags().Lookup(userPrefixFlag).Usage,
						Default:    args.userPrefix,
						Validators: []interactive.Validator{interactive.MaxLength(maxLengthUserPrefix)},
					})
					if err != nil {
						return fmt.Errorf("Expected a valid prefix for the configuration: %s", err)
					}
					args.userPrefix = prefix
				}
				roleName, _ := aws.GetResourceIdFromARN(args.installerRoleArn)
				if roleName != "" {
					if !output.HasFlag() && runtime.Reporter.IsTerminal() && mode == interactive.ModeAuto {
						runtime.Reporter.Infof("Using %s for the installer role", args.installerRoleArn)
					}
					err := aws.ARNValidator(args.installerRoleArn)
					if err != nil {
						return fmt.Errorf("Expected a valid ARN: %s", err)
					}
					roleExists, _, err := runtime.AWSClient.CheckRoleExists(roleName)
					if err != nil {
						return fmt.Errorf(
							"There was a problem checking if role '%s' exists: %v",
							args.installerRoleArn,
							err,
						)
					}
					if !roleExists {
						return fmt.Errorf("Role '%s' does not exist", args.installerRoleArn)
					}
					isValid, err := runtime.AWSClient.ValidateAccountRoleVersionCompatibility(
						roleName, aws.InstallerAccountRole, MinorVersionForGetSecret)
					if err != nil {
						return fmt.Errorf("There was a problem listing role tags: %v", err)
					}
					if !isValid {
						return fmt.Errorf(
							"Role '%s' is not of minimum version '%s'",
							args.installerRoleArn,
							MinorVersionForGetSecret,
						)
					}
				}
			}

			args.userPrefix = strings.Trim(args.userPrefix, " \t")

			if len([]rune(args.userPrefix)) > maxLengthUserPrefix {
				return fmt.Errorf("Expected a valid prefix for the configuration: "+
					"length of prefix is limited to %d characters", maxLengthUserPrefix)
			}
		}

		oidcConfigInput := oidcconfigs.OidcConfigInput{}
		if !args.managed {
			oidcConfigInput, err = oidcconfigs.BuildOidcConfigInput(args.userPrefix, args.region)
			if err != nil {
				return err
			}
		}

		oidcConfigStrategy, err := getOidcConfigStrategy(mode, &oidcConfigInput)
		if err != nil {
			return err
		}
		oidcConfigId, err := oidcConfigStrategy.execute(runtime)
		if err != nil {
			return err
		}
		if !args.rawFiles && oidcConfigId != "" {
			cmd := oidcprovider.CreateOidcProvider.NewCommand()
			args := []string{"--oidc-config-id", oidcConfigId, "--mode", mode}
			cmd.ParseFlags(args)
			runner := oidcprovider.CreateOidcProvider.Runner()
			err = runner(nil, runtime, cmd, args)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

type CreateOidcConfigStrategy interface {
	execute(runtime *rosa.Runtime) (string, error)
}

type CreateUnmanagedOidcConfigRawStrategy struct {
	oidcConfig *oidcconfigs.OidcConfigInput
}

func (s *CreateUnmanagedOidcConfigRawStrategy) execute(runtime *rosa.Runtime) (string, error) {
	bucketName := s.oidcConfig.BucketName
	discoveryDocument := s.oidcConfig.DiscoveryDocument
	jwks := s.oidcConfig.Jwks
	privateKey := s.oidcConfig.PrivateKey
	privateKeyFilename := s.oidcConfig.PrivateKeyFilename
	err := helper.SaveDocument(string(privateKey), privateKeyFilename)
	if err != nil {
		return "", fmt.Errorf("There was a problem saving private key to a file: %s", err)
	}
	discoveryDocumentFilename := fmt.Sprintf("discovery-document-%s.json", bucketName)
	err = helper.SaveDocument(discoveryDocument, discoveryDocumentFilename)
	if err != nil {
		return "", fmt.Errorf("There was a problem saving discovery document to a file: %s", err)
	}
	jwksFilename := fmt.Sprintf("jwks-%s.json", bucketName)
	err = helper.SaveDocument(string(jwks[:]), jwksFilename)
	if err != nil {
		return "", fmt.Errorf("There was a problem saving JSON Web Key Set to a file: %s", err)
	}
	if !output.HasFlag() && runtime.Reporter.IsTerminal() {
		runtime.Reporter.Infof(
			"Please refer to documentation to use generated files to create an OIDC compliant configuration.",
		)
	}

	return "", nil
}

type CreateUnmanagedOidcConfigAutoStrategy struct {
	oidcConfig *oidcconfigs.OidcConfigInput
}

const (
	discoveryDocumentKey = ".well-known/openid-configuration"
	jwksKey              = "keys.json"
)

func (s *CreateUnmanagedOidcConfigAutoStrategy) execute(runtime *rosa.Runtime) (string, error) {
	bucketUrl := s.oidcConfig.IssuerUrl
	bucketName := s.oidcConfig.BucketName
	discoveryDocument := s.oidcConfig.DiscoveryDocument
	jwks := s.oidcConfig.Jwks
	privateKey := s.oidcConfig.PrivateKey
	privateKeySecretName := s.oidcConfig.PrivateKeySecretName
	installerRoleArn := args.installerRoleArn
	var spin *spinner.Spinner
	if !output.HasFlag() && runtime.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		runtime.Reporter.Infof("Setting up unmanaged OIDC configuration '%s'", bucketName)
	}
	if spin != nil {
		spin.Start()
	}
	err := runtime.AWSClient.CreateS3Bucket(bucketName, args.region)
	if err != nil {
		return "", fmt.Errorf("There was a problem creating S3 bucket '%s': %s", bucketName, err)
	}
	err = runtime.AWSClient.PutPublicReadObjectInS3Bucket(
		bucketName, strings.NewReader(discoveryDocument), discoveryDocumentKey)
	if err != nil {
		return "", fmt.Errorf("There was a problem populating discovery "+
			"document to S3 bucket '%s': %s", bucketName, err)
	}
	err = runtime.AWSClient.PutPublicReadObjectInS3Bucket(bucketName, bytes.NewReader(jwks), jwksKey)
	if err != nil {
		if spin != nil {
			spin.Stop()
		}
		return "", fmt.Errorf("There was a problem populating JWKS "+
			"to S3 bucket '%s': %s", bucketName, err)
	}
	secretARN, err := runtime.AWSClient.CreateSecretInSecretsManager(privateKeySecretName, string(privateKey[:]))
	if err != nil {
		return "", fmt.Errorf("There was a problem saving private key to secrets manager: %s", err)
	}
	oidcConfig, err := v1.NewOidcConfig().
		Managed(false).
		SecretArn(secretARN).
		IssuerUrl(bucketUrl).
		InstallerRoleArn(installerRoleArn).
		Build()
	if err == nil {
		oidcConfig, err = runtime.OCMClient.CreateOidcConfig(oidcConfig)
	}
	if err != nil {
		if spin != nil {
			spin.Stop()
		}
		return "", fmt.Errorf("There was a problem building your unmanaged OIDC Configuration %v.\n"+
			"Please refer to documentation and try again through:\n"+
			"\trosa register oidc-config --issuer-url %s --secret-arn %s --role-arn %s",
			err, bucketUrl, secretARN, installerRoleArn)
	}
	if output.HasFlag() {
		err = output.Print(oidcConfig)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	if runtime.Reporter.IsTerminal() {
		if spin != nil {
			spin.Stop()
		}
		output := fmt.Sprintf(InformOperatorRolesOutput, oidcConfig.ID())
		runtime.Reporter.Infof(output)
	}

	return oidcConfig.ID(), nil
}

type CreateUnmanagedOidcConfigManualStrategy struct {
	oidcConfig *oidcconfigs.OidcConfigInput
}

func (s *CreateUnmanagedOidcConfigManualStrategy) execute(runtime *rosa.Runtime) (string, error) {
	commands := []string{}
	bucketName := s.oidcConfig.BucketName
	discoveryDocument := s.oidcConfig.DiscoveryDocument
	jwks := s.oidcConfig.Jwks
	privateKey := s.oidcConfig.PrivateKey
	privateKeyFilename := s.oidcConfig.PrivateKeyFilename
	privateKeySecretName := s.oidcConfig.PrivateKeySecretName
	err := helper.SaveDocument(string(privateKey), privateKeyFilename)
	if err != nil {
		return "", fmt.Errorf("There was a problem saving private key to a file: %s", err)
	}
	createBucketConfig := ""
	if args.region != aws.DefaultRegion {
		createBucketConfig = fmt.Sprintf("LocationConstraint=%s", args.region)
	}
	createS3BucketCommand := awscb.NewS3ApiCommandBuilder().
		SetCommand(awscb.CreateBucket).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.CreateBucketConfiguration, createBucketConfig).
		AddParam(awscb.Region, args.region).
		Build()
	commands = append(commands, createS3BucketCommand)

	putBucketTaggingCommand := awscb.NewS3ApiCommandBuilder().
		SetCommand(awscb.PutBucketTagging).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.Tagging, fmt.Sprintf("'TagSet=[{Key=%s,Value=%s}]'", tags.RedHatManaged, tags.True)).
		Build()
	commands = append(commands, putBucketTaggingCommand)

	PutPublicAccessBlockCommand := awscb.NewS3ApiCommandBuilder().
		SetCommand(awscb.PutPublicAccessBlock).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.PublicAccessBlockConfiguration,
			"BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=false,RestrictPublicBuckets=false").
		Build()
	commands = append(commands, PutPublicAccessBlockCommand)

	readOnlyPolicyFilename := fmt.Sprintf("readOnlyPolicy-%s.json", bucketName)
	err = helper.SaveDocument(fmt.Sprintf(aws.ReadOnlyAnonUserPolicyTemplate, bucketName), readOnlyPolicyFilename)
	if err != nil {
		return "", fmt.Errorf("There was a problem saving bucket policy document to a file: %s", err)
	}
	putBucketBucketPolicyCommand := awscb.NewS3ApiCommandBuilder().
		SetCommand(awscb.PutBucketPolicy).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.Policy, fmt.Sprintf("file://%s", readOnlyPolicyFilename)).
		Build()
	commands = append(commands, putBucketBucketPolicyCommand)
	commands = append(commands, fmt.Sprintf("rm %s", readOnlyPolicyFilename))

	discoveryDocumentFilename := fmt.Sprintf("discovery-document-%s.json", bucketName)
	err = helper.SaveDocument(discoveryDocument, discoveryDocumentFilename)
	if err != nil {
		return "", fmt.Errorf("There was a problem saving discovery document to a file: %s", err)
	}
	putDiscoveryDocumentCommand := awscb.NewS3ApiCommandBuilder().
		SetCommand(awscb.PutObject).
		AddParam(awscb.Body, fmt.Sprintf("./%s", discoveryDocumentFilename)).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.Key, discoveryDocumentKey).
		AddParam(awscb.Tagging, fmt.Sprintf("'%s=%s'", tags.RedHatManaged, tags.True)).
		Build()
	commands = append(commands, putDiscoveryDocumentCommand)
	commands = append(commands, fmt.Sprintf("rm %s", discoveryDocumentFilename))
	jwksFilename := fmt.Sprintf("jwks-%s.json", bucketName)
	err = helper.SaveDocument(string(jwks[:]), jwksFilename)
	if err != nil {
		return "", fmt.Errorf("There was a problem saving JSON Web Key Set to a file: %s", err)
	}
	putJwksCommand := awscb.NewS3ApiCommandBuilder().
		SetCommand(awscb.PutObject).
		AddParam(awscb.Body, fmt.Sprintf("./%s", jwksFilename)).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.Key, jwksKey).
		AddParam(awscb.Tagging, fmt.Sprintf("'%s=%s'", tags.RedHatManaged, tags.True)).
		Build()
	commands = append(commands, putJwksCommand)
	commands = append(commands, fmt.Sprintf("rm %s", jwksFilename))
	createSecretCommand := awscb.NewSecretsManagerCommandBuilder().
		SetCommand(awscb.CreateSecret).
		AddParam(awscb.Name, privateKeySecretName).
		AddParam(awscb.SecretString, fmt.Sprintf("file://%s", privateKeyFilename)).
		AddParam(awscb.Description, fmt.Sprintf("\"Secret for %s\"", bucketName)).
		AddParam(awscb.Region, args.region).
		AddTags(map[string]string{
			tags.RedHatManaged: "true",
		}).
		Build()
	commands = append(commands, createSecretCommand)
	commands = append(commands, fmt.Sprintf("rm %s", privateKeyFilename))
	fmt.Println(awscb.JoinCommands(commands))
	if runtime.Reporter.IsTerminal() {
		runtime.Reporter.Infof("Please run commands above to generate OIDC compliant configuration in your AWS account. " +
			"To register this OIDC Configuration, please run the following command:\n" +
			"rosa register oidc-config\n" +
			"For more information please refer to the documentation")
	}

	return "", nil
}

type CreateManagedOidcConfigAutoStrategy struct {
	oidcConfigInput *oidcconfigs.OidcConfigInput
}

func (s *CreateManagedOidcConfigAutoStrategy) execute(runtime *rosa.Runtime) (string, error) {
	var spin *spinner.Spinner
	if !output.HasFlag() && runtime.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		runtime.Reporter.Infof("Setting up managed OIDC configuration")
	}
	if spin != nil {
		spin.Start()
	}
	oidcConfig, err := v1.NewOidcConfig().Managed(true).Build()
	if err != nil {
		return "", fmt.Errorf("There was a problem building the managed OIDC Configuration: %v", err)
	}
	oidcConfig, err = runtime.OCMClient.CreateOidcConfig(oidcConfig)
	if err != nil {
		if spin != nil {
			spin.Stop()
		}
		return "", fmt.Errorf("There was a problem registering your managed OIDC Configuration: %v", err)
	}
	s.oidcConfigInput.IssuerUrl = oidcConfig.IssuerUrl()
	if output.HasFlag() {
		err = output.Print(oidcConfig)
		if err != nil {
			return "", fmt.Errorf("%s", err)
		}
		return "", nil
	}
	if runtime.Reporter.IsTerminal() {
		if spin != nil {
			spin.Stop()
		}
		output := fmt.Sprintf(InformOperatorRolesOutput, oidcConfig.ID())
		runtime.Reporter.Infof(output)
	}

	return oidcConfig.ID(), nil
}

func getOidcConfigStrategy(mode string, input *oidcconfigs.OidcConfigInput) (CreateOidcConfigStrategy, error) {
	if args.rawFiles {
		return &CreateUnmanagedOidcConfigRawStrategy{oidcConfig: input}, nil
	}
	if args.managed {
		return &CreateManagedOidcConfigAutoStrategy{oidcConfigInput: input}, nil
	}
	switch mode {
	case interactive.ModeAuto:
		return &CreateUnmanagedOidcConfigAutoStrategy{oidcConfig: input}, nil
	case interactive.ModeManual:
		return &CreateUnmanagedOidcConfigManualStrategy{oidcConfig: input}, nil
	default:
		return nil, weberr.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
	}
}
