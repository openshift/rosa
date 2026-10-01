# Go Guidelines

Use this file for Go code and review. Follow nearby package patterns, and use
[command guidelines](command-guidelines.md) for Cobra-specific decisions and
[error conventions](error-conventions.md) for error behavior.

- Avoid variable shadowing, especially `err`, `ctx`, and AWS or OCM clients.
- Define interfaces when needed, preferably on the consumer side. Return
  concrete types unless the codebase already establishes an interface boundary.
- Do not store `context.Context` in structs.
- Distinguish nil from empty slices when behavior depends on the difference.
- Use named constants when a value matters to behavior or readability.
- Do not rely on map iteration order or goroutine scheduling in code or tests.
- Avoid `init()` except where registration patterns require it.
- Keep goroutines bounded and cancelable during polling, retries, and tests.
- Prefer small, focused functions and early returns over deep nesting.
- Document new exported types and functions with Go doc comments.
- Prefer concrete types or small interfaces over reflection or `any` when clearer.
- Use explicit names and nearby repository acronym casing, including `...Hcp`.
- Reuse standard-library or vendored implementations when they fit.
- Avoid unrelated import changes; change imports only when the task requires it.
