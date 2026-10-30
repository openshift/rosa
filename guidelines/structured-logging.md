# Structured Logging Conventions

## Scope

Use this file when adding diagnostic logging to `pkg/` or `internal/core/`
code, or when deciding whether a message belongs in `reporter` output or in
a structured log record. It defines the relationship between `pkg/reporter`
(user-facing terminal output) and `pkg/log` (structured diagnostic logging),
and the migration path from the former to the latter. See
[ARCHITECTURE.md](ARCHITECTURE.md) for the CLI/core layer split,
[error-conventions.md](error-conventions.md) for how errors and recoverable
`reporter` messages are handled today, [workflow-conventions.md](workflow-conventions.md)
for the Request/Result/Client conventions structured logging must not
disturb, and [refactor/pkg-architecture.md](refactor/pkg-architecture.md)
for the per-package classification this document's new package fits into.

## Principle

`reporter` and structured logging serve two different audiences and neither
replaces the other:

- **`reporter`** talks to the person running the command. It is part of the
  command's user-facing output contract: what a human sees on their
  terminal, or what a script sees on stdout with `--output json`.
- **The structured logger (`pkg/log`)** records diagnostic detail: what the
  program did internally, with identifiers and fields attached, for
  troubleshooting, support, and future observability tooling. It is never
  part of a command's output contract.

This document introduces structured logging as **additive**, not as a
`reporter` replacement. Both mechanisms coexist while the CLI/core-layer
migration in `ARCHITECTURE.md` is in progress. There is no flag day where
`reporter` calls stop working.

## Relationship Between Logging And Reporter Output

