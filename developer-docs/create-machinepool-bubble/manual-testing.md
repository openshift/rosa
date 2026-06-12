# Manual testing guide — machine pool demo commands

How to exercise the **fake** HCP golden-path demos:

| Command | UI |
|---------|-----|
| `rosa create machinepool-demo -i` | Survey (production primitive) |
| `rosa create machinepool-bubble -i` | Bubble Tea (migration POC) |

Both commands use the same fixtures and production validators. Neither calls OCM or AWS.

**Related docs:** [golden-path.md](./golden-path.md), [interactive-flow.md](./interactive-flow.md)

---

## Prerequisites

1. Build the CLI from **this branch** and run **that binary** — not an older `rosa` already on your `PATH`:

   ```bash
   cd /path/to/rosa   # your clone of ROSAENG-4069-bubble-tea (or equivalent)
   make rosa
   ./rosa create --help | grep machinepool-demo   # must show machinepool-demo
   ```

   If `rosa create` does not list `machinepool-demo` / `machinepool-bubble`, you are not running the build from this branch. Common cases:

   | Symptom | Cause |
   |---------|--------|
   | `unknown shorthand flag: 'i' in -i` | `-i` is being parsed by the parent `create` command because the subcommand does not exist in that binary |
   | No `machinepool-demo` in Available Commands | `which rosa` points at an installed release (e.g. `~/go/bin/rosa`), not `./rosa` |

   To use the demo build by default after `make rosa`:

   ```bash
   go install ./cmd/rosa    # installs to $GOBIN or ~/go/bin, overwriting the old binary
   ```

   Or invoke explicitly: `./rosa create machinepool-demo -i` from the repo root.

2. Use a real terminal (TTY). The Bubble Tea command exits with an error if stdout is not interactive.

3. No cluster, AWS profile, or OCM login is required.

---

## Quick smoke test

