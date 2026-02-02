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

const (
	StageURL      = "https://console.dev.redhat.com/openshift/details/s/"
	ProductionURL = "https://console.redhat.com/openshift/details/s/"
	StageEnv      = "https://api.stage.openshift.com"
	ProductionEnv = "https://api.openshift.com"
)

const (
	BillingAccount        = "090777400063"
	ChangedBillingAccount = "487962084830"
)

const (
	OCMRolePreifx  = "rosacli-ocm-role"
	UserRolePreifx = "rosacli-user-role"
)
