package commandbuilder

import (
	"fmt"
	"sort"
	"strings"
)

const ParamNewLineSeparator = " \\\n"

type Service string

const (
	IAM Service = "iam"
	S3  Service = "s3api"
)

type Command string

const (
	//IAM
	CreateRole                    Command = "create-role"
	DeleteRole                    Command = "delete-role"
	CreatePolicy                  Command = "create-policy"
	DeletePolicy                  Command = "delete-policy"
	CreatePolicyVersion           Command = "create-policy-version"
	DeleteRolePolicy              Command = "delete-role-policy"
	AttachRolePolicy              Command = "attach-role-policy"
	DetachRolePolicy              Command = "detach-role-policy"
	TagPolicy                     Command = "tag-policy"
	TagRole                       Command = "tag-role"
	CreateOpenIdConnectProvider   Command = "create-open-id-connect-provider"
	DeleteOpenIdConnectProvider   Command = "delete-open-id-connect-provider"
	DeleteRolePermissionsBoundary Command = "delete-role-permissions-boundary"
	//S3
	CreateBucket Command = "create-bucket"
	PutObject    Command = "put-object"
)

type Param string

const (
	//IAM
	Tags                     Param = "tags"
	RoleName                 Param = "role-name"
	AssumeRolePolicyDocument Param = "assume-role-policy-document"
	PermissionsBoundary      Param = "permissions-boundary"
	Path                     Param = "path"
	PolicyName               Param = "policy-name"
	PolicyDocument           Param = "policy-document"
	PolicyArn                Param = "policy-arn"
	Url                      Param = "url"
	ClientIdList             Param = "client-id-list"
	ThumbprintList           Param = "thumbprint-list"
	OpenIdConnectProviderArn Param = "open-id-connect-provider-arn"
	SetAsDefault             Param = "set-as-default"

	//S3
	Bucket                    Param = "bucket"
	Region                    Param = "region"
	Acl                       Param = "acl"
	CreateBucketConfiguration Param = "create-bucket-configuration"
	Body                      Param = "body"
	Key                       Param = "key"
)

type CommandBuilder struct {
	service Service
	command Command
	params  []string
	tags    map[string]string
}

func (b *CommandBuilder) SetService(awsService Service) *CommandBuilder {
	b.service = awsService
	return b
}

func (b *CommandBuilder) SetCommand(awsCommand Command) *CommandBuilder {
	b.command = awsCommand
	return b
}

func (b *CommandBuilder) AddParam(awsParam Param, value string) *CommandBuilder {
	if value != "" {
		b.params = append(b.params, createParamString(awsParam, value))
	}
	return b
}

func (b *CommandBuilder) AddTags(value map[string]string) *CommandBuilder {
	if b.tags == nil {
		b.tags = make(map[string]string, len(value))
	}
	for k, v := range value {
		b.tags[k] = v
	}
	return b
}

func (b *CommandBuilder) AddParamNoValue(awsParam Param) *CommandBuilder {
	b.params = append(b.params, fmt.Sprintf("\t--%s", awsParam))
	return b
}

func (b *CommandBuilder) Build() string {
	serviceString := ""
	if b.service != "" {
		serviceString = string(b.service)
	}

	commandString := ""
	if b.command != "" {
		commandString = fmt.Sprintf(" %s%s", b.command, ParamNewLineSeparator)
	}

	paramsString := ""
	if len(b.tags) != 0 {
		b.AddParam(Tags, createTags(b.tags))
	}
	if len(b.params) != 0 {
		sort.Strings(b.params)
		paramsString = strings.Join(b.params, ParamNewLineSeparator)
	}
	return fmt.Sprintf(
		"aws %s%s%s",
		serviceString,
		commandString,
		paramsString,
	)
}

func NewIAMCommandBuilder() *CommandBuilder {
	return &CommandBuilder{service: IAM}
}

func NewS3CommandBuilder() *CommandBuilder {
	return &CommandBuilder{service: S3}
}

func createParamString(awsParam Param, value string) string {
	return fmt.Sprintf("\t--%s %s", awsParam, value)
}

func createTags(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		keys = append(keys, fmt.Sprintf("Key=%s,Value=%s", k, v))
	}
	sort.Slice(keys, func(i, j int) bool {
		l1, l2 := len(keys[i]), len(keys[j])
		if l1 != l2 {
			return l1 < l2
		}
		return keys[i] < keys[j]
	})
	return strings.Join(keys, " ")
}

func JoinCommands(commands []string) string {
	return strings.Join(commands, "\n\n")
}
