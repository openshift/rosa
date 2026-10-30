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
| Stream | stdout (`Infof`), stderr (`Warnf`/`Errorf`) | stderr under `--debug`, discarded otherwise |
| Format | Plain text, optionally ANSI-colored | Structured key/value fields (`slog`) |
| Affected by `--output json/yaml`? | Yes: `output.StructuredReporter` reformats `Warnf`/`Errorf` as JSON | No, always the same stream and format, independent of `--output` |
| Levels | Info / Warn / Error, plus a global Debug gate (`--debug`) | Off entirely without `--debug`. With it, every level (`Debug` through `Error`) writes |
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

### Choosing A Level

Use OpenTelemetry's severity model as the reference for what each `pkg/log`
level means:

- **Debug**: detail useful only for troubleshooting a specific problem, not
  interesting on a normal run (`"ensuring IAM role"` before the call that
  might fail).
- **Info**: a normal operational event worth recording as part of the
  diagnostic trail, not a problem (`"ensured IAM role"` once the call
  succeeds).
- **Warn**: an unexpected condition the workflow recovered from on its own,
  processing continues.
- **Error**: an operation failed. The workflow still returns an `error` to
  its caller per [error-conventions.md](error-conventions.md). The log
  record is diagnostic detail alongside that returned error, not a
  substitute for it.

