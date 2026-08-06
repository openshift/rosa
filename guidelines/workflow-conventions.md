# Workflow Request and Result Conventions

## Scope

Use this file when designing or modifying reusable workflow functions in the
core layer (`pkg/`). It defines the type conventions for data that crosses
the boundary between the CLI layer and core layer, as described in
[ARCHITECTURE.md](ARCHITECTURE.md#boundary-rules).

## Lifecycle

A ROSA CLI operation follows this data flow:

```
 User input          CLI Layer                 Core Layer
 ───────────    ───────────────────    ─────────────────────────
 flags/args  →  parse, prompt,     →  workflow function receives
                resolve defaults,     Request, validates, executes,
                build Request         returns (Result, error)
                                                    │
                                                    ▼
                                      CLI formats Result for
                                      display (table/JSON/YAML)
```

The **Request** and **Result** types are the boundary contract. The CLI layer
constructs a Request from resolved user input. The core layer returns a Result
(or a domain type) with an error. Neither type carries CLI, terminal, or
presentation concerns.

## Request Types

### Naming

Name request types `<Verb><Resource>Request`:

```go
type CreateMachinePoolRequest struct { ... }
type EditIngressRequest struct { ... }
type UpgradeClusterRequest struct { ... }
```

For list and describe operations with few parameters, a dedicated request type
is still preferred over loose parameters. It keeps the workflow function
signature stable when new filters or options are added later:

```go
type ListMachinePoolsRequest struct {
    ClusterID string
}

type DescribeKubeletConfigRequest struct {
    ClusterID string
    Name      string
}
```

### What a Request Contains

A Request carries the **resolved domain values** needed to perform the
operation. Every field should pass the core-layer test: would this value
be meaningful to a TUI, a headless program, or a REST API?

Typical contents:

- Identifiers: cluster ID, resource name, account ID.
- Domain parameters: instance type, replica count, labels, taints.
- Configuration values: autoscaling bounds, CIDR ranges, version strings.
- Behavioral flags with domain meaning: `DryRun bool`, `BestEffort bool`.

### What a Request Must Not Contain

A Request must never carry:

- `*cobra.Command`, `*pflag.FlagSet`, or any Cobra/pflag type.
- Reporter, spinner, logger, or terminal-state references.
- Interactive prompt state (`interactive.Enabled()`, survey validators).
- Output format preferences (`--output json`).
- The `rosa.Runtime` struct (it bundles CLI lifecycle with client access).
- Display strings, ANSI codes, or pre-formatted output.

If a workflow needs an AWS or OCM client, accept the client interface as a
separate parameter to the workflow function, not as a field on the Request.

### Construction

The CLI layer is responsible for constructing a fully resolved Request. This
means:

1. Parse flags and positional arguments.
2. Run interactive prompts (when interactive mode is enabled).
3. Apply CLI-side defaults for any value the user did not provide.
4. Look up dependent values (e.g., resolve a cluster name to a cluster ID).
5. Build the Request with all fields populated.

The core layer never constructs a Request from CLI inputs. It receives one
ready to use.

```go
// cmd/create/machinepool/cmd.go (CLI layer)
func run(cmd *cobra.Command, args []string) {
    // ... resolve flags, prompt, look up cluster ...

    request := machinepool.CreateMachinePoolRequest{
        ClusterID:    clusterID,
        Name:         name,
        InstanceType: instanceType,
        Replicas:     replicas,
    }
    result, err := machinepoolService.CreateMachinePool(ctx, ocmClient, request)
    // ... handle error, format result for display ...
}
```

### Optional Values and Zero-Value Behavior

Use Go's type system to communicate whether a field is required or optional:

| Situation | Convention |
|-----------|-----------|
| Required field | Use the value type directly (`string`, `int`). Validation rejects the zero value. |
| Optional field | Use a pointer (`*string`, `*int`). `nil` means "not specified." |
| A group of fields is all-or-nothing | Use an embedded struct pointer. `nil` means the group is absent. |

The pointer convention makes optionality visible at the type level without
requiring the reader to check comments, validation code, or domain rules.
A value type means the caller must always provide the field (even when
zero is a valid value). A pointer means the caller may omit it.

```go
type CreateMachinePoolRequest struct {
    ClusterID    string       // required
    Name         string       // required
    InstanceType *string      // optional; nil means "use API default"
    Replicas     *int         // optional; nil means "use API default"
    MinReplicas  *int         // optional; nil means "autoscaling not configured"
    MaxReplicas  *int         // optional; nil means "autoscaling not configured"
    SpotConfig   *SpotConfig  // optional; nil means "no spot instances"
}
```

For boolean fields, prefer naming that makes `false` the common default so
that an unset pointer (`nil`) and an explicit `false` behave the same way.
When the boolean is required or its zero value is always the common case,
a value type is acceptable:

```go
SkipHealthCheck *bool  // optional; nil/false = perform health check

DryRun bool            // required flag; false = normal execution
```

### Defaults

There are three layers of defaults. Keep them in their respective layers:

| Default type | Owner | Example |
|--------------|-------|---------|
| **CLI default** | CLI layer, applied when building the Request | `--replicas` defaults to 2 when the user omits the flag |
| **Domain default** | Core layer, applied when a Request field is zero/nil | Workflow sets instance type to `m5.xlarge` when `InstanceType` is empty |
| **API default** | External service (OCM, AWS), applied server-side | OCM applies default network settings when not specified |

The core layer documents domain defaults near the workflow function or on the
Request type. CLI defaults live in the `cmd/` flag definitions. API defaults
are owned by the external service and should not be duplicated in the
codebase.

### Validation

The core layer validates the Request before performing any operation. Return
a descriptive error for invalid input rather than silently correcting it or
exiting the process.

Validation belongs in the same package as the Request type, either as a method
on the Request or as a standalone function:

```go
func (r *CreateMachinePoolRequest) Validate() error {
    if r.ClusterID == "" {
        return fmt.Errorf("cluster ID is required")
    }
    if r.Name == "" {
        return fmt.Errorf("machine pool name is required")
    }
    if r.MinReplicas != nil && r.MaxReplicas == nil {
        return fmt.Errorf("max replicas is required when min replicas is set")
    }
    return nil
}
```

The CLI layer may also validate early (e.g., checking that a required flag is
present before prompting for additional values), but the core layer must not
assume the caller validated first.

## Result Types

### Naming

Name result types `<Verb><Resource>Result`:

```go
type CreateMachinePoolResult struct { ... }
type ListMachinePoolsResult struct { ... }
type UpgradeClusterResult struct { ... }
```

### When to Use a Result Type

Use a dedicated Result type when the workflow produces a **compound outcome**
that the caller needs to act on:

- The primary resource plus metadata (e.g., warnings, the operation performed).
- A list of resources plus pagination or summary information.
- Multiple related resources affected by a single operation.

When the workflow returns a **single domain object** and success/failure is
the only other signal, return the domain type directly:

```go
// Single domain object: return it directly.
func (s *service) DescribeMachinePool(req DescribeMachinePoolRequest) (*MachinePool, error)

// Compound outcome: use a Result type.
func (s *service) CreateMachinePool(req CreateMachinePoolRequest) (*CreateMachinePoolResult, error)
```

When the workflow performs an action and the caller only needs to know
whether it succeeded, returning only `error` is acceptable:

```go
func (s *service) DeleteMachinePool(req DeleteMachinePoolRequest) error
```

### What a Result Contains

A Result exposes **domain outcomes** that the caller can inspect, branch on,
or pass to another workflow:

- The primary domain object (created, updated, or retrieved resource).
- Structured warnings or advisories (not pre-formatted warning strings).
- Operation metadata: was this a no-op, which resources were affected.

```go
type CreateMachinePoolResult struct {
    MachinePool *MachinePool
    Warnings    []string
}

type ListMachinePoolsResult struct {
    MachinePools []*MachinePool
}
```

### What a Result Must Not Contain

A Result must never carry:

- Pre-formatted display strings or fixed-width table rows.
- ANSI color codes or terminal-width-aware layout.
- Manual-mode CLI command strings (e.g., `aws iam create-role ...`).
- Reporter messages or log lines.
- Exit codes.

The CLI layer owns all formatting decisions. It reads structured data from the
Result and renders it as a table, JSON, YAML, or interactive output.

### Domain Types

Result types often wrap domain types. A domain type represents a resource or
concept in the ROSA domain model:

```go
type MachinePool struct {
    ID           string
    Name         string
    ClusterID    string
    InstanceType string
    Replicas     int
    // ...
}
```

Domain types are reusable across workflows. They are not tied to a single
operation. Place domain types in the same package as the workflows that
produce and consume them.

When the existing OCM SDK type (`*cmv1.MachinePool`) is the natural return
value and no additional fields are needed, returning the SDK type directly is
acceptable. Introduce a local domain type when:

- The workflow combines data from multiple SDK calls.
- The workflow needs to expose a simpler or narrower view than the SDK type.
- The SDK type includes internal or transport-level fields the caller should
  not see.

## Package Ownership

Request, Result, and domain types live in the `pkg/` package that owns the
workflow function. They are core-layer types. The workflow function, its
Request, its Result, and its domain types form a cohesive unit:

```
pkg/machinepool/
    machinepool.go       // workflow functions (service interface + implementation)
    request.go           // CreateMachinePoolRequest, EditMachinePoolRequest, ...
    result.go            // CreateMachinePoolResult, ListMachinePoolsResult, ...
    types.go             // MachinePool, SpotConfig, ... (domain types)
    validation.go        // validation functions and helpers
```

Small packages may keep everything in a single file. The file organization is
a guideline, not a rule; what matters is that the types are in `pkg/`, not in
`cmd/` or `internal/cli/`.

## Workflow Function Signatures

Workflow functions accept a `context.Context`, a Request, and any needed
service clients, and return a Result or domain type with an error:

```go
type MachinePoolService interface {
    CreateMachinePool(ctx context.Context, ocmClient ocm.Client, req CreateMachinePoolRequest) (*CreateMachinePoolResult, error)
    EditMachinePool(ctx context.Context, ocmClient ocm.Client, req EditMachinePoolRequest) (*MachinePool, error)
    DeleteMachinePool(ctx context.Context, ocmClient ocm.Client, req DeleteMachinePoolRequest) error
    ListMachinePools(ctx context.Context, ocmClient ocm.Client, req ListMachinePoolsRequest) (*ListMachinePoolsResult, error)
    DescribeMachinePool(ctx context.Context, ocmClient ocm.Client, req DescribeMachinePoolRequest) (*MachinePool, error)
}
```

Do not accept `*rosa.Runtime` as a parameter. It bundles CLI lifecycle
(Reporter, Cobra runner wrappers) with client access. Accept the specific
client interfaces the workflow needs.

## Relationship to Existing Types

The codebase currently uses `*UserOptions` and `*Options` types for flag
resolution:

- `CreateMachinepoolUserOptions`: Raw flag values (strings, ints, bools)
  defined close to the Cobra command.
- `CreateMachinepoolOptions`: Wrapper with a `Bind()` method for positional
  argument resolution.

These are CLI-layer types. They are inputs to the flag-resolution and
prompting process, not inputs to the core workflow. The lifecycle is:

```
UserOptions (CLI)  →  resolve/prompt  →  Request (core boundary)  →  workflow  →  Result (core boundary)
```

As the codebase migrates toward the target architecture, existing
`*UserOptions` and `*Options` types remain in the CLI layer (`cmd/` or
`internal/cli/`). New Request and Result types are added in `pkg/` as
workflows are extracted. There is no need to rename existing types; the naming
difference (`UserOptions` vs. `Request`) reinforces which layer owns each
type.

## Review Prompts

- Does the Request contain only resolved domain values, free of CLI types?
- Does the Result expose structured data, not pre-formatted display strings?
- Are optional fields using the right zero-value or pointer convention?
- Is validation performed in the core layer, not assumed from the CLI?
- Would this Request and Result be equally useful from a TUI, automation tool,
  or REST API?
