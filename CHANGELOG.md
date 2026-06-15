## 1.2.64 (15 Jun, 2026)

FEATURES:
   * Expose CreateOCMRole to CAPA (#3262)
   * Added support for --no-console to rosa create ocm-role (#3252)
   * Updated rosa list ocm-role to display console access

ENHANCEMENTS:
 * Bug fixes
   * Fixing id:38788,id:56783,id:60278,id:73391,id:81399
 * Chores
   * Fix linter issues in pkg/aws/{helpers,policies}

## 1.2.63 (15 May, 2026)

ENHANCEMENTS:
 * Bug fixes
   * Default machinepools to m7i.xlarge
   * Refactor proxy URL validation logic
 * Ci
   * Trust e2e git worktree
   * Stabilize e2e image builds

## 1.2.62 (04 May, 2026)

FEATURES:
   * Support --with-feature flag to filter machinetypes on features
   * Add cluster channel support
   * Implement validation for managed policies in prod
   * Allow min-replicas=0 for HCP nodepool autoscaling

ENHANCEMENTS:
 * Bug fixes
   * Refactor proxy URL validation logic
   * Support set min_replicas=0 when editing nodepool
   * Format --output json errors as JSON for whoami
   * Remove/revise validation for hcp cluster nodes size
   * Add channel flag in the prompt message for creation in future
   * Rename logforwarding_test_suite.go to logforwarding_suite_test.go
   * Remove extra question mark in the prompt
   * Revise the query string to avoid leading wildcards in LIKE statement
 * Chores
   * Update OWNERS approvers
   * Add amandahla to approvers
   * Update Go version to 1.25.8 in order to fix
   * Docs update about script
   * Update git hooks and add local verification scripts
   * Updated template for PR's
 * Documentation
   * Remove duplicate contributor guide
   * Expand agent readiness guidance
   * Address agent guidance review comments
   * Add repo-local agent guidance
 * Bump
   * Ocm-sdk-go to v0.1.499: vendor
   * Ocm-sdk-go to v0.1.499

## 1.2.61 (03 Mar, 2026)

FEATURES:
   * Implement validation for managed policies in prod
   * Added indication of FIPS state on describe command

ENHANCEMENTS:
 * Bug fixes
   * Created helper for role validation
   * Removed gating around CapRez + autoscaling
   * BuildCommand to include --etcd-encryption-kms-arn when FIPS is enabled
   * Improve error message for proxy passwords with special characters
   * Removed capacity preference field when empty
   * Add missing capacity-reservation-preference field in 'rosa describe machibepool'
   * Adjust when ze property not available
   * CIDR range hardcoded value
   * Add output flag to create/logfwder
   * Remove tech preview from zero egress status
   * Check for any linked OCM or user roles
   * Added autonode status on the describe command
   * Improved AutoNode IAM Role error message
   * Update logForwarder
 * Chores
   * Bump ocm-sdk-go to v0.1.497 for version Channel support
   * Updated the owners file
   * Bump ocm-common to v0.0.32 for deprecation middleware
 * Documentation
   * Add intro text to clibyexample template

## 1.2.60 (09 Jan, 2026)

ENHANCEMENTS:
 * Bug fixes
   * Removing fedramp checks that prevent setting HCP billing accounts

## 1.2.58 (18 Dec, 2025)

FEATURES:
   * Block capacity-reservation-preference on govcloud
   * Disable usage of logfwding on fedramp
   * Block CapRez in govcloud on a flag level
   * Create command for logforwarders
   * List command for logforwarders
   * Delete command for logforwarders
   * Technology Preview gating for WinLI
   * Edit log forwarder command
   * Describe log forwarder command
   * Log Forwarding for HCP day1 support
   * Log Forwarding for HCP day1 support
   * Add logForward to cluster
   * Introduce filtering for winli create/nodepool
   * Validation error for caprez on classic
   * Describe/nodepool support for image type
   * Capacity reservation preference for create/nodepool
   * Win-Li support (machinepool type flag)
   * Pull versions from CVO instead of CIS for roles
   * Use Cluster.Versions for classic list/upgrades
   * Deprecate available upgrades for rosa list versions

ENHANCEMENTS:
 * Bug fixes
   * Attach to builder, instead of replace
   * Edit logforwarder from yaml only if same type
   * Allow both S3 and CW day2 create
   * Edit/logforwarder always use CW/S3 yaml fields
   * Allow empty groups, if so, require apps
   * Change regex to support new ID convention
   * Add caprez preference to describe/machinepool
   * Allow empty list of apps for cloudwatch
   * Support multiple groupLogVersions
   * Only set logForwarders on HCP clusters
   * Logforward index out of range
   * Fix all open CVEs on ROSA CLI
   * Linter fixes for !3117
   * Dont allow winli+fedramp
   * Only show available options for caprez preference
   * Send EITHER capres ID or preference to api
   * Fix typo in caprez help msg
   * Accidental assignment of incorrect value
   * Fix linter errors for #3093
   * Fix linter errors for #3091
   * Discontinue UWM for ROSA HCP clusters
   * Filter DNS domains by organization in list command
   * Hardcoded VPC CIDR values in cloudformation template
   * Only list the dns-domain under current organization
   * Handle HTTP errors in rosa download command and improve error messages
   * Ensure subnet fetching in all scenarios
   * Changed behavior of sts flag on promoted command
   * Remove proxy's special characters value to avoid meeting invalid value error
 * Chores
   * Bump SDK to v0.1.486
   * Update FedRAMP CI profile OCP versions
   * Bump SDK to 0.1.482
   * Bump Go to 1.24 + fix issues
   * Bump golangci-lint v2.6.1
   * Bump unneeded reviewers+approvers
   * Add tool to generate docs

## 1.2.57 (09 Oct, 2025)

FEATURES:
   * Add support for enabling AutoNode to 'rosa edit cluster'
   * Introduce billing account selection for classic
   * Unhide CapRes flag for m2
   * IDMS Type for ROSA-HCP Update

ENHANCEMENTS:
 * Bug fixes
   * Created verification to when subnets are not set on proxy
   * Revert "OCM-17465 | fix: created verification to when subnets are not set on proxy"
   * Added verifications to exclude fedramp from autonode feature
   * Bug with private API being broken for govcloud
   * Do not prompt capres in govcloud env
   * Changed messaging for validation error in HCP cluster
   * Created verification to when subnets are not set on proxy
   * It should not report error when create iamserviceaccount without setting '-path' miltiple times
 * Chores
   * Bump ROSA CLI version to 1.2.57

## 1.2.56 (10 Sep, 2025)

FEATURES:
   * Add hidden options for setting master/infra machine types with rosa create cluster
   * Remove attaching ec2 policy to worker role for zero egress
   * IDMS support for ROSA-HCP
   * Deprecate warnings for UWM + HCP
   * Introduce capacity reservation to nodepools
   * Deprecate user workload monitoring support msg
   * Add comprehensive IAM service account management commands
   * Deprecate user workload monitoring toggle
   * Introduce warning + dry run mode for upgrading clusters
   * Add support for handling deprecation headers in ROSA CLI

ENHANCEMENTS:
 * Bug fixes
   * Delete inline policies
   * Dlt/iamserviceaccount + unused options.go file
   * Fix delete interactive + auto modes
   * Prompt user for what input method they want
   * Correct delete prompt for iamserviceaccount
   * Fix broken ARN fetching for OIDC providers
   * Removed optional from subnet selection
   * Do not prompt for capacity reservation ID when autoscaling
   * Allow nodepool creation without capacity reservation id
   * Exclude public subnets for privatelink classic
   * Fix a couple tests that are failing in fedramp
   * Fix retrieving versions for listing CLI versions
 * Chores
   * Revert version to make HEAD
   * Update ocm-sdk-go to v0.1.476
   * Bump ROSA CLI version to 1.2.56
   * Update warning around upgrade dry runs
   * Bump SDK to 0.1.475
   * Bump SDK version and go mod+tidy
 * Documentation
   * Add Claude Code development guidelines and repository instructions
 * Other
   * Revert "OCM-17719 | feat: Deprecate user workload monitoring toggle"
   * Revert "OCM-17719 | feat: Deprecate user workload monitoring support to be st…"

## 1.2.55 (29 Jul, 2025)

FEATURES:
   * Added private ingress + private API prompts for HCP
   * Rename CAPA logger something more identifiable
   * Update help message for private-link to 'deprecate' for HCP
   * Update mirror and base releases folder URI
   * New public routines in cloudformation client
   * Introduce help messages for prompts missing them in create/cluster
   * Introduce warning around creating accountroles
   * Added new columns and flags to list machine pool command
   * Added private ingress + private API prompts for HCP
   * Expose functionality for creating Account/Operator roles and OIDC config/provider
   * Introduce warning + validation for govcloud env choice

ENHANCEMENTS:
 * Bug fixes
   * Fix broken map for loop
   * Fix kubelet configs and add policy template for govcloud
   * Revert package changes from #2906
   * Fix private+privateIngress clusters + validation
   * Pass in non-args privteIngress value
   * Introduce private API to private STS validation
   * Updated strings and tests to match
   * Remove check stopping private ingress changes
   * Bump Go version to 1.23.1
   * Only supply ingresses[0].listening when HCP
   * No region set for command which needs no region, `list/clusters`
   * Wrong OIDC Provider import
 * Chores
   * Bump ROSA CLI version to v1.2.55
   * Bump ocm-common dependency
   * Bump CLI version on master to 1.2.55
   * Change slack channel + handle in contributing guide
   * Bump OCM sdk version to 0.1.465 + vendor/tidy
 * Other
   * Revert "OCM-17311 | chore: Bump CLI version on master to 1.2.55"
   * Revert "OCM-17058 | feat: Update mirror and base releases folder URI"

## 1.2.54 (23 Jun, 2025)

FEATURES:
   * Enable HCP in govcloud environments
   * Removed duplicated-from-API validation (machinepools)
   * Added new warnings to account-role checks and validation for the red-hat-managed tag
   * Remove default channelgroup from edit/cluster
   * Introduce `-y` for migrating cluster network type
   * Bump konflux to check for release_1.2.53
   * Un-hide `channel-group` flag in create/cluster
   * Introduce channel-group in edit/cluster
   * Sort CF template parameters in help output

ENHANCEMENTS:
 * Bug fixes
   * Bump Go version to 1.23
   * Do not enable interactive if channel group is edited
   * Improved UX for error handling in private clusters
 * Chores
   * Change slack channel + handle in contributing guide
   * Edit makefile comment
   * Bump ROSA version to 1.2.53
   * Add konflux bot to OWNERS file
 * Other
   * Konflux build pipeline service account migration for rosa
   * Red Hat Konflux update rosa

## 1.2.53 (29 Apr, 2025)

FEATURES:
   * Introduce `-y` for migrating cluster network type
   * Added new warnings to account-role checks and validation for the red-hat-managed tag
   * Introduce `-y` for migrating cluster network type
   * Bump konflux to check for release_1.2.53
   * Bump jwt version, to avoid security flaw
   * Ability to specify 4 availability zones
   * Allow specifying availability zones for CF stack creation

ENHANCEMENTS:
 * Bug fixes
   * Bump Go version to 1.23
   * Bump Go version to 1.23
   * Fix AZ conditions
   * Explicit AZs take priority over number of AZs
 * Chores
   * Bump ROSA version to 1.2.53
   * Bump prerelease version

## 1.2.52 (27 Mar, 2025)

FEATURES:
   * /UX\ Output of `list account-roles` sorted by '-HCP-'

ENHANCEMENTS:
 * Bug fixes
   * Bug with upgrading cluster and missing account roles
 * Chores
   * Update version to v1.2.52-RC1

## 1.2.51 (26 Feb, 2025)

FEATURES:
   * Disable interactive mode when editing HCP autoscaler
   * Change help messages for autoscaler cmds
   * Add UX informational msgs when create/nodepool
   * HCP autoscaler simpler describe output
   * Support editing HCP cluster autoscaler
   * Remove HCP gate on `describe/autoscaler
   * Disable SDN->OVN migration flags for 1.2.50

ENHANCEMENTS:
 * Bug fixes
   * Calculate default NP values when create/edit np
   * Create+Edit/np info msg fixes with calculations
   * Spelling of flag in external auth provider
   * Rename key values used for migration API request
   * Change error output for failed inflight check
   * UX issue with default value for migration subnets
 * Chores
   * Update version to 1.2.51
   * Bump SDK to fix API bug
 * Other
   * Revert "OCM-13520 | feat: Disable SDN->OVN migration flags for 1.2.50"

## 1.2.50 (11 Feb, 2025)

FEATURES:
   * Disable SDN->OVN migration feature for 1.2.50
   * Disable SDN->OVN migration flags for 1.2.50
   * Add support for json/yml output with migrations describe
   * Change describe formatting and edit interactive mode
   * Edit/cluster support for SDN->OVN migrations
   * Describe cluster support for SDN -> OVN migration
   * Deprecate unused control-plane flag from upgrade/cluster
   * Removing validation for no changes on np
   * Adding extra info message for user to specify template-dir when calling custom template
   * Removing validation for no changes on np
   * Default values for CAS Max Nodes Total and validation added for count of mp worker nodes
   * Describe Cluster changes for hcpsharedvpc
   * Manual mode for by_cluster_key oproles for hcpsharedvpc
   * By_cluster_key operator role creation hcpsharedvpc
   * Allow use of flag for deleting hcpsharedvpc policies
   * Build rosa command for creating hcp shared vpc cluster
   * HCP Shared VPC create/cluster interactive mode
   * Create/cluster support for HCP shared VPC
   * Create/cluster validation for hcpsharedvpc flags
   * Add hcpsharedvpc flags + deprecation warnings to create/cluster
   * Add manual mode for deleting hcp sharedvpc policies [oproles]
   * Manual mode for deleting hcpsharedvpc account role policies
   * Delete operator roles auto mode (and changes to account roles)

ENHANCEMENTS:
 * Bug fixes
   * Rename key values used for migration API request
   * UX issue with default value for migration subnets
   * Do not error/exit when fail to describe migrations
   * Add profile flag to very network command
   * Do not exit when unable to save refresh token
   * Prompt users for role arn of assumed role when listing regions
   * Remove empty string checks for http/https proxy
   * Filter DNS domains when create/cluster if hosted CP
   * Make route53 role arn usable for classic oproles
   * Make vpcendpointrolearn and hcp HZID prompts hcp only
   * Remove duplicate addition of createRole command
   * Add required flags to oprole creation for cluster manual mode
   * Create/oproles manual- use path from user roles
   * Make additional-allowed-principals required for hcpsharedvpc
   * Replace deprecated flags when building cluster cmd
   * Oproles add path to hcpsharedvpc policy ARNs manual mode
   * Output formatting for deleting account roles
   * Include path in create acc/op roles manual mode
   * Delete accountroles hcpsharedvpc interactive
   * Use path flag with hcpsharedvpc policy creation
   * Do not create new policy version when policy is hcpsharedvpc
   * Combine all policies rather than eating the output
   * Swap manual+auto questions for dlt/accountroles
   * Fixing manual-mode + log event output
   * Swap function args to make usability straightforward
   * UX changes for create/accountroles default values
 * Chores
   * Bump SDK to fix API bug
   * Bump SDK 0.1.455 -> 0.1.456
   * Bump SDK version from 0.1.454 -> 0.1.455
   * Bump SDK version to v0.1.454
 * Other
   * Revert "OCM-13520 | feat: Disable SDN->OVN migration flags for 1.2.50"
   * Bump RC version 1->2
   * Revert "OCM-10147 | feat: removing validation for no changes on np"

## 1.2.49 (17 Dec, 2024)

FEATURES:
   * Adding extra info message for user to specify template-dir when calling custom template
   * Default values for CAS Max Nodes Total and validation added for count of mp worker nodes
   * Describe Cluster changes for hcpsharedvpc
   * Manual mode for by_cluster_key oproles for hcpsharedvpc
   * By_cluster_key operator role creation hcpsharedvpc
   * Allow use of flag for deleting hcpsharedvpc policies
   * Build rosa command for creating hcp shared vpc cluster
   * HCP Shared VPC create/cluster interactive mode
   * Create/cluster support for HCP shared VPC
   * Create/cluster validation for hcpsharedvpc flags
   * Add hcpsharedvpc flags + deprecation warnings to create/cluster
   * Add manual mode for deleting hcp sharedvpc policies [oproles]
   * Manual mode for deleting hcpsharedvpc account role policies
   * Delete operator roles auto mode (and changes to account roles)
   * Delete accountrole sharedvpc policies in auto/interactive
   * Manual mode for accountroles doesnt always create policies
   * Interactive mode for create/operatorroles (hcp shared vpc)
   * Interactive mode for create/accountroles
   * Manual mode for account roles with shared-vpc for hcp
   * Refactor op + acc roles sharedvpc policies
   * --hosted-cp functionality for rosa list dns-domains
   * Add HCP shared-vpc support to accountroles
   * Manual mode for create&delete operator-roles with hcp sharedvpc
   * HCP sharedVPC functionality to create/operatoroles
   * Adding extra validation to verify user is logged in
   * Adding default cf template for binary builds
   * Adding extra validation to verify user is logged in
   * Add --hosted-cp functionality to create/dns-domain

ENHANCEMENTS:
 * Bug fixes
   * Make route53 role arn usable for classic oproles
   * Make vpcendpointrolearn and hcp HZID prompts hcp only
   * Remove duplicate addition of createRole command
   * Add required flags to oprole creation for cluster manual mode
   * Create/oproles manual- use path from user roles
   * Output formatting for deleting account roles
   * Make additional-allowed-principals required for hcpsharedvpc
   * Replace deprecated flags when building cluster cmd
   * Oproles add path to hcpsharedvpc policy ARNs manual mode
   * Delete accountroles hcpsharedvpc interactive
   * Include path in create acc/op roles manual mode
   * Use path flag with hcpsharedvpc policy creation
   * Do not create new policy version when policy is hcpsharedvpc
   * Fixing manual-mode + log event output
   * Combine all policies rather than eating the output
   * Swap manual+auto questions for dlt/accountroles
   * Swap function args to make usability straightforward
   * UX changes for create/accountroles default values
   * Do not print create policy commands more than once op roles
   * Do not create sharedvpc policies when answer is No
   * Do not print create policy commands more than once op roles
   * Set default value to true if both sharedvpc roles are supplied
   * Manual mode [oproles]- create policy commands single print
   * Dont create policies for op roles when they exist
   * Add arn validation for hcp sharedvpc role arns
   * Make shared vpc role ARNs required for HCP
   * Refactor manual mode for create/oproles
   * Removing refresh token to be a validation requirement
   * Updating naming for Rosa create network log events
   * Only check for HCP only flags when not using `--mode auto`
   * Fix for regression with classic sharedvpc oproles
   * Take paths from user-provided sharedvpc role arns
   * Creation of operator roles for classic fixes
   * Pre-command validation for shared vpc flags
   * Change help messages for create/operratorroles sharedvpc flags
   * Disable failing tests which are blocking merge
   * Fixed help message for binary builds
   * Fixed tempalte dir env var to work
   * Fixed network command to be able to run custom templates
   * Fixed help message for binary builds
   * Fixed default name in info message
   * Updated info message for when no template is specified
   * Changing default template dir in help message
   * Duplicate commands when manual create/accountroles
 * Chores
   * Bump release RC version
   * Bump SDK version 0.1.447 -> 0.1.448
 * Other
   * Revert "[release_1.2.49] OCM-12851| feat: default values for CAS Max Nodes Total and validation added for count of mp worker nodes"

## 1.2.48 (15 Nov, 2024)

FEATURES:
   * Adding extra validation to verify user is logged in
   * Adding default cf template for binary builds
   * Adding extra validation to verify user is logged in
   * Update cluster autoscaler max value (180->249)
   * Fixing manual mode operator deletion command
   * Adding dmoskale to rosa cli approvers
   * Adding validation of the roles manage policies as a step when an upgrade policy is being requested by a cluster.
   * Adding --template-dir flag & TEMPLATE_DIR env var
   * Add EC2 container registry policy to worker role for zero egress
   * Improved help message for rosa create network

ENHANCEMENTS:
 * Bug fixes
   * Fixed help message for binary builds
   * Fixed tempalte dir env var to work
   * Fixed help message for binary builds
   * Fixed default name in info message
   * Fixed network command to be able to run custom templates
   * Changing default template dir in help message
   * Updated info message for when no template is specified
   * Duplicate commands when manual create/accountroles
   * Fixing enable-delete-protection flag
   * Fixing tags and network not appearing in help
   * Skip validation of container registry policy for create cluster
   * Env var regression with create/network
   * Pass config ID into create/provider
   * Exit with status 0
   * Rearranging creation order to fix endpoint issue
   * Only attempt provider creation with auto mode
   * Adding log outputs for empty values
   * Only run oidcprovider command when mode=auto
   * Adding Availability zone default count
   * Re-enable adding EC2 policy to worker role
   * Revert adding EC2 policy to worker role
   * Not exit when user choosing N for registry config
 * Chores
   * Update version to RC3
   * St release version to 1.2.48-RC1
   * Update ROSA CLI with the latest ocm sdk to use addons_mgmt
   * Bump master to 1.2.48

## 1.2.47 (30 Oct, 2024)

FEATURES:
   * Adding --template-dir flag & TEMPLATE_DIR env var
   * Improved help message for rosa create network
   * Support list access requests
   * Adding Rosa CLI create Network Command
   * Add rosa describe accessrequest command
   * Day1 additional SG support for HCP
   * Allow billing account update via the cluster edit command
   * Add hunterkepley to approvers section
   * Add "rosa cli" to !latest error main.go
   * Support approve/deny access request in ROSA CLI

ENHANCEMENTS:
 * Bug fixes
   * Env var regression with create/network
   * Pass config ID into create/provider
   * Exit with status 0
   * Rearranging creation order to fix endpoint issue
   * Adding log outputs for empty values
   * Adding Availability zone default count
   * Only attempt provider creation with auto mode
   * Only run oidcprovider command when mode=auto
   * Reapplying tags
   * Fix error in rosa init
   * AvailabilityZoneCount creates correct amount of resources
   * Adding a default template if no template is specified
   * Include zero egress status in cluster describe command
   * Etcd encryption describe should be shared between hcp and classic
   * Use different language when updating vs creating cluster for enabling the registry set of configuration options.
   * Fixing typo
   * First check if pool exists
   * Allow trailing comma for taints/labels
   * Sanitize user input so that command with registry arguments is executable.
   * Refactor multiple ca tests from getClusterRegistryConfig
   * Specifying --registry-config-allowed-registries-for-import when editing a cluster should not enable interactive mode.
   * Remove multiple ca tests from getClusterRegistryConfig
   * Loosen registry regex check
   * A few more bug fixes for registry config
   * A few bug fixes for registry config
   * When filters are empty consider true
   * Error msg for editing nodepool; autoscaling+ 0 replicas
   * Registry config platform allowlist should hide from help
 * Chores
   * Bump RC from 2 -> 3
   * Adjust goreleaser to use openshift/rosa
 * Other
   * Update info.go

## 1.2.46 (04 Oct, 2024)

FEATURES:
   * Client support for configurable registries
   * Describe client support for configurable registries
   * Bump ocm-common to lower np min disk size to 75
   * Clarify that 'billing-account' expects an account ID
   * Use error message built from SDK
   * Bump sdk to v0.1.440
   * Show disk size in output for list node pools
   * Print custom worker disk size in describe nodepool

ENHANCEMENTS:
 * Bug fixes
   * Loosen registry regex check
   * A few more bug fixes for registry config
   * A few bug fixes for registry config
   * When filters are empty consider true
   * Error msg for editing nodepool; autoscaling+ 0 replicas
   * Registry config platform allowlist should hide from help
   * Adjust error message for ingress identifier
   * Regression, HCP nodepool creation validation used for classic
   * When classic topology return nil
   * Adjust check for policy tags
   * Update description for 36293
   * Make target ensuring release builds and publishes to github
   * Use default user agent
   * Fix create/provider when cluster name provided
   * Add unit to disk size on describe nodepool
   * Fix issue with classic clusters without --oidc-config-id flag
   * Add Limited Support Reason Override support
   * Do not accept clusterID for thumbprint if programmatic
   * Update test ids: 43070, 74661
   * Fix etcd kms key arn space
   * Add etcd to reduce confusion
   * Fixing OCM-10512 bug
   * Fixing OCM-10777 + OCM-10818
   * Fix bash syntax error in prow_ci.sh
 * Chores
   * Bump 1.2.46 to release
   * Bump 1.2.46 to RC2
   * Adjust goreleaser to use openshift/rosa
   * Add output to release script for errata jira list

## 1.2.45 (16 Sep, 2024)

FEATURES:
   * Show disk size in output for list node pools
   * Print custom worker disk size in describe nodepool
   * Enable custom worker disk sizes on node pools
   * Adding unit tests for CreateMachinePool + CreateNodePool
   * Moving create machinepool and nodepool functions from cmd -> pkg
   * Refactoring on autoscaler commands to use new runner pattern
   * Add role console url in the output for policy attach
   * Add custom rosa client version
   * Add a user agent config
   * Output content of trust policy attached to a role
   * Move fetching OIDC Thumbprint to backend
   * Output IAM policies that are being attached to IAM roles

ENHANCEMENTS:
 * Bug fixes
   * Fix create/provider when cluster name provided
   * Add unit to disk size on describe nodepool
   * Fix issue with classic clusters without --oidc-config-id flag
   * Do not accept clusterID for thumbprint if programmatic
   * Fixing OCM-10777 + OCM-10818
   * Fix etcd kms key arn space
   * Add etcd to reduce confusion
   * Assign cluster ID var in create/oidc_provider
   * Error when setting multi-az true for HCP
   * Validate int32 for user input in autoscaler attributes
   * Add missing aliases for describe/delete autoscaler commands
   * Don't allow cluster name longer than 15 characters when installed into shared vpc
   * Remove manual cluster ID flag update for create/oidcprovider
   * Add cluster key back to oidcprovider create/cluster call
   * Describe etcd KMS key when configure
   * Fix provider creation in cluster create
   * Don't overwrite existing autoscaler value in edit flow
   * Interactive installer inconsistency in subnets
   * Provide OIDC config ID to create/provider programmtically
   * Update staging qaprodauth console url for details page
   * Missing operator roles when filtering due to extended naming cutting the postfix
   * Trim double quotes of component route parameter
   * Forbid to create hcp cluster without billing account
   * Update edit machinepool error message for missing id
 * Chores
   * Bump release to 1.2.45-rc4
   * Chore : bump release to 1.2.45-rc3
   * Update goreleaser config to ignore merge commits
   * Increment master version to 1.2.45 (1 above 1.2.44-rc1)

## 1.2.44 (28 Aug, 2024)

FEATURES:
   * Unhide no-cni flag at cluster creation
   * Persist refreshed tokens
   * Upgrade ocm-sdk-go to 0.1.435
   * Adding taints effect validation
   * Adding extra validation to fix panic introduced with httpTokens

ENHANCEMENTS:
 * Bug fixes
   * Don't overwrite existing autoscaler value in edit flow
   * Missing operator roles when filtering due to extended naming cutting the postfix
   * A regression caused the external ID not to be forwarded
   * Change options validations to allow 0 min replicas as well
   * Cmd/version arguments not being passed
   * Upgrade machinepool producing error
   * Delete classic and hcp account roles automatically
   * Update error message for missing machine pool ID
   * Set correct default value for replicas in update mp
   * Change replica (min, max, and normal) minimum to 0
   * Issue with interactive + replicas edit machinepool
   * Bug with min/max replicas (edit machinepool)
   * Do not allow managed policies role without hcp flag
 * Chores
   * Initial 1.2.44 rc cut
   * Update module gitlab.com/c0b/go-ordered-json to include license
   * Switch from ghodss/yaml to sigs.k8s.io/yaml
 * Documentation
   * Add example for enabling UWM day-2

## 1.2.43 (07 Aug, 2024)

FEATURES:
   * Adding taints effect validation
   * Adding extra validation to fix panic introduced with httpTokens
   * Support IMDSv2 in HCP
   * Edit machinepool refactored to use new runner
   * Moving create machinepool and nodepool functions from cmd -> pkg
   * Support both yaml and json format for tuningconfigs
   * Upgrade ocm-sdk-go to 0.1.428
   * Move edit machinepool non-cmd funcs to pkg and test

ENHANCEMENTS:
 * Bug fixes
   * Change options validations to allow 0 min replicas as well
   * Set correct default value for replicas in update mp
   * Change replica (min, max, and normal) minimum to 0
   * Issue with interactive + replicas edit machinepool
   * Bug with min/max replicas (edit machinepool)
   * Remove dot import from machine pools
   * Test expected ROSA CLI command structure to prevent accidental command removal
   * Fixing regression problems for OCM-9620,OCM-9655,OCM-9654,OCM-9625
   * Elevate QE approvers to main ROSA approvers to reduce TOIL and enable QE team
   * Add back maxSurge and maxUnavailabe for create HCP NPs
   * Fixed slow tests for break glass credential implementation
   * Re-add incorrectly removed edit machinepool command
   * Enable maxSurge and maxUnavailable for interactive mode
   * Re-add max-surge/max-unavailable code
   * Fix incorrect autoscaling enablement flag usage
   * Make regex match works on bash
   * Hcp clusters do not allow multi-az machinepool creation
 * Chores
   * Update next release to 1.2.43
   * Updates fedramp client id configuration

## 1.2.42 (15 Jul, 2024)

FEATURES:
   * Support multi-arch-enabled parameter
   * Create/update of maxUnavailable and maxSurge for HCP nodepools
   * Show state description for HCP upgrade policies

ENHANCEMENTS:
 * Bug fixes
   * Prevent same role arns for IAM roles of the same cluster
   * Update ocm-common dependency to latest v0.04
   * Typo in logger info message for output of command
   * Validate node drain grace period when edit node pools
   * Do not enable interactive for max-surge and max-available
   * Add password validator for create admin
   * Unsupported combination of flags for operator-roles
   * Update error message for linking ocm-role/user-role
   * Remove empty line between mode descriptions
   * Update ocm-sdk-go to fix Windows SSL certificate issues
   * Hide the aws billing account getting exposed in cluster installation log
   * Include prefix check for clusters running registered oidc configs
   * Fetch subnets for availability zone
 * Chores
   * Bump ocm-sdk-go
   * Bump ocm-sdk-go to v0.1.425
   * Update to version 1.2.42 in master branch ahead of next release
   * Update the fips flag description
 * Other
   * Revert "OCM-9094 | feat: support multi-arch-enabled parameter"

## 1.2.41 (24 Jun, 2024)

FEATURES:
   * Revert maxUnavailable and maxSurge for NodePool
   * Add additional allowed principals config for hcp clusters
   * Handle 412 response in upgrade commands
   * Hide attach/detach commands
   * Create/update of maxUnavailable and maxSurge for HCP nodepools
   * Bump ocm-sdk-go to v0.1.422
   * Removing interactive mode as default for prefix prompt
   * E2E delete machinepool cmd tests; new runner
   * Display attached policies for sts roles
   * Remove version from editing machinepool

ENHANCEMENTS:
 * Bug fixes
   * Update ocm-sdk-go to fix Windows SSL certificate issues
   * Use a tmp file per username
   * Err not being caught on rosa version
   * Use space delimiter in AWS tags error msg to avoid confusion
   * Colon not present in some field for describing autoscaler to a cluster
   * Id:53031 update the error message
   * [HCP] always set replicas when updating machinepools
   * Fix delete oidc provider failed
   * Mandatory fields should enter interactive mode if not provided
   * Use aws default http client
   * Enable VPC total clean
   * Only check redhat managed policies when trying upgrade roles
   * Unhide attach/detach commands
   * Kubelet-config edit prompt no longer shows up when kubelet-configs are not targeted
   * Hide unnecessary kubeconfig info
   * Fix error message for invalid username validation for idp creation
   * Don't show empty lines when filtering version on acc roles
   * Remove redudant empty lines in manual mode
   * Always add attach command for managed policy during rosa upgrade roles
   * Delete should exit 1 if fails
   * Only list kubeletconfigs when absolutely necessary when creating a machinepool
   * Add validation for username/password of idp creation
   * Add attach policy command for rosa upgrade roles
   * Fix problem in delete operator roles manual mode
 * Chores
   * Bump ocm-sdk-go to v0.1.425
   * Update download name for image testing pipeline
   * Remove thomasmckay from ROSA OWNERS file
   * Bump gci version to correctly handle newer golang versions
   * Bump dependency golang.org/x/net and google.golang.org/protobuf

## 1.2.40 (30 May, 2024)

FEATURES:
   * Move delete machinepool/nodepool funcs to pkg
   * Update rosa edit machinepool command for kubeletconfig support
   * Update create machinepool command to support kubeletconfigs
   * Added ability to describe Kubeletconfigs for HCP clusters
   * Added support for delete of KubeletConfig for HCP clusters
   * Added the ability to edit KubeletConfigs for HCP clusters
   * Updated to support create kubeletconfig for HCP clusters
   * Validation for user creating account-roles
   * Added list of kubeletconfigs to describe HCP machinepool output
   * Add the ability to list KubeletConfigs for a cluster
   * Refactor list machine pool cmd to use new default runner
   * Ensure account roles have expected attached policies
   * Add support for ListInstanceTypes method
   * Move list machinepool non-cmd funcs to pkg, split, test
   * Change default value back
   * Move non-cmd funcs to pkg, split, test

ENHANCEMENTS:
 * Bug fixes
   * Kubelet-config edit prompt no longer shows up when kubelet-configs are not targeted
   * Only list kubeletconfigs when absolutely necessary when creating a machinepool
   * Hide the duplicate output of operator roles
   * Show correct describe message for expired breakglass
   * Allow users to use positional args for specifying the name of kubeletconfig
   * Force users into interactive mode if omitting a name when creating a KubeletConfig for HCP clusters
   * Return message if there is no breakglass can be revoked
   * Ensure the user is prompted if they change the kubelet-config on their nodepool
   * Check kubeletconfig exists by name if user specifies it for Classic cluster
   * Ensure users can only supply a single kubeletconfig for HCP MachinePools
   * Ensure --kubelet-configs flag is not supported for ROSA Classic MachinePools
   * Ensure the ID of the KubeletConfig is printed by rosa describe kubeletconfig
   * Ensure --name option is required for working with KubeletConfigs on HCP clusters
   * Add region parameters for deleting oidc config command
   * Ensured that name is optional when creating a KubeletConfig
   * Allow min_replicas 0 with edit machinepools
   * Add detach commands for arbitrary policies
   * Print role arn for hcp
   * Get the orgin junit file from ArtifactDir
   * List machinepool should show inline
   * Allow 0 min replicas for classic cluster autoscaling mp
   * Warn revoked break glass credential
   * Lower mapAZcreated to 1
   * Display external-auth-providers-enabled flag
   * Display min/max in describe machinepool
   * Edit autoscaling max replicas of nodepool force to set min replicas
   * Remove duplicate oidcprovider cmd call
   * Skips policy compatibility check when version supplied is empty
   * Add region deprecation disablement to sub-commands
   * Prevents overwriting the client id if set for fedramp with keycloak
   * Ensure arbitrary policy not removed during cluster roles deletion
   * Check policy attached in detach manual mode
   * Provide the default value to the external auth
   * Change default value for disable region dep. flag
   * Adjust describe ingress binding
   * Update aws sdk to fix imds v2 issue
   * Unhide breakglass and externalauth
   * Hide attach and detach commands
   * Ensure version option args are initialised
   * Change not found to info
   * Fix deprecation warn printing when creating cluster
 * Chores
   * Prepare 1.2.40 release
   * Bump to rc2 version
   * Added additional aliases to the `rosa list kubeletconfig` command.
   * Updated OCM SDK to v0.1.419
   * Increment master version (to 1.2.40)
 * Documentation
   * Add presubmit readme for rosa CLI testing
 * Other
   * Revert "OCM-6900 | go mod vendor required for release"
   * Go mod vendor required for release

## 1.2.39 (14 May, 2024)

FEATURES:
   * Change default value back
   * Detach cmd to detach policy
   * Classic mp aws tags on create/describe
   * Add cmd to attach policy
   * Add describe ingress command
   * Allow to supply user tags on creation of hcp machine pool
   * Refactor describe machine pool cmd to use new default runner
   * Show a warning when a user attempts to run a command on a version other than the latest version from mirror.
   * Show a warning when a user attempts to run a command on a version other than the latest version from mirror.
   * Add cache for checking version mirror
   * Move describe machinepool funcs to pkg, test, split up, make interface
   * Show user tags on describe machine pool for HCP
   * Support cluster admin day-1 creation for HCP cluster
   * Deprecate the region flag in commands which do not utilize it
   * Select security groups interactive - filtering

ENHANCEMENTS:
 * Bug fixes
   * Allow min_replicas 0 with edit machinepools
   * Allow 0 min replicas for classic cluster autoscaling mp
   * Add region deprecation disablement to sub-commands
   * Change default value for disable region dep. flag
   * Remove duplicate oidcprovider cmd call
   * Fix deprecation warn printing when creating cluster
   * Display external-auth-providers-enabled flag
   * Update aws sdk to fix imds v2 issue
   * Adjust describe ingress binding
   * Update aws sdk to fix imds v2 issue - manual edit
   * Unhide breakglass and externalauth
   * Hide attach and detach commands
   * Ensure version option args are initialised
   * Valide rolename and policyarn in rosa attach/detach cmd
   * Small bugfix for rosa describe machinepool
   * Error message for unmanaged policies
   * Get the merged commit but not the pull request commit
   * Remove older attributes for component routes
   * Filter empty subnet id
   * Remove dependency on OCM for checking ROSA version
   * Align nodepool actions with machinepools
   * Block HCP operator-roles with unmanaged policies account role
   * Dsiable region deprecation flag when flag is used
   * Only show message when region is changed and no output mode set
   * Fixed bug where deprecation shows for parent cmd
   * Align list users with list idps
   * Add info msg for empty case
   * Adjust describe machine pool output
   * List instance types filter by region requires installer role arn
   * Allow empty GitHub IDP hostname
   * Invert describe out for cluster delete protection
   * Add poll length for break glass credential
   * GitHub IDP Add hostname validation
   * Error message when describe credential that status is revoked
 * Chores
   * Bump ocm-sdk to v0.1.418
   * Bump ocm sdk to version 0.1.416 to fix windows ocm cert expiry
   * Increment master version (to 1.2.39)
 * Other
   * Sync with main

## 1.2.38 (24 Apr, 2024)

FEATURES:
   * Allow users set delete protection on cluster
   * Display AWS Billing Account on non-HCP
   * Add actions for break glass credential
   * Add fedramp gating to all managedservice funcs+tests
   * Add keyring support for configuration
   * Added rosa describe autoscaler command
   * Display login error if --govcloud supplied with commercial region

ENHANCEMENTS:
 * Bug fixes
   * Invert describe out for cluster delete protection
   * List instance types filter by region requires installer role arn
   * Add info after successfully created breakglasscredentials
   * Error message should distinguish between labels and taints
   * Ensure "mode" in interactive mode defaults to users input
   * Add json output test for external auth
   * Add validation for revoking breakglasscredential
   * Support create breakglasscredential without optional flags
   * Provide better message for external auth creation
   * GitHub IDP validation
   * Added test case
   * Added back the condition to check for empty env flag
   * Revert "OCM-6119 | fix: block users from passing region flag when creating an oidc provider"
   * Disable the feature for custom username of cluster admin
   * Add checks for mandatory parameters
   * Upgrade role error should print extra info when invoke from upgrade cluster
   * Revert "OCM-6883 | fix: rosa describe admin can list the admin with custom name"
   * Error out if cluster is enabled with external auth
   * Use cluster name in error message
   * Panic when not supplying '=' delimiter component routes
   * Rosa describe admin can list the admin with custom name
   * Needs to also go through admin fedramp url aliases
   * Simplified the condition for FedRAMP
   * Update --best-effort help message
   * Add validator for node drain grace period in interactive mode
   * Remove else condition and move FedRAMP logic before normal check
   * Support local envs in GetEnv()
   * Allow user specify `cluster-admin` as admin username
   * Add validations for external auth
 * Chores
   * Release 1.2.38
   * 1.2.38-RC3 cut with cherrypicked release blocker
   * Bump ocm sdk to version 0.1.416 to fix windows ocm cert expiry
   * 1.2.38-RC2 cut with cherrypicked release blocker
   * Update the list jira helper to ignore reverts when listing release jiras
   * Request to access maintainer access to ROSA repo
   * Allow GH made reverts on commit check

## 1.2.37 (03 Apr, 2024)

FEATURES:
   * Add rhRegion flag to rosa login command
   * Disable ManagedService creation/updates in FedRAMP
   * ROSA CLI V2
   * CRUD for external auth providers
   * Handle trying to make hcp cluster on fedramp properly
   * Allow to edit component routes of ingress
   * Node drain grace period for hosted control planes
   * Support customize username for cluster admin creation
   * Making billing-mode standard default
   * Make auth and device code flags visible
   * Extend name length to 54 characters
   * Create node pool with SG IDs
   * Remove shorthand for --oidc-config-id
   * Create private functions, bubble up errors and simplify them
   * Add token and config commands
   * Describe node pools with additional security groups
   * Add filtering by config ID for list oidc-providers
   * List instance-types by region
   * --verbose option to version command
   * Supressing help message when error is thrown

ENHANCEMENTS:
 * Bug fixes
   * Add json output of the external auth
   * Create node pool with security group - check version
   * Hcp private cluster subnet validation
   * Describe node pool - adjust output
   * Create machine pool with security group - filter options
   * Validation for classic managed policies
   * Update OCM-SDK
   * CLI support versions 4.12 for HCP cluster creation
   * 4.14 nightly version not compatible with ROSA HCP
   * Improved help text for config get/set commands
   * Upgrade cluster should not exit prematurely after calling upgrade role
   * Improve domain-prefix help message
   * Added more info to get config error message when config is not found
   * Add validation to not pass token or fedramp flag for OAuth login
   * Removed output flag from rosa token command
   * Block users from passing region flag when creating an oidc provider
   * Generates a valid operator role prefix from the cluster name
   * Fail commands when unknown arguments are passed
   * Shouldn't automatically start an interactive mode for domain prefix as it's optional
   * Change describe node pool output
   * Bound math.MaxInt64 to avoid overflow on non-64bit arch
   * Remove toolchain from go.mod
   * Change 'RunE' calls to just 'Run'
   * Revert spinner on upgrade account role policies
   * Align error messages for billing accounts
   * Filter out local/wavelength zone in cluster creation
   * Ensure 'make clean' removes all binaries
   * Update list jira command to filter out issues from previous releases
   * Temporarily remove build output from rosa version command
   * Use the version constant
 * Chores
   * Bump rosa cli version to 1.2.37
   * Bump ocm-sdk-go v0.1.410
   * Ensure all mocks and tests are excluded from coverage statistics
   * Make all codecov reporting informational
   * Bump ocm-sdk-go v0.1.409
   * Remove error from reporter.Builder.Build method signature
   * Add robpblake as an approver for PR reviews
   * Bump ocm-sdk-go v0.1.405
   * Bump ocm-sdk-go v0.1.404
   * Prepare 1.2.36 release
   * Update scripts to use fix version instead of labels
   * Expose nodepool upgrades on creation
 * Documentation
   * Include an example of a valid commit that follows the conventional commits

## 1.2.36 (21 May, 2026)

FEATURES:
   * Add token and config commands
   * Create private functions, bubble up errors and simplify them
   * Add filtering by config ID for list oidc-providers
   * List instance-types by region
   * --verbose option to version command
   * Make ID in UX fix for 'rosa edit addon' nonpositional
   * Update OCM SDK to the latest version
   * Removing duplicate err msg output
   * Added a -y flag to remove user interaction when linking roles
   * Add Device Code Flow
   * Edit error messages in by_clusterkey to have %v
   * Add new parameter to enable external auth config
   * Added single quotes to original code
   * Removed unecessary err check
   * Bump SDK release to 0.1.398
   * Use default version
   * Update link ocm role error message
   * Print Git SHA as build information in rosa version
   * Improve error message for `rosa link ocm-role`
   * Reverted golang version to 1.20
   * Added additional message for better understanding
   * Add oauth login using PKCE
   * Improve create op-role UX by not checking for account role version on manual mode
   * Add --client option to rosa version command
   * Secure password by hashing it
   * Use GenerateRandomPassword in ocm-common
   * Add new parameter to disable CNI at cluster creation for HCP
   * Include 'Display Name' in cluster description
   * Changed default platform type to aws-classic & aws-hosted-cp
   * Version, verify rosa-client: Support --debug

ENHANCEMENTS:
 * Bug fixes
   * Remove toolchain from go.mod
   * Update list jira command to filter out issues from previous releases
   * Temporarily remove build output from rosa version command
   * Align error messages for billing accounts
   * Ensure 'make clean' removes all binaries
   * Use the version constant
   * Validate aws region within .WithAWS
   * Mark --no-cni flag as hidden
   * Add more instruction in billing account error
   * Updating OWNERS file
   * Ensure codecov reports are being published for master
   * Triggering retries associated with 'InvalidClientTokenId'
   * Adjusted Codecov.yml to be more lenient on small fixes
   * Correct the message shown in interactive mode when creating a machinepool
   * Check if op roles exist always again
   * Make rosa the default goal in the Makefile
   * Ensure rosa is the default Makefile target
   * Update correct values for subnet count validation
   * Add the diff target to Makefile
   * Detailed throttle retry message
   * Move logic only related to auto into ModeAuto check
   * Update description on link user roles command
   * Default version computation should be done only once
   * Default version computation should be done only once
   * Add passed disk size to error message
   * Run go mod vendor
   * Revise help message for rosa grant/revoke command
   * Wrap error message for invalid credentials
   * Align usage of -arn suffix on role parameters to create cluster command
   * Add retry count to throttle retry message
   * Add progress indicator for account role policy upgrades
   * Validate machinepool labels for /
   * Add error message when supplying worker disk size on unsupported version
   * Improve logging when fetching instance type data for creation of machine pools
   * Ensure that --etcd-encryption-kms-arn is printed when set and --etcd-encryption is true
   * Ensure account role creation is not prompted for other role type flag
   * Update default-mp-labels arg to worker-mp-labels
   * Revise error message for rosa grant/revoke
   * Add cluster arg to help message
   * Remove option to create classic roles if a user passes hosted-cp parameter
   * Align role-arn flag across commands
   * Add confirmation prompt to deleting cluster autoscaler
   * Moved code to auto block
   * Fix bug with 'edit addon' UX change
   * Remove `--billing-model` flag from rosa edit addon
   * Improve logging when fetching instance type data for creation of machine pools
   * Reword usage help text for delete and link ocm role commands
   * Resort to last cert in cert chain
   * Use cluster ID to fetch operator-roles
   * Align cluster description value `Enabled`
   * Use helper.Contains for looping
   * Adjust message when operator roles already exists
   * Handle bad disk size input
   * Describe machinepool help info
   * Add billing-account param to last command
   * ROSA CLI N/W Verifier Error Message for IPI VPC Clusters
   * Better message for ingress unsupported att on HCP
   * If no subnets available refer to doc
   * Remove condition that checks cluster state when creating oidc provider
   * --output flag for 'rosa verify network'
   * `list users` can list `cluster-admin` user
 * Chores
   * Prepare 1.2.36 release
   * Expose nodepool upgrades on creation
   * Update scripts to use fix version instead of labels
   * Extract hypershift version validation to a separate function
   * Pkg/aws: expose the Creator transform from caller identity
   * Remove dependency on AWS client
   * Clean up old TODO
   * Move release scripts from Wiki to Git repository
   * Updates for golang 1.21
   * *: go mod tidy with 1.21
   * Prepare 1.2.35 release
   * Bump and use ocm-commons
   * Pkg/ocm: expose upgrade policy on creation
   * Sort OWNERS alphabetically
   * Bump ocm-sdk-go v0.1.395
   * Update OWNERS file to reflect current team members
   * Bump ocm-common
   * Bump ocm-commons

## 1.2.35 (21 May, 2026)

FEATURES:
   * Added single quotes to original code
   * Edit error messages in by_clusterkey to have %v
   * Bump SDK release to 0.1.398
   * Use default version
   * Update link ocm role error message
   * Print Git SHA as build information in rosa version
   * Version, verify rosa-client: Support --debug
   * Improve error message for `rosa link ocm-role`
   * Add --client option to rosa version command
   * Reverted golang version to 1.20
   * Added additional message for better understanding
   * Add oauth login using PKCE
   * Improve create op-role UX by not checking for account role version on manual mode
   * Secure password by hashing it
   * Add new parameter to disable CNI at cluster creation for HCP
   * Use GenerateRandomPassword in ocm-common
   * Include 'Display Name' in cluster description
   * Changed default platform type to aws-classic & aws-hosted-cp

ENHANCEMENTS:
 * Bug fixes
   * Mark --no-cni flag as hidden
   * Check if op roles exist always again
   * Move logic only related to auto into ModeAuto check
   * Detailed throttle retry message
   * Triggering retries associated with 'InvalidClientTokenId'
   * Update description on link user roles command
   * Default version computation should be done only once
   * Revise help message for rosa grant/revoke command
   * Default version computation should be done only once
   * Wrap error message for invalid credentials
   * Align usage of -arn suffix on role parameters to create cluster command
   * Add passed disk size to error message
   * Validate machinepool labels for /
   * Run go mod vendor
   * Add retry count to throttle retry message
   * Add cluster arg to help message
   * Align role-arn flag across commands
   * Update default-mp-labels arg to worker-mp-labels
   * Improve logging when fetching instance type data for creation of machine pools
   * Ensure account role creation is not prompted for other role type flag
   * Ensure that --etcd-encryption-kms-arn is printed when set and --etcd-encryption is true
   * Revise error message for rosa grant/revoke
   * Remove option to create classic roles if a user passes hosted-cp parameter
   * Add confirmation prompt to deleting cluster autoscaler
   * Moved code to auto block
   * Remove `--billing-model` flag from rosa edit addon
   * Improve logging when fetching instance type data for creation of machine pools
   * Reword usage help text for delete and link ocm role commands
   * Resort to last cert in cert chain
   * Align cluster description value `Enabled`
   * Fix bug with 'edit addon' UX change
   * Add error message when supplying worker disk size on unsupported version
   * Handle bad disk size input
   * `list users` can list `cluster-admin` user
   * Describe machinepool help info
   * Adjust message when operator roles already exists
   * ROSA CLI N/W Verifier Error Message for IPI VPC Clusters
   * Better message for ingress unsupported att on HCP
   * Use cluster ID to fetch operator-roles
   * Add billing-account param to last command
   * Use helper.Contains for looping
   * Remove condition that checks cluster state when creating oidc provider
   * If no subnets available refer to doc
   * --output flag for 'rosa verify network'
 * Chores
   * Prepare 1.2.35 release
   * Sort OWNERS alphabetically
   * Pkg/ocm: expose upgrade policy on creation
   * Bump ocm-sdk-go v0.1.395
   * Bump and use ocm-commons
   * Update OWNERS file to reflect current team members
   * Bump ocm-commons
   * Bump ocm-common

## 1.2.34 (21 May, 2026)

FEATURES:
   * Use GenerateRandomPassword in ocm-common
   * Add new parameter to disable CNI at cluster creation for HCP
   * Changed default platform type to aws-classic & aws-hosted-cp

ENHANCEMENTS:
 * Bug fixes
   * Use helper.Contains for looping
   * Adjust message when operator roles already exists
   * Describe machinepool help info
   * Add billing-account param to last command
   * ROSA CLI N/W Verifier Error Message for IPI VPC Clusters
   * Better message for ingress unsupported att on HCP
   * If no subnets available refer to doc
   * Remove condition that checks cluster state when creating oidc provider
   * --output flag for 'rosa verify network'
   * Typos in messages
   * Properly exit on error during upgrade operator-roles
   * Typo 'openShift'
   * Improve list account-roles runtime
   * `list users` can list `cluster-admin` user
 * Chores
   * Bump ocm-common
   * Bump ocm-commons

## 1.2.33 (21 May, 2026)

FEATURES:
   * Add codecov to CI pipeline for ROSA
   * Add FedRAMP Prod Sector Env aliases

ENHANCEMENTS:
 * Bug fixes
   * Validate min replicas for hosted cp clusters
   * Default HCP installer role to 'ManagedOpenshift' prefix
   * Add a warning message when using best effort deletion
   * Unset httpTokens for HCP
   * Show imds param on cluster helper command
   * Parenthesis placement
   * Better messages for ingress prompts
   * Set IMDS as option prompt
   * Output platform and tags
   * Chunk slice before sending request to AWS
   * Add prompt to then show default ingress custom options
   * Only show private subnets when private link cluster is selected
   * Use spaces instead of \t
   * Don't show create operator roles midflow
   * Use tabs instead of spaces
   * EC2 is typed as Ec2 in ROSA CLI
   * Update aliases for oidc config
   * Revise help message for create-cluster-admin
   * Update example command for rosa edit cluster
   * Add subnet info when available
   * Remove duplicated describe att
   * Only output follow-up commands if necessary
 * Chores
   * Bump to v1.2.33
   * Update OCM-SDK-GO version to v0.1.390

## 1.2.32 (21 May, 2026)

FEATURES:
   * Allow passing platform type for subnet based verification
   * Order imports - contributing file
   * Add linter checks to the PR pipeline
   * Format existing imports
   * Refactor `GeneratePolicyFiles` to simplify code
   * Add `fmt-imports` to the Makefile
   * Add custom tags flag to rosa verify network

ENHANCEMENTS:
 * Bug fixes
   * Revert - fetch passed arn in all roles
   * Send custom tags when doing network verify for a cluster
   * Do not prompt classic on govcloud; do not allow --hosted-cp on govcloud
   * Remove create ingress cmd
   * Create HCP trust policies in manual mode
   * Fetch passed arn in all roles
   * Do not allow fips flag for HCP
   * Improve error message for create operator roles
   * Create HCP operator roles by prefix - filter by policy
   * Skip permission policies creation for HCP
   * Only asks for sgs if not been set in interactive
   * Helper message infra/control sgs
   * Describe cluster show sg ids
   * Fix GetVersionList for fedramp
   * Elaborate on --best-effort help description
   * Remove suggestion to 'create oidc-config' during 'create account-roles'
   * Fixed account role creation in fedramp
   * Provide sensible default for PIDs limit in interactive mode
   * Create machine pool interactive mode
 * Chores
   * Bump sdk v0.1.387
   * Use consts for SG flag calls

## 1.2.31 (21 May, 2026)

FEATURES:
   * Link the enforce of billing account hcp for GA
   * Show dynamically technology preview messages for hcp
   * Added rosa edit kubeletconfig sub-command
   * Allow non sts clusters to verify network
   * Added rosa create kubeletconfig sub-command
   * Added the rosa delete kubeletconfig sub-command
   * Added rosa describe kubeletconfig sub-command
   * Added rosa describe kubeletconfig sub-command
   * Additional sg infra and control plane
   * Allow cluster ID association via on demand NV
   * Add information about inflight check failures on cluster describe

ENHANCEMENTS:
 * Bug fixes
   * Fix GetVersionList for fedramp
   * Describe cluster show sg ids
   * Only asks for sgs if not been set in interactive
   * Helper message infra/control sgs
   * Create HCP operator roles by prefix - filter by policy
   * Fixed account role creation in fedramp
   * Provide sensible default for PIDs limit in interactive mode
   * Create machine pool interactive mode
   * Ensure we validate maximum pids limit when creating/editing KubeletConfig
   * Ensure we display what the user entered for '-c' flag on KubeletConfig command output
   * Remove technology preview from command help
   * Added -y flag to rosa edit kubeletconfig command
   * Include --properties in create cluster output
   * Allow users to reset mp taints
   * Update hcp billing account link
   * Create HCP cluster - filter out roles with classic policies
   * Create cluster - improve no account roles found message
   * Create HCP account roles with unmanaged policies
   * Adjust subnet ids passed
   * Create cluster - filter classic ROSA account roles
   * Fix CheckAndParseVersion when coming from preRelease
   * Adjust min format so it allows nightlies
   * Disable node drain grace period for hosted clusters
   * Updating vendor
   * Validate upgrade versions from prerelease scenarios
   * Align list upgrade output when no avl upgrades
   * Classic upgrade should align left
 * Chores
   * Use consts for SG flag calls
   * Bump release to 1.2.31
   * Bump sdk v0.1.385
   * Bump sdk v0.1.380
   * Bump sdk to v0.1.377

## 1.2.30 (10 Nov, 2023)

ENHANCEMENTS:
 * Bug fixes
   * Do not ask billingAccount for hcp before GA
 * Chores
   * Release 1.2.30

## 1.2.29 (03 Nov, 2023)

FEATURES:
   * Fixing cluster creating bug
   * Moved kms Arn regexp validator to common lib
   * Support best-effort mode in cluster deletion
   * Vendor changes
   * Moved associated to GetRole functions to common folder
   * Disable edit machinepool --version command
   * Use HCP specific endpoints for versions and upgrades
   * AWS billing support
   * Create operator roles for shared VPC with cluster key
   * Added default push request template

ENHANCEMENTS:
 * Bug fixes
   * 'go mod vendor' to fix build
   * Create cluster - filter classic ROSA account roles
   * Fix CheckAndParseVersion when coming from preRelease
   * Validate upgrade versions from prerelease scenarios
   * Updating vendor
   * Classic upgrade should align left
   * Log should be aligned to left
   * Hide billingaccount for classic
   * Doesn't check /labels for HCP
   * Better help message
   * Force --enable-autoscaling flag when autoscaling flags are used
   * Remove unnecessary warning messages about Red Hat owned subnets when user explicitly specifies subnets to use
   * Validate version and HCP default ingress attributes
   * Use output print slice instead of helper
   * Adjust security groups for machine pool help
   * Validate sgs flag out of byo vpc
   * CLI options dependency
   * 'rosa describe admin' cannot find 'cluster-admin' htpasswd idp and user (updated)
   * Improve subnet not foudn error message
   * Fedramp admin stage urls
   * Only go interactive on non hcp
   * Change references to default machine pool to worker machine pool

## 1.2.28 (21 May, 2026)

FEATURES:
   * Day2 security groups
   * Show additional sg ids on machine pools
   * Support day 1 additional compute sg ids
   * Add the ability to list clusters using an Account Role
   * Add interactive mode for shared VPC
   * Validate disk size during machinepool creation
   * Add a rosa describe machinepool command

ENHANCEMENTS:
 * Bug fixes
   * Doesn't check /labels for HCP
   * Better help message
   * Force --enable-autoscaling flag when autoscaling flags are used
   * Use output print slice instead of helper
   * Adjust security groups for machine pool help
   * Validate sgs flag out of byo vpc
   * CLI options dependency
   * Only go interactive on non hcp
   * Only show interactive SGs if there are any
   * Improve create operator roles error message
   * Adjust listed security groups
   * Improve create cluster in shared VPC info message
   * Rosa can list resources when cluster in hibernating state
   * Prompts with default value are not 'optional'
   * Remove root disk size from describe output
   * Newer version available point to console.redhat.com
   * Update help message on rosa upgrade role command to be consistent
   * Align version computation create machinepool for HCP
   * No version change in edit machinepool if upgrade scheduled
   * Don't show RH managed VPC options when selecting subnets
   * Don't ask for disk size for hosted cp
   * Error message to alert customer of new ingress attributes
   * Allow users to reset machinepool labels in interactive mode
   * Machine type when using disk size
   * CLI options not validated client-side
   * Cluster-autoscaler enabled for HCP
   * Autoscaler default values
 * Chores
   * Interactive helpers
   * Add arm64 builds to gitignore

## 1.2.27 (21 May, 2026)

FEATURES:
   * Add support for json/yaml output on all rosa list commands
   * Do not allow HCP to be used on govcloud for create/accountroles
   * Add 'rosa edit autoscaler' command
   * Add root disk size validation
   * Print acknowledgement when -y is passed
   * Add 'rosa create autoscaler' command
   * Changed Password Validator function to use shared library
   * Allow setting GPU limitations
   * Add automatic upgrades for node pools
   * Add 'Workload Monitoring' to describe/cluster
   * Support machine pools with zero replicas on day two
   * Add warning message for public ingress when switching HCP cluster visibility
   * Hypershift node pool upgrades

ENHANCEMENTS:
 * Bug fixes
   * Don't ask for disk size for hosted cp
   * Align version computation create machinepool for HCP
   * No version change in edit machinepool if upgrade scheduled
   * Error message to alert customer of new ingress attributes
   * Machine type when using disk size
   * CLI options not validated client-side
   * Cluster-autoscaler enabled for HCP
   * Autoscaler default values
   * Resource limits range defaults
   * Autoscaler created when not enabled
   * Check if autoscaler already exists before creation
   * Negative durations are allowed
   * Edit autoscaler examples
   * Better assignment to isGovCloud + const
   * Moved WithAWS() inline with NewRuntime()
   * Addressed review comments
   * Redid logic to use already existing parse func & creator struct
   * Used StringValue instead of dereferencing a string
   * Exit when error from using `--hosted-cp` on govcloud
   * Exit when unable to get caller
   * Created AWSClient in the right spot [preventing nil ptr]
   * Added 'gate' to the info message
   * Moved condition + updated message
   * Validation issues of configuring autoscaler interactively
   * Make prompt print even when -y is passed
   * Cores range validation
   * Set max-nodes-total default value to 180
   * Re-attach policies after detaching from operator roles
   * Reusing str
   * Adjust so that it doesn't show new ingress attributes when not supported
   * Responded to review comment
   * More accuracy to --worker-disk-size
   * Create operator-roles for shared VPC
   * Moved getMode to top of cluster creation so it checks for invalid mode firstly
   * Reversed 'bool'
   * Automatic control plane upgrade when no update is available
   * Validate oic-provider exists in manual mode
   * Edit nodepool was broken if no upgrade available
   * Inform user about visibility change impact in interactive mode
   * Add info message to create account roles
   * Handle verify network 'running' state
   * Handle output for listing operator roles
   * Validate cluster admin flags
   * Better warning message for RH managed vpc
   * Only show warning message about EOL if there is an EOL timestamp
   * Add a min/max validation
   * In interactive mode, version is required for node pools
   * Utilization-threshold missing from interactive mode
   * Change the wording for autoscaler params
   * Describe cluster compute node shows wrong value as we should not look on cluster.nodes.compute anymore
   * Add check for duplcate deletion of same cluster
   * Default values for boolean fields of autoscaler
   * Updated installation document
 * Chores
   * Remove tbrisker from owners
   * Bump ocm sdk version to v0.1.367
   * Extract autoscaler code to new modules
   * Fix pronoun usage in command help
   * Update cluster pattern in test
   * Build arm64 as part of release flow

## 1.2.26 (28 Aug, 2023)

FEATURES:
   * Hypershift node pool upgrades
   * Cluster autoscaler delete functionality
   * Updating create-cluster command with autoscaler configurations
   * Improve help message for the shared VPC flow
   * Warning and prompt for tech preview hibernation
   * Removing default pool from rosa cli
   * Support shared vpc in rosa
   * Add warning and prompt to continue when subnets are not found
   * Add shared pvc fields to cluster describe cmd
   * Ability to set machine pool root disk size
   * Create operator-roles - ensure shared VPC policy
   * Upgrade operator roles for the shared VPC flow
   * Add vpc id and order asc by vpc id
   * Bump the OCM-SDK-GO version
   * Include policies,namespace,name,in-use when listing operator roles
   * Warns when choosing a close to eol version
   * Add fedramp staging01 aliases
   * Create operator roles for the shared VPC flow

ENHANCEMENTS:
 * Bug fixes
   * Remove time.compare func for eol check
   * Handle output for listing operator roles
   * In interactive mode, version is required for node pools
   * Utilization-threshold missing from interactive mode
   * Change the wording for autoscaler params
   * Describe cluster compute node shows wrong value as we should not look on cluster.nodes.compute anymore
   * Default values for boolean fields of autoscaler
   * Add empty default ingress attribute in config
   * Use latest policies if user version is empty
   * Allow to choose load balancer type for sts clusters
   * ROSA CLI for IMDSv2 - Help displays same prompt instead of anything helpful
   * Adding default value of http tokens metadata for describe command
   * Take correct action based on vpc prompt
   * Use utility function to retrieve acc role suffix
   * Checks hcp role tag in all flows
   * Interactively validate issuer url and secret arn
   * Show roles with missing policies and order by prefix
   * Accept int64 max value for disk size
   * Add name to subnet details when selecting interactively
   * Check args not changed instead of empty is more reliable
   * Create operator-roles - hide hosted cp managed policies
   * Remove/hide HCP managed policies before release
   * No need to choose installer arn when already specified in args
   * Remove admin identifier for stage
   * More clarity into why it might have not found the prefix
   * Allow supplying empty values to reset component routes
   * Allow specifying empty values to remove route-selectors/excluded-namespaces
   * Add --region to verify network message
   * Replace deprecated rand.Seed()
 * Chores
   * Bump ocm-sdk to v0.1.362
   * Use SDK constants for upgrades instead of hardcodes

## 1.2.25 (10 Aug, 2023)

FEATURES:
   * Import openshift-online/ocm-common shared library as module
   * Move clusterNode validations code to shared library
   * Ability to set machine pool root disk size
   * Update vendor with apimachinery resource
   * Day1 operations for managed ingress attributes
   * Day2 operations for managed ingress attributes
   * Supports use local credentials cluster properties
   * Support htpasswd from-file flag in interactive mode
   * Add "USER DEFINED" in dns domain list
   * Support dns domain creation in rosa cli

ENHANCEMENTS:
 * Bug fixes
   * Allow supplying empty values to reset component routes
   * Allow specifying empty values to remove route-selectors/excluded-namespaces
   * Trim spaces in each excluded namespace
   * Clearer message when skipping option
   * Adjust help usage for cluster routes attributes
   * Adding default value for http tokens
   * Setting up the help in route-selector attribute
   * Bump OCM-GO-SDK
   * Add hostname and tls to builder
   * Determine role-arn for verify network
   * Replace deprecated ioutil.TempFile
   * Enables interactive if role arn is not supplied
   * The interactive mode should not exit after input invalid ingress label
   * Shouldnt ask for mode if supplied in args
   * Add mention for how labels are selected
   * Mention default values for args
   * Adding Http tokens value to describe command
   * Do not Allow IDP name cluster-admin
   * Empty checks against nil
   * Sanitize tags in place
   * Ensure policy versions use the cluster channel
   * Remove nudge command when creating account roles
   * Handle nil pointer in user tag validations
   * Update valid cluster states for verify network
   * Ensure operator roles always use the latest version
   * Update 'verify network' status message
   * Improve info message for the command to use to list upgrades
   * Version validation to check against correct channel group
   * Add 'rosa verify network' help message
   * Made info message only print when on auto mode
 * Chores
   * Remove changes document
   * Bump golang to 1.20

## 1.2.24 (21 May, 2026)

FEATURES:
   * Allow creation of multiple htpasswd idps
   * Add -c flag for listing operator-roles
   * Support delete dns domain
   * Add audit log details in rosa describe cluster
   * Support cluster admin creation during cluster creation
   * Feat:  Support loading users from htpasswd file
   * Support list dns domains in rosa cli
   * Adds admin flag for fedramp
   * Update the displayed output
   * Ensure only one of users or username flags is provided
   * 'rosa verify network' command
   * Register oidc config command
   * Remove environment check for managed policies
   * Expose `--hosted-cp` and `classic`  flags

ENHANCEMENTS:
 * Bug fixes
   * Ensure policy versions use the cluster channel
   * Remove nudge command when creating account roles
   * Handle nil pointer in user tag validations
   * Version validation to check against correct channel group
   * Ensure operator roles always use the latest version
   * Enable creating ROSA clusters with tags containing `:`
   * Add validation for roles
   * Hide available upgrades for HCP in list versions
   * Create operator-roles - hide hosted cp managed policies
   * Add output support for list dns domains
   * Remove/hide HCP managed policies before release
   * Show and choose default version first
   * Made prompt for OIDC Config ID not appear when flag given
   * Align FedRAMP non-admin token url and client ID for integration
   * Add existence check for OCM roles
   * Enable clusteradmin if user set --cluster-admin-user arg
   * Reuse operator roles - validate compatible policies
   * Edited error message to no longer include 'please try again'
   * Added command to INFO message for registering oidc config
   * Removed unneeded log about creating cluster -- affects UI
   * Validation for 'value' in tags
   * Remove debug msg that prints the user/password list
   * Accept symbols =,.@- in role name section of arn
   * Use default cidr if empty outside interactive
   * Adjust error message for HCP account roles
   * Squashed commit of the following:
   * Error message was misleading
   * Fix/filter-subnets-by-cidr
   * Check oidc provider from clusters within current aws account
 * Chores
   * Bump sdk to v0.1.352
   * Bump ocm-sdk v0.1.349
   * Bump ocm-sdk-go to v0.1.348
 * Other
   * Small refactor to prepare for nodepool upgrades
   * Accept path setting thru iam role check - regex
   * Bump ocm-sdk-go to v0.1.347
   * Create Hypershift dedicated account roles
   * ROSA CLI support for configurable ingress load balancer type

## 1.2.23 (26 Jun, 2023)

FEATURES:
   * Ensure only one of users or username flags is provided
   * Register oidc config command
   * Add automatic upgrades for hypershift control plane
   * Add 'in use' column list oidc providers
   * Add list oidc providers command

ENHANCEMENTS:
 * Bug fixes
   * Remove debug msg that prints the user/password list
   * Accept symbols =,.@- in role name section of arn
   * Use default cidr if empty outside interactive
   * Squashed commit of the following:
   * Error message was misleading
   * Fix/filter-subnets-by-cidr
   * Skip version check for automated upgrade
   * Ensure all users are copied
   * Ensure policy when calling delete policy versions was not checking for errors
   * Correct error message when 'rosa link user-role' with account-id under different org
   * Remove secondary yes check from confirmation
   * Add flag to bypass confirmation on edit cluster
   * Act on output flag for list service command
   * STS default warning does not show for hcp
   * Remove dial issuer url when creating operator roles by prefix
   * Create cluster - fix error messages
   * Enable interactive mode if missing name or spec path for tuning configs
 * Documentation
   * Add instructions to get rosa binary from latest
 * Other
   * Release v1.2.23
   * Release v1.2.23-rc2
   * Release v1.2.23-rc1
   * Small refactor to prepare for nodepool upgrades
   * Accept path setting thru iam role check - regex
   * Bump ocm-sdk-go to v0.1.347
   * Update OCM-SDK-GO to version 0.1.345
   * Exit link userrole cmd when errors
   * Add the ability to list all clusters
   * Bump ocm-sdk-go to v0.1.343
   * Accept multiple users for htpasswd IDP
   * Add debug reporter to indicate role prefix being used
   * Commit changes requested
   * Change account role name pattern roles used with HCP
   * Add specific commit binary image to README.md
   * Changing parameter name from HttpTokensState to Ec2MetadataHttpTokens
   * Revert "OCM-140 | chore : reverting IMDSv2"
   * Adds pull always to command
   * Adds explanation and solution to possible auth errors
   * Add Technology Preview to --help output for hosted-cp
   * Handle errors when retrieve versiongates

## 1.2.22 (17 May, 2023)

FEATURES:
   * Removed prompt for subscription billing account
   * Unhide hcp params
   * Remove hypershift capability check in list regions
   * Asks for OIDC Configuration kind in interactive

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.22
   * Goes through both operator roles type when deleting by prefix
   * If no such entity move operator roles creation forward
   * Properly checks the path when validating trusted relationship
   * Only validate installer arn if role name is not empty
   * Remidiating gramatical typo in hibernation help messages
   * Validate installer role arn after selection
   * Behavior when creating operator-roles
   * Check error when retrieving operator-roles
   * Create hcp account roles in interactive mode
 * Chores
   * Chore : reverting IMDSv2
 * Other
   * Hide hosted CP flags
   * ROSA Cli for IMDSv2 for OSD/ROSA
   * Remove os.Exit in cluster creation cmd

## 1.2.21 (05 May, 2023)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.21
   * Only ask mode if flag is not changed
   * Only print success message if no errors occur
   * Fix : only create route selector builder on ingress edit if route selectors exist
   * Fix : update action to run on each commit in a pull request
 * Other
   * SDA-8963 Improve machine type not found message
   * SDA-8718 Update rosa create message for hosted cluster
   * Improve error msg displayed when deletion of admin user fails
   * Allow editing ingress privacy for private hypershift cluster

## 1.2.20 (03 May, 2023)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.20
   * Aws path differs from ocm expected path

## 1.2.19 (03 May, 2023)

FEATURES:
   * Add messages and checks to ensure a better flow UX wise
   * Add interactive option for delete account roles
   * Defaults to managed oidc configs

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.19
   * Upgrade account roles - fix error message
   * Provide warning message when no OIDC config is found
   * OIDC provider behavior when calling internally from oidc config flow
   * Skip replicas when passing tuning configs to edit
   * Rosa upgrade hcp account roles - handle edge cases
   * Accept only official default prefix when selecting default prefix
 * Other
   * SDA-8689 Adjust default replicas on hosted clusters
   * Added billing account information to rosa describe cluster command

## 1.2.18 (01 May, 2023)

FEATURES:
   * Use OIDC Config ID to create OIDC provider and operator roles
   * Skip subnet choice when AZ is supplied
   * Add tuning config support for node pools
   * Add create/list/describe/update/delete tuning configs

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.18
   * Fix node pool edit/create behaviour when no tuning config in a cluster
   * Delete tuning config bad definition
   * Adjust behavior for creating with -o
   * Improve upgrade roles / operator-roles info messages
   * Add flag to bypass confirmation for hibernate/resume
 * Chores
   * Bump ocm-sdk to v0.1.338
   * Bump ocm-sdk v0.1.337
   * Removing init and global variables for hibernate/resume
 * Other
   * Golangci-lint feedback: simplified IsValidAWSAccount function
   * PR feedback: updated validation message and moved check for empty string out of IsValidAWSAccount function
   * Refactor oidcconfig and move a reuse-able code to a new helper packagex
   * Added billing account parameter for hypershift clusters
   * SDA-8690 Update replicas help message for hosted-cp cluster
   * Expose scheduling time for hypershift upgrade

## 1.2.17 (24 Apr, 2023)

FEATURES:
   * Implement delete upgrade command for hypershift
   * Only allow aws auto mode when choosing and output flag enabled
   * Add --etcd-encryption-kms-arn to cluster build command when in use
   * Add output json/yaml to create oidc config
   * Add aws account ID to filter unmanaged OIDC configs
   * Add bucket policy to allow public read
   * Check if there are clusters using operator roles prefix
   * Add oidc config id to message

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.17
   * Correctly check if an upgrade is already scheduled
   * Implement list upgrades for Hypershift
   * Ask for drain period before scheduling the update
   * Shared operator roles with managed policies
   * Upgrade account roles - support hosted CP managed policies
   * Add more validations before terminating run for upgrade roles
   * Trims trailing slash in ocm api when checking env
   * Remove --tags param from example for usage
   * Add message to helper --delete param on init cmd
   * Small issue when customer sets up oidc provider with issuer url '/' at the end
   * Better messages when listing operator roles for specific version
   * SDA-8804,8805 | fix: better behavior for listing operator roles
   * Fix version validation error handling
   * Use parameter installer role arn when passed
   * Version tests
 * Chores
   * Add initial github action to checking commit message format
   * Remove deprecated lib `io/ioutil`
 * Documentation
   * Update contribution guide with commit hygiene
 * Other
   * Warn about `version` flag in combination with HCP managed policies
   * Align label and taints validation between create and edit
   * Disable quoting of values - makes request/response dumps readable

## 1.2.16 (10 Apr, 2023)

FEATURES:
   * Add list operator-roles command
   * Add type of cluster in list clusters command
   * Expose oidc config commands and params
   * Better UX when using oidc config id and related commands
   * Extra message when operator roles prefix is already in use
   * Add message informing which role is being deleted
   * Update oidc config commands to use /oidc_configs endpoints
   * Use oidc-config-id in cluster flow
   * Allow creating operator roles using prefix and byo oidc options
   * Add possibility to delete operator roles from prefix
   * Add option to remove oidc provider created from BYO OIDC
   * Add description when available for error state in describe cluster
   * Add possibility to reuse operator roles

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.16
   * Proper naming for topology description in list clusters
   * Force interactive enable if required params for hcp are not supplied
   * Checking Account roles against proper Openshift version for the 'create service' command.
   * If -y is specified no need to go into interactive mode creating oidc-config
   * Always show message for unregistering oidc configuration
   * Improve taint validation
   * Interactive mode --classic-oidc-config param
   * Fixing some ui issues for oidc configs
   * Ux for oidc-config cmds and cluster creation
   * Favoring regex when deleting operator roles by prefix
   * Interactive mode in unmanaged oidc config creation
   * Cluster flow oidc provider flow should use issuer url instead
   * Check specific prefix instead of all op roles that start with prefix
   * Use oidcConfigIdFlag instead of var
   * Fix hostedcp multiaz subnets validation on interactive mode
   * Adhering to aws change where it now returns 404
   * Add check for cluster version compatibility when reusing the operator roles
   * Use role name instead of role arn for manual creation by prefix
   * Add reachability check for oidc endpoint url when creating operator role by prefix
   * Does not check flags when is progmatically called
   * Check if cmd was progmatically called before erroring
   * Fix inconsistencies across commands providing a watch flag
   * Block --watch when on manual mode for creating cluster
   * Add red-hat-managed tag to oidc config and oidc provider resources
 * Other
   * Ensure stdout is only printed in terminal mode
   * Upgraded github.com/openshift-online/ocm-sdk-go v0.1.327 => v0.1.330
   * Oc client version local check only
   * Better way of checking for hcp-enabled versions
   * Added hosted-cp to rosa list versions
   * Add control plane upgrades for Hypershift cluster
   * Delete account-roles - delete both types for the default flow
   * SDA-8551 Sync replicas validation for hosted clusters
   * Fixed rebase issue
   * Update versions to include hypershift_enabled and add validation
   * SDA-8220 Support autorepair parameter for hosted clusters
   * Delete account roles - classic ROSA
   * Revert "[SDA-8369] Update versions to include hypershift_enabled and add validation"
   * Addressed review comments
   * Hypershift -> HostedControlPlane
   * Add `hypershift_enabled` to versions (similar to `rosa_enabled`)
   * Upgrade cluster with hosted CP policies
   * Delete account-roles - handle Hypershift roles
   * Create Hypershift operator roles
   * Hypershift - create cluster
   * Create account roles - prompt accurate message for cluster creation
   * Create account roles - hide `hosted-cp` flag
   * SDA-8224 Deprecate MultiAZ/SingleAZ validation for hosted clusters
   * Create Hypershift account roles with managed policies
   * Bump SDK to v0.1.324 and go mod vendor+tidy
   * Go mod tidy
   * Ran go mod vendor
   * Bump sdk to 0.1.322
   * SDA-8218 Support version parameter on hosted machine pools
   * Create OCM admin roles in manual mode - add all tags to policy
   * Update k8s.io/apimachinery to v0.26.2
   * SDA-7231 Update rosa describe for hypershift
   * SDA-8040 Align machinepool condition for enter in interactive mode
   * Create cluster - roles with managed policies
   * SDA-8325 Add subnets field for Default Worker
   * Describe cluster - add managed policies field to the output
   * Use cluster attribute `ManagedPolicies` to identify cluster with managed policies

## 1.2.15 (23 Feb, 2023)

FEATURES:
   * Set byo oidc enabled when specifying byo oidc attributes
   * Check if any clusters are using the oidc config

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.15
   * Missing '--' for the oidc endpoint url flag
   * Using k8s/apimachinery/validation for labels
   * Fix etcd encryption default enforecment
   * Release v1.2.15
   * Add a few more validations to bucket/folder name
   * To fix empty DNS domain when DNS not ready: SDA-7418
   * Allow empty label match editing ingress interactive
   * Adding some validations to bucket name
   * Improve error messages for deleting oidc-config
 * Chores
   * Update changes for 1.2.15
 * Other
   * Bump ocm-sdk-go version to v0.1.319
   * Use latest OCP version instead of the default version
   * Change managed policies flag name to `aws-policies`
   * Add labels and taints to the list machinepools command
   * Add support for labels and taints to HostedCP NodePools
   * Improve logging so that it's more obvious what is wrong
   * Update `CHANGES.adoc` with the recent bug fix
   * Add etcd encyprtion kms arn support
   * Add a `AWS managed` column to list role commands
   * Attach three policies to the installer role - managed policies
   * Ensure ARNs have the correct partition

## 1.2.14 (08 Feb, 2023)

FEATURES:
   * Add command delete oidc-config and minor fixes
   * Add spinner creating oidc config
   * Add user prefix to oidc configuration

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.14
   * Add question for private key secret arn
   * Oidc endpoint url should be of https scheme
   * Forcing creation only works for unmanaged policies
   * Show info report when deleting operator roles
   * Add region when creating manual s3 bucket for oidc config
   * Only show root failure if it is not a suggestion error
   * Incorrect flags in message and hidden for upgrade roles
   * Permissions boundary shouldn't be asked if supposed to skip interactive
   * Reprompt user when passed invalid addon argument
 * Other
   * Add confirm flag
   * SDA-7895 Enable editing subnet in machinepools for hosted clusters
   * Refactor `create account roles command` to use interfaces
   * Create cluster - validate account roles have managed policies attached
   * Add port for OAuth Callback URI in OpenID
   * IDP related minor cleanup
   * Edit machinepool min replicas
   * Minor refactor to improve code clarity wrt addons

## 1.2.13 (25 Jan, 2023)

FEATURES:
   * Add force param to forcefully ensure policies
   * Add download rosa option
   * Store private key for byo oidc in secrets manager

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.13
   * [SDA-7757] byo OIDC secret arn support (#1018)
   * Fix managed policies cluster upgrade path
   * Skip missing acc roles during upgrade
 * Other
   * Delete OCM role with managed policies
   * Upgrade to Go 1.19
   * Create OCM role with managed policies
   * Delete roles with managed policies
   * Bump ocm-sdk-go to v0.1.310
   * Remove GitHub IDP dependency to console availability
   * Skip region check if we use shard pinning
   * SDA-7767 Let user define machinepool name on hosted cluster

## 1.2.12 (18 Jan, 2023)

FEATURES:
   * Consider current version incompatible
   * Add manual mode
   * Command create oidc-config

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.12
   * Accept pre release version during upgrade
   * Delete correct operator roles and support hypershift
   * Only checking '"' character and leaving regex validation for CS
   * Using ARNValidator instead of arn Parse when parsed is not used
   * Fix tags passing in cluster creation interactive mode
   * Edited query for GetClusterUsingSubscription to fix deletions
   * Review comments
   * Add mocks
   * Fix OAuth callback URL logic for HyperShift clusters
   * Code review
   * Check current values to see if there are no real changes
   * Incorrect OIDC Provider Sometimes Targeted for Deletion
 * Other
   * Support output flag (#1014)
   * Ensure console URL is available before offering it to the user
   * Upgrade roles command - handle managed policies
   * Used better flag
   * Upgrade operator roles with managed policies
   * Added constant for "auto" string
   * Now updating stsBuilder.AutoMode properly
   * Add the default-mp-labels flag to cluster create command on interactive
   * Re-added tags question
   * Fixed panic
   * Upgrade account roles with managed policies
   * Revert "[SDA-7662] Display Tags question in interactive mode"
   * Removed len(tags) > 0 from if for Tags question in interactive

## 1.2.11 (03 Jan, 2023)

FEATURES:
   * Hide region from other globally available commands
   * Using LCP to retrieve operator policy prefix
   * Retrieve operator role prefix from backend
   * Validates machine pool label

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.11
   * Release v1.2.11
   * Fix linter errors - add constant for string "true"
   * Fix bug - create managed account roles
   * Fix scaling bug and improve interactive mode
   * Using tabwritters options instead of manually formatting
   * Fix ux issues related to rosa describe
   * Remove channel group from recreate output, this is treated within creation flow
   * Use default/in-place value for addon param first
   * Removing local contains in favor of helper.Contains
   * Hide region arg in account roles commands
   * Allow editing default machine pool labels
   * Accomodate inline policies in new upgrade roles flow
   * Using lower case before comparing expected acc role arns
   * Add error message when CA is passed but github hostname is not
   * Using unified path on upgrade roles/operator-roles
   * Upgrading from pre release would fail to validate version
   * Remove auth url info from LDAP idp when listing
   * Phrasing
   * Spacing listing users
   * Check if any new operator roles have been created
 * Chores
   * Refactor sort strings helper
 * Other
   * SDA-7521 Support instanceType selection on NodePools
   * Create operator role with existing policies
   * Refactor get policy details and ARN
   * Create account roles with existing policies
   * Refactor `GetPolicies` function
   * Bump OCM SDK GO version to v0.1.303
   * Make rosa describe upgrade
   * Add default machine pool labels validations
   * Enable day1 default machine pool labels
   * Adjust NodePool headers
   * Describe cluster - print `infra_id` to the output
   * Log a warning if the user's organization doesn't have the needed capability
   * Support nodepools under machinepool commands
   * Upgrade to ocm-sdk-go 0.1.299

## 1.2.10 (01 Dec, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.10
   * Going through all policies to check upgrade
 * Other
   * Check HostedCP version support also in interactive mode + align versions

## 1.2.9 (28 Nov, 2022)

FEATURES:
   * Checking undefined aws region
   * Add channel group and option to choose version for policy tags in upgrade roles cmd
   * New upgrade roles command and some refactors
   * Add aws command builder unit tests
   * Add warn messages about sts/non sts modes
   * Check if rosa cli is up to date
   * Don't send update request if there are no changes
   * Set interactive enabled if local flags are unchanged, except for cluster flag

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.9
   * Handle empty strings before validation
   * Always show warning, don't go into interactive if mint mode or non sts flags are enabled
   * Invert condition for no reason to update
   * Prompt mode for upgrade cluster when sts and mode is empty
   * Remove not needed vars in favor of using the args
   * Order of messages
   * Sort prefixes to ensure consistancy when they are the same rank
   * Fix bug - create a machine pool with a different region configured in the AWS CLI
   * Don't show if redirecting to file
   * Message non sts
   * Message when non sts
   * Specify which flag in message
   * Add tags check when b.tags nil
   * Fix hosted cluster parameter in create cluster
   * Favor replicas instead of deprecated compute-nodes param
   * Lint
 * Chores
   * Manual aws command builder
   * Clean up
   * Setting up a function to look into given params expected to be unchanged
 * Other
   * Remove redundant quotation
   * Adding check for Changes in replicas flag too
   * Move HostedCP region supports check to the backend side
   * Create machine pool - display spinner when fetching instance types
   * Create a machine pool - prevent choosing a spot instance for a local AZ
   * Create machinepool - filter supported instances by availability zones
   * Show Limited Support status when calling `rosa describe cluster`
   * Bump OCM SDK GO version to v0.1.293
   * Revert "[SDA-6643] STS is now default mode for cluster creation, added flags for mint mode/non-sts mode"
   * Unhide flag
   * Bump OCM SDK GO version to v0.1.292
   * Output current environment when it is not production
   * Create/oidcprovider bug sets interactive.Enable
   * Unhide tags during cluster create
   * Add `--yes` to create cluster cmd
   * Transformer added to change escaped empty strings to real empty strings
   * STS is now default mode for cluster creation, added flags for non-sts
   * Update owners file
   * [Hypershift] Filter regions where HostedCP is avalaible
   * [Hypershift] Filter regions where HostedCP is avalaible
   * Removed --channel-group  from --help options.
   * Revert "[Hypershift] Filter regions where HostedCP is avalaible"
   * [Hypershift] Filter regions where HostedCP is avalaible
   * [SDA-6984] Add support for nightly builds for HyperShift
   * Add market specific billing options for addon installations
   * Update hypershift naming convention for latest SDK
   * Bump ocm sdk to 0.1.289
   * Hosted-cp flag now forces byo vpc prompt

## 1.2.8 (13 Oct, 2022)

FEATURES:
   * Add validation to path ocm/user roles

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.8
   * Change message from one minute wait for several minutes
   * Aws acc id on whoami
   * Aws empty path is different than ours
   * Invert path detected message condition
   * Adding leading space before all path args when building commands
   * Accepts empty path
   * Consider empty path valid creating acc roles
   * Remove error to add support for path in ARN
   * Block managed services path option
   * Differentiate between '/' and /
   * Setting path arg in a new line for all commands
   * Adding conditions for piping the output
   * Clearer message
   * Adding arn path validator to create account roles --path arg
   * Path args need not to be explicitly set for interactive mode to ask about it
 * Chores
   * Bump go ocm sdk v0.1.288
   * Less hacky
   * Add gdbranco to reviewers and approvers
 * Other
   * [SDA-6984] Added unit tests
   * [SDA-6984] Remove channel group in error message when unsupported OCP version is provided for hosted cluster
   * Add renan-campos to reviewers, approvers, and maintainers
   * [SDA-6760] Add validation for minimum supported OCP version in HyperShift

## 1.2.7 (03 Oct, 2022)

FEATURES:
   * Adding message about operator roles and policies path
   * Deprecate 'compute-nodes' args in favor of 'replicas' in create cluster cmd
   * Add --output json/yaml to create admin, omit when passwordArg is set
   * Remove operator role path in create cluster in favor of master role path
   * Removing path from ocm-role as it is not supported. oidcProvider already didn't had a path arg
   * Unify operator role and policy with path from account roles
   * Unify acc roles its policies paths
   * Adding -o yaml/json option to cmd whoami

ENHANCEMENTS:
 * Bug fixes
   * Lint
   * Message
   * Unwanted change
   * Defer cleanup
   * Path compatibility issue with inline policies from acc roles
   * Lint
   * Enable path arg visibility
   * Remove path arg from -h ocm-roles description
   * Adding trim spaces and tabs when validating cluster name
   * Lint
   * Adding back ocm-roles path option and keeping it hidden
   * Remove deprecated linter
   * Review changes
   * Using installer instead of control plane role for path
   * Remove operator-role-path from generated create cluster command as it was deprecated
   * Getting path from master instance role
   * Missing changes for --role-path
   * Lint
   * Adjusting order of calls to make sure deletion calls aren't being duplicate, this caused a 500 error on login after deleting and recreating admin from a newer rosa cli
   * ':' character was at the wrong place
   * Reporting correct message back if specific version was chosen
   * Description of version arg
   * Changing description for channel group
   * Fix manual create operator policy sda-6740
   * Fix setting interactive mode enable for addon installation billing mode
   * Fix- It failed to set empty value with "" for no_proxy filed via interactive mode
   * Fix - Google IDP doesn't work when created with ROSA CLI
 * Chores
   * Add gdbranco github user to owners file
   * Adding strategy and function to check if created on old ROSA
   * Rebase
 * Other
   * [SDA-6075] Add upgrade policy to rosa struct information when displayed with the rosa describe cluster with -o json or -o yaml
   * Rosa STS mode auto conflicts with the watch option
   * Added redhatmanaged=true tag to operator roles in manual mode
   * Added RedHatManaged=True to manual operator/account/user roles creation
   * Update stage console URL
   * [Hypershift] Arg validation for Hypershift clusters
   * [Hypershift] Modify `describe cluster` to differentiate between classic vs Hosted-cp
   * [Hypershift] Enable subnet validation for Hosted clusters
   * Fetch all regions for non-interactive mode
   * Adding escaped carrier to start of --path argument in ocm-role
   * Removing unnecessary hypershift check for managed services.
   * Checking addon params
   * Upgrade	cluster	manual mode - prompt the aws operator role upgrade commands
   * [Hypershift] Modify `describe cluster` to differentiate between classic vs Hosted-cp
   * Set mode only once in operator roles upgrade
   * Add also local-proxy env config
   * Improve `EnsurePolicy` error message
   * Create cluster - list region filtered by OCP version
   * Added redhatmanaged=true tag to roles when `rosa upgrade operator-roles` is ran
   * Add support for Hypershift cluster creation
   * Upgrade OCM-SDK-GO version to 0.1.287
   * Path for account and operator roles and policies
   * Ensure prerequisites for deleting operator and account role policies
   * Hide arn path flags
   * Move operator policies from account to operator commands
   * Red-hat-managed=true tag now added to operatorroles
   * Add aliases for local development
   * Deleted account and operator policies
   * Only display supported machine types by region
   * Allow using local AWS credentials
   * Ensure policy version succeeds
   * Bump ocm sdk to 0.1.285
   * Allow setting billing model for addong installations
   * Missing '--operator-roles-path' in 'To create this cluster again...'
   * Compare arn path for existing policy/role
   * Revert PR#787
   * Adding an info message after `rosa delete admin`
   * Add red-hat-managed tag to roles and policies
   * Add arn path to ocm and user role
   * Refactor `GetCluster` function
   * Create cluster - use a GET request to describe cluster details
   * Support for path in iam roles and policies
   * Use root CA to generate OIDC thubmnail
   * Refactored ROSA to create operator policies when running `rosa create cluster`
   * Add samira to maintainers

## 1.2.6 (05 Aug, 2022)

ENHANCEMENTS:
 * Other
   * Release v1.2.6
   * Ran go mod vendor after rebasing
   * Updated SDK version and ran go mod vendor
   * Switch from github.com/pkg/errors to stdlib
   * Create cluster - for single AZ, only allow to select one AZ
   * Removed change to query
   * Replaced display_name with name in query
   * Removed DisplayName from cluster
   * Ensure there is no default network type
   * Update rosa-authenticator configuration
   * Add fake cluster parameter to create services
   * Remove token from error output
   * Remove AWS info from debug output

## 1.2.5 (20 Jul, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.5
   * Create cluster - validate availability zones count interactively
   * Fix for - Not able to remove or add a new cluster-admin in rosa cli
   * Fix typo in error message when looking up account role prefix
 * Other
   * Add environment-specific configuration (#702)
   * Delete admin should not deleted htpasswd idp as the htpasswd list is not empty

## 1.2.4 (12 Jul, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Fix bug - create cluster - validate subnets number for private link
   * Fix bug - fetch the subnets from the cluster region
   * Fix bug - create a cluster with an invalid number of subnets
   * Fix error message - create non-BYOVPC - select availability zone
 * Other
   * Add oriAdler to OWNERS in order to tag a ROSA release
   * ROSA release 1.2.4
   * Removing htpasswd idp even if there are no users in this idp
   * Accept major minor version for cluster upgrade
   * Not able to remove or add a new cluster-admin in rosa cli
   * Setting useVPCExist to true when subnet ids are provided
   * Set `clusterKey` properly to support `r.FetchCluster`
   * Removed validator object
   * Create cluster - validate subnets count interactively
   * List machine pools - add a subnets column
   * Create a single AZ machine pool implicitly by providing a subnet
   * Create cluster - detect multi-AZ cluster
   * A different approach to parsing the flags.
   * Add basic STS addon installation flow
   * Skip role version comparison for unversioned roles
   * Refactor Role PolicyDoc creation
   * Set `clusterKey` properly when calling commands programmatically
   * Select a single subnet for a single AZ machine pool - BYOVPC clusters
   * Refactor function for more general use
   * Update ocm-sdk-go to v.0.1.275
   * Addressing pr comments
   * Edit service can update parameters that weren't originally defined.
   * Migrate commands to fetch cluster using runtime
   * Add FetchCluster method to runtime
   * Migrate commands to use runtime GetClusterKey
   * Add GetClusterKey to runtime
   * Migrate remaining commands to use runtime
   * Migrate list commands to use runtime
   * Migrate whoami to use runtime
   * Migrate verify to use runtime
   * Migrate uninstall addon to use runtime
   * Migrate unlink to use runtime
   * Migrate revoke to use runtime
   * Migrate resume to use runtime
   * Migrate logs to use runtime
   * Migrate login to use runtime
   * Migrate edit service to use runtime
   * Migrate link to use runtime
   * Drop unused CheckStackReadyForCreateCluster method
   * Migrate initialize to use runtime
   * Migrate hibernate cluster to use runtime
   * Migrate grant user to use runtime
   * Migrate edit machinepool to use runtime
   * Migrate edit ingress to use runtime
   * Migrate edit cluster to use runtime
   * Migrate edit addon to use runtime
   * Migrate FindExistingHTPasswdIDP method to use runtime
   * Migrate create service to use runtime
   * Migrate create ingress to use runtime
   * Migrate create machinepool to use runtime
   * Migrate create oidcprovider to use runtime
   * Migrate create ocmrole to use runtime
   * Migrate create userrole to use runtime
   * Migrate create operatorroles to use runtime
   * Migrate create idp to use runtime
   * Migrate delete service to use runtime
   * Migrate delete upgrade to use runtime
   * Migrate delete operatorrole to use runtime
   * Migrate delete userrole to use runtime
   * Migrate delete oidcprovider to use runtime
   * Migrate delete ocmrole to use runtime
   * Migrate delete ingress to use runtime
   * Migrate delete idp to use runtime
   * Migrate delete cluster to use runtime
   * Migrate delete admin to use runtime
   * Migrate delete accountroles to use runtime
   * Migrate describe commands to use runtime
   * Provide shell completion for IdP types
   * Allow to select availability zones when creating a non-BYOVPC cluster
   * The additional-trust-bundle-file can't be set via interactive mode if the cluster is not set proxy fields
   * Initial implementation of runtime
   * Migrate some commands to use runtime

## 1.2.3 (20 Jun, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.3
   * Fix bug - remove replicas constraint when editing single AZ machine pool
   * Fix order of instance types
   * Fix a bug when editing no-proxy field
   * Addon install -	fix bug	- do not print not-set parameters
 * Other
   * Add GetAllowedActions PolicyDocument method
   * Make PolicyDocument creators return pointer
   * Make checkPermissionsUsingQueryClient a method of PolicyDocument
   * Add String() to PolicyDocument
   * Create a single AZ machine pool - availability zone flag
   * Prompt the user to select multi or single AZ only in an interactive mood
   * Ensure all flags passed during managed service creation are used.
   * Add more throttle metrics
   * Select a single AZ for a machine pool in a multi-AZ cluster
   * Simplify logging package
   * Add helpers for creating a policy document and allowing actions
   * Add GetPrefixFromOperatorRole and TrimRoleSuffix helpers
   * Add Operator Role to cluster
   * Support -c flag when using "rosa describe addon-installation"
   * Support host-prefix during managed service creation
   * Add AllowsAction method to PolicyDocument
   * Move GenerateRolePolicyDoc method to policy_document
   * Unify multiple SaveDocument implementations
   * Refactor GetRolePolicyDocument into InterpolatePolicyDocument
   * Drop unused aws.ReadPolicyDocument method
   * Extract policy document structs to separate file
   * Support creation of managed services with non-custom configurations
   * Allow login with encrypted tokens
   * Remove external org ID if empty
   * Allow tokens without 'typ' claim
   * Creating htpassword idp still prompts for username even if provided
   * Unhide ocm/user link/unlink role
   * Command to list parameters of add-on installation
   * Customizable network configuration in service creation
   * Bumping ocm-sdk-go to v0.1.272
   * The wildcard domain is not allowed to set in no_proxy field
   * Reject '*' when validating no-proxy field
   * Reduce extra call to OCM when manipulating addon installation
   * Add group support for OpenID IDP in ROSA CLI
   * Adding private-link flag to managed service create
   * Make CredRequest API
   * Update to OCM SDK 0.1.268
   * ROSA - Allow for additional, customer-provided "no_proxy" values for cluster-wide proxy
   * Add credential requests to describe addon command
   * Update templates
   * Run go mod tidy
   * Bump OCM-SDK to 0.1.266
   * Create user-role - improve help message
   * Remove version dependency from rosa
   * Addon install - add non-interactive commands
   * List parameters when describing managed services
   * Adding command to update managed service

## 1.2.2 (11 May, 2022)

ENHANCEMENTS:
 * Other
   * Release v1.2.2
   * Unhide ui roles
   * Adding new alias for managed service commands
   * SDA-5889-Fix User Role Error
   * Supporting flag values that contain equal signs
   * Allow addons be edited, regardless of addon state
   * Output validation error message when creating service with invalid
   * Adding single-az byo-vpc support for create service
   * Update dev script

## 1.2.1 (22 Apr, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.1
   * Fix Throttle issue for Operator roles
   * Fix login error
 * Other
   * Supporting different regions for create service command
   * Add metric for throttle
   * Only prompt for HTPasswd IDP name when actually creating a new IDP
   * Upgrade cluster to 4.10.* - add delay after roles creation
   * Add username & password requirements to the flags' help messages
   * Warn that deleting HTPasswd IDP with cluster-admin user will delete the admin
   * Support for cmk multi-region keys

## 1.2.0 (18 Apr, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.2.0
   * Fix bug - delete account roles - make the `prefix` flag optional
   * Fix error message - rosa create ocm-role
   * Fix error message - rosa delete ocm-role
   * Fix `rosa describe admin` to look at HTPasswd IDP users to determine existence of admin
 * Other
   * Adding support for byo-vpc in creating services
   * Updated ocm sdk to v0.1.262
   * Remove AUTH URL from HTPasswd entries of `rosa list idps`
   * Add username & password validations in CLI
   * Enhancing usability of managed service commands
   * Added command to delete managed services
   * Added command to describe managed services
   * Added command to list managed services
   * Added command to create managed services
   * Add global color flag
   * Fetch sts policies from ocm

## 1.1.12 (05 Apr, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.12
   * Fix json output for cluster creation
   * Fix early exit in cluster creation(json+mode=auto)
   * Fix cluster creation hanging with auto+watch flags
   * Fix throttle delay
   * Fix bug - create ocm-role - prompt the role ARN
   * Fix help for --compute-nodes
 * Other
   * Update to ocm-sdk-go v0.1.258
   * Get Cluster Name from Name Instead of DisplayName
   * Add max throttle delay to avoid exponential backoff
   * Automatically select default account roles
   * Add support for 4.10 upgrade
   * Add more permissions to ocm admin role
   * Permit overriding confirmation prompt for cluster upgrades
   * Added link to help menu
   * Add KMS permission to installer and more permissions for ocm role
   * Introducing HTPasswd IDP
   * Set minimum retry delay for AWS API calls
   * Add quota service permissions to the `installer policy`
   * Refactor `list ocm-role` to use a map of linked role
   * Sort roles to display linked ones first

## 1.1.11 (08 Mar, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.11
   * Fix operator role issue
   * Fix operator roles issue for old rosa versions

## 1.1.10 (07 Mar, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.10
   * Fix operator policies for 4.10
   * Improve `rosa unlink user role` error message
   * Fix bug - validate role type before deletion
   * Fix upgrade issue
   * Fix bug - `delete ocm-role` should be hidden in rosa cli
   * Fix bug - change the interactive message of `rosa delete user-role`
   * Fix bug - deletion of a role with the wrong account ID in role ARN
   * Fix bug - forbid deletion of ocm-role in case user cannot unlink role
   * Fix cosmetic issues rosa upgrade
   * Fix bug - capitalize `rosa unlink user-role message`
   * Fix bug - improve the help message of 'rosa unlink ocm-role'
   * Fix bug - add warning when creating ocm-role with duplicate name
 * Documentation
   * Create command 'rosa list ocm-roles'
 * Other
   * Revert "Introducing HTPasswd IDP"
   * Revert "HTPasswd bug fixes corresponding with some CS changes"
   * Add new support policy and policy for ovn networking
   * Sda-5576-Fix upgrades to 4.9 or less with 4.10 operator roles
   * Add support for seamless upgrade from any rosa version
   * HTPasswd bug fixes corresponding with some CS changes
   * Add policies for 4.10
   * Introducing HTPasswd IDP
   * List roles - display a spinner while fetching the roles
   * Add 'rosa delete user-role' command
   * Display HTPasswd IDP when listing a cluster's IDPs
   * Add 'rosa delete ocm-role' command
   * Create command 'rosa unlink user-role'
   * Added policies for ocm admin role
   * Create command 'rosa unlink ocm-role'
   * Create command 'rosa list user-roles'
   * Modify function `handleErr` to include the error type in the new error
   * Initial MachineTypeList implementation
   * Refactor setting available quota for MachineTypeList
   * Refactor GetMachineTypes to use MachineTypeList
   * Refactor ValidateMachineType to use MachineTypeList
   * Refactor GetAvailableMachineTypeList
   * Unify quota calculation logic for MachineType
   * Drop unused GetMachineTypeList method
   * Update linter configuration to newer version

## 1.1.9 (01 Feb, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.9
   * Fix bug - remove duplicated error message when deleting a cluster
 * Other
   * The attribute name in error message should be same with the correct proxy attribute in body
   * Add IAM List and Get role permissions to support policy
   * ROSA CLI Interactive install - make the choice default STS

## 1.1.8 (27 Jan, 2022)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.8
   * Fix missing vendored module
   * Fix issue with delete operatorrole/oidcprovider role
   * Fixed upgade' to 'upgrade'
   * Fixed issue with operator role upgrade
   * Fix etcdEncryption
   * Fix linter errors
 * Other
   * Add word wrapping to list gates output
   * Add cluster flag for list gates
   * Add ack gate support
   * Revert "Verify chosen machine pool type is available"
   * Update OWNERS file
   * Verify chosen machine pool type is available
   * Avoid nil pointer dereference in cluster create
   * Remove openshift version from operator role name
   * Add version gate ackto ROSA
   * Add gate support in rosa cli cluster upgrade
   * Add rosa cli version to header
   * Enable interactive mode when missing required flags
   * Clean up upgrade command
   * Add network type selection
   * Bump go version to 1.16
   * Update to Ginkgo 2
   * Update to version 4 of JWT library
   * Generate static assets for STS support permissions
   * Send rosa cli login event to pendo
   * Add stop and run instance permissions for support
   * Adding password argument to create admin

## 1.1.7 (08 Dec, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.7
   * Fix org admin validation
   * Fix call to link cmd
   * Fix crash when calling link cmd internally
 * Other
   * Added validation for ocm-role
   * Make `--admin` flag idempotent
   * Add --admin option to create ocm-role command
   * Add pendo eventor account roles manual mode
   * Remove hard dependency on default region
   * Add permission for describe region and route tables
   * Ignoring environment config
   * Ignore .envrc (DirEnv)
   * Change rosa init help message
   * Improve UX in ROSA edit cluster and ROSA delete roles
   * Add org admin validation for ocm-role
   * Add permission to describe VPCs
   * Allow FIPS mode support
   * Allow linking multiple role ARNs
   * Support editing cluster-wide proxy
   * Add org external id to ocm role name

## 1.1.6 (22 Nov, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.6
   * SDA-5022 : fix operator role upgrade being blocked by account role upgrade
   * Fix interactive setting of `mode` option
   * Fix minor typo
   * Fix proxy config validations
 * Other
   * Added edit support for UVM
   * Add interactive mode for link user/ocm role
   * Added support for operator prefix
   * Create OCM Role
   * Add ocm user role
   * Added support for master-iam-role
   * Clean/fix role validation for upgrade
   * SDA-5018 : improve cluster upgrade manual flow to prompt user to upgrade roles
   * SDA-5017 : improve cluster upgrade manual mode to print operator role commands
   * Validate sts roles on sts cluster upgrade
   * Add ROSACLI/version to User-Agent string
   * Changing cluster proxy attirbutes to pointers
   * Removed --enable_proxy argument
   * Add rosa upgrade account/operator role
   * Check for pre-existing operator roles and error if they exist
   * Clarify `verify permissions` cmd is only for non-STS clusters
   * Updated pendo event for rosa
   * Handle minor issues in delete handling
   * Add missing update message for default machine pool
   * Support cluster-wide proxy during cluster creation
   * Update OWNERS file

## 1.1.5 (21 Oct, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.5
   * Fix issue with delete account roles for older rosa
   * Fix NPE when fetching AWS statement principals
   * Fix sts mode validation
   * SDA-4911 : Fix creating operator roles prefix
 * Other
   * Add --mode to create command output
   * Respect disable-uwm flag default
   * Removed operator roles check from oidcprovider
   * Update delete cluster
   * SDA-4912 add retryer to aws client
   * Add '--sts' to interactive command output
   * Print spot instances when listing machinepools
   * Unhide Spot instances
   * SDA-4916 add validation to sts cluster create mode flag
   * Group account roles by prefix
   * Added account role deletion
   * Delete oidc provider and operator roles
   * Attach permission policies to roles
   * Merge compatible policies
   * Autocomplete cluster names on --cluster flag
   * Add providers for various shells

## 1.1.4 (13 Oct, 2021)

ENHANCEMENTS:
 * Other
   * Release v1.1.4
   * SDA-4773 : Support --mode on create cluster --sts
   * Don't print info logs when redirecting `create account-roles`
   * Bump OCM SDK
   * Rename master instance role to control plane
   * Do not validate Role ARN on IAM clusters
   * SDA-4744 : Add account role validation on cluster create

## 1.1.3 (30 Sep, 2021)

ENHANCEMENTS:
 * Other
   * Release v1.1.3
   * Allow compatible policies to create clusters
   * SDA-4829 update getThumbprints to use http package instead of tls
   * Silently ignore AccessDenied errors when validating resources
   * Bump OCM SDK to v0.1.209
   * Add kmskey for sts
   * Add StopInstances action to support Hibernation
   * Remove ROSA init account command
   * Support custom properties
   * Update user tag regexp to include unicode spaces
   * Add disable workload monitoring to ROSA
   * Add script to list JIRA tickets addressed on current release
   * Add rosa list account roles

## 1.1.2 (02 Sep, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.2
   * Fix broken links
   * Fix help and error messages
 * Other
   * Suppress spinner on non-terminal output
   * Force AWS PrivateLink for private STS clusters
   * Use default version on create account-roles
   * Add validation to user tags
   * Remove account roles prefix flag
   * Use AWS Tags to find pre-configured account roles
   * Add STS flag
   * Replace account role ARNs with account roles prefix
   * Ensure operator roles are unique
   * Return error when request fails
   * Allow empty URL and CA Path in interactive mode
   * Allow setting --output flag
   * Add check and prompt for required true addon parameters

## 1.1.1 (20 Aug, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Fix validation of spot max price
   * Fix prefix prompt text
 * Other
   * Updated error message
   * Move flag up a level
   * Do not redact install log output
   * Allow creating roles with permissions boundary
   * Add validator for host prefix
   * Ensure regexp validation allows empty values
   * Add validators for labels and taints
   * Add validator for CIDRs
   * Determine whether output is meant for terminal
   * Bump golang version to 1.15
   * Bump OCM SDK version to v0.1.199
   * Add jhernand to reviewer list
   * Use interactive validators
   * Use interactive validators
   * Use interactive validators
   * Use interactive validators
   * Use interactive validators
   * Provide real-time validators
   * Add customer managed key for rosa cluster
   * Exit once done watching logs
   * Update OWNERS file
   * Update to minimum required SCP
   * Remove optional policy checks
   * Track mode for AWS resource creation
   * Use standard config path for ocm.json
   * Remove etcd encryption from interactive mode
   * Add confirmation prompt with default of 'Y'
   * Validate operator roles prefix

## 1.1.0 (30 Jul, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.1.0
   * Fix spot instance decimal representation
   * Fix help text for mode flag
 * Other
   * Add permissions required for PrivateLink
   * Prompt user to create policies
   * Auto-find policies for roles
   * Ensure blocking pending clusters are non-STS
   * Ensure all role ARNs are required
   * Update trust policy
   * Hide spot instance flags
   * Add support for machine pool spot instances
   * Ensure that roles and policies can be upgraded
   * Error out when using invalid mode
   * Verify if OIDC Provider already exists
   * Add input validation for role names
   * Add user-friendly error
   * Validate operator roles exist
   * Update get addon parameters to use addon-inquiries request
   * Bump ocm-sdk v0.1.197
   * Output sample create cluster command
   * Automatically populate operator IAM roles
   * Update help text for etcd encryption
   * Add oidc-provider command
   * Report all insufficient quotas
   * Find cluster by external ID
   * Add operator-roles command
   * Reduce EBS quota checks
   * Reattempt login in case of sso outage
   * Update URLs for upcoming move to console.redhat.com
   * Output Role ARN once created
   * Provide a way to externally call command
   * Refactor AWS client code
   * Confirm delete operation
   * Update AWS SDK
   * Add tags to AWS resources
   * Replace --delete-stack flag with --delete
   * Add 'account' to init command
   * Add new account-roles resource
   * Send WARN output to STDERR
   * Ensure that JSON output for empty arrays looks correct
   * Update cmd/create/idp/cmd.go
   * Add directory with development scripts
   * Added hibernation and resume support to rosa cli
   * There is no "user" anymore
   * Add region tag for older versions
   * Add flag for JSON and YAML output
   * Filter clusters by AWS account ID
   * Bump SDK version
   * Bump SDK version
   * Add etcd-encryption flag to buildCommand
   * Move to interactive package
   * Move to separate package
   * Split resources into files
   * Refactor OCM client code
   * Move to ocm package
   * Do not expose internal API structure
   * Move all OCM API wrappers to ocm package

## 1.0.9 (15 Jun, 2021)

ENHANCEMENTS:
 * Other
   * Release v1.0.9
   * Add option to enable etcd encryption
   * Added quota validation for listing instance types
   * Ensure operator IAM roles
   * Remove interfacer linter
   * Add missing flags to re-create script
   * Ensure versions with STS support
   * Display STS configuration
   * Add Priya to reviewers list

## 1.0.8 (02 Jun, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.0.8
   * Fix tests with missing TagUser call
 * Other
   * Add support role ARN attribute
   * Ensure interactive mode for STS credentials without role ARN
   * Expose all flags
   * Normalize instace role parameters
   * Support STS users (#351)
   * Added new rosa list instance-types api
   * Increase golangci timeout to 5 minutes
   * Added wait for accountclaims to get ready
   * Disable IAM user checks for STS
   * Include note about quota limitations
   * Removed default region from CloudFormation stack check
   * Added SSO Validation

## 1.0.7 (20 May, 2021)

ENHANCEMENTS:
 * Other
   * Release v1.0.7
   * Display description during Pending state
   * Remove default region
   * Added custom IAM Roles
   * Updated the details link
   * Allow setting 0 replicas to autoscaling machine pool (Not default)

## 1.0.6 (12 May, 2021)

ENHANCEMENTS:
 * Other
   * Release v1.0.6
   * Output OIDC Endpoint URL if available
   * Add support for STS clusters
   * Correctly use the --disable-scp-checks parameter when supplied to init command
   * Hide references to PrivateLink
   * Enable PrivateLink on clusters

## 1.0.5 (16 Apr, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Use correct region instead of default
 * Other
   * Release v1.0.5

## 1.0.4 (07 Apr, 2021)

ENHANCEMENTS:
 * Other
   * Release v1.0.4
   * Log event when creating client with STS credentials

## 1.0.3 (06 Apr, 2021)

ENHANCEMENTS:
 * Other
   * Track ad-hoc authenticated events
   * Enable skip SCP check on init

## 1.0.2 (25 Mar, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.0.2
   * Fix command help example
   * Fix example command
 * Other
   * Redact KUBECONFIG line
   * Add helpful error message when using STS credentials
   * Filter out misleading output
   * Display legal terms URL
   * Display logs when cluster is in error state
   * Remove instance type
   * Error when editing non-editable parameters

## 1.0.1 (18 Mar, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.0.1
   * Fix example and help text
 * Other
   * Display availability
   * Display expected format in error
   * When setting CLI params skip unset values
   * Remove unnecessary interactive flag
   * Parse help flag when overriding flag parsing

## 1.0.0 (16 Mar, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v1.0.0
   * Fix description of cluster flags
 * Other
   * Add cluster network configuration
   * Disallow grant and revoke on cluster-admin
   * Align sort with OCM version list
   * Enforce interactive mode if required params are missing
   * Remove flag from global create
   * Remove unnecessary flags
   * Print link to get new token on expired session
   * Skip interactive mode if any flag is set
   * Format the success message
   * Error out on invalid min-replica
   * Skip autoscaling prompt when setting replicas
   * Send empty string when CIDR is nil
   * Check existence of addon installation before installing
   * Allow editing labels and taints
   * Remove count flag
   * Display default machine pool as Default
   * Report better errors for incompatible installation states
   * Use integer for numeric params
   * Disallow editing params of a non-ready addon
   * Disallow editing addons without parameters
   * Allow schema-less hosted domains on Google IDP
   * Add missing region flags
   * Align version list with cluster creation
   * Better message when deleting non-existent ingress
   * Allow installation parameters in CLI
   * Update URL for integration environment
   * Display upgrade state whenever showing existing upgrades
   * Display datetime format in error output
   * Accept numeric parameters as floats
   * Allow editing of addon parameters

## 0.1.10 (24 Feb, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v0.1.10
   * Fix text capitalization
 * Documentation
   * Update copyright year for man pages
   * Remove from repo and refer users to official docs
 * Other
   * Ensure that required inputs are same type as non-required
   * Verify compatibility of addons on cluster
   * Filter list of addons to those compatible with ROSA
   * Remove vendor dir from linter
   * Use quota_cost to determine compatibility
   * Rebuild docs on list cluster command
   * Add region flag to list cmd
   * Display total worker nodes across all machine pools
   * Validation message should show parameter name
   * Move region and profile flags

## 0.1.9 (18 Feb, 2021)

ENHANCEMENTS:
 * Other
   * Release v0.1.9
   * Use vendor directory

## 0.1.8 (17 Feb, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v0.1.8
   * Fix list of recommendations
   * Fix Makefile build command
   * Fix empty flavour when validating cluster creation
   * Fix programmatically-run commands
 * Other
   * Validate node drain grace period
   * Use Run instead of PreRun
   * Remove asset build dependency

## 0.1.7 (16 Feb, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v0.1.7
   * Fix go-bindata command and downgrade go version
   * Fix parameter defaults in interactive prompt
   * Update version
   * Fix doc typos
   * Fix default taints in interactive mode
   * Fix example
 * Other
   * Use correct privacy flag on describe
   * Allow the creation of fake clusters
   * Add hidden flag to set cluster flavour
   * Cleaning up some leftover obsolete code from autoscaling PR
   * Clean up argument and flag requirements
   * Show success message on write operations
   * Validate mapping method input
   * Only force interactive mode when necessary
   * Avoid calling API after failed validation
   * Remove suggestion to run init
   * Output command to rerun cluster creation
   * Trim user-provided machine-friendly names
   * Specify UTC for schedule time
   * Display explicit values in grace period help
   * Ensure interactive mode for schedule
   * Align command with auto-generated docs

## 0.1.6 (20 Jan, 2021)

ENHANCEMENTS:
 * Other
   * Release v0.1.6
   * Use explicit login flag checks
   * Add multi-az status to describe
   * Sort machine types by CPU cores
   * Remove explicit enable

## 0.1.5 (15 Jan, 2021)

ENHANCEMENTS:
 * Other
   * Release v0.1.5
   * Add openshift version to describe output
   * Support add-on installation parameters
   * Support addon uninstallation form cluster
   * Require min/max replicas on interactive mode iff existing machinepool autoscaling is disabled

## 0.1.4 (06 Jan, 2021)

ENHANCEMENTS:
 * Bug fixes
   * Release v0.1.4
   * Fix formatting and add generated docs
   * Fix local validation for worker nodes and machinepool replicas
 * Other
   * Use install command instead of create
   * Enable all commands
   * Allow listing of all available addons
   * Add autoscaling support
   * Disable `maligned` linter
   * Add link to retrieve tokens
   * Display scheduled upgrades
   * Show warnings when user makes cluster private
   * Hide 'env' parameter
   * Validate that compute nodes are multiple of 3
   * Set default version
   * Rename repository from moactl to rosa
   * Determine if user exists before revoking
   * Advise user to run init for failed credentials
   * Advise user to run init for failed credentials
   * Update OWNERS
   * Adding Orange team members to OWNERS file

## 0.1.3 (04 Dec, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.1.3
   * Allow scheduling, listing, canceling cluster upgrades
   * Add taints to machinepool commands
   * Remove paid AMI flag and finalize ROSA transition
   * [rosa create cluster] Verify provided subnets for Existing VPC exist in AWS
   * Dont ignore subnets from command line args if provided
   * Ask user before showing subnets

## 0.1.2 (24 Nov, 2020)

ENHANCEMENTS:
 * Bug fixes
   * Release v0.1.2
   * Fix interactive mode
 * Other
   * [rosa create cluster] Return more clear error message when no versions are found.
   * Add support for existing VPC
   * Enabling Interactive mode if no arguments specified
   * Update implementation to include the default values in the interactive mode only
   * Added Confirmation option for default network parameters
   * Always use interactive mode on unset required flags
   * Remove API ingress when listing ingress

## 0.1.1 (05 Nov, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.1.1
   * Rotate osdCcsAdmin credentails on creation of each cluster (#118)
   * Allow managing 'default' machinepool
   * Added Details Page Link
   * Added validation for name
   * Support full CRUD operations for machine pools
 * Init
   * Verify permissions for osdccsadmin using ValidateSCP

## 0.1.0 (30 Oct, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.1.0
   * Update flow to use grant and revoke
   * Remove shard info from describe cluster
   * Red Hat OpenShift Service on AWS
   * Enable interactive mode
   * Rename IDP to Cluster-Admin

## 0.0.16 (22 Oct, 2020)

ENHANCEMENTS:
 * Bug fixes
   * Release v0.0.16
   * Fix idp name generation
 * Other
   * Default to free AMI
   * Fallback to interactive mode
   * Ensure CCS is enabled before asking to disable SCP checks
   * Do not show cluster-admin user
   * Always show help text for claims
   * Display Provision Shard if available
   * Adressing code review items
   * Addressing vkareh review
   * Advise user to store password securely
   * Check region after profile credentials have been validated
   * Fail name check before calling API
   * Fall back to full error message
   * Added Display Name and Domain name to describe
   * Add Check Admin User function, and add tests to verify
   * Add tags to template, not working

## 0.0.15 (15 Oct, 2020)

ENHANCEMENTS:
 * Other
   * Release 0.0.15
   * Init test cluster name to less than 15 char
   * Add flag to disable SCP checks
   * Default to using paid AMI
   * Bring the default number of nodes down
   * Add admin resource to login to cluster
   * Allow using AWS_PROFILE env var
   * Keep error opIds and codes behind debug flag
   * Add extra scopes to OpenID IdP
   * Allow insecure connections on LDAP IdP
   * Provide help link for mapping method
   * Make hosted_domain required unless mapping method is lookup
   * Show example command for install logs
   * Provide guidance on using GitHub organizations when creating IdP
   * Ensure osdCcsAdmin exists before attempting cluster dry-run
   * Add CONTRIBUTING.md file, with some details about CI

## 0.0.14 (08 Oct, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.0.14
   * Move main.go to moactl directory, add make install target
   * Only download go-bindata when not available
   * Simulate cluster creation
   * Add --dry-run flag
   * Add support for GitLab
   * Updated OCM SDK version
   * Added New Error Message Implementation
   * Add support for certificate bundles

## 0.0.13 (30 Sep, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.0.13
   * Split configuration to ensure early failure
   * Remove validations from create command
   * Adding validations to cluster create command
   * Improve warnings when cluster is pending
   * Add support for AWS profiles
   * Warn the user that it will take about 1 minute to add IdP
   * Fixed lint and reverted wrong line
   * Review Comments
   * Add Provision Type and Reason for error cluster

## 0.0.12 (24 Sep, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.0.12
   * Check for only 100 vCPU
   * Added Detailed Granular Status to match with ocm UI
   * Update based on output of newer commands

## 0.0.11 (22 Sep, 2020)

ENHANCEMENTS:
 * Other
   * Use OCM SDK to get token expiration
   * Expose channel-group
   * Correct typos and incorrect commands in README
   * Allow selective override of the paid AMI
   * Use OpenShift versions that have MOA marketplace images
   * Avoid nil pointer dereferencing
   * Only warn when oc client missing

## 0.0.10 (14 Sep, 2020)

ENHANCEMENTS:
 * Bug fixes
   * Release v0.0.10
 * Other
   * Allow querying for channel-groups
   * Added Timestamp to created date
   * Ensure token is required
   * Use default region for CloudFormation stack
   * Add 'Channel Group' attribute to 'moactl describe cluster'
   * Do not check ViewBilling policy
   * Add progress indicator when waiting for logs
   * Add command to list enabled versions
   * Do not use colors on Windows
 * Create
   * Add credential check for osdCcsAdmin when cluster starts to be created

## 0.0.9 (27 Aug, 2020)

ENHANCEMENTS:
 * Other
   * Release 0.0.9
   * Update Makefile for CI and remove PR check script
 * Verify-permissions
   * Revert code refactors

## 0.0.8 (27 Aug, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.0.8
   * Ensure no output on error

## 0.0.7 (26 Aug, 2020)

ENHANCEMENTS:
 * Other
   * Release 0.0.7
   * Do not error out on invalid version
   * Change how SDK logs are propagated
   * Ensure region is set when creating AWS client
   * Add command to list available regions
   * List regions using user AWS creds

## 0.0.6 (13 Aug, 2020)

ENHANCEMENTS:
 * Bug fixes
   * Release v0.0.6
   * Fix help text
   * Fix confirmation output
   * Fix function call from broken dependency
   * Fix long line
 * Documentation
   * Add list and describe commands for add-ons
 * Other
   * Added Detailed Error Message for Throttling
   * Confirm add-on installation
   * Hide addons until it's feature-complete
   * Detach logs once operation is complete
   * Describe cluster automatically after creation
   * Allow user to watch cluster uninstallation logs
   * Allow user to watch cluster installation logs
   * Add command to download openshift-client tools
   * Add command to verify OpenShift client tools
   * Remove global list of add-ons
   * Update API endpoints
   * Add uninstall logs
   * Add separate install/uninstall logs
   * Update SDK client
   * Make command more intuitive
   * Change from ginkgo to to go test
   * Show AWS account ID used to create cluster
   * Deprecate --name in favor of --cluster-name
   * Add tests for EnsureOsdCcsAdminUser
   * Check cloudformation stack exists
   * Improve moactl verify quota error messages
   * Return error if using root account
   * Direct the user to check add-on status after install
   * Allow user to specify IdP name
   * Set compute node defaults based on AZ

## 0.0.5 (21 Jul, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.0.5
   * Validate only permissions in the OSD SCP policy document
   * Validate permissions in the AWS client region
   * Don't validate AWS Organization List Policies
   * Updates from second moa hackday

## 0.0.4 (20 Jul, 2020)

ENHANCEMENTS:
 * Other
   * Release v0.0.4
   * Update adding IDP section
   * Link to aws scp doc
   * GitHub IdP: Change label name for Hostname
   * Display optional marker for non-required fields
   * Confirm operation
   * Adding a tldr section to quickstart
   * Fallback to interactive mode
   * Compare quota correctly to display available add-ons
   * Updates to the quickstart
   * Add moactl logs example
   * Add sentence describing whoami
   * Add whoami

## 0.0.3 (06 Jul, 2020)

ENHANCEMENTS:
 * Bug fixes
   * Fix linter errors
   * Fix small linter issues
 * Other
   * Release v0.0.3
   * Add support for Windows binary build
   * Adding quickstart
   * Add list and create commands for add-ons
   * Make client-id a non-password field
   * Add interactive mode to OpenID
   * Add interactive mode to LDAP
   * Add interactive flag to create idp
   * Add interactive mode to edit cluster
   * Add interactive flag to create
   * AWS Region: Allow passing --region to verify and init commands
   * Custom cluster properties.
   * Add golangci version for CI
   * Use Create[Reporter|Logger]OrExit
   * Switch to use CreateLoggerOrExit
   * Define CreateLoggerOrExit

## 0.0.2 (05 Jun, 2020)

ENHANCEMENTS:
 * Bug fixes
   * Fix unnecessary conversion
   * Add golangci-lint configuration and fix all lint warnings
   * Fix command help after creating IdP
 * Other
   * Release v0.0.2
   * Update token URL
   * Expand error messages
   * Track version of moactl used for cluster creation
   * Prefix output with source API
   * Allow the use of --cluster as identifier
   * Switch to use CreateReporterOrExit
   * Define CreateReporterOrExit
   * Add command to display account information
   * Fallback to JWT for account data
   * Get arbitrary token data
   * Allow the use of --cluster as identifier
   * Check cluster_admin_enabled before listing cluster-admins
   * Limit API retires and set minimum throttle delay between reties

## 0.0.1 (28 May, 2020)

ENHANCEMENTS:
 * Bug fixes
   * Fix example command order
   * Fix example command order
   * Fix CF capabilities
   * Fix oc version check
   * Add cluster-admins to delete command
   * Fix error message typo
 * Documentation
   * Add Examples to commands
   * Generate documentation for moactl
   * Clean up auto-generated assets
 * Other
   * Release v0.0.1
   * Allow using sub-domain identifier
   * Add command to update cluser
   * Add permissions command to verify
   * Add command to update API and ingress endpoints
   * Add verify quota command
   * Move cluster creation to cluster package.
   * Add a cluster provider
   * Update all imports to the new repo
   * Add command to output cluster logs
   * Add flag to delete CloudFormation stack
   * Only login when necessary
   * Verify whether oc is installed
   * Ensure CF IAM user logical name matches constructed name
   * Add IAM policy simulator
   * Compile templates at build time
   * Remove go-bindata metadata
   * Get IAM access key and secret from CF output
   * Create MOA cluster using new Product API
   * Enable support for cluster-admins group
   * Format and Lint code
   * Hide token entry from stdin
   * Display cluster state on list and describe
   * Add Debugf function
   * Print better info message when cluster is created
   * Re-structure command hierarchy
   * Add aliases
   * Add bash completion command
   * Change how cluster is determined
   * Add man pages and rst to gitignore
   * Add all supported cluster creation flags
   * Clean up error messages
   * Remove storage and load balancer quota flags
   * Flip example argument order to match updated ordering
   * Add ingress resource to cluster
   * Move every action log to Debugf level
   * Move to top level directory
   * Initial commit
   * Add login and logout commands
   * Move OCM connection creation to separate package
   * Make `tags` and `properties` subpackages
   * Initial _BYOC_ implementation
   * Add describe command
   * Add init command to prepare AWS account
   * Improve logging
   * Initial empty commit
   * Move cluster loading to a helper function
   * Refactor create command to use internal AWS client
   * Use builder pattern for AWS client
   * Move user creation to init phase
   * Login to Red Hat during init phase
   * Move idp to be top-level resource
   * Move key validation to helper
   * Use internal AWS client for all cluster commands
   * Add env flag to determine OCM environment
   * Add support for GitHub Identity Provider
   * Only call login if necessary
   * Split provider logic into separate files
   * Define order of precedence of token
   * Create IAM user osdCcsAdmin with Cloudformation

