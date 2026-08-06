# ROSA CLI Architecture

## High-Level Shape (current state)

Today the codebase is organized as follows. The [Target Architecture](#target-architecture)
section below describes where each concern is moving; output, prompting, and
reporter code currently in `pkg/` is being migrated to `internal/cli/`.

- `cmd/rosa/` owns Cobra command registration, flag wiring, and help text.
- `pkg/` owns business logic, service integrations, and helpers. It also
  currently contains output, prompting, and reporter code that will move to
  `internal/cli/` as the migration progresses.
- `pkg/aws/` and `pkg/ocm/` are the main external-system boundaries; their interfaces and mocks are reused across commands and tests.
- `pkg/output/`, `pkg/reporter/`, and `pkg/interactive/` currently define most
  user-facing output, error, and prompt behavior. In the target architecture
  these are CLI-layer concerns under `internal/cli/`.
- `cmd/docs/` and `make generate-docs` cover generated CLI docs.
- `cmd/rosa/structure_test/` guards the command tree and supported flag contracts.

## Command Layering

### Target Architecture

The repository is moving toward a two-layer architecture that cleanly separates CLI presentation from reusable core logic. The goal is to allow non-CLI consumers (automation tools, TUIs, REST APIs, or other first- and third-party applications) to import and use ROSA's core operations without pulling in Cobra, terminal prompting, or formatted output.

The strict test for where code belongs: **would this logic be equally useful in a TUI, a headless program, a CLI, or a REST API?** If the answer is no, it belongs in the CLI layer.

```
┌──────────────────────────────────────────────────────────────┐
│                       CLI Layer                              │
│  Owns: Cobra wiring, flag parsing, interactive prompting,    │
│        output rendering, confirmation dialogs, spinners,     │
│        workflow orchestration, os.Exit, manual-mode command  │
│        string generation                                     │
│                                                              │
│  ┌────────────┐  ┌──────────────┐                            │
│  │   cmd/     │  │ internal/cli │                            │
│  │            │  │              │                            │
│  │ Thin Cobra │  │ Shared CLI   │                            │
│  │ commands   │  │ helpers:     │                            │
│  │            │  │ - flags      │                            │
│  │ - wiring   │  │ - prompting  │                            │
│  │ - Run func │  │ - rendering  │                            │
│  │ - os.Exit  │  │ - reporters  │                            │
│  │            │  │ - runtime    │                            │
│  └─────┬──────┘  └──────┬───────┘                            │
│        │                │                                    │
│        └───── calls ────┘                                    │
└──────────────────┬───────────────────────────────────────────┘
                   │
                   │  Accepts resolved values and returns
                   │  data structures + errors. Never prompts,
                   │  never formats, never exits.
                   │
┌──────────────────▼───────────────────────────────────────────┐
│                    Core / Library Layer                      │
│  Owns: Business logic, domain types, validation, service     │
│        operations, SDK integrations, constants               │
│                                                              │
│  ┌────────────┐  ┌──────────────┐                            │
│  │   pkg/     │  │internal/core │                            │
│  │            │  │              │                            │
│  │ Public API │  │ Private impl │                            │
│  │            │  │              │                            │
│  │ - domain   │  │ - AWS client │                            │
│  │   types    │  │ - OCM client │                            │
│  │ - service  │  │ - config     │                            │
│  │   ops      │  │ - logging    │                            │
│  │ - valid-   │  │ - caching    │                            │
│  │   ation    │  │ - version    │                            │
│  │ - consts   │  │   checks     │                            │
│  └────────────┘  └──────────────┘                            │
└──────────────────────────────────────────────────────────────┘
```

### Directory Responsibilities

Each directory mirrors the layer it belongs to:

#### `cmd/` : CLI Layer (public commands)

Thin Cobra command files. Each command file should:

- Define the `cobra.Command` struct (use, short, long, example, args).
- Wire flags onto the command.
- Provide a `Run` function that resolves flag/interactive values into plain Go
  types, then delegates to a core-layer service function.
- Handle `os.Exit`, reporter output, and error presentation.

A command file should **not** contain business logic, validation rules, SDK calls,
or data transformation. If a `Run` function is growing beyond flag resolution and
delegation, logic is leaking up.

##### Sibling Command Imports

A `cmd/` package may import another `cmd/` package **only for command
registration** (wiring subcommands onto a parent). Beyond registration:

1. A `cmd/` package must not call another command package's `Cmd.Run()`,
   mutate another command's flags via `Cmd.Flags().Set()`, or read another
   command's package-global state.
2. When two commands need the same operation, extract it into `pkg/` or
   `internal/core/`. Both commands call the core function with resolved values.
3. Shared constants and helper functions currently exported from `cmd/`
   packages (e.g., `ClusterAdminUsername`, `FindIDPWithAdmin`) should migrate
   to `pkg/` so they are available without `cmd/`-to-`cmd/` imports.

#### `internal/cli/` : CLI Layer (shared CLI infrastructure)

Shared CLI-specific code that multiple `cmd/` files need but that no library consumer
would ever import. This includes:

- **Flag plumbing**: `--region`, `--profile`, `--debug`, `--color`, `--output`,
  `--interactive`, `--yes`, `--govcloud`, `--cluster` flag definitions.
- **Interactive prompting**: `survey`-based `GetString`, `GetBool`, `GetOption`, etc.
- **Output rendering**: JSON/YAML/table formatting, `Print`/`PrintWarn`/`PrintError`,
  `StructuredReporter`, fixed-width display string builders, tabwriter output.
- **Terminal awareness**: TTY/ANSI capability detection, terminal-width queries,
  and console layout calculations. Core functions return structured data; only the
  CLI layer decides how to fit it on screen.
- **Reporters**: Terminal-aware colored Info/Warn/Error/Debug output to stdout/stderr.
- **Runtime**: The `Runtime` struct bundling Reporter + Logger + OCMClient + AWSClient,
  `DefaultRunner`/`RuntimeVisitor` Cobra wrappers, and lifecycle management.
- **Command registration**: The command registry that wires subcommands to root.
- **Manual-mode command builders**: `commandbuilder` packages that generate human-readable `aws ...` or `rosa ...` strings for copy-paste.
- **Argument parsing**: Unknown-flag handling, pflag normalization, region deprecation.
- **Shell completion**: Completion registration, completion callbacks, and
  `cobra.ShellCompDirective` usage.

#### `pkg/` : Core Layer (public API)

Stable, exported packages that any consumer can import. These define the domain:

- **Domain types**: Machine pool options, autoscaler config, kubelet config, IAM
  service account types, log forwarding config, break-glass credentials, etc.
- **Service operations**: Create/edit/delete/list/describe for machine pools, node
  pools, ingresses, kubelet configs, autoscalers, external auth providers, etc.
  These accept resolved values (not `*cobra.Command`) and return data structures
  and errors (not formatted strings or `os.Exit`).
- **Validation**: Label/taint parsing, replica range checks, duration format
  validation, URL validation, version feature gates.
- **Constants**: Tag keys, property keys, environment variable names.

Service operations accept **Request** types and return **Result** types (or
domain types) with errors. See
[`workflow-conventions.md`](workflow-conventions.md) for naming, construction,
optional-value handling, and the full Request/Result lifecycle.

A function in `pkg/` must **never**:
- Import Cobra, pflag, survey, or the reporter/output/interactive packages.
- Call `os.Exit`.
- Write to `os.Stdout` or `os.Stderr`.
- Read global state like `interactive.Enabled()` or `fedramp.Enabled()`.
- Build fixed-width display strings or formatted human-readable output.
- Prompt the user for input.

If a function needs environment-specific configuration (e.g., FedRAMP mode), it
receives that as a parameter, not by reading package-level state.

#### `internal/core/` : Core Layer (private implementation)

Internal packages that support `pkg/` but are not part of the public contract:

- **AWS client**: SDK v2 wrapping for IAM, STS, EC2, S3, CloudFormation, etc.
- **OCM client**: OpenShift Cluster Manager API client for clusters, node pools,
  IDPs, upgrades, versions, gates, etc.
- **Config**: OCM configuration load/save, keyring access, JWT parsing.
- **Logging**: Logger factories, SDK logger adapters, HTTP debug dumpers.
- **Caching**: File-backed TTL cache for version lists.
- **Networking**: CloudFormation stack creation/polling for VPC setup.
- **Version checking**: Mirror.openshift.com retrieval, version comparison.

These packages may be promoted to `pkg/` if external demand warrants a stable API
around them.

### What Lives Where : Quick Reference

| Concern | Owner | Examples |
|---------|-------|----------|
| Cobra `Command` struct, `Use`, `Short`, `Long` | `cmd/` | `cmd/create/machinepool/cmd.go` |
| Flag definitions (`--replicas`, `--name`, etc.) | `cmd/` or `internal/cli/` | `AddClusterFlag`, `AddFlag` |
| Interactive prompting (`GetString`, `GetBool`) | `internal/cli/` | `interactive.GetString(...)` |
| Confirmation dialogs (`--yes`, `confirm.Prompt`) | `internal/cli/` | `confirm.Confirm(...)` |
| Output rendering (JSON/YAML/table, display strings) | `internal/cli/` | `output.Print(...)`, `PrintAutoscaler` |
| Reporter (Info/Warn/Error/Debug to terminal) | `internal/cli/` | `reporter.Infof(...)` |
| Manual-mode AWS CLI command generation | `internal/cli/` | `commandbuilder.NewCommand(...)` |
| `os.Exit` | `cmd/` or `internal/cli/` | `cmd/` Run functions, `rosa.DefaultRunner` error path |
| Workflow orchestration (resolve flags, prompt, call service, render) | `cmd/` Run function | `CreateMachinepoolRunner` |
| Business logic (create pool, validate labels, build OCM request) | `pkg/` | `machinepool.CreateMachinePool(...)` |
| Request/Result types (workflow boundary contract) | `pkg/` | `machinepool.CreateMachinePoolRequest`, `CreateMachinePoolResult` |
| Domain types and validation | `pkg/` | `machinepool.MachinePool`, `ValidateLabels` |
| Shared error types (CLI/core boundary) | `pkg/` | `errors.ValidationError` |
| Constants (tag keys, property keys, env var names) | `pkg/` | `aws/tags`, `properties`, `constants` |
| AWS SDK operations | `internal/core/` | `aws.Client.CreateRole(...)` |
| OCM API operations | `internal/core/` | `ocm.Client.CreateCluster(...)` |

### Current State

The codebase does not yet follow this target architecture. Today:

- Many `pkg/` packages accept `*cobra.Command` parameters, call `interactive.GetString`,
  check `output.HasFlag()`, call `reporter.Errorf` + `os.Exit(1)`, and build formatted
  display strings.
- Several `pkg/options/` packages build complete `cobra.Command` structs; these are
  `cmd/` code living in the wrong place.
- The `pkg/rosa.Runtime` struct bundles CLI-specific lifecycle (Reporter, Cobra runner
  wrappers) with general-purpose client access.
- About 16 sites across 15 `cmd/` files call a sibling command's `Cmd.Run()`
  or mutate its flags via `Cmd.Flags().Set()`. `cmd/create/cluster` is the
  largest hub, chaining into `describe/cluster`, `create/operatorroles`,
  `create/oidcprovider`, and `logs/install`.
- The `internal/` directory exists but does not yet follow the target structure.
  Today it contains `internal/ocmrole/` (OCM role creation logic with CLI
  dependencies). In the target architecture, packages under `internal/` should
  be organized into `internal/core/` (private core implementation) or
  `internal/cli/` (shared CLI infrastructure).

The migration is incremental. See [`pkg-architecture.md`](refactor/pkg-architecture.md) for
the per-package classification and split work required.

### Migration Guidelines

When working on existing code:

- **New commands** should follow the target pattern: thin `cmd/` file that resolves
  inputs and delegates to a `pkg/` service function that accepts plain Go types and
  returns data + errors.
- **Existing commands** being modified should move in the target direction when the
  change scope justifies it, but a bug fix does not require a full layer separation.
- **New `pkg/` functions** must not introduce new CLI dependencies. If a function
  needs a value that comes from a flag or interactive prompt, accept it as a parameter.
- **New `internal/` packages** should be placed under `internal/core/` or
  `internal/cli/` according to the layer they belong to. Existing `internal/`
  packages that predate this structure will be reorganized as they are modified.

### Backward Compatibility

The migration is an internal restructuring. User-facing behavior must be
preserved unless an intentional change is explicitly scoped, reviewed, and
documented in the PR.

**Preserved contract:**

- **Flags**: Names, types, defaults, and semantics stay the same.
- **Exit codes**: All error paths currently use exit code 1. When extracting
  `os.Exit(1)` from core code, the `cmd/` layer must translate the returned
  error back to `os.Exit(1)`. Do not introduce new exit codes during migration.
- **JSON/YAML output**: The shape of `--output json` and `--output yaml`
  responses must not change. Scripts and automation depend on these structures.
- **Interactive prompts**: Question text, option lists, and default values must
  remain the same.

**Allowed cosmetic changes:**

- Error message casing and punctuation may be normalized to satisfy linter
  rules (e.g., lowercasing error strings for `staticcheck ST1005`) when a file
  is already being modified. These are lint-driven corrections, not semantic
  changes. When changing error messages, update all tests that assert on the
  exact string.

### Boundary Rules

The layer boundary is a hard line. These rules are not aspirational and apply to
all new code immediately, and to modified code as scope permits:

1. **Core layer functions accept resolved values.** A machine pool creation function
   receives a struct of options, not a `*cobra.Command` to extract flags from.
2. **Core layer functions return data and errors.** They never call `os.Exit`, print
   to stdout/stderr, or build display strings.
3. **The CLI layer owns all user interaction.** Prompting, confirmation, spinners,
   progress bars, and output formatting live exclusively in `cmd/` or `internal/cli/`.
4. **No global state in the core layer.** Functions do not read `interactive.Enabled()`,
   `fedramp.Enabled()`, `output.HasFlag()`, or similar package-level booleans.
   Configuration is passed as parameters.
5. **Display strings are not data.** Functions that need to communicate structured
   results return structs, slices, or maps. The CLI layer formats them for display.

### Import Direction

Dependencies flow downward: **CLI layer → Core layer**. Never the reverse.

The table below is the normative reference. Each cell states whether the row
package **may** or **must not** import the column package.

| Importer ↓ \ Target → | `cmd/` | `internal/cli/` | `pkg/` | `internal/core/` |
|------------------------|--------|------------------|--------|-------------------|
| **`cmd/`**             | **registration only** | **may** | **may** | **may** |
| **`internal/cli/`**    | **must not** | —          | **may** | **may**          |
| **`pkg/`**             | **must not** | **must not** | —     | **may**          |
| **`internal/core/`**   | **must not** | **must not** | **may** | —              |

Reading the table:

- **`cmd/`** may import from any other layer. It is the top of the dependency graph.
- **`internal/cli/`** may import from `pkg/` and `internal/core/`, but never from
  `cmd/`. Shared CLI infrastructure serves commands; it does not know about
  specific commands.
- **`pkg/`** may import from `internal/core/`, but never from `cmd/` or
  `internal/cli/`. This is the rule that keeps the public API free of CLI concerns.
- **`internal/core/`** may import from `pkg/` (for shared types and constants) but
  never from `cmd/` or `internal/cli/`. It is the bottom of the dependency graph.

The bidirectional permission between `pkg/` and `internal/core/` is intentional.
Both are core-layer directories with different visibility, and Go's compiler
enforces acyclicity at the package level, preventing any actual import cycle.

Prohibited imports must never appear in `pkg/` or `internal/core/`
import statements:

- `github.com/spf13/cobra`
- `github.com/spf13/pflag`
- `github.com/AlecAivazis/survey`
- Any package under `internal/cli/` (once it exists)
- Any package currently classified as CLI Presentation in
  [`pkg-architecture.md`](refactor/pkg-architecture.md): `reporter`, `output`,
  `interactive`, `arguments`, `color`, `debug`, `commands`, `rosa` (runtime),
  `options/*`, `aws/profile`, `aws/region`, `aws/commandbuilder`

This list applies to **new code immediately**. Existing violations are resolved
incrementally.

## External Boundaries

- AWS-facing behavior lives behind repo-specific helpers, wrappers, and mocks rather than raw SDK calls sprinkled through command code.
- OCM-facing behavior should stay consistent with the client and output patterns already used under `pkg/ocm/`.
- Architecture and setup expectations can differ between ROSA classic and ROSA with HCP; code and docs should say which mode they apply to.

## Validation And Generated Assets

- Local hooks enforce staged formatting on commit and full verification before push.
- `make basic-checks` and `make pre-push-checks` are the main local confidence paths before opening or updating a PR.
- Generated boundaries matter:
  - `assets/bindata.go`
  - `pkg/*/mocks/`
  - `cmd/create/idp/mocks/`
  - vendored dependencies under `vendor/`
- Command tree or flag changes usually require updates under `cmd/rosa/structure_test/` and may require `make generate-docs`.

## Risk Hotspots

- Login, token storage, credentials, STS, IAM, OIDC, and break-glass flows are security-sensitive and should not be changed casually.
- Cluster creation, edit, upgrade, and machinepool flows tend to combine CLI, AWS, and OCM behavior, so small changes can ripple.
- Dependency bumps for AWS or OCM libraries can change behavior outside the edited file; call them out explicitly and validate them end to end.