Run either command and accept defaults through all 24 prompts using the [golden path script](#golden-path-script-all-24-steps) below. Expect:

- No panics or hangs
- A final line similar to: `Machine pool '…' created successfully (demo dry run — no OCM or AWS calls were made)`
- A **Collected settings** summary listing your answers

Repeat with the other command and confirm the same answers produce the same summary shape.

---

## Golden path script (all 24 steps)

Use these answers for the **longest happy path**. Step order matches `pkg/machinepool/machinepool.go` (`CreateNodePools`).

| # | Prompt | Golden-path answer | Notes |
|---|--------|-------------------|-------|
| 1 | Machine pool name | `demo-pool` | Must match DNS-like name rules |
| 2 | Image Type | `default` | First option / default |
| 3 | OpenShift version | `4.16.0` | Default in fixtures |
| 4 | Select subnet for a hosted machine pool? | **No** | **Yes** re-prompts with golden-path message |
| 5 | AWS availability zone | `us-east-1a` | Either fixture AZ is fine |
| 6 | Subnet ID | `subnet-0demoaaa111111111` | Pick either listed subnet |
| 7 | Enable autoscaling? | **Yes** | **No** re-prompts with golden-path message |
| 8 | Min replicas | `2` | Default |
| 9 | Max replicas | `4` | Must be ≥ min |
| 10 | Labels | `demo/role=worker` | Or empty |
| 11 | Taints | `dedicated=workers:NoSchedule` | Or empty |
| 12 | Additional 'Machine Pool' Security Group IDs | `sg-0demo1111111111111` | Multi-select; optional to pick more |
| 13 | Tags | `Environment=demo,Team=rosa` | Or empty |
| 14 | Instance type | `m7i.xlarge` | First fixture type |
| 15 | Autorepair | `Yes` | Default |
| 16 | Tuning configs | `tuning-default` | Multi-select; optional second item |
| 17 | Capacity Reservation ID | *(empty)* | Enter / skip |
| 18 | Capacity Reservation Preference | `none` | Full list when ID is empty |
| 19 | Kubelet config | `kubelet-standard` | **At most one** selection |
| 20 | Configure the use of IMDSv2 | `optional` | Default |
| 21 | Root disk size (GiB or TiB) | `100 GiB` | Default |
| 22 | Node drain grace period | *(empty)* | Or `30 minutes` |
| 23 | Max surge | `1` | Default |
| 24 | Max unavailable | `0` | Default |

### Survey controls

- **Single select:** arrow keys + Enter
- **Multi-select:** space to toggle, Enter to confirm (can select zero or more)
- **Bool:** arrow to Yes/No + Enter
- **String/int:** type value + Enter; invalid values re-prompt on the same step

### Bubble Tea controls

- **Single select (list):** all options shown at once for small lists (same as Survey); cyan `>` on the focused row; ↑/↓ + Enter to choose
- **Optional lists (e.g. Image Type):** includes **Skip** and the same long prompt text as Survey (`optional…`, `default = '…'`); Skip stores an empty value
- **Filter (list):** start typing to filter (Survey-style), or press `/`; `Esc` clears the filter
- **Large lists:** if a step has more than 25 options, the list shows a scroll window (use ↑/↓); type to filter to narrow choices first
- **Multi-select:** ↑/↓ move, **space** toggle, Enter confirm
- **Bool (list):** ↑/↓ between Yes/No + Enter (both options always visible)
- **Text/int:** type + Enter; error stays on same step
- **Cancel:** `Ctrl+C` or `Esc` → `interactive input cancelled`
- **Completed answers:** each accepted answer is printed above the active prompt as `? Question: answer` (Survey-style scrollback via `tea.Println`)

---

## Golden-path branch enforcement

These are **demo-only** guardrails, not production behavior.

| Step | Wrong answer | Expected behavior |
|------|--------------|-------------------|
| 4 — Select subnet? | **Yes** | Message: *This demo follows the golden path: answer No to subnet selection…* then the same question again |
| 7 — Enable autoscaling? | **No** | Message: *This demo follows the golden path: enable autoscaling (answer Yes).* then the same question again |

Production `rosa create machinepool` does **not** loop on these branches; it continues with the user's choice.

---

## Validation spot checks

Use any step below to confirm **real validators** fire. After the error, fix the input on the **same** step.

### Machine pool name (step 1)

| Input | Expected |
|-------|----------|
| *(empty)* | Required / invalid name |
| `1bad` | Must start with a lowercase letter |
| `UPPER` | Invalid (uppercase not allowed) |
| `demo-pool` | Accept |

### Min / max replicas (steps 8–9)

| Input | Expected |
|-------|----------|
| Min `abc` | Not a valid integer |
| Min `-1` | Non-negative when autoscaling enabled |
| Min `10`, Max `5` | Max must be ≥ min |
| Min `501` | HCP limit: ≤ 500 |
| Min `2`, Max `4` | Accept |

### Labels (step 10)

| Input | Expected |
|-------|----------|
| `bad label=value` | Invalid label key |
| `demo/role=worker` | Accept |
| *(empty)* | Accept |

### Taints (step 11)

| Input | Expected |
|-------|----------|
| `not-a-taint` | Expected `key=value:effect` format |
| `key=value:BadEffect` | Unsupported effect |
| `dedicated=workers:NoSchedule` | Accept |
| *(empty)* | Accept |

### Tags (step 13)

| Input | Expected |
|-------|----------|
| `BadKey=value` | Invalid tag key |
| `duplicate=a,duplicate=b` | Duplicate key error |
| `Environment=demo` | Accept |
| *(empty)* | Accept |

### Kubelet config (step 19)

| Input | Expected |
|-------|----------|
| Two configs selected | *only a single kubelet config is supported* |
| One config | Accept |
| None | Accept |

### Capacity preference (step 18)

| Setup | Input | Expected |
|-------|-------|----------|
| ID empty | `open` | Accept |
| ID empty | `only` with no ID | Accept (preference-only without ID is valid in validator) |
| Enter ID `cr-12345` first | anything except `capacity-reservations-only` | Preference must be `capacity-reservations-only` when ID is set |

### Root disk size (step 21)

| Input | Expected |
|-------|----------|
| `not-a-size` | Parse / validation error |
| `50 GiB` | Accept (if within product limits) |
| `100 GiB` | Accept (default) |

### Node drain grace period (step 22)

| Input | Expected |
|-------|----------|
| `1 day` | Invalid unit (minutes/hours only) |
| `hour` | Not numeric |
| `30 minutes` | Accept |
| *(empty)* | Accept |

### Max surge / max unavailable (steps 23–24)

| Input | Expected |
|-------|----------|
| `not-valid` | Validation error |
| `1` / `0` | Accept (defaults) |
| `25%` | Accept if format valid per production rules |
| *(empty)* | Accept |

---

## Suggested test sessions

### Session A — Golden path GIF / demo recording

1. `rosa create machinepool-demo -i` — follow the [script](#golden-path-script-all-24-steps) exactly.
2. `rosa create machinepool-bubble -i` — same answers.
3. Compare: prompt order, multi-select UX, completed-answer summary (Bubble only), total time.

### Session B — Validation matrix

Pick **one** command; at each highlighted step above, deliberately enter the bad value, confirm the error, then correct it. No need to repeat on both UIs unless comparing error wording.

### Session C — Multi-select focus

On steps 12, 16, and 19:

- Select **zero** items → should complete (optional fields).
- Select **one** SG + one tuning + one kubelet → golden path summary shows parsed IDs/names.
- On kubelet, try selecting **two** → must fail before advancing.

### Session D — Cancel / resilience

Bubble Tea only:

1. Start `rosa create machinepool-bubble -i`, answer a few steps, press `Ctrl+C`.
2. Expect exit code 1 and `interactive input cancelled`.
3. Restart and complete the flow.

---

## What success looks like

```
Creating machine pool 'demo-pool' on cluster 'demo-hcp-cluster' (Survey demo dry run)
Machine pool 'demo-pool' created successfully (demo dry run — no OCM or AWS calls were made)
Collected settings:
  Name:              demo-pool
  ...
```

Confirm:

- Subnet shows parsed ID (`subnet-0demoaaa111111111`), not the full display string.
- Security groups show parsed `sg-…` IDs.
- Autoscaling shows `true (min 2, max 4)`.
- No network calls (can run offline).

---

## What is out of scope

- Automated unit/teatest files for these demos
- Real cluster create or `rosa create machinepool` parity
- Classic (non-HCP) machine pools
- FedRAMP / GovCloud prompts
- Non-interactive flags (only `-i` / `--interactive` is supported)

---

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| `machine pool bubble demo requires an interactive terminal` | Piping output or running in CI without TTY; use a local terminal |
| Stuck on subnet/autoscaling | Answered **Yes** / **No** on golden-path steps; switch to the required answer |
| Multi-select does nothing on Enter with no selection | Expected — empty selection is valid for optional multi-selects |
| Structure test failure after adding flags | Update `cmd/rosa/structure_test/command_args/rosa/create/machinepool-*/command_args.yml` |

---

## Maintenance

Update this guide when:

- `pkg/machinepooldemo/` fixtures or prompt order change
- `golden-path.md` step list changes
- New validators are wired into the demo runners
