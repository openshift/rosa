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
	// nolint:gosec

	//#nosec GSC-G505 -- Import blacklist: crypto/sha1

	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zgalor/weberr"
	"gopkg.in/square/go-jose.v2"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	DEFAULT_LENGTH_RANDOM_LABEL = 4
)

var args struct {
	region string
}

var Cmd = &cobra.Command{
	Use:     "oidc-config",
	Aliases: []string{"oidcconfig"},
	Short:   "Create OIDC config for an STS cluster.",
	Long: "Create OIDC config in an S3 bucket for the " +
		"client AWS account and populates it to be compliant with OIDC protocol.",
	Example: `  # Create OIDC config rosa create oidc-config`,
	Run:     run,
}

func init() {
	flags := Cmd.Flags()

	aws.AddModeFlag(Cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
	arguments.AddRegionFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS()
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
			Question: "OIDC config creation mode",
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

	oidcConfigStrategy, err := getOidcConfigStrategy(mode)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
	bucketUrl, privateKeyFilename := oidcConfigStrategy.execute(r)
	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("To create a cluster with this oidc config, please run:\n"+
			"rosa create cluster --sts --oidc-endpoint-url %s "+
			"--bound-service-account-signing-key-path ./%s", bucketUrl, privateKeyFilename)
	}
}

type CreateOidcConfigStrategy interface {
	execute(r *rosa.Runtime) (string, string)
}

type CreateOidcConfigAutoStrategy struct{}

const (
	discoveryDocumentKey = ".well-known/openid-configuration"
	jwksKey              = "keys.json"
)

func (s *CreateOidcConfigAutoStrategy) execute(r *rosa.Runtime) (string, string) {
	randomLabel := helper.RandomLabel(DEFAULT_LENGTH_RANDOM_LABEL)
	bucketName := fmt.Sprintf("oidc-%s", randomLabel)
	bucketUrl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucketName, args.region)
	err := r.AWSClient.CreateS3Bucket(bucketName, args.region)
	if err != nil {
		r.Reporter.Errorf("There was a problem creating S3 bucket '%s': %s", bucketName, err)
		os.Exit(1)
	}
	privateKey, publicKey, err := createKeyPair()
	if err != nil {
		r.Reporter.Errorf("There was a problem generating key pair: %s", err)
		os.Exit(1)
	}
	privateKeyFilename := fmt.Sprintf("private-key-%s.key", bucketName)
	err = helper.SaveDocument(string(privateKey[:]), privateKeyFilename)
	if err != nil {
		r.Reporter.Errorf("There was a problem saving private key to a file: %s", err)
		os.Exit(1)
	}
	discoveryDocument := generateDiscoveryDocument(bucketUrl)
	err = r.AWSClient.PutPublicReadObjectInS3Bucket(
		bucketName, strings.NewReader(discoveryDocument), discoveryDocumentKey)
	if err != nil {
		r.Reporter.Errorf("There was a problem populating discovery "+
			"document to S3 bucket '%s': %s", bucketName, err)
		os.Exit(1)
	}
	jwks, err := buildJSONWebKeySet(publicKey)
	if err != nil {
		r.Reporter.Errorf("There was a problem generating JSON Web Key Set: %s", err)
		os.Exit(1)
	}
	err = r.AWSClient.PutPublicReadObjectInS3Bucket(bucketName, bytes.NewReader(jwks), jwksKey)
	if err != nil {
		r.Reporter.Errorf("There was a problem populating JWKS "+
			"to S3 bucket '%s': %s", bucketName, err)
		os.Exit(1)
	}
	return bucketUrl, privateKeyFilename
}

type CreateOidcConfigManualStrategy struct{}

