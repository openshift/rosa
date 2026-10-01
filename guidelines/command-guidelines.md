# Command Guidelines

## Scope

Use this file when adding or changing Cobra commands, flags, help text, prompt flow, or command execution behavior under `cmd/rosa/`.

## Layering

- Keep Cobra-specific wiring in `cmd/` and move non-Cobra logic into `pkg/`.
- Start from the nearest comparable command area before introducing a new pattern; machinepool commands are the preferred reference for newer work.
- Keep user-facing output and prompt behavior aligned with the existing `output`, `reporter`, and `interactive` patterns.
- Prefer existing helpers and functions when they fit the task.

## Entrypoint And Exit Behavior

- Follow the entrypoint and exit pattern already used in the surrounding command area.
- Many existing commands still use `Run: run`, and some legacy areas still use direct `os.Exit()` calls; do not normalize a command area accidentally as part of an unrelated change.
- If a task intentionally changes the entrypoint or exit pattern, treat it as a scoped refactor and validate the surrounding command area together.
- Keep error handling aligned with the surrounding package, including reporter usage, exit behavior, and wrapped messages. See [error conventions](error-conventions.md).

## Flags And Command Contracts

- Update `cmd/rosa/structure_test/command_structure.yml` when the command tree changes.
- Update the matching `cmd/rosa/structure_test/command_args/**/command_args.yml` when supported flags change.
- If help output or generated CLI docs change, run `make generate-docs`.

## Platform API Dispatch

Each command that supports the Platform API (hyperfleet) path uses a `dispatch`
function as its Cobra `Run:` entrypoint. It checks `hyperfleet.Enabled()` and
routes to the v2 runner, when implemented, or the original v1 `run()` function.
Until a v2 runner exists, the enabled path reports an unsupported-command
error and exits.

When adding Platform API support to a command:

- Implement v2 logic in a separate `run_v2.go` file in the command package,
  with `runV2` as the entrypoint. Use `v2` rather than a product name in command
  file or function names.
- Replace the stub error in `dispatch.go` with a call to `runV2`.
- Keep the original `run()` function intact and keep Platform API conditionals
  out of it.
- Build the v2 runner's own runtime, for example with
  `NewRuntime().WithHyperFleet()`, without affecting the v1 path.

## Review Prompts

- Is the new logic really command wiring, or should it live in `pkg/`?
- Does the change preserve the existing output, prompt, and error-reporting behavior for this command area?
- Are structure tests, flag contracts, and generated docs still aligned with the command after the change?
