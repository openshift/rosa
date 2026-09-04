# Workflow Request and Result Conventions

## Scope

Use this file when designing or modifying reusable workflow functions in the
core layer (`pkg/`). It defines the type conventions for data that crosses
the boundary between the CLI layer and core layer, as described in
[ARCHITECTURE.md](ARCHITECTURE.md#boundary-rules). For error handling,
wrapping, and CLI translation conventions, see
[error-conventions.md](error-conventions.md).

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

Validation is split between the CLI layer and the core layer. Each layer owns
a distinct set of concerns.

#### Invocation validation (CLI layer)

Invocation validation catches problems that only make sense in the context of
a CLI command. It runs in the `cmd/` runner before the Request is built.

Invocation validation is responsible for:

- **Flag and argument syntax**: required flags are present, mutually exclusive
  flags are not combined, positional arguments are well-formed.
- **Type coercion and parsing**: Cobra handles basic type validation for typed
  flags (e.g., `IntVar` rejects `"three"`). For string flags with format
  constraints -- CIDR ranges, version strings, comma-separated key=value
  pairs -- the CLI layer parses and validates the raw input before building
  the Request. By the time the Request is constructed, all values should be
  correctly typed and parsed.
- **Interactive prompts**: prompting for missing values when interactive mode
  is enabled.
- **Environment prerequisites**: the cluster exists, STS is enabled, the
  caller has the right AWS identity.
- **Input resolution**: resolving `file://` references to file contents,
  looking up a cluster name to get an OIDC provider ARN, calling `GetCreator`
  to determine the AWS partition.
- **Format-specific checks**: validating that a policy ARN matches the ARN
  format, so a typo is rejected before the OIDC-provider lookup (an AWS API
  call) or a `file://` read runs.

```go
// cmd/create/iamserviceaccount/cmd.go (invocation validation)

// Verify the cluster supports this operation. This is an environment
// prerequisite, not a request-invalid check, so it stays a plain wrapped
// error rather than a rosaerrors.ValidationError; see "Classifying
// validation errors" below.
if cluster.AWS().STS().RoleARN() == "" {
    return fmt.Errorf("cluster '%s' is not an STS cluster", cluster.Name())
}

// Validate flag values before building the Request. These checks are
// duplicated in CreateIAMServiceAccountRequest.Validate() too: pkg/ may
// eventually be called by something other than this CLI, so the core
// layer must not assume these checks already ran.
if len(userOptions.PolicyArns) == 0 && userOptions.InlinePolicy == "" {
    return &rosaerrors.ValidationError{Message: "at least one policy ARN or inline policy must be specified"}
}
for i, policyARN := range userOptions.PolicyArns {
    if _, err := arn.Parse(policyARN); err != nil {
        return &rosaerrors.ValidationError{
            Field:   "PolicyARNs",
            Message: fmt.Sprintf("policy ARN at index %d is invalid", i),
            Err:     err,
        }
    }
}

// Resolve file:// references before passing to the core layer
if after, ok := strings.CutPrefix(inlinePolicy, "file://"); ok {
    policyBytes, err := os.ReadFile(after)
    // ...
    inlinePolicy = string(policyBytes)
}
```

Note that the ARN check returns `&rosaerrors.ValidationError{...}` rather
than a plain `fmt.Errorf`, using the same type and wording as the domain
check it duplicates (see [Classifying validation errors](#classifying-validation-errors)).
An invocation check that fails fast still needs to be classified as an
invalid-input error like any other, so the CLI layer can react the same way
regardless of which layer caught it.

By contrast, the STS check above and a `GetCreator` lookup failure are not
request-invalid conditions. The request may be perfectly well-formed, and
the environment or AWS call itself is what failed. Those stay plain wrapped
errors (`fmt.Errorf("...: %w", err)`), not `rosaerrors.ValidationError`.
`rosaerrors.ValidationError` is reserved for "the input the caller gave was
invalid," never for "an external prerequisite or lookup failed."

Invocation validation may overlap with domain validation for usability.
Catching "no policies specified" at the flag level gives the user immediate
feedback before expensive lookups (OIDC provider, AWS identity) run. The
domain layer will catch the same condition independently.

##### Ordering within invocation validation

Within the CLI layer, order checks from cheapest to most expensive, so a
caller gets feedback from the earliest phase capable of producing it, not
from whichever phase happens to run first. The boundary that matters is
whether a check makes a new network call, not merely whether it depends on
some prior state:

1. **No new network call**: flag-value checks (required fields present, ARN
   format, mutually exclusive combinations) and checks against state already
   fetched earlier in the same command (e.g., testing a field on the
   `cluster` object `FetchCluster()` already returned). Cheapest, so these
   run first.
2. **Local resolution**: reading local state that needs no network call
   either, such as resolving a `file://` reference, together with any
   validation of what that resolves to (e.g., rejecting empty content).
3. **New AWS API calls**: lookups that hit AWS (`GetCreator`, the OIDC
   provider lookup). Run only after phases 1 and 2 have passed, since there
   is no point paying for an API call the request was already going to fail
   before reaching.
4. **Domain validation (`Request.Validate()`)**: runs inside the workflow
   function after the Request is built from the results of phases 1 to 3, and
   re-checks invariants independently of the CLI layer (see
   [Domain validation](#domain-validation-core-layer) below).

`cmd/create/iamserviceaccount/cmd.go` follows this ordering: the STS check
(phase 1, since `cluster` was already fetched) and the pure input checks
(also phase 1) run first, then inline-policy resolution (phase 2), then
`GetCreator` and the OIDC provider lookup (phase 3), then the Request is
built and passed to a workflow function that calls `Validate()` (phase 4). A
new command should follow the same ordering rather than interleaving cheap
and expensive checks based on the order fields happen to appear in the flag
definitions.

#### Domain validation (core layer)

Domain validation protects business invariants that must hold regardless of
how the workflow is called -- CLI, TUI, test, or automation. It runs inside
the workflow function via `Validate()` on the Request, before any side
effects.

Domain validation is responsible for:

- **Required fields**: all fields the workflow needs are present and non-empty.
- **Domain naming rules**: Kubernetes service account and namespace names
  follow RFC 1123 / DNS subdomain conventions.
- **Cross-field constraints**: a role name is required when multiple service
  accounts share a single role.
- **Conditional requirements**: GovCloud environments require an account ID
  and partition.
- **Value integrity**: optional pointer fields are not empty strings when
  provided (e.g., `InlinePolicy` is `*string`; `nil` means absent, but a
  pointer to `""` is invalid).

```go
// pkg/iamserviceaccount/create.go (domain validation)

func (r *CreateIAMServiceAccountRequest) Validate() error {
    if r.ClusterName == "" {
        return fmt.Errorf("cluster name is required")
    }
    for _, sa := range r.ServiceAccounts {
        if err := ValidateServiceAccountName(sa.Name); err != nil {
            return fmt.Errorf("invalid service account name %q: %w", sa.Name, err)
        }
    }
    if r.RoleName == nil && len(r.ServiceAccounts) > 1 {
        return fmt.Errorf("role name is required when specifying multiple service accounts")
    }
    // ...
}
```

Domain validation belongs in the same package as the Request type, either as
a method on the Request or as a standalone function. The workflow function
calls `Validate()` before performing any operation and must not assume the
CLI layer validated first.

#### Validator lists

A single linear if-chain works for a handful of invariants, but it gets hard
to read and to test in isolation as invariants accumulate: an early `return`
hides every check that comes after it, so a caller only ever sees the first
violation, and there is no way to exercise one invariant without exercising
all of the ones before it.

Prefer a list of small, independently testable validator functions once a
Request has more than a few invariants. Each validator checks exactly one
concern and returns its own violations as a `[]error` (nil if none), rather
than pre-joining them with `errors.Join` itself. `Validate()` collects every
validator's slice into one flat list and calls `errors.Join` exactly once,
so a caller sees every violation in a single response instead of fixing
them one at a time, and no validator ends up joining a join:

```go
// pkg/iamserviceaccount/create.go (domain validation)

var createValidators = []func(*CreateIAMServiceAccountRequest) []error{
    validateClusterName,
    validateOIDCProviderARN,
    validateServiceAccountsPresent,
    validateServiceAccountIdentifiers,
    validatePolicies,
    validateRoleName,
    validateGovcloud,
}

func (r *CreateIAMServiceAccountRequest) Validate() error {
    var errs []error
    for _, validate := range createValidators {
        errs = append(errs, validate(r)...)
    }
    return errors.Join(errs...)
}

func validateClusterName(r *CreateIAMServiceAccountRequest) []error {
    if r.ClusterName == "" {
        return []error{&rosaerrors.ValidationError{Field: "ClusterName", Message: "cluster name is required"}}
    }
    return nil
}
```

A validator that checks more than one thing (e.g. `validateGovcloud`) builds
its own local `[]error` with plain `append` and returns it directly --
it never calls `errors.Join` itself. Only `Validate()` calls `errors.Join`,
over the fully flattened list. Each validator returns
`&rosaerrors.ValidationError{...}` rather than a plain `fmt.Errorf`; see
[Classifying validation errors](#classifying-validation-errors) below for
why and for the full pattern.

`pkg/iamserviceaccount/create.go` implements the full validator list for
`CreateIAMServiceAccountRequest`. New Request types with more than a few
invariants should follow the same shape rather than growing a single
`Validate()` method.

#### Classifying validation errors

A workflow function (e.g. `CreateIAMServiceAccount`) calls `Validate()`
internally, before performing any side effects. Its caller never calls
`Validate()` directly, so a plain `error` returned from the workflow function
does not tell the caller whether the request was invalid (nothing happened)
or whether a later, operational step failed (e.g. a role was created but
attaching a policy to it failed). Callers that need to react differently to
those two cases must be able to tell them apart with `errors.As`, not by
inspecting message text.

Have individual validator functions return `&rosaerrors.ValidationError{Field,
Message}` (`pkg/errors`, imported as `rosaerrors` since it shares its name
with the standard library `errors` package) instead of a plain `fmt.Errorf`:

```go
func validateClusterName(r *CreateIAMServiceAccountRequest) []error {
    if r.ClusterName == "" {
        return []error{&rosaerrors.ValidationError{Field: "ClusterName", Message: "cluster name is required"}}
    }
    return nil
}
```

`Field` is the request field the check applies to, when it maps to exactly
one field; leave it empty for a check that spans more than one (see
`validatePolicies`'s "at least one policy ARN or inline policy is required"
for an example). `Error()` never renders `Field` — it exists purely as
structured metadata for a consumer that wants to branch or report on
*which* field failed (a future `--output json` mode, a REST API, a TUI)
without parsing the message. Introducing `Field` therefore never changes
existing user-facing text.

When a check wraps a lower-level error (`ValidateServiceAccountName`,
`arn.Parse`), set `Err` instead of interpolating the error into `Message`,
so the cause survives for `errors.Is`/`errors.As`:

```go
if err := ValidateServiceAccountName(sa.Name); err != nil {
    errs = append(errs, &rosaerrors.ValidationError{
        Field:   "ServiceAccounts",
        Message: fmt.Sprintf("invalid service account name %q", sa.Name),
        Err:     err,
    })
}
```

Then, at the point a workflow function calls `Validate()` internally, wrap
its aggregate result the same way — this time with only `Err` set:

```go
if err := req.Validate(); err != nil {
    return nil, &rosaerrors.ValidationError{Err: fmt.Errorf("invalid request: %w", err)}
}
```

This aggregate wrap guarantees every `Validate()` failure is classified as
a validation error regardless of what individual checks return, without
relying on every check remembering to use the typed error. A caller can
then do:

```go
var validationErr *rosaerrors.ValidationError
if errors.As(err, &validationErr) {
    // invalid input: no side effects occurred
}
```

`errors.As` recurses through both the aggregate wrapper and the
`errors.Join` tree of individual checks automatically, so this one call
finds a match regardless of which check (or how many) failed.

**`rosaerrors.ValidationError` is shared by every workflow.** Do not define
a per-workflow type such as `CreateIAMServiceAccountValidationError` or
`DeleteIAMServiceAccountValidationError` — callers only ever need to know
"was this input invalid," never which workflow produced it. Only the
`Validate()` failure path (and the individual checks it runs) return this
type; a workflow's other failure paths (AWS/OCM calls, network errors) stay
plain wrapped errors and correctly do not match `*rosaerrors.ValidationError`.

`rosaerrors.ValidationError` is for exactly one condition: "the request was
invalid." When a caller needs to distinguish some other specific condition —
not "was input invalid," but something like "this role already exists in a
different profile" — use a sentinel (`var ErrX = errors.New(...)`) or a small
typed error carrying data, scoped to that one condition, matching existing
precedent: `cmd/whoami/cmd.go`'s `errNotLoggedIn`, `cmd/create/ocmrole`'s
`ErrRoleExistsWrongProfile`, `cmd/initialize/cmd.go`'s `errInitExitZero`.
Don't create a sentinel or type speculatively for every validator or every
failure path — only where a caller genuinely needs to branch on it.

#### Intentional duplication

Some checks appear in both layers. This is acceptable when:

- The CLI check provides **early feedback** before expensive operations (API
  calls, file I/O, interactive prompts).
- The domain check ensures the **invariant holds for all callers**, not just
  the CLI.

Do not remove a domain validation just because the CLI also checks the same
condition. Do not add domain validation solely to catch CLI-specific concerns
(e.g., flag syntax, interactive mode state).

#### What does not belong in domain validation

- Cobra flag registration errors (handled by Cobra itself).
- Interactive prompt flow control (`interactive.Enabled()`).
- Output format selection (`--output json`).
- File system operations (`file://` resolution, config file reads).
- AWS or OCM client construction and authentication.

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

This is why the *interface declaration* is scoped per workflow, but the
*concrete client implementation* that satisfies it does not have to be
written once per workflow. When several workflows need the same
underlying operations and the only obstacle is a signature mismatch (e.g.
`pkg/aws.Client.EnsureRole` takes a `reporter.Logger` where a workflow's
interface expects `context.Context`), write that translation once as a
small shared type, not as a hand-rolled adapter struct copied into each
CLI command. `pkg/aws/rolebridge.Client` is this in practice: it adapts
`aws.Client`'s `EnsureRole`/`AttachRolePolicy`/`PutRolePolicy` to
`context.Context`-based signatures, and any workflow's
`<Verb><Resource>Client` interface that needs those same three methods is
satisfied by it directly, with no per-workflow adapter file. Delete a
bridge like this once the underlying signature mismatch it exists to paper
over is fixed at the source.

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
