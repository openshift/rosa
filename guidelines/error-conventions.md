# Error Conventions

## Scope

Use this file when writing or modifying code that produces, propagates, or
translates errors across the CLI/core boundary described in
[ARCHITECTURE.md](ARCHITECTURE.md). For Request/Result lifecycle conventions,
see [workflow-conventions.md](workflow-conventions.md).

## Principle

Core layer code (`pkg/`) returns errors. CLI layer code (`cmd/`) decides how
to present them to the user and whether to exit the process. The core layer
never controls the user experience of a failure.

## Core Layer Error Rules

### Error string formatting

Error strings must start with a lowercase letter and must not end with
punctuation, per Go convention (enforced by `staticcheck`'s ST1005). They
are often wrapped into a larger sentence by a caller:

```go
// Good
return fmt.Errorf("failed to create role: %w", err)

// Bad: capitalized and punctuated as if it were a standalone sentence.
return fmt.Errorf("Failed to create role: %w.", err)
```

### Return errors, never exit

Core layer functions must return errors to their caller. They must not:

- Call `os.Exit`.
- Call `reporter.Errorf` or any reporter method.
- Call `log.Fatal` or any function that terminates the process.
- Depend on Cobra usage-error behavior (e.g., `cmd.Usage()`).

```go
// Good: return the error for the caller to handle.
func (s *Service) CreateIAMServiceAccount(ctx context.Context, client CreateIAMServiceAccountClient,
    req CreateIAMServiceAccountRequest) (*CreateIAMServiceAccountResult, error) {
    if err := req.Validate(); err != nil {
        return nil, &rosaerrors.ValidationError{Err: fmt.Errorf("invalid request: %w", err)}
    }
    // ...
}

// Bad: core code should not exit or log to the user.
func CreateRole(reporter reporter.Logger, roleName string) {
    // ...
    reporter.Errorf("Failed to create role: %s", err)
    os.Exit(1)
}
```

Functions in `pkg/rosa/runtime.go` currently call `os.Exit` in methods like
`FetchCluster` and `WithAWS`. This is a known legacy pattern. New code must
not follow it.

### Wrap with context using %w

When a function catches an error from a lower layer, wrap it with
`fmt.Errorf` and the `%w` verb so the original error remains available for
matching with `errors.Is` and `errors.As`:

```go
roleARN, err := client.EnsureRole(ctx, roleName, trustPolicy, ...)
if err != nil {
    return nil, fmt.Errorf("failed to create role: %w", err)
}
```

The wrapping message should describe what the current function was trying to
do, not repeat the underlying error's message. The result reads as a chain:

```
failed to create role: AccessDenied: User is not authorized to perform iam:CreateRole
```

### Do not wrap with static messages that lose the cause

Wrapping without `%w` discards the error chain:

```go
// Bad: the original error is converted to a string and cannot be matched.
return fmt.Errorf("failed to create role: %s", err)

// Good: the original error is preserved in the chain.
return fmt.Errorf("failed to create role: %w", err)
```

## Error Types

### When to use each form

| Form | When to use | How the caller matches |
|------|-------------|----------------------|
| `fmt.Errorf("...: %w", err)` | Default. Wrapping an error from a lower layer. | `errors.Is(err, target)` or `errors.As(err, &target)` on the wrapped cause |
| Typed error (struct) | The caller needs to inspect structured fields or branch on the error category. | `errors.As(err, &typedErr)` |
| Sentinel (`var ErrX = errors.New(...)`) | The caller needs to branch on one specific, fixed condition. | `errors.Is(err, ErrX)` |

Don't create a typed error or sentinel speculatively for every failure path.
Most errors are adequately served by `fmt.Errorf` with `%w` wrapping; add a
type or sentinel only where a caller genuinely needs to branch on it.

### `rosaerrors.ValidationError`: the one shared classification every workflow uses

`pkg/errors.ValidationError` marks the one error condition every workflow
that follows the [Request/Validate() pattern](workflow-conventions.md#validation)
needs its caller to distinguish: "the request was invalid, no side effects
occurred" versus "an operational step further into the workflow failed."
See [Classifying validation errors](workflow-conventions.md#classifying-validation-errors)
for the full pattern and rationale.

`pkg/errors` shares its name with the standard library `errors` package
(the package almost every caller also needs, for `errors.Is`/`errors.As`),
so it is imported under an alias:

```go
// pkg/iamserviceaccount/create.go
import rosaerrors "github.com/openshift/rosa/pkg/errors"
```

Individual validator functions return it directly, with `Field` set when
the check maps to exactly one request field:

```go
func validateClusterName(r *CreateIAMServiceAccountRequest) []error {
    if r.ClusterName == "" {
        return []error{&rosaerrors.ValidationError{Field: "ClusterName", Message: "cluster name is required"}}
    }
    return nil
}
```

`Field` is structured metadata only — `Error()` never renders it, so adding
`Field` to a check never changes its user-facing text. It exists for a
consumer that wants to branch or report on *which* field failed without
parsing the message (a future `--output json` mode, a REST API, a TUI). When
a check wraps a lower-level error, set `Err` too so the cause survives for
`errors.Is`/`errors.As` instead of being flattened into the message string.

The workflow function then wraps `Validate()`'s aggregate result the same
way, this time with only `Err` set:

```go
func (s *Service) CreateIAMServiceAccount(ctx context.Context, client CreateIAMServiceAccountClient,
    req CreateIAMServiceAccountRequest) (*CreateIAMServiceAccountResult, error) {
    if err := req.Validate(); err != nil {
        return nil, &rosaerrors.ValidationError{Err: fmt.Errorf("invalid request: %w", err)}
    }
    // ... EnsureRole/AttachRolePolicy/PutRolePolicy failures stay plain
    // wrapped errors; they correctly do not match *rosaerrors.ValidationError.
}
```

This aggregate wrap guarantees classification holds regardless of what any
individual check returns, rather than relying on every check remembering to
use the typed error.

The CLI layer matches it via `errors.As` whenever it needs to know whether
the request was invalid, independent of what it does with that information:

```go
result, err := service.CreateIAMServiceAccount(ctx, adapter, req)
if err != nil {
    return err
}
```

`errors.As` recurses through the aggregate wrapper and the `errors.Join`
tree of individual checks automatically, so one call finds a match
regardless of which check failed, without the CLI layer needing to walk the
tree itself.

`rosaerrors.ValidationError` is shared by every workflow — do not define a
per-workflow type such as `CreateIAMServiceAccountValidationError`. Callers
only ever need to know "was this input invalid," never which workflow
produced it.

### Showing command usage

Show `cmd.Usage()` only when a required flag was not supplied at all, not
for every `rosaerrors.ValidationError`. A malformed value (an invalid ARN,
an empty inline policy) already has its own error message; printing the
full flag list next to it reads as "your flags are wrong" when the value
was the problem.

`cmd/create/iamserviceaccount/cmd.go` calls `cmd.Usage()` inline, only at
the specific invocation checks that catch an entirely missing required
value, before returning the `rosaerrors.ValidationError`:

```go
if len(userOptions.ServiceAccountNames) == 0 {
    cmd.Usage()
    return &rosaerrors.ValidationError{
        Field: "ServiceAccounts", Message: "at least one service account name is required",
    }
}
```

It does not call `cmd.Usage()` for a malformed-value check (an invalid ARN
format), nor for the aggregate `rosaerrors.ValidationError` that
`service.CreateIAMServiceAccount` can return from domain validation: those
failures are explained well enough by their own message.

### Other error categories: typed errors and sentinels

For an operational condition that needs its own distinct CLI handling — not
"was input invalid," but something workflow-specific like "this role already
exists in a different profile" — use a sentinel or a small typed error
carrying data, scoped to that one condition, following existing repo
precedent: `cmd/whoami/cmd.go`'s `errNotLoggedIn`, `cmd/create/ocmrole`'s
`ErrRoleExistsWrongProfile`, `cmd/initialize/cmd.go`'s `errInitExitZero`.

## CLI Layer Translation

### The DefaultRunner pattern

`rosa.DefaultRunner` is the standard error exit point for newer commands.
A `CommandRunner` function returns an error; `DefaultRunner` prints it via
`reporter.Errorf` and calls `os.Exit(1)`:

```go
// pkg/rosa/runner.go
func DefaultRunner(visitor RuntimeVisitor, runner CommandRunner) func(command *cobra.Command, args []string) {
    return func(command *cobra.Command, args []string) {
        ctx := context.Background()
        r := NewRuntime()
        defer r.Cleanup()

        if visitor != nil {
            visitor(ctx, r, command, args)
        }

        if err := runner(ctx, r, command, args); err != nil {
            r.Reporter.Errorf("%s", err)
            os.Exit(1)
        }
    }
}
```

Commands using `DefaultRunner` should return errors, not call `os.Exit`
or `reporter.Errorf` for terminal failures.

### Adding remediation guidance

When the CLI layer can identify the error category, it can add guidance
that would be inappropriate in the core layer (because the core layer does
not know it is running inside a CLI). `cmd/create/iamserviceaccount/cmd.go`
does this today by showing command usage for a missing-required-flag error
(see [Showing command usage](#showing-command-usage) above). A command could
go further and add remediation text for an operational failure the same way:

```go
result, err := service.CreateIAMServiceAccount(ctx, adapter, req)
if err != nil {
    var validationErr *rosaerrors.ValidationError
    if errors.As(err, &validationErr) {
        return err
    }
    // Illustrative: operational errors could get remediation context the
    // core layer can't add. Not yet done for this command.
    return fmt.Errorf("%w\n\nVerify that your AWS credentials are valid and "+
        "that your IAM user has permission to create roles", err)
}
```

Remediation strings are CLI concerns. The core layer should never embed
guidance like "run `aws iam ...`" or "check your credentials" in its errors.

### Exit codes

The CLI layer decides the exit code. The core layer never returns or
suggests an exit code. Currently, ROSA uses `0` for success and `1` for
all failures. [ARCHITECTURE.md](ARCHITECTURE.md#backward-compatibility)
prohibits introducing new exit codes during the migration to the target
architecture. When that constraint is relaxed, the CLI layer would map
error types to codes — the typed error infrastructure (`rosaerrors.ValidationError`
and future categories) supports this, but it is not in use today.

Since every failure exits `1` today, ordering matters more than the code
itself: cheap checks that are likely to fail should run before expensive
ones, so a doomed command exits on the cheapest possible check instead of
paying for a network call first. See
[Ordering within invocation validation](workflow-conventions.md#ordering-within-invocation-validation)
for the phase breakdown.

## Diagnostic Context

### What to include in wrapped errors

Include context that helps identify which operation failed and on what
input. Include identifiers (role name, cluster ID, policy ARN) but not
large payloads (full policy documents, API response bodies) or sensitive
values (credentials, tokens, secrets), even when they would fit inline:

```go
// Good: identifies the operation and the resource.
return fmt.Errorf("failed to attach policy '%s' to role '%s': %w", policyARN, roleName, err)

// Bad: includes the entire policy document.
return fmt.Errorf("failed to put inline policy %s on role %s: %w", policyDocument, roleName, err)

// Bad: leaks a credential into logs, terminal history, and bug reports.
return fmt.Errorf("failed to authenticate with token %s: %w", token, err)
```

### Reporter for non-fatal messages

Use `reporter.Infof` and `reporter.Debugf` in the CLI layer for progress
messages, warnings, and diagnostic output that are not errors:

```go
r.Reporter.Infof("Created IAM role '%s' with ARN '%s'", result.RoleName, result.RoleARN)
r.Reporter.Debugf("Trust policy attached for OIDC provider '%s'", req.OIDCProviderARN)
```

The core layer must not call reporter methods. If the core layer needs to
surface non-fatal warnings, return them as structured data in the Result
type (see [workflow-conventions.md](workflow-conventions.md#what-a-result-contains)).

## Review Prompts

- Does the core layer return all errors without calling `os.Exit` or
  reporter methods?
- Are errors wrapped with `%w` so the cause chain is preserved?
- Does the wrapping message describe what the current function was doing?
- Can the CLI layer distinguish error categories when it needs to show
  different messages or remediation guidance?
- Are exit codes decided by the CLI layer, not the core layer?
- Is remediation guidance in the CLI layer, not embedded in core errors?
