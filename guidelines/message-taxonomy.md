# Message And Output Taxonomy

## Scope

Use this file when deciding where a piece of output belongs: what a user
sees while a command runs, what a script or `--output json/yaml` consumer
receives, what goes to a log, and which layer is allowed to produce each
one. It names the categories the rest of the guidelines already imply:
[error-conventions.md](error-conventions.md) and
[workflow-conventions.md](workflow-conventions.md) each describe part of
this picture, and this document gives them one shared vocabulary. See
[ARCHITECTURE.md](ARCHITECTURE.md) for the CLI/core layer split this
taxonomy assumes, and [structured-logging.md](structured-logging.md) for
the `pkg/log` mechanism behind the Diagnostic logging category below.

## Principle

Every message a command produces belongs to exactly one of the categories
below, and each category has exactly one owning layer:

- **The core layer (`pkg/`) never produces user-facing output.** It returns
  data and errors (see [ARCHITECTURE.md](ARCHITECTURE.md#boundary-rules)). A
  workflow function cannot call `reporter.Infof`, build a display string,
  or decide what the user sees. The one exception is structured diagnostic
  logging via `pkg/log`, which a workflow may call directly: it carries no
  presentation concerns and is never part of the output contract (see
  [Diagnostic logging](#diagnostic-logging) below).
- **The CLI layer (`cmd/`, and `pkg/reporter`/`pkg/output`/`pkg/interactive`
  today, moving to `internal/cli/`) owns every other category below.** It
  decides what the person running the command sees, what a script sees,
  and what the `reporter.Debugf`/`pkg/logging` half of diagnostic logging
  captures.

`pkg/reporter`, `pkg/output`, and `pkg/interactive` physically live under
`pkg/` today, but they are CLI-layer packages, not core-layer packages.
See [ARCHITECTURE.md's current-state note](ARCHITECTURE.md#high-level-shape-current-state).
Their location does not change their ownership. A new workflow must still
treat them as off-limits imports per
[ARCHITECTURE.md's prohibited-imports list](ARCHITECTURE.md#import-direction).

## The Categories

| Category | Produced by | Mechanism | Stream | Returned, emitted, logged, or rendered? |
|----------|-------------|-----------|--------|------------------------------------------|
| Progress | CLI layer | `reporter.Infof` | stdout | Emitted through the `reporter.Logger` interface, during the operation |
| Final result (success message) | CLI layer | `reporter.Infof`, reading a workflow's `Result` | stdout | Emitted through `reporter.Logger`, once, after the workflow returns |
| Warning | CLI layer | `reporter.Warnf` | stderr | Emitted through `reporter.Logger`, auto-reformatted as JSON under `--output json/yaml` |
| Error | Core layer returns it, CLI layer presents it | `error` return value → `reporter.Errorf` | stderr | **Returned** by the core layer, **emitted** by the CLI layer, and terminates the process |
| Diagnostic logging | CLI layer (`reporter.Debugf`) and core layer (`pkg/log` for structured diagnostics, `pkg/logging` for SDK wire-level only) | `reporter.Debugf`, `pkg/log`'s `slog.Logger`, and `pkg/logging`'s `logrus` loggers | stdout (`Debugf`) / stderr (`pkg/log`) / logger-specific (SDK adapters) | **Logged**, gated by `--debug`, never part of the command's output contract |
| Machine-readable result | CLI layer | `output.Print(resource)` | stdout | **Rendered** from the workflow's `Result`/domain type, never from a `reporter` string |
| Interactive input | CLI layer | `pkg/interactive`, `pkg/interactive/confirm` | stdin/stdout | Solicited rather than produced, gated by `--interactive`/`-i` and `--yes`/`-y` |

The core layer participates in two rows: it returns the `error` value that
Error starts from, and it may emit `pkg/log` records directly for
Diagnostic logging, since those carry no presentation concerns. Every
other category is constructed, formatted, and emitted entirely by the CLI
layer, typically from the `Result` or domain type a workflow returned
(see [workflow-conventions.md#result-types](workflow-conventions.md#result-types)).

### Progress

A progress message tells the person watching the terminal that the
operation is still running: it announces something in-flight, not a
finished outcome. It uses `reporter.Infof`, usually inside a loop or before
a wait:

```go
// pkg/ocm/clusters.go
reporter.Infof("Waiting for cluster '%s' with the same creator ARN to start installing",
    pendingCluster.ID())
time.Sleep(30 * time.Second)
```

`pkg/helper/helpers.go`'s `DisplaySpinnerWithDelay` pairs a progress
message with a terminal-only spinner:

```go
func DisplaySpinnerWithDelay(reporter reporter.Logger, infoMessage string, delay time.Duration) {
    if reporter.IsTerminal() {
        spin := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
        reporter.Infof(infoMessage)
        spin.Start()
        time.Sleep(delay)
        spin.Stop()
    } else {
        time.Sleep(delay)
    }
}
```

The `Infof` call sits inside the same `IsTerminal()` branch as the spinner,
not outside it: on a piped or redirected stdout, `DisplaySpinnerWithDelay`
emits neither the spinner animation nor the message, it only sleeps. That
differs from a plain, unguarded `reporter.Infof` progress call like the one
above, which fires on every invocation regardless of `IsTerminal()`. See
[Interactive And Non-Interactive Execution](#interactive-and-non-interactive-execution)
below for how that split plays out across different call sites.

### Final result (success message)

A final message announces that the operation is done, usually built from
fields on the workflow's `Result`. It is the one-shot counterpart to a
progress message's repeated or in-flight updates:

```go
// cmd/create/cluster/cmd.go
if !output.HasFlag() || r.Reporter.IsTerminal() {
    r.Reporter.Infof("Cluster '%s' has been created.", clusterName)
    r.Reporter.Infof(
        "Once the cluster is installed you will need to add an Identity Provider " +
            "before you can login into the cluster. See 'rosa create idp --help' " +
            "for more information.")
}
```

```go
// cmd/create/iamserviceaccount/cmd.go
result, err := service.CreateIAMServiceAccount(ctx, adapter, req)
if err != nil {
    return err
}
r.Reporter.Infof("Created IAM role '%s' with ARN '%s' using OIDC '%s'",
    result.RoleName, result.RoleARN, oidcProviderARN)
```

A progress message can repeat, race against a timer, or interleave with a
spinner. A final message, when it fires, fires once rather than repeating,
after the workflow call returns successfully, and reads fields off the
returned `Result` rather than off the loop's local state. Whether it fires
at all for a given invocation can still be conditional, see
[Machine-Readable Output Is Not Built From Console
Messages](#machine-readable-output-is-not-built-from-console-messages) and
the [worked example](#worked-example-iamserviceaccount-create) below for
cases where a guard skips it entirely. Progress and final messages are both
`reporter.Infof`, so the distinction is structural (when it fires, and
whether it reads a finished `Result`), not mechanical: there is no
separate "final" method on `reporter.Logger`. Reviewers should be able to
point at the `Result`/domain value a final message reads from. A progress
message typically has none.

### Warning

A warning is a non-blocking advisory: execution continues past it. It uses
`reporter.Warnf`, which writes to stderr and does not return an `error`:

```go
// cmd/create/accountroles/creators.go
r.Reporter.Warnf("If policies created are not attached, or are missing, try re-running "+
    "\"rosa create account-roles\" with \"%s\"", forcePolicyCreationFlag)
```

The call site keeps going (it calls `EnsurePolicy` immediately afterward).
This is what separates a warning from an error below: a warning has no
return value for the caller to act on, so nothing in the language forces
the caller to stop.

The core layer must not call `reporter.Warnf`. If a workflow needs to
surface a recoverable condition, it returns structured data on its
`Result` instead. See
[workflow-conventions.md's `Warning` struct](workflow-conventions.md#what-a-result-contains),
and the CLI layer decides whether and how to render it as a warning.

### Error

An error is a terminal failure for the operation. The core layer **returns**
it. Only the CLI layer decides how to present it and whether to exit. The
canonical translation point is `rosa.DefaultRunner`:

```go
// pkg/rosa/runner.go
if err := runner(ctx, r, command, args); err != nil {
    r.Reporter.Errorf("%s", err)
    os.Exit(1)
}
```

`reporter.Errorf` both writes to stderr and returns an `error` of its own,
which is why it reads as a single call at legacy call sites that still
build their own exit path (`r.Reporter.Errorf(...); os.Exit(1)`, as seen in
`pkg/rosa/runtime.go`). New code should return the error and let
`DefaultRunner` perform that translation, per
[error-conventions.md](error-conventions.md#the-defaultrunner-pattern)
rather than calling `reporter.Errorf` and `os.Exit` directly.

Errors differ from warnings and diagnostic logs on one axis each:

- **Error versus warning**: an error stops the operation (a returned `error`
  the caller must handle). A warning does not (`Warnf` has no return
  value and the caller already decided to continue).
- **Error versus diagnostic log**: an error is always user-facing. The
  person running the command needs to know their operation failed. A
  diagnostic log entry is not part of the command's output contract at
  all. See [Diagnostic logging](#diagnostic-logging) below.

### Diagnostic logging

Diagnostic logging is internal detail: useful for troubleshooting, not
addressed to the person running the command. It is never part of a
command's output contract: it does not appear in `--output json/yaml`
payloads, and its presence or absence never changes a command's exit code
or displayed result. Today this takes three forms, all gated by `--debug`
(`debug.Enabled()`):

```go
// pkg/reporter/reporter.go
func (r *Object) Debugf(format string, args ...any) {
    if !debug.Enabled() {
        return
    }
    r.Infof(format, args...)
}
```

```go
// pkg/iamserviceaccount/create.go: core layer, structured
logger := log.FromContext(ctx)
logger.InfoContext(ctx, "ensured IAM role", "role", roleName, "roleArn", roleARN)
```

```go
// pkg/logging/logger.go
func NewLogger() (result *logrus.Logger) {
    result = logrus.New()
    result.SetFormatter(&logrus.TextFormatter{...})
    if debug.Enabled() {
        result.SetLevel(logrus.DebugLevel)
    }
    return result
}
```

`reporter.Debugf` writes plain text to stdout like `Infof`, just gated by
`--debug`. `pkg/log` is the structured, context-propagated counterpart for
workflow business logic: unlike `reporter`, it carries no CLI presentation
concerns, so a `pkg/` workflow may import it directly, writing structured
key/value records to stderr under `--debug` and to `io.Discard` otherwise.
`pkg/logging`'s `logrus`-based loggers feed the OCM and AWS SDK clients
their own wire-level debug output (`pkg/logging/ocm_logger.go`,
`pkg/logging/aws_logger.go`) and are unrelated to both: they exist to let
the SDKs log, not to let ROSA code log.

These three mechanisms sit in different layers. `reporter` (including
`Debugf`) is CLI-only and a prohibited import for `pkg/`
(see [ARCHITECTURE.md](ARCHITECTURE.md#import-direction)): a workflow
function must never call it. `pkg/log` and `pkg/logging`, by contrast, are
core-layer code. [pkg-architecture.md](refactor/pkg-architecture.md)
classifies `pkg/logging` as `internal/core/rosalog`, and `pkg/aws`,
`pkg/ocm`, and `pkg/version` already call
`logging.NewLogger()`/`logging.NewOCMLogger()` directly today to configure
SDK client loggers. It is not, however, a general-purpose diagnostic
logging facility for workflow business logic: it hands back a
`*logrus.Logger` meant for an SDK client builder, not a per-call logger
with contextual fields a workflow would reach for to record, say, "ensured
IAM role 'x' with ARN 'y'". `pkg/log` is that general-purpose facility: a
workflow retrieves it from `ctx` via `log.FromContext` rather than
constructing or receiving one. See
[structured-logging.md](structured-logging.md) for the full `pkg/log`
design, the `reporter`/`pkg/log` ownership split, and the
[migration path](structured-logging.md#migration-path-for-existing-reporter-usage)
off the `reporter.Logger` parameters some `pkg/` packages still take today.

### Machine-readable result

Structured output is a different code path from everything above it, not a
reformatting of it. `output.Print` takes the workflow's `Result` or domain
type directly and marshals it to JSON or YAML:

```go
// pkg/output/output.go
func Print(resource interface{}) error {
    var b bytes.Buffer
    switch reflect.TypeOf(resource).String() {
    case "*v1.Cluster":
        if cluster, ok := resource.(*cmv1.Cluster); ok {
            cmv1.MarshalCluster(cluster, &b)
        }
    // ...
    default:
        if err := defaultEncode(resource, &b); err != nil {
            return err
        }
    }
    // ...
    str, err := parseResource(b)
    fmt.Print(str)
    return nil
}
```

See [Machine-Readable Output Is Not Built From Console
Messages](#machine-readable-output-is-not-built-from-console-messages) for
why this must stay a separate path rather than a serialization of
`reporter` strings.

### Interactive input

`pkg/interactive` (prompts, `GetString`/`GetBool`/etc., gated by
`--interactive`/`-i`) and `pkg/interactive/confirm` (`--yes`/`-y`
confirmation prompts) solicit input rather than produce it, but they are
part of the same CLI-only ownership rule: the core layer must never read
`interactive.Enabled()` or call `confirm.Prompt`
(see [ARCHITECTURE.md's boundary rules](ARCHITECTURE.md#boundary-rules)). See
[Interactive And Non-Interactive Execution](#interactive-and-non-interactive-execution)
for how `-i` and `-y` interact with the rest of the taxonomy.

## Progress Messages Are Distinguishable From Final Results

The distinction is structural, not a different reporter method:

- A **progress** message fires during the operation, potentially more than
  once (inside a polling loop, before a spinner), and does not depend on a
  finished `Result`.
- A **final result** message, when execution reaches it, fires once per
  invocation rather than repeating, and is built from fields on the
  returned `Result` or domain value. Reaching it at all can be conditional:
  a guard like `output.HasFlag()` or a check on a `Result` field can skip
  it for a given invocation, the way each of
  `cmd/create/iamserviceaccount/cmd.go`'s two final messages is (see the
  [worked example](#worked-example-iamserviceaccount-create) below).

A reviewer can check this by asking: does this `Infof` call read from a
value the workflow just returned, or from state local to a loop that
hasn't finished yet? `cmd/create/iamserviceaccount/cmd.go`'s two `Infof`
calls both read `result.RoleName`/`result.InlinePolicyName` after
`service.CreateIAMServiceAccount` returns. They are final messages, with
no progress message in that command today.

## Warnings Are Distinguishable From Errors And Diagnostic Logs

Three different questions separate these three categories. A message
belongs to whichever category answers "yes" to its question:

1. **Does it stop the operation?** Yes → [Error](#error). No → continue to
   the next question. An error is a returned `error`. A warning is not.
2. **Is it addressed to the person running the command, or is it internal
   detail for troubleshooting?** Addressed to the user →
   [Warning](#warning). Internal detail → [Diagnostic
   logging](#diagnostic-logging).
3. A warning is visible by default (stderr, always on). A diagnostic log
   is visible only under `--debug`. If removing `--debug` would hide the
   message and nobody would notice during normal use, it is diagnostic,
   not a warning.

Mechanically: `Warnf` and `Errorf` both write to stderr, but only `Errorf`
returns an `error` the caller must handle, and only `Warnf`/`Errorf` get
the automatic JSON reformatting under `--output json/yaml` (see
[output.StructuredReporter](#machine-readable-output-is-not-built-from-console-messages)).
`Debugf` never gets that treatment: `StructuredReporter.Debugf` delegates
to the plain reporter unconditionally, so with `--debug` set, `Debugf`
still writes plain text to stdout even under `--output json/yaml`, where
it can appear interleaved with the structured payload rather than folded
into it. `--debug` is the only gate on `Debugf`; `--output json/yaml` is
not one.

## Machine-Readable Output Is Not Built From Console Messages

`--output json`/`--output yaml` is served by `output.Print`, which
marshals the workflow's `Result` or domain type directly, never by
capturing or re-parsing `reporter.Infof`/`Warnf`/`Errorf` text. The two
code paths are kept separate at the command level:

```go
// pattern used throughout cmd/describe/... and cmd/list/...
if output.HasFlag() {
    f, err := formatCluster(cluster, scheduledUpgrade, upgradeState, displayName, migrations)
    if err != nil {
        return err
    }
    return output.Print(f)
}
// else: build a human-readable table and print it
```

Warnings and errors are the one place the two paths do automatically
converge, through `output.StructuredReporter`, not because JSON output is
built from console text, but because both representations are derived
from the same underlying `error`/message at the point it is reported:

```go
// pkg/output/reporter.go
func (r *StructuredReporter) Errorf(format string, args ...any) error {
    err := fmt.Errorf(format, args...)
    if !PrintError(err) {
        return r.inner.Errorf(format, args...)
    }
    return err
}
```

`PrintError`/`PrintWarn` (`pkg/output/output.go`) check
`output.IsStructuredOutput()` first. When it's set, they print
`{"error": "..."}`/`{"warning": "..."}` to stderr and the plain text
`E:`/`W:`-prefixed form is suppressed. `Infof`/`Debugf` are never
restructured this way: progress, final, and diagnostic messages have no
JSON form at all, by design, since `--output json/yaml` describes the
*result*, not the narration around it. A command must never derive its
`--output json/yaml` payload by scraping or templating `reporter` strings.
It must build the payload from the `Result`/domain type and let
`output.Print` do the serialization.

## Interactive And Non-Interactive Execution

Three independent signals change how these categories behave, and a
command can see any combination of them:

- **`reporter.IsTerminal()`**: is stdout a terminal (not piped or
  redirected)? Different call sites gate different things on it.
  `DisplaySpinnerWithDelay` gates its message and its spinner together: a
  piped invocation gets neither, only the sleep. `cmd/create/cluster/cmd.go`'s
  `!output.HasFlag() || r.Reporter.IsTerminal()` guard is narrower: it
  skips its final messages only when `--output json/yaml` is also set and
  stdout is not a terminal, so a plain piped `rosa create cluster` (no
  `--output` flag) still gets its `Infof` progress/final text.
- **`interactive.Enabled()`** (`--interactive`/`-i`): does the command
  prompt for missing values at all? Off by default: a non-interactive
  invocation must supply every required value as a flag, since there is no
  prompt fallback.
- **`confirm.Yes()`** (`--yes`/`-y`): does a confirmation prompt
  auto-answer instead of blocking on stdin? `confirm.Prompt` checks `yes`
  before calling `survey.AskOne` at all, so `--yes` is what makes a
  destructive command usable in a script or CI job without a TTY attached.

A plain, unguarded `reporter.Infof`/`Warnf`/`Errorf` call (no `IsTerminal()`
or `output.HasFlag()` check wrapped around it) always fires on its stream
in both terminal and piped execution. `Debugf` behaves the same with
respect to terminal versus piped execution, but is gated internally by
`--debug` regardless (see [Diagnostic logging](#diagnostic-logging)
above): an unguarded `Debugf` call fires in both terminal and piped
execution when `--debug` is set, and not at all when it isn't. A call site
that *is* guarded by `IsTerminal()`/`output.HasFlag()` can be skipped
outright, not just reformatted: `DisplaySpinnerWithDelay`
skips its message whenever stdout is not a terminal, `cmd/create/cluster/cmd.go`
skips its final messages only when `--output json/yaml` is set and stdout
is not a terminal, and `cmd/create/iamserviceaccount/cmd.go` skips its
final messages whenever `--output json/yaml` is set at all, terminal or
not, because that branch returns before reaching them. Non-interactive
execution changes two independent things, not one, gated by two different
flags. First, fail-fast-versus-prompt is per field, not a blanket rule: it
only applies to a value a command explicitly wires behind
`interactive.Enabled()` plus an `interactive.Get*` call
(`pkg/options/iamserviceaccount/describe.go`'s
`validateAndPromptServiceAccountName` prompts when `-i` is set and the
value is missing, then falls through to a plain fail-fast error either
way if it's still empty). A value a command never wires that way, like
every required flag in `cmd/create/iamserviceaccount/cmd.go`, simply fails
fast regardless of `-i`; `-i` never enters into it. Second, and entirely
separately, a confirmation prompt is gated only by `-y`, regardless of
`-i`, and piped stdin is not a supported way to answer it: `confirm.Prompt`
discards `survey.AskOne`'s error, leaving its `response` variable at its
zero value (`false`), but `AskOne` still reads raw bytes from stdin
underneath regardless of whether it's a terminal. If stdin reaches EOF
immediately (closed, or redirected from `/dev/null`), that discarded error
produces a quiet "no". If stdin is a pipe that stays open without
producing input, the same read can block indefinitely instead, same as an
interactive terminal prompt would. A script or CI job that might hit a
confirmation prompt should always pass `--yes` rather than relying on
piped stdin to behave predictably.
When adding a new command, decide explicitly which category each message
belongs to, and whether, and under what exact condition, it should be
gated by `IsTerminal()` or `output.HasFlag()`. Don't assume "it works when
I ran it in my terminal" covers the piped, scripted, and `--output
json/yaml` cases.

## Worked Example: `iamserviceaccount create`

`pkg/iamserviceaccount/create.go` and `cmd/create/iamserviceaccount/cmd.go`
exercise every CLI-owned category except Progress, Warning, and
Interactive input in one place, with a clean Request/Result boundary in
between. Those three are absent because this command has no natural
occasion for them today, not because they were skipped:

```go
// pkg/iamserviceaccount/create.go: core layer, returns data and errors only
func (s *Service) CreateIAMServiceAccount(
    ctx context.Context,
    client CreateIAMServiceAccountClient,
    req CreateIAMServiceAccountRequest,
) (*CreateIAMServiceAccountResult, error) {
    if err := req.Validate(); err != nil {
        return nil, &rosaerrors.ValidationError{Err: fmt.Errorf("invalid request: %w", err)}
    }
    // ... EnsureRole / AttachRolePolicy / PutRolePolicy ...
    return &CreateIAMServiceAccountResult{
        RoleName:           roleName,
        RoleARN:            roleARN,
        AttachedPolicyARNs: req.PolicyARNs,
        InlinePolicyName:   inlinePolicyName,
    }, nil
}
```

```go
// cmd/create/iamserviceaccount/cmd.go: CLI layer, owns every message category
if len(userOptions.ServiceAccountNames) == 0 {
    _ = cmd.Usage()
    return &rosaerrors.ValidationError{ // Error: invocation validation, returned
        Field: "ServiceAccounts", Message: "at least one service account name is required",
    }
}
// ...
result, err := service.CreateIAMServiceAccount(ctx, adapter, req)
if err != nil {
    return err // Error: core-layer failure, returned up to DefaultRunner
}

if output.HasFlag() {
    if err := output.Print(result); err != nil { // Machine-readable result
        return fmt.Errorf("failed to print IAM service account result: %w", err)
    }
    return nil
}

r.Reporter.Infof("Created IAM role '%s' with ARN '%s' using OIDC '%s'", // Final result
    result.RoleName, result.RoleARN, oidcProviderARN)
if result.InlinePolicyName != "" {
    r.Reporter.Infof("Attached inline policy '%s' to role '%s'", result.InlinePolicyName, result.RoleName) // Final result
}
return nil
```

Reading this top to bottom against the taxonomy:

- **Error** (invocation validation): `cmd.go`'s flag checks return
  `&rosaerrors.ValidationError{...}` directly. No `reporter.Errorf` call
  happens in the core or CLI code. `rosa.DefaultRunner` performs the
  `Errorf`+`os.Exit(1)` translation once the `CommandRunner` returns.
- **Error** (core-layer failure): `service.CreateIAMServiceAccount`
  returns a plain wrapped error (`fmt.Errorf("failed to create role:
  %w", err)`) when `EnsureRole` fails. `cmd.go` returns it unchanged.
- **Final result**: both `r.Reporter.Infof` calls run only when execution
  reaches them, not unconditionally: `output.HasFlag()` returns before
  either one when `--output json/yaml` is set. Even when it isn't, the
  first call fires once, after `result` is populated, reading fields
  directly off it; the second is itself conditional
  (`if result.InlinePolicyName != ""`), so it fires only when an inline
  policy was actually requested. No progress message exists in this
  command because the operation completes synchronously with no wait loop.
- **Machine-readable result**: `cmd.go` checks `output.HasFlag()` right
  after `result` comes back and, when set, calls `output.Print(result)`
  and returns, skipping the `reporter.Infof` calls entirely. No change to
  `CreateIAMServiceAccount` was needed: `CreateIAMServiceAccountResult`
  was already structured enough to marshal directly, once its fields
  carried `json`/`yaml` tags. This is the same `output.HasFlag()`/
  `output.Print(result)` shape `cmd/create/spotterminationqueue/cmd.go`
  uses for its own `Result` type. Skipping the two `Infof` calls at the
  bottom of `cmd.go` is not sufficient by itself: `EnsureRole` and
  `AttachRolePolicy` (`pkg/aws/policies.go`) still take a `reporter.Logger`
  parameter directly and call `Infof` unconditionally on success, reached
  through `rolebridge` *before* `output.HasFlag()` is ever checked. `cmd.go`
  passes those two calls a `discardReporter{}` instead of `r.Reporter`
  whenever `output.HasFlag()` is true, so that legacy call path cannot
  print plain text to stdout ahead of the JSON/YAML result either. Adopting
  `--output json/yaml` on a command that still routes through a legacy
  `reporter.Logger`-parameter client must account for this, not just the
  command's own `Infof` calls.
- **Warning**: none in this command. A recoverable condition would need to
  come back as a `Warning` field on `CreateIAMServiceAccountResult` per
  [workflow-conventions.md](workflow-conventions.md#what-a-result-contains),
  not a `reporter.Warnf` call inside the workflow.
- **Interactive input**: also none, though `BuildIamServiceAccountCreateCommandWithOptions`
  registers `--interactive`/`-i` via `interactive.AddModeFlag`/`AddFlag`.
  Every value this command needs is a required flag; nothing in
  `CreateIamServiceAccountRunner` calls `interactive.GetString` or any
  other prompt to fill one in. `cmd/describe/iamserviceaccount/cmd.go`
  prompts for a missing service account name, namespace, or role name the
  same way a future interactive `create` flow would.
- **Diagnostic logging**: `CreateIAMServiceAccount` retrieves a logger via
  `log.FromContext(ctx)` and emits structured `pkg/log` records around each
  AWS call, alongside whatever `pkg/aws/rolebridge`'s underlying
  `aws.Client` calls may log internally under `--debug`. None of this is
  visible without `--debug`, and none of it changes the `reporter.Infof`
  calls below. See
  [structured-logging.md's walkthrough of this same command](structured-logging.md#representative-path-iamserviceaccount-create)
  for the exact fields and call sequence.

The `Result` type's own doc comment states the ownership rule this example
demonstrates: "The CLI layer uses this to format output for the user. The
result carries domain values only, not preformatted console messages."

## Review Prompts

- Does every message belong to exactly one category, and does its owning
  layer match this document (core layer never produces user-facing output
  and CLI layer owns every category except the `pkg/log` half of
  Diagnostic logging)?
- Is a progress message distinguishable from a final result because it
  fires during the operation rather than once, from a finished `Result`?
- Does a warning leave execution running, an error stop it, and a
  diagnostic log stay invisible without `--debug`, rather than any two of
  these blurring together?
- Is `--output json/yaml` built from the `Result`/domain type via
  `output.Print`, never from `reporter` strings?
- Does the command behave correctly when piped or run non-interactively:
  no silent prompt-block without `-i`, no destructive confirmation without
  `-y`, and no essential message silently dropped because it was gated on
  `IsTerminal()` by mistake?