OpenTelemetry's model also defines `Fatal`, for failures the application
cannot continue from. `pkg/log` has no `Fatal` level, and will not: `slog`
has no built-in `Fatal` method, and more to the point, logging a message
and then terminating the process is a CLI-layer decision in this codebase,
not a log level. Core-layer code returns an `error`, `rosa.DefaultRunner`
reports it via `reporter.Errorf` and calls `os.Exit(1)`, see
[The DefaultRunner pattern](error-conventions.md#the-defaultrunner-pattern)
and [Exit codes](error-conventions.md#exit-codes). A `pkg/log` level that
called `os.Exit` itself would put process termination inside core-layer
logging code, exactly what [ARCHITECTURE.md](ARCHITECTURE.md#boundary-rules)
bans for `pkg/`. An `Error`-level log record and a returned `error` can
describe the same failure at the same moment: the record is diagnostic
detail, the returned `error` is what determines whether, and how, the CLI
layer exits.

## The `pkg/log` Package

`pkg/log` is a small package built directly on the standard library
`log/slog` (Go 1.26 in this module) rather than a new dependency or an
extension of `pkg/logging`'s `logrus` setup (which exists for a different
purpose, dumping raw outbound request and response traffic for the AWS SDK,
the OCM SDK, and other outbound HTTP clients, and is left unchanged. See
[Existing `logrus` usage](#existing-logrus-usage) and
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
the writer and the level, and is responsible for keeping structured logging
off by default:

```go
// pkg/rosa/runtime.go (CLI layer)
logWriter := io.Writer(io.Discard)
logLevel := slog.LevelInfo
if debug.Enabled() {
    logWriter = os.Stderr
    logLevel = slog.LevelDebug
}
structuredLogger := log.New(logWriter, logLevel)
```

Without `--debug`, `NewRuntime` builds the structured logger around
`io.Discard`: nothing is written anywhere, at any level, on a normal run.
This is deliberate. `pkg/log` is diagnostic output for support and
troubleshooting, not something every user should see by default just for
running a command, `reporter` already owns normal user-facing output. With
`--debug`, the logger writes every level, `Debug` through `Error`, to
`stderr`. There is currently no way to choose a level independently of
`--debug` (e.g. "show `Warn` and above without full `Debug` detail"), and no
way to write structured logs to a file instead of `stderr`. Both are real,
expected follow-up work, not introduced here, see
[Out of scope](#what-this-does-not-cover) below.

## Propagation Without Global CLI Coupling

The logger travels on `context.Context`, not as a parameter added to every
workflow and client interface. `rosa.DefaultRunner` attaches the root
logger once, already carrying a `command` field so every command gets it
for free instead of hand-typing its own command name:

```go
// pkg/rosa/runner.go
ctx := context.Background()
r := NewRuntime()
ctx = r.AttachLogger(ctx, command)
```

Every `pkg/` workflow function already takes `ctx` as its first parameter
(per [workflow-conventions.md](workflow-conventions.md#workflow-function-signatures)),
so any of them can retrieve the logger with `log.FromContext(ctx)` with no
signature changes. `FromContext` returns a discard logger when nothing was
attached, so a workflow called from a test, a future TUI, or a REST API
works with zero logging setup and no nil checks.

This happens automatically once a `cmd/` command is wired through
`rosa.DefaultRunner`, which calls `r.AttachLogger` for every such command
already. That wiring is not yet true repo-wide: commands still on the
legacy `Run: run` pattern (`func run(_ *cobra.Command, _ []string)`,
calling `rosa.NewRuntime()` directly) don't get a `ctx` with the logger
attached for free. A legacy command is not blocked from structured logging
though: it can call the same `r.AttachLogger(ctx, command)` `DefaultRunner`
calls internally, and pass that `ctx` into the workflow it calls, instead
of reconstructing the attach call by hand. Skipping this step, under either
pattern, means a workflow's `log.FromContext(ctx)` silently returns the
discard logger with no attached fields and no signal that logging is a
no-op. Converting a command to `DefaultRunner` is convenient but not
required, and either way is tracked separately from this document's
migration path, which covers `reporter.Logger` call sites, not runner
wiring.

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

`cmd/create/iamserviceaccount/cmd.go` attaches a `cluster` field to the
logger before calling the workflow (`command` is already on it, attached by
`DefaultRunner`):

```go
ctx = log.WithLogger(ctx, log.FromContext(ctx).With("cluster", cluster.Name()))

adapter := rolebridge.New(r.AWSClient, r.Reporter)
service := &iamserviceaccount.Service{}
result, err := service.CreateIAMServiceAccount(ctx, adapter, req)
```

The existing `r.Reporter.Infof(...)` calls immediately after this are
untouched. The person running the command sees exactly the same output as
before. On a normal run, `NewRuntime` builds the structured logger around
`io.Discard`, so none of the `DebugContext`/`InfoContext` calls below write
anywhere. With `--debug`, the structured logger writes to `stderr`
alongside `reporter`'s `Debugf` output, so the person running the command
with `--debug` sees both: `reporter`'s plain text debug lines on `stderr`,
and this workflow's structured records (`role`, `roleArn`, `policyArn`
fields and the rest) also on `stderr`, interleaved.

Inside `pkg/iamserviceaccount/create.go`, `Service.CreateIAMServiceAccount`
retrieves the logger from `ctx` and emits structured records at each phase,
with every record automatically carrying the `command` field `DefaultRunner`
attached and the `cluster` field attached above:

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
callers. Deleting `pkg/aws/rolebridge` needs all three of the IAM methods it
bridges to carry a `context.Context`, not just `EnsureRole`/`AttachRolePolicy`:
`PutRolePolicy` already takes neither a `reporter.Logger` nor a
`context.Context` today, so it needs a `ctx context.Context` parameter added
even though it has no `reporter.Logger` to remove. Only once `EnsureRole`
and `AttachRolePolicy` take `ctx` instead of `reporter.Logger`, and
`PutRolePolicy` takes `ctx` as well, does `aws.Client` satisfy a
`context.Context`-based `<Verb><Resource>Client` interface like
`CreateIAMServiceAccountClient` directly, and `rolebridge` can be deleted,
exactly the condition already noted in `rolebridge`'s package doc and in
[refactor/pkg-architecture.md](refactor/pkg-architecture.md#split-work-required).

## Existing `logrus` Usage

`logrus` is not banned. The policy going forward: `logrus` stays only for
dumping raw outbound request and response traffic, the wire-level detail of
what was sent to and received from a third-party API. `pkg/logging` builds
loggers for exactly that, used by both the AWS/OCM SDK client builders and,
through the same `pkg/logging.RoundTripperBuilder`, `pkg/version`'s HTTP
client for `mirror.openshift.com` version lookups. None of this is
SDK-specific, it is wire-dump logging generally, and `pkg/log` is not a
better fit for it: a wire dump is raw bytes and headers, not a structured
event with contextual fields, so converting it to `slog` would not gain
anything. Everywhere else, new diagnostic logging should use `pkg/log`, not
`logrus`, even inside a package that already imports `logrus` for one of the
uses below.

Existing `logrus` usage outside wire-dumping falls into two different
categories, with two different fixes:

- **`pkg/network/network.go` and `pkg/network/helper.go`** construct their
  own `logrus.Logger` and use it to report CloudFormation stack-creation
  progress and per-resource status, with ANSI color codes, printed by
  default since `logrus`'s default level is Info and these calls are never
  gated by `--debug`. This is user-facing output standing in for `reporter`,
  not diagnostic logging, so it is not a `pkg/log` migration candidate. It
  needs the same fix as a `reporter.Logger`-parameter call site under
  [User-facing progress](#migration-path-for-existing-reporter-usage) above:
  return the message as data from `pkg/network`'s core-layer functions and
  let `cmd/create/network/cmd.go` report it via `reporter`. This is actually
  an undocumented instance of the same problem the `reporter.Logger`-based
  call sites above already have, and arguably a less visible one: those at
  least take a `reporter.Logger` parameter, naming the CLI dependency at the
  call boundary, where `pkg/network` builds its own `logrus.Logger`
  internally with no parameter at all. Not performed as part of this change.
- **`pkg/version/retriever.go`** has direct `logger.Debugf` calls for
  genuine diagnostic detail (cache-set failures, available versions found)
  unrelated to any wire dump. This is a real `pkg/log` migration candidate.
  The catch: `RosaVersion.IsLatest` and `Retriever`'s methods do not take a
  `context.Context` today, so converting needs `ctx` threaded through
  `NewRosaVersion`, those interfaces, and their callers
  (`cmd/rosa/main.go`, `cmd/verify/rosa/cmd.go`,
  `pkg/commands/registry.go`) first. Future work, not performed here.

## What This Does Not Cover

- `pkg/logging` (the `logrus`-based factory for wire-level request and
  response dumping, used by the AWS SDK, the OCM SDK, and other outbound
  HTTP clients) is a separate concern and is unchanged by this document.
  See [Existing `logrus` usage](#existing-logrus-usage) above for where
  `logrus` remains and what is expected to migrate to `pkg/log`.
- No new `--log-level` or `--log-format`/`--log-file` flag is introduced.
  Structured logging today is a single on/off switch tied to `--debug`:
  off (writes to `io.Discard`) without it, every level to `stderr` with it.
  Choosing a level independently of `--debug` (e.g. `Warn` and above without
  full `Debug` detail), and routing structured logs to a file instead of
  `stderr`, are both real follow-up work this document does not implement.
- Converting `pkg/aws`, `pkg/ocm`, `pkg/policy`, or `pkg/helper` off
  `reporter.Logger` parameters is described above as future work, not
  performed as part of introducing `pkg/log`.
- Carrying partial progress (e.g. a role already created) alongside an error
  when a later step in the same workflow call fails. `CreateIAMServiceAccount`
  returns `nil` on error even after `EnsureRole` succeeds, folding the
  already-created role name into the wrapped error text instead
  (per [error-conventions.md](error-conventions.md#what-to-include-in-wrapped-errors)),
  not into a structured partial `Result`. A way to carry that progress as
  structured data on the error path, rather than only in error-message text,
  is a [workflow-conventions.md](workflow-conventions.md) Result-design
  question, not something this document introduces or resolves.
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
