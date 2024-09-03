package constants

const (
	Yes                    = "Yes"
	No                     = "No"
	YStreamPreviousVersion = "y-1"
	TrueString             = "true"
)

// Ec2MetadataHttpTokens for hcp cluster
const (
	DefaultEc2MetadataHttpTokens  = "optional"
	RequiredEc2MetadataHttpTokens = "required"
	OptionalEc2MetadataHttpTokens = "optional"
)

var JumpAccounts = map[string]string{
	"production": "710019948333",
	"staging":    "644306948063",
}