func (s *CreateOidcConfigManualStrategy) execute(r *rosa.Runtime) (string, string) {
	commands := []string{}
	randomLabel := helper.RandomLabel(DEFAULT_LENGTH_RANDOM_LABEL)
	bucketName := fmt.Sprintf("oidc-%s", randomLabel)
	bucketUrl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucketName, args.region)
	createBucketConfig := ""
	if args.region != "us-east-1" {
		createBucketConfig = fmt.Sprintf("LocationConstraint=%s", args.region)
	}
	createS3BucketCommand := awscb.NewS3CommandBuilder().
		SetCommand(awscb.CreateBucket).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.CreateBucketConfiguration, createBucketConfig).
		Build()
	commands = append(commands, createS3BucketCommand)
	privateKey, publicKey, err := createKeyPair()
	if err != nil {
		r.Reporter.Errorf("There was a problem generating key pair: %s", err)
		os.Exit(1)
	}
	privateKeyFilename := fmt.Sprintf("private-key-%s.key", bucketName)
	err = helper.SaveDocument(string(privateKey[:]), privateKeyFilename)
	if err != nil {
		r.Reporter.Errorf("There was a problem saving private key to a file: %s", err)
		os.Exit(1)
	}
	discoveryDocument := generateDiscoveryDocument(bucketUrl)
	discoveryDocumentFilename := fmt.Sprintf("discovery-document-%s.json", bucketName)
	err = helper.SaveDocument(discoveryDocument, discoveryDocumentFilename)
	if err != nil {
		r.Reporter.Errorf("There was a problem saving discovery document to a file: %s", err)
		os.Exit(1)
	}
	putDiscoveryDocumentCommand := awscb.NewS3CommandBuilder().
		SetCommand(awscb.PutObject).
		AddParam(awscb.Acl, "public-read").
		AddParam(awscb.Body, fmt.Sprintf("./%s", discoveryDocumentFilename)).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.Key, discoveryDocumentKey).
		Build()
	commands = append(commands, putDiscoveryDocumentCommand)
	jwks, err := buildJSONWebKeySet(publicKey)
	if err != nil {
		r.Reporter.Errorf("There was a problem generating JSON Web Key Set: %s", err)
		os.Exit(1)
	}
	jwksFilename := fmt.Sprintf("jwks-%s.json", bucketName)
	err = helper.SaveDocument(string(jwks[:]), jwksFilename)
	if err != nil {
		r.Reporter.Errorf("There was a problem saving JSON Web Key Set to a file: %s", err)
		os.Exit(1)
	}
	putJwksCommand := awscb.NewS3CommandBuilder().
		SetCommand(awscb.PutObject).
		AddParam(awscb.Acl, "public-read").
		AddParam(awscb.Body, fmt.Sprintf("./%s", jwksFilename)).
		AddParam(awscb.Bucket, bucketName).
		AddParam(awscb.Key, jwksKey).
		Build()
	commands = append(commands, putJwksCommand)
	fmt.Println(awscb.JoinCommands(commands))
	return bucketUrl, privateKeyFilename
}

func getOidcConfigStrategy(mode string) (CreateOidcConfigStrategy, error) {
	switch mode {
	case aws.ModeAuto:
		return &CreateOidcConfigAutoStrategy{}, nil
	case aws.ModeManual:
		return &CreateOidcConfigManualStrategy{}, nil
	default:
		return nil, weberr.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
	}
}

func createKeyPair() ([]byte, []byte, error) {
	bitSize := 4096

	// Generate RSA keypair
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate private key")
	}
	encodedPrivateKey := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Generate public key from private keypair
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate public key from private")
	}
	encodedPublicKey := pem.EncodeToMemory(&pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   pubKeyBytes,
	})

	return encodedPrivateKey, encodedPublicKey, nil
}

type JSONWebKeySet struct {
	Keys []jose.JSONWebKey `json:"keys"`
}

// buildJSONWebKeySet builds JSON web key set from the public key
func buildJSONWebKeySet(publicKeyContent []byte) ([]byte, error) {
	block, _ := pem.Decode(publicKeyContent)
	if block == nil {
		return nil, errors.Errorf("Failed to decode PEM file")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse key content")
	}

	var alg jose.SignatureAlgorithm
	switch publicKey.(type) {
	case *rsa.PublicKey:
		alg = jose.RS256
	default:
		return nil, errors.Errorf("Public key is not of type RSA")
	}

	kid, err := keyIDFromPublicKey(publicKey)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch key ID from public key")
	}

	var keys []jose.JSONWebKey
	keys = append(keys, jose.JSONWebKey{
		Key:       publicKey,
		KeyID:     kid,
		Algorithm: string(alg),
		Use:       "sig",
	})

	keySet, err := json.MarshalIndent(JSONWebKeySet{Keys: keys}, "", "    ")
	if err != nil {
		return nil, errors.Wrapf(err, "JSON encoding of web key set failed")
	}

	return keySet, nil
}

// keyIDFromPublicKey derives a key ID non-reversibly from a public key
// reference: https://github.com/kubernetes/kubernetes/blob/v1.21.0/pkg/serviceaccount/jwt.go#L89-L111
func keyIDFromPublicKey(publicKey interface{}) (string, error) {
	publicKeyDERBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to serialize public key to DER format")
	}

	hasher := crypto.SHA256.New()
	hasher.Write(publicKeyDERBytes)
	publicKeyDERHash := hasher.Sum(nil)

	keyID := base64.RawURLEncoding.EncodeToString(publicKeyDERHash)

	return keyID, nil
}

const (
	discoveryDocumentTemplate = `{
	"issuer": "%s",
	"jwks_uri": "%s/keys.json",
	"response_types_supported": [
		"id_token"
	],
	"subject_types_supported": [
		"public"
	],
	"id_token_signing_alg_values_supported": [
		"RS256"
	],
	"claims_supported": [
		"aud",
		"exp",
		"sub",
		"iat",
		"iss",
		"sub"
	]
}`
)

func generateDiscoveryDocument(bucketURL string) string {
	return fmt.Sprintf(discoveryDocumentTemplate, bucketURL, bucketURL)
}
