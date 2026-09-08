# Package Classification: `pkg/`

This document classifies every package under `pkg/` to guide the separation of reusable core/library code from CLI-specific implementation.

The test applied to each package: **would this logic be equally useful in a TUI, a headless program, a CLI, or a REST API?** If the answer is "no," it is not core API.

This is a **north star** document. Each package is placed where it _should_ live once its CLI concerns are extracted. Packages that are not yet clean are marked **Needs split** with details in the [Split Work Required](#split-work-required) section at the end.

## Categories

| Label | Meaning | Target Location |
|-------|---------|-----------------|
| **Public Core API** | Domain logic, service operations, validation, types, and constants that any consumer needs regardless of presentation. Zero CLI, terminal, or formatting dependencies. | `pkg/` (stable, exported) |
| **Private Core Implementation** | Internal plumbing that supports the core API: SDK client wrappers, caching, config loading, logging adapters, HTTP helpers. Useful across consumers but not a stable public contract. | `internal/core/` |
| **CLI Presentation / Interaction** | Packages that exist because there is a terminal: Cobra/pflag wiring, interactive prompts, colored output, reporter, output formatting, command registration, manual-mode command-string generation. | `cmd/` (commands) or `internal/cli/` (shared CLI code) |
| **Test-Only Support** | Test helpers, mocks, matchers. Not shipped. | `internal/testing/` or colocated |

---

## Public Core API

Target location: `pkg/` (stable, exported). Each entry shows **target name** with current name in parentheses when they differ.

| Target | Status | Description |
|--------|--------|-------------|
| `pkg/aws/tags` | Clean | Tag key constants (`rosa_cluster_name`, `red-hat-managed`, etc.) and EC2 tag-matching utilities. |
| `pkg/breakglasscredential` | Needs split | Break-glass credential creation via OCM and expiration parsing. |
| `pkg/clusterautoscaler` | Needs split | Autoscaler OCM config building and validation rules (min/max ranges, duration formats). |
| `pkg/clusterregistryconfig` | Needs split | Registry config spec building for hosted clusters and validation rules. |
| `pkg/constants` | Needs split | Environment variable names (`ROSA_TOKEN`, `AWS_PROFILE`, `OCM_CONFIG`, etc.). OIDC flag-name and help-message constants in `oidc_constants.go` are CLI-specific and should be separated. |
| `pkg/externalauthprovider` | Needs split | External auth provider creation via OCM, URL validation, and claim mapping logic. |
| `pkg/fedramp` | Needs split | FedRAMP/GovCloud environment configuration: URL and token endpoint mappings per environment, and GovCloud account-field validation. |
| `pkg/utils` (`pkg/helper`) | Needs split | General-purpose utilities: random labels, string/slice operations, shell quoting, file saving, `IsBYOVPC` check. |
| `pkg/utils/urlutil` (`pkg/helper/url`) | Clean | URL validation utilities: credential character checks, IPv6 literal host detection. |
| `pkg/utils/fileutil` (`pkg/input`) | Needs split | YAML/JSON file unmarshalling and input helpers. |
| `pkg/utils/versions` (`pkg/helper/versions`) | Needs split | Version list retrieval from OCM, comparison, filtering, and segment normalization (`FormatMajorMinorPatch`). Returns version strings and errors. Used by multiple `cmd/` and `pkg/` packages. |
| `pkg/autonode` (`pkg/helper/autonode`) | Needs split | AutoNode IAM role configuration and validation logic. |
| `pkg/download` (`pkg/helper/download`) | Needs split | HTTP file download with temp-file atomic rename and structured error wrapping (`formatDownloadError`). Returns errors, not display strings. Terminal progress rendering (`WriteCounter`, `PrintProgress`) moves to `internal/cli/`. |
| `pkg/machinepool` (merge `helper/features`) | — | Version-gated feature support checks. Only used by `pkg/machinepool`; merge as unexported functions. |
| `pkg/machinepool` (merge `helper/machinepools`) | Needs split | Label, taint, and tag parsing and validation (`ParseLabels`, `ParseTaints`, `ValidateNodeDrainGracePeriod`, etc.). Reusable logic merges into `pkg/machinepool`; CLI orchestration (`GetTaints`, `GetAwsTags`, `GetLabelMap` with `*cobra.Command`, `interactive.GetString`, `r.Reporter.Errorf`, `os.Exit`) moves to `internal/cli/`. |
| `pkg/operatorroles` (split from `pkg/helper/roles`) | Needs split | Reusable operator role logic: role name generation, ARN validation, managed policy validation, Red Hat managed tag checks. Extracted from `pkg/helper/roles`; CLI orchestration stays in `internal/cli/roles`. |
| `pkg/rolepolicybinding` (`pkg/helper/rolepolicybindings`) | Needs split | Role-policy binding status checking, missing-binding detection, and `TransformToRolePolicyDetails`. |
| `pkg/iamserviceaccount` | Clean | Types, validation, trust policy generation, tag construction, and role name generation for IAM service accounts. |
| `pkg/ingress` | Needs split | Ingress describe/list service with OCM API calls. |
| `pkg/kubeletconfig` | Needs split | KubeletConfig pod pids limit validation and OCM CRUD operations. |
| `pkg/logforwarding` | Needs split | Log forwarding config types, YAML unmarshalling, and OCM builder binding. |
| `pkg/machinepool` | Needs split | Machine pool and node pool CRUD via OCM, replica/autoscaling validation, spot instance logic, root disk sizing, version compatibility checks. Most entangled package. |
| `pkg/properties` | Clean | OCM cluster property key constants (`rosa_cli_version`, `fake_cluster`, etc.). |
| `pkg/version` | Needs split | Version retrieval from mirror.openshift.com, comparison logic, cache-backed version list. Reusable API for TUIs, automation, and other non-CLI consumers. Cobra-accepting functions (`ShouldRunCheck`) and `output.HasFlag()` checks move to `internal/cli/`. |
| `pkg/errors` | Clean | Shared `ValidationError` type (`Field`, `Message`, `Err`) distinguishing invalid input from operational failures across `Request.Validate()`-based workflows. No dependencies of its own; imported under the `rosaerrors` alias since it shares its name with the standard library `errors` package. |

## Private Core Implementation

Target location: `internal/core/`. Each entry shows **target name** with current name in parentheses when they differ.

| Target | Status | Description |
|--------|--------|-------------|
| `internal/core/aws` (`pkg/aws`) | Needs split | AWS SDK v2 client wrapping IAM, STS, EC2, S3, CloudFormation, Organizations, SecretsManager, ServiceQuotas. |
| `internal/core/aws/api_interface` (`pkg/aws/api_interface`) | Clean | Go interfaces mirroring AWS SDK v2 service clients. Abstraction for mocking and dependency injection. |
| `internal/core/buildinfo` (`pkg/info`) | Clean | Build metadata: `DefaultVersion`, build hash, `DefaultUserAgent`. Used by `aws` and `ocm` clients for user-agent strings. |
| `internal/core/httputil` (`pkg/clients`) | Clean | Minimal HTTP client interface wrapping `net/http`. Testing seam used by `pkg/version`. |
| `internal/core/ocm` (`pkg/ocm`) | Needs split | OCM API client: clusters, node pools, machine pools, IDPs, ingresses, upgrades, addons, billing, OIDC, kubelet configs, versions, gates. |
| `internal/core/ocmconfig` (`pkg/config`) | Clean | OCM client configuration: load/save from file or keyring, JWT token parsing, connection builder. |
| `internal/core/policy` (`pkg/policy`) | Needs split | IAM policy attach/detach operations and quota validation. Manual-mode command string generation (`ManualAttachArbitraryPolicy`, `ManualDetachArbitraryPolicy`) moves to `internal/cli/`. |
| `internal/core/rosalog` (`pkg/logging`) | Clean | Logger factories: logrus setup, AWS SDK logger adapter, OCM SDK logger adapter, HTTP debug round-tripper. |
| `internal/core/sharedvpcroles` (`pkg/roles`) | Needs split | Shared VPC role policy helpers and input validation. |
| `internal/core/versioncache` (`pkg/cache`) | Clean | File-backed gob cache with TTL expiration. Version list caching. |
| `internal/core/vpcnetwork` (`pkg/network`) | Clean | CloudFormation stack creation and polling for VPC network setup. |

## CLI Presentation / Interaction

Target location: `cmd/` (commands) or `internal/cli/` (shared CLI code). Each entry shows **target name** with current name in parentheses when they differ.

| Target | Description |
|--------|-------------|
| `internal/cli/arguments` (`pkg/arguments`) | CLI flag parsing, unknown-flag handling, region deprecation warnings, pflag normalization. |
| `internal/cli/color` (`pkg/color`) | Manages the `--color` flag and terminal color detection. |
| `internal/cli/commandbuilder` (`pkg/aws/commandbuilder`) | Builds human-readable AWS CLI command strings (e.g., `aws iam create-role ...`) for "manual mode" output. |
| `internal/cli/commandbuilder/roles` (`pkg/aws/commandbuilder/helper/roles`) | Generates manual-mode AWS CLI commands for operator roles. |
| `internal/cli/commands` (`pkg/commands`) | Single-function command registry wiring all subcommands onto root `cobra.Command`. |
| `internal/cli/rolebridge` (`pkg/aws/rolebridge`) | Adapts `aws.Client`'s reporter.Logger-based `EnsureRole`/`AttachRolePolicy` (and context-free `PutRolePolicy`) to `context.Context`-based signatures, so workflow packages can declare `context.Context`-based client interfaces without depending on `pkg/reporter`. Temporary: delete once those `aws.Client` methods stop taking a `reporter.Logger`. |
| `internal/cli/debug` (`pkg/debug`) | Adds the `--debug` pflag and tracks its boolean state. |
| `internal/cli/interactive` (`pkg/interactive`) | `survey`-based `GetString`, `GetBool`, `GetInt`, `GetOption`, `GetPassword`, validation combinators, `--interactive` pflag. |
| `internal/cli/interactive/confirm` (`pkg/interactive/confirm`) | `--yes` flag and confirmation prompts using `survey`. |
| `internal/cli/interactive/consts` (`pkg/interactive/consts`) | Single constant (`SkipSelectionOption`) used exclusively by the interactive layer. |
| `internal/cli/interactive/logforwarding` (`pkg/interactive/logforwarding`) | Interactive prompts for CloudWatch/S3 log forwarding config. |
| `internal/cli/interactive/oidc` (`pkg/interactive/oidc`) | Interactive OIDC config ID selection with terminal prompts. |
| `internal/cli/interactive/roles` (`pkg/interactive/roles`) | Interactive installer role ARN selection with AWS lookups, spinner, and `os.Exit`. |
| `internal/cli/interactive/securitygroups` (`pkg/interactive/securitygroups`) | Interactive security group ID selection with `os.Exit`. |
| `internal/cli/ocmoutput` (`pkg/ocm/output`) | Output formatting for node pools and machine pools: autoscaling display, replica counts, labels, tags. Builds fixed-width display strings. |
| `internal/cli/output` (`pkg/output`) | Output formatting subsystem: `--output` flag (json/yaml), `Print`/`PrintWarn`/`PrintError` to stdout/stderr, OCM resource marshalling, `StructuredReporter`. Absorbs `pkg/object` as `output.Data` (`map[string]any`); delete standalone `pkg/object/` package. |
| `internal/cli/profile` (`pkg/aws/profile`) | Adds the `--profile` pflag. |
| `internal/cli/region` (`pkg/aws/region`) | Adds the `--region` pflag. |
| `internal/cli/reporter` (`pkg/reporter`) | `Logger` interface and terminal-aware reporter printing colored messages to stdout/stderr. |
| `internal/cli/roles` (`pkg/helper/roles`) | Needs split. CLI workflow orchestration: auto/manual mode branching, `confirm.Prompt`, `r.Reporter.*`, `fmt.Println`, `os.Exit`. Reusable logic (role name generation, ARN validation, managed policy validation, tag checks) moves to `pkg/operatorroles`. |
| `internal/cli/runtime` (`pkg/rosa`) | Central `Runtime` struct bundling Reporter, Logger, OCMClient, AWSClient, plus `DefaultRunner`/`RuntimeVisitor` Cobra wrappers. The CLI lifecycle harness. |
| `cmd/` (`pkg/options/iamserviceaccount`) | Builds full `cobra.Command` structs and wires flags for IAM service account create/delete/describe. |
| `cmd/` (`pkg/options/machinepool`) | Builds `cobra.Command` and wires flags for machine pool creation. |
| `cmd/` (`pkg/options/network`) | Builds `cobra.Command` and instantiates `reporter.CreateReporter()`. |

## Test-Only Support

Target location: `internal/testing/` or colocated with the package under test.

| Target | Description |
|--------|-------------|
| colocated (`pkg/aws/api_interface/api_interface_tests`) | Test companions for the api_interface package. |
| colocated (`pkg/aws/mocks`) | Generated gomock implementations of AWS api_interface types. |
| `internal/testing/` (`pkg/test`) | Test helpers: output capture, OCM mock server setup, mock runtime builders, command structure verification, flag arg generation/verification. |
| `internal/testing/matchers` (`pkg/test/matchers`) | Custom Gomega matcher (`MatchExpected`). |

---

## Split Work Required

Every package marked **Needs split** above has CLI concerns to extract. See
[Boundary Rules](../ARCHITECTURE.md#boundary-rules) for the rationale behind each
rule; this section lists only what must move.

### Public Core API → extract to CLI layer

| Target | Extract to CLI Layer |
|--------|----------------------|
| `pkg/breakglasscredential` | Cobra flag definitions, `interactive.GetString`, pflag `Changed()` checks, formatted output strings. |
| `pkg/clusterautoscaler` | Cobra/pflag flag definitions, ~25 interactive prompt sites, `output.PrintBool`/`output.PrintStringSlice`, `PrintAutoscaler` display builder. |
| `pkg/clusterregistryconfig` | Cobra/pflag flag definitions, ~8 interactive prompt sites, `interactive.Enabled()` checks. |
| `pkg/constants` | OIDC flag-name and help-message constants (`InstallerRoleArnFlag`, `InformOperatorRolesOutput`, etc.). Env var names stay. |
| `pkg/externalauthprovider` | Cobra/pflag flag definitions, ~10 interactive prompt sites, `interactive.Enabled()` checks. |
| `pkg/fedramp` | `--govcloud`/`--admin` pflag definitions, `Enabled()` global reader, `HasFlag`/`HasAdminFlag` accepting `*cobra.Command`. URL/token endpoint data stays. |
| `pkg/utils` (`pkg/helper`) | `DisplaySpinnerWithDelay` (`reporter.IsTerminal()`, `reporter.Infof()`). All other helpers are clean. |
| `pkg/utils/versions` (`pkg/helper/versions`) | `rosa.Runtime` dependency. Accept OCM client and logger as parameters. |
| `pkg/autonode` (`pkg/helper/autonode`) | `*cobra.Command` parameter, `interactive.GetBool`/`GetString`, `fedramp.Enabled()`. |
| `pkg/download` (`pkg/helper/download`) | `WriteCounter.PrintProgress()` (`fmt.Printf`), `Download()` `fmt.Print("\n")`. Accept progress callback or `io.Writer`. |
| `pkg/machinepool` (merge `helper/machinepools`) | `GetTaints`, `GetAwsTags`, `GetLabelMap`: `*cobra.Command` parameter, `interactive.Enabled()`, `interactive.GetString`, `r.Reporter.Errorf` + `os.Exit(1)` (6 sites). These three functions move to `internal/cli/`; pure parsing and validation functions merge into `pkg/machinepool`. |
| `pkg/machinepool` | ~60 `interactive.*` sites, `confirm.*`, `output.*`, `r.Reporter.*`, `os.Exit(1)` (3 sites), `tabwriter`/`fmt.Print`, display strings. Most entangled package. |
| `pkg/rolepolicybinding` (`pkg/helper/rolepolicybindings`) | Display strings containing `rosa attach policy` CLI commands. Return structured data instead. |
| `pkg/ingress` | `output.HasFlag()`, `output.Print()`, `fmt.Print()`, `*pflag.FlagSet` parameter, display strings. |
| `pkg/utils/fileutil` (`pkg/input`) | `CheckIfHypershiftClusterOrExit`: `os.Exit` via `exitFunc`, `r.Reporter.Errorf`. Return error instead. |
| `pkg/kubeletconfig` | `interactive.GetString`/`GetInt`, `confirm.ConfirmRaw`, `interactive.Enable()` mutation, `r.Reporter.Infof`, display strings. |
| `pkg/logforwarding` | `LogForwarderObjectAsString` display builder, `FlagName`/`LogFwdConfigHelpMessage` CLI constants, `ConstructPodGroupsInteractiveOptions`. |
| `pkg/version` | `ShouldRunCheck` accepting `*cobra.Command`, `output.HasFlag()`. Version retrieval and comparison stay in `pkg/`. |

### Private Core Implementation → extract to CLI layer

| Target | Extract to CLI Layer |
|--------|----------------------|
| `internal/core/aws` (`pkg/aws`) | `os.Exit` (7 sites via `CreateNewClientOrExit` and `helpers.go`), `fedramp.Enabled()` (3 sites), `fmt.Printf` error output (4 sites). `EnsureRole`/`AttachRolePolicy` take a `reporter.Logger` parameter and call it directly; move to returning structured progress data and let the CLI layer report it, once callers no longer depend on the current signature (`pkg/aws/rolebridge` bridges this in the meantime). |
| `internal/core/ocm` (`pkg/ocm`) | Cobra `--cluster` flag with shell completion, `reporter.IsTerminal()`, `output.HasFlag()`, `os.Exit(1)` (2 sites), `CreateNewClientOrExit`. |
| `internal/core/policy` (`pkg/policy`) | `reporter.Logger` parameter on `AutoAttachArbitraryPolicy`. `ManualAttachArbitraryPolicy` and `ManualDetachArbitraryPolicy` generate `aws iam ...` command strings for terminal display; move to `internal/cli/`. `AutoDetachArbitraryPolicy` returns human-readable status messages; return structured results instead. |
| `internal/core/sharedvpcroles` (`pkg/roles`) | `rosa.Runtime` and `arguments` (CLI flag package) dependencies. |

### CLI Presentation → already CLI, needs cleanup

| Target | Cleanup Required |
|--------|-----------------|
| `internal/cli/roles` (`pkg/helper/roles`) | Split the package. Move reusable logic to `pkg/operatorroles`: `GeOperatorRolePrefixFromClusterName`, `GetOperatorRoleName`, `CheckHasRedHatManagedTag`, `ValidateUnmanagedAccountRoles`, `ValidateAdditionalAllowedPrincipals`, and the managed policy validation functions (decoupled from `r.Reporter`). Keep CLI orchestration in `internal/cli/roles`: auto/manual mode branching (`createOperatorRole`), `confirm.Prompt` (`upgradeMissingOperatorRole`), `BuildMissingOperatorRoleCommand`, `r.Reporter.*`, `fmt.Println`, `os.Exit(1)` (2 sites). |

For boundary rationale, see [Boundary Rules](../ARCHITECTURE.md#boundary-rules).
