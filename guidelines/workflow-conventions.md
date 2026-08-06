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
| Required field | Use the value type directly (`string`, `int`). Validation rejects the zero value only if the domain disallows it. |
| Optional field | Use a pointer (`*string`, `*int`). `nil` means "not specified." |
| A group of fields is all-or-nothing | Use an embedded struct pointer. `nil` means the group is absent. |

The pointer convention makes optionality visible at the type level without
requiring the reader to check comments, validation code, or domain rules.
A value type means the caller must always provide the field, but it cannot
distinguish "the caller omitted this" from "the caller explicitly supplied
the zero value." When zero is not a valid domain value (an empty cluster ID,
a zero replica count where at least one is required), validation should
reject it, and that ambiguity does not matter. When zero is itself a valid
domain value (a replica count of zero is a legitimate autoscaling floor),
validation must not reject it just because it is the zero value, and the
field should use a pointer instead if the caller also needs to express
"not specified."

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
    if r.MaxReplicas != nil && r.MinReplicas == nil {
        return fmt.Errorf("min replicas is required when max replicas is set")
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
type Warning struct {
    Code    string
    Message string
}

type CreateMachinePoolResult struct {
    MachinePool *MachinePool
    Warnings    []Warning
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

### Multiple Backends for the Same Verb and Resource

When the same verb and resource is served by independent backend trees (for
example, separate v1 and v2 command/runner stacks), do not force one Request
type to cover both. Give each tree its own Request and Result types, even
with identical names, and let Go package namespacing disambiguate them (e.g.,
`rosav1.CreateIAMServiceAccountRequest` vs.
`rosav2.CreateIAMServiceAccountRequest`). Each type still follows the naming,
construction, and validation conventions in this document independently
within its own tree.

Do not extract shared code between the two trees, even where a Request,
domain type, or client helper looks identical today. v1 and v2 are expected
to evolve independently, and v1 is temporary: it will be removed once v2
replaces it. A shared package couples the two together in the meantime, so a
change made for v2 can accidentally affect v1's behavior (or vice versa), and
once v1 is removed, that "shared" package is left as a single-consumer
package that only ever served v2. Duplicating the code between the two trees
costs some repetition, but it keeps v1 stable and isolates risk between the
two, and it avoids guessing at an abstraction before there are two real,
long-lived consumers to justify one.

## Workflow Function Signatures

Workflow functions accept a `context.Context`, a Request, and any needed
client interfaces, and return a Result or domain type with an error:

```go
func CreateMachinePool(ctx context.Context, client CreateMachinePoolClient, req CreateMachinePoolRequest) (*CreateMachinePoolResult, error)
func EditMachinePool(ctx context.Context, client EditMachinePoolClient, req EditMachinePoolRequest) (*MachinePool, error)
func DeleteMachinePool(ctx context.Context, client DeleteMachinePoolClient, req DeleteMachinePoolRequest) error
func ListMachinePools(ctx context.Context, client ListMachinePoolsClient, req ListMachinePoolsRequest) (*ListMachinePoolsResult, error)
func DescribeMachinePool(ctx context.Context, client DescribeMachinePoolClient, req DescribeMachinePoolRequest) (*MachinePool, error)
```

Do not accept `*rosa.Runtime` as a parameter. It bundles CLI lifecycle
(Reporter, Cobra runner wrappers) with client access. Accept the specific
client interfaces the workflow needs.

### Client Interfaces Are Scoped Per Workflow, Not Per Package

Define one client interface per workflow function, named `<Verb><Resource>Client`
to match the Request/Result naming convention, containing only the methods
that workflow calls:

```go
type CreateMachinePoolClient interface {
    CreateMachinePool(ctx context.Context, clusterID string, pool *MachinePool) (*MachinePool, error)
}

type DeleteMachinePoolClient interface {
    DeleteMachinePool(ctx context.Context, clusterID, id string) error
}
```

"One interface per workflow" does not mean "one method per interface." A
workflow that needs several client calls still gets a single interface
covering all of them, e.g. a create workflow that also attaches policies:

```go
type CreateIAMServiceAccountClient interface {
    EnsureRole(ctx context.Context, name string, policy string, ...) (string, error)
    AttachRolePolicy(ctx context.Context, roleName string, policyARN string) error
    PutRolePolicy(ctx context.Context, roleName string, policyName string, policy string) error
}
```

The point is not sharing that interface with any other workflow, not
minimizing method count. Two workflows may need identically-named methods
(e.g., both `DescribeMachinePool` and some other workflow call `GetMachinePool`);
each still declares its own interface. Go's structural typing lets one
concrete client type satisfy every one of these interfaces without them
referencing each other or the client needing to know they exist.

Do not collapse a workflow's multiple client calls behind one higher-level
method (e.g., a single `Create(...)` that internally calls `EnsureRole`,
`AttachRolePolicy`, and `PutRolePolicy`) just to shrink the interface or
simplify a test mock. The ordering and looping between those calls (create
the role, then attach each managed policy, then attach the inline policy) is
the reusable behavior this workflow exists to provide. Hiding it inside one
opaque client method pushes that logic into every adapter that implements
the interface, and workflow-level tests lose the ability to assert on it
(e.g., that a failed validation makes zero client calls, or that policies
attach only after the role exists).

Do not group workflows behind a shared reader/writer or `<Resource>Service`
interface by default. Sharing an interface across workflows means a
dependency added for one workflow's needs shows up in every other workflow's
signature, even ones that never call it. Prefer a package-level function
over a method on a shared `Service` struct when the workflow has no state to
hold. Introduce a shared `Service`, or group functions behind one interface,
only when multiple workflows demonstrably need to be swapped or mocked
together, not merely because they operate on the same resource.

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