| | `reporter.Logger` | `pkg/log` |
|---|---|---|
| Audience | The person running the command | Diagnostics: support, logs, future tooling |
| Stream | stdout (`Infof`), stderr (`Warnf`/`Errorf`) | Always stderr |
| Format | Plain text, optionally ANSI-colored | Structured key/value fields (`slog`) |
| Affected by `--output json/yaml`? | Yes: `output.StructuredReporter` reformats `Warnf`/`Errorf` as JSON | No, always the same stream and format, independent of `--output` |
| Levels | Info / Warn / Error, plus a global Debug gate (`--debug`) | Debug / Info / Warn / Error per record |
| Structured fields | No, printf-style strings only | Yes, key/value pairs, with contextual fields via `.With(...)` |
| Importable from `pkg/`? | No, banned by `ARCHITECTURE.md`'s core-layer import rules (enforced by `depguard`) | Yes, carries no CLI presentation concerns |
| Propagation | Explicit `reporter.Logger` parameter (see [Propagation](#propagation-without-global-cli-coupling) for why this is what structured logging avoids) | Carried on `context.Context`, already threaded through every workflow signature |

Because the two write to different streams for different purposes, a single
operation can emit both without conflict: `reporter.Infof` announces
"Created IAM role 'x'" for the person watching the terminal, while the
structured logger records the same event with fields (`role`, `roleArn`,
`policyArn`) for troubleshooting (see
[Representative path](#representative-path-iamserviceaccount-create) below).

Diagnostic logs are never treated as user-facing command output: they never
go to stdout, are never included in `--output json`/`--output yaml`
payloads, and their presence or absence never changes a command's exit code
or displayed result.

## The `pkg/log` Package

`pkg/log` is a small package built directly on the standard library
`log/slog` (Go 1.26 in this module) rather than a new dependency or an
extension of `pkg/logging`'s `logrus` setup (which exists for a different
purpose, configuring AWS/OCM SDK debug loggers, and is left unchanged. See
[Out of scope](#what-this-does-not-cover) below). `slog.Logger` already
provides levels, structured fields, and a pluggable `Handler` for testing,
so `pkg/log` does not wrap it in a new interface:

```go
// pkg/log
func New(w io.Writer, level slog.Level) *slog.Logger
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context
func FromContext(ctx context.Context) *slog.Logger
```

`New` builds the root logger. It takes the output writer and level as plain
values rather than reading them from anywhere. `pkg/log` must not import
`pkg/debug` or any other CLI-layer package (the same core-layer import rule
that excludes `pkg/reporter` from `pkg/`, see
[ARCHITECTURE.md](ARCHITECTURE.md#boundary-rules)). The CLI layer decides
the level:

```go
// pkg/rosa/runtime.go (CLI layer)
logLevel := slog.LevelInfo
if debug.Enabled() {
    logLevel = slog.LevelDebug
}
structuredLogger := log.New(os.Stderr, logLevel)
```

## Propagation Without Global CLI Coupling

The logger travels on `context.Context`, not as a parameter added to every
workflow and client interface. `rosa.DefaultRunner` attaches the root logger
once:

```go
// pkg/rosa/runner.go
ctx := context.Background()
r := NewRuntime()
ctx = log.WithLogger(ctx, r.StructuredLogger)
```

Every `pkg/` workflow function already takes `ctx` as its first parameter
(per [workflow-conventions.md](workflow-conventions.md#workflow-function-signatures)),
so any of them can retrieve the logger with `log.FromContext(ctx)` with no
signature changes. `FromContext` returns a discard logger when nothing was
attached, so a workflow called from a test, a future TUI, or a REST API
works with zero logging setup and no nil checks.

This only holds once a `cmd/` command is wired through `rosa.DefaultRunner`,
which is not yet true repo-wide: commands still on the legacy `Run: run`
pattern (`func run(_ *cobra.Command, _ []string)`, calling
`rosa.NewRuntime()` directly) have no `ctx` at all, so a workflow they call
gets `FromContext`'s discard logger with no attached fields and no signal
that logging is silently a no-op. Converting a command to `DefaultRunner` is
a prerequisite for that command's workflows to emit structured logs, and is
tracked separately from this document's migration path, which covers
`reporter.Logger` call sites, not runner wiring.

This is what "supplied to reusable workflows without global CLI coupling"
means in practice, and it is a deliberate contrast with the pattern
`reporter.Logger` uses today. `pkg/aws`'s `EnsureRole` and `AttachRolePolicy`
take a `reporter.Logger` parameter directly:

```go
// pkg/aws/policies.go (existing)
func (c *awsClient) EnsureRole(reporter reporter.Logger, name string, policy string, ...) (string, error)
```

Because `reporter.Logger` is banned from `pkg/` workflow packages, any
workflow that needs these methods through a `context.Context`-based client
interface (per
[workflow-conventions.md](workflow-conventions.md#client-interfaces-are-scoped-per-workflow-not-per-package))
cannot call them directly. It needs an adapter that holds a
`reporter.Logger` on the CLI's behalf. `pkg/aws/rolebridge` is exactly that
adapter, and its package doc says as much: it exists only to bridge this
mismatch, and should be deleted once `EnsureRole`/`AttachRolePolicy` stop
taking a `reporter.Logger`. `pkg/log` avoids ever needing this kind of
adapter for logging: because it has no CLI dependency, a `pkg/` client
interface can retrieve it from `ctx` internally, with no parameter and no
bridge required from any caller.

## Adding Structured Logs To A Workflow

```go
// Good: pull the logger from ctx, add fields relevant to the event.
func (s *Service) DoSomething(ctx context.Context, client DoSomethingClient, req DoSomethingRequest) (*Result, error) {
    logger := log.FromContext(ctx)
    logger.InfoContext(ctx, "ensured resource", "id", result.ID)
    // ...
}
```

```go
// Bad: a reporter.Logger parameter is a banned core-layer import and forces
// every caller to construct or adapt a CLI-specific reporter.
func DoSomething(reporter reporter.Logger, req DoSomethingRequest) (*Result, error)

// Bad: constructing a local logger inside pkg/ code bypasses the CLI
// layer's level/output configuration and duplicates pkg/logging's role.
func DoSomething(req DoSomethingRequest) (*Result, error) {
    logger := logrus.New()
    // ...
}
```

Prefer `InfoContext`/`DebugContext`/`WarnContext`/`ErrorContext` (the
context-aware `slog` methods) over the non-context variants, since `ctx` is
already available in every workflow signature and lets a future `Handler`
enrich records with request-scoped values (e.g. a trace ID) without any
caller changes.

## Representative Path: `iamserviceaccount create`

`cmd/create/iamserviceaccount/cmd.go` attaches contextual fields, the
cluster name and command, to the logger before calling the workflow:

```go
ctx = log.WithLogger(ctx, log.FromContext(ctx).With(
    "cluster", cluster.Name(),
    "command", "create iamserviceaccount",
))

adapter := rolebridge.New(r.AWSClient, r.Reporter)
service := &iamserviceaccount.Service{}
result, err := service.CreateIAMServiceAccount(ctx, adapter, req)
```

The existing `r.Reporter.Infof(...)` calls immediately after this are
untouched. The person running the command still sees exactly the same
output as before.

Inside `pkg/iamserviceaccount/create.go`, `Service.CreateIAMServiceAccount`
retrieves the logger from `ctx` and emits structured records at each phase,
with every record automatically carrying the `cluster` and `command` fields
attached above:

```go
logger := log.FromContext(ctx)
// ...
logger.DebugContext(ctx, "ensuring IAM role", "role", roleName, "path", path)
roleARN, err := client.EnsureRole(ctx, roleName, trustPolicy, ...)
// ...
logger.InfoContext(ctx, "ensured IAM role", "role", roleName, "roleArn", roleARN)
// ...
logger.InfoContext(ctx, "attached managed policy", "role", roleName, "policyArn", policyARN)
// ...
logger.InfoContext(ctx, "attached inline policy", "role", roleName, "policyName", inlinePolicyName)
```

`pkg/iamserviceaccount` imports `pkg/log` and nothing else new, no
`reporter`, no adapter, no CLI dependency. This is why this workflow was
picked as the representative path: it already followed the target
Request/Result/Client shape in `workflow-conventions.md`, so adding
structured logging required no signature or interface changes at all,
only pulling the logger already available on `ctx`.

These log calls bracket the AWS calls made through `client` (a
`pkg/aws/rolebridge.Client`) at the workflow layer. The `cluster` and
`command` fields land on every record above. They do not reach inside
`aws.Client`'s `EnsureRole`/`AttachRolePolicy`/`PutRolePolicy` themselves:
`rolebridge.Client` discards the `context.Context` it's given (see that
package's doc comment), so nothing propagates from `ctx` into the AWS SDK
call boundary. This is an existing limitation of the bridge, not something
this change attempts to fix. It closes on its own once `rolebridge` is
deleted per the [migration path](#migration-path-for-existing-reporter-usage)
below, at which point `EnsureRole`/`AttachRolePolicy` take a real
`context.Context` and can log from inside `pkg/aws` with the same
propagated fields.

## Migration Path For Existing Reporter Usage

Several `pkg/` packages already call `reporter.Logger` methods directly,
a "Needs split" legacy pattern documented in
[refactor/pkg-architecture.md](refactor/pkg-architecture.md): `pkg/aws/policies.go`
(`EnsureRole`, `createRole`, `AttachRolePolicy`), `pkg/ocm/helpers.go`
(`ValidateOperatorRolesMatchOidcProvider`), `pkg/policy/policy_service.go`
(`AutoAttachArbitraryPolicy`), `pkg/helper/helpers.go`
(`DisplaySpinnerWithDelay`), and `pkg/rosa/runtime.go`'s own `Debugf` calls
inside `FetchCluster`. None of these are touched by this change. Converting
them is future work, done incrementally, package by package, not as a
single rewrite. `reporter.Logger` parameters and `pkg/log` context
propagation are expected to coexist until that migration is complete.

When converting one of these call sites, classify each message first:

1. **User-facing progress**: the person running the command should see
   this (e.g. "Created IAM role"). Move the call up to the CLI layer, or
   return the information as structured data on the workflow's Result type
   so the CLI layer can report it, per
   [error-conventions.md](error-conventions.md#reporter-for-recoverable-messages).
   Do not convert this kind of message into a log record: it would stop
   being visible to the user.
2. **Diagnostic detail**: internal branching, identifiers, timings, or
   anything primarily useful for troubleshooting rather than for the
   person watching the command run. Convert it to `log.FromContext(ctx)`
   plus fields, following the [pattern above](#adding-structured-logs-to-a-workflow).

Once every call site on a given method is converted, drop the
`reporter.Logger` parameter from that method's signature and update its
callers. For `pkg/aws`'s `EnsureRole`/`AttachRolePolicy` specifically, this
also removes the only reason `pkg/aws/rolebridge` exists: once those methods
take a `context.Context` and no longer require a `reporter.Logger`, any
workflow's `<Verb><Resource>Client` interface is satisfied by `aws.Client`
directly, and `rolebridge` can be deleted, exactly the condition already
noted in `rolebridge`'s package doc and in
[refactor/pkg-architecture.md](refactor/pkg-architecture.md#split-work-required).

## What This Does Not Cover

- `pkg/logging` (the `logrus`-based factory for AWS/OCM SDK debug loggers)
  is a separate concern (SDK wire-level debug output) and is unchanged by
  this document.
- No new `--log-level` or `--log-format` flag is introduced. The structured
  logger's level today follows the same `--debug` gate `reporter.Debugf`
  already uses.
- Converting `pkg/aws`, `pkg/ocm`, `pkg/policy`, or `pkg/helper` off
  `reporter.Logger` parameters is described above as future work, not
  performed as part of introducing `pkg/log`.
- Naming cleanup, not performed here: `reporter.Logger` should become
  `reporter.Reporter` (see the `TODO` on that interface), since "Logger" on
  a type meant for user-facing output invites exactly the confusion this
  document exists to resolve. Similarly, `Runtime.Logger` (the `logrus`
  logger for AWS/OCM SDK wire-level debug output) should become
  `Runtime.SdkLogger`, freeing the plain `Runtime.Logger` name for what is
  currently `Runtime.StructuredLogger` (see the `TODO` in
  `pkg/rosa/runtime.go`), so the struct reads as `Reporter`, `Logger`,
  `SdkLogger` rather than `Reporter`, `Logger`, `StructuredLogger`.

## Review Prompts

- Does the message belong to the person running the command (`reporter`) or
  is it diagnostic detail (`pkg/log`)? Don't move a user-facing message into
  a log record just because it's convenient.
- Does the new or modified `pkg/` code retrieve its logger from `ctx` via
  `log.FromContext`, rather than taking a `reporter.Logger` parameter or
  constructing its own logger?
- Do structured log calls attach the fields a reader would need to find and
  correlate the record (resource identifiers), without embedding large
  payloads or secrets? The same diagnostic-context rule
  [error-conventions.md](error-conventions.md#what-to-include-in-wrapped-errors)
  applies to errors.
- Does the change avoid writing structured logs to stdout, and avoid
  letting `--output json`/`--output yaml` payloads depend on whether
  logging occurred?
- If this change removes the last `reporter.Logger`-parameter call site on
  an `aws.Client` method that `pkg/aws/rolebridge` bridges, has the bridge
  been deleted too?
