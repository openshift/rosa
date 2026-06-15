# HCP `create machinepool` — interactive flow and golden path

Reference for **`ROSAENG-4069`** and the machine-pool Bubble Tea POC on branch `ROSAENG-4069-bubble-tea`.

**Related docs**

- [README.md](../README.md) — index, POC videos (Survey parity vs UX-first)
- [survey-to-bubbletea-migration.md](./survey-to-bubbletea-migration.md) — how to migrate the next Survey command to Bubble Tea
- [../survey-bubbletea-migration-pm-brief.md](../survey-bubbletea-migration-pm-brief.md) — PM-level takeaway

**Source code:** `cmd/create/machinepool/cmd.go` → `pkg/machinepool/machinepool.go` (`CreateNodePools`).  
**Demo commands:** `rosa create machinepool-demo -i`, `machinepool-bubble -i`, `machinepool-bubble-new -i`.

**Legend**

| Outcome | Meaning |
|---------|---------|
| **Re-prompt** | Survey validation failed; user stays on the same question |
| **Exit** | Command stops (returned error or `os.Exit(1)`) |
| **Skip** | Step not shown (flag already set or prerequisite not met) |

**Demo scope:** **Hosted Control Plane (HCP) clusters only** (`cluster.Hypershift().Enabled()` → `CreateNodePools`). Classic machine pools (`CreateMachinePool`) are out of scope.

---

## What is the golden path?

The **golden path** is the longest **happy-path** walk through `rosa create machinepool -i` on an **HCP cluster** that:

- Hits **every Survey primitive** used in `CreateNodePools`
- Takes **every answer-driven branch that adds prompts** (not exits)
- Includes **all three** `GetMultipleOptions` steps when the cluster and AWS environment support them
- Ends in a successful **`CreateNodePool`** (or a faked success in the Bubble Tea POC)

It is the **stress-test scenario** for the machine-pool Bubble Tea demo—not the path most customers run daily.

### Why we use this path

| Reason | Detail |
|--------|--------|
| **Representative complexity** | Machine pool create is among the hardest interactive flows in ROSA (see `developer-docs/cli-paths.md`). |
| **MultiSelect coverage** | Survey `GetMultipleOptions` has **no Bubbles equivalent**. Security groups, tuning configs, and kubelet config are the main migration risks. |
| **Branching** | Subnet/AZ tree and autoscaling yes/no show that a single linear wizard is not enough. |
| **Input variety** | Exercises `GetString`, `GetOption`, `GetBool`, `GetInt`, and `GetMultipleOptions` in one run. |
| **Test design** | One documented scenario for manual runs, GIFs, and automated model/teatest tests. |

### Assumptions (golden-path script)

- **100% interactive** — `-i` / `--interactive`; no flags pre-filled
- **HCP only** — `CreateNodePools`
- **Commercial AWS** — not FedRAMP/GovCloud
- **Validation retries** — invalid answers re-prompt on the same step (not part of the golden-path script)

### Answer-driven choices (golden path)

| Decision | Choose | Effect |
|----------|--------|--------|
| FedRAMP / GovCloud? | **No** | Image type, capacity reservation, and kubelet prompts run |
| Image type | **Default** *(or Windows if LI preview is active)* | Same prompt count; Windows adds a warn only |
| Select subnet for hosted pool? | **No** | Routes through availability zone |
| Subnets in chosen AZ? | **Multiple** | Extra subnet `GetOption` after AZ |
| Enable autoscaling? | **Yes** | Min + max replicas (not single replica count) |
| Day-2 security groups supported? | **Yes** (cluster version) | Security group multi-select can run |
| SGs available in VPC? | **Yes** | **`GetMultipleOptions`: security groups** |
| Tuning configs on cluster? | **Yes** | **`GetMultipleOptions`: tuning configs** |
| Capacity Reservation ID | **Empty** | Full preference list (`none` / `only` / `open`) |
| Kubelet configs on cluster? | **Yes** | **`GetMultipleOptions`: kubelet config** (max one) |
| Cluster autoscaler `MaxNodesTotal` | **Set** *(optional)* | Info messages before create; not an extra prompt |

---

## Phase 0 — before any Survey prompt

| Step | What happens | Validation fails |
|------|----------------|------------------|
| Cluster key | `--cluster` resolved via OCM | **Exit** — invalid cluster key format |
| Cluster state | Cluster must be `ready` | **Exit** — cluster not ready |
| Labels flag | If `--labels` set, parse format | **Exit** |
| Image type flag | If `--type` set: HCP only, not GovCloud, valid type | **Exit** |
| AWS client | Built for cluster region | **Exit** — AWS client failure |
| OCM autoscaler | Loaded (info messages later) | **Exit** — OCM error |

**HCP flag pre-checks (before prompts)**

| Condition | Outcome |
|-----------|---------|
| `--multi-availability-zone` | **Exit** — not supported on HCP |
| Both `--subnet` and `--availability-zone` | **Exit** |

**Interactive mode activation**

- User passes `--interactive` / `-i`, **or**
- Machine pool `--name` is empty → CLI prints `Enabling interactive mode` and turns interactive on automatically
- With `-i`, many prompts run **even when flags were already provided**

---

## Production interactive flow (`CreateNodePools`)

Cluster routing:

```text
Cluster Hypershift (HCP) enabled?
├── Yes → CreateNodePools   ← this document
└── No  → CreateMachinePool (classic — out of scope)
```

Steps below follow **production** prompt order in `pkg/machinepool/machinepool.go`. Demo commands use the same steps and validators but **reorder the metadata block** — see [Demo golden path (24 steps)](#demo-golden-path-24-steps).

### 1. Machine pool name

| | |
|---|---|
| **Prompt** | `GetString` — "Machine pool name" (required) |
| **Validation** | Regex `^[a-z]([-a-z0-9]*[a-z0-9])?$` |
| **Fails** | **Re-prompt**; post-check → **Exit** if still invalid |

### 2. Image type *(HCP, not GovCloud)*

| Prompt | `GetOption` — "Image Type" (optional; default / Windows) |
|--------|----------------------------------------------------------|

| After selection | Outcome |
|-----------------|---------|
| Invalid type | **Exit** |
| Windows + tech preview not active | **Exit** |
| Windows + preview active | **Warn**; continue |

### 3. OpenShift version *(HCP)*

Fetch version list → `GetOption` — "OpenShift version" (required) → `ValidateVersion`. **Re-prompt** on Survey error; **Exit** if version not in allowed range.

### 4–5. Subnet / availability zone *(HCP)*

| Step | Prompt |
|------|--------|
| 4a | `GetBool` — "Select subnet for a hosted machine pool" |
| 4b | If yes → `GetOption` — "Subnet ID" |
| 5a | If no subnet yet → `GetOption` — "AWS availability zone" |
| 5b | Multiple subnets in AZ → info message + subnet pick |

Invalid answers **Re-prompt**; AWS/fetch errors **Exit**.

### 6. Replicas / autoscaling

| Condition | Prompt |
|-----------|--------|
| Autoscaling off, flags unset | `GetBool` — "Enable autoscaling" |
| Autoscaling **yes** | `GetInt` — "Min replicas", "Max replicas" |
| Autoscaling **no** | `GetInt` — "Replicas" |

HCP replica validators (max 500, min ≤ max, etc.): **Re-prompt** on failure.

### 7. Additional security groups

**If** interactive, flag unset, and cluster version supports day-2 security groups:

| Prompt | `GetMultipleOptions` — "Additional 'Machine Pool' Security Group IDs" (optional) |
|--------|-----------------------------------------------------------------------------------|

Survey error → **Exit**; no SGs in VPC → **Skip** (empty selection).

### 8. Instance type

Spinner + OCM/AWS fetch → `GetOption` — "Instance type" (required). HCP filters by image type. **Re-prompt** / **Exit** on invalid selection.

### 9. Labels and taints

| Prompt | Type | Validation fails |
|--------|------|------------------|
| "Labels" | `GetString` — `key=value` list | **Re-prompt**; parse error → **Exit** |
| "Taints" | `GetString` — `key=value:Effect` list | **Re-prompt**; parse error → **Exit** |

### 10. Autorepair *(HCP)*

`GetBool` — "Autorepair" (default: true). **Re-prompt** on failure.

### 11. Tuning configs *(HCP)*

**If** cluster has tuning configs in OCM: `GetMultipleOptions` — "Tuning configs" (optional). **Skip** if none available.

### 12. Capacity reservation *(HCP, not GovCloud)*

| Step | Prompt |
|------|--------|
| 12a | `GetString` — "Capacity Reservation ID" (optional) |
| 12b | `GetOption` — "Capacity Reservation Preference" |

Preference vs ID rules: **Exit** on violation.

### 13. Kubelet configs *(HCP, not GovCloud)*

`GetMultipleOptions` — "Kubelet config" (max **1**). More than one selected → **Re-prompt**; none on cluster → **Warn** and skip.

### 14. IMDSv2 *(HCP)*

`GetOption` — "Configure the use of IMDSv2 for ec2 instances". **Re-prompt** / **Exit** on invalid value.

### 15. Root disk size

`GetString` — "Root disk size (GiB or TiB)". **Re-prompt** during prompt; **Exit** after parse failure.

### 16. Node drain grace period *(HCP)*

`GetString` — "Node drain grace period" (optional). **Re-prompt** / **Exit** on invalid value.

### 17. Upgrade strategy *(HCP)*

`GetString` — "Max surge" then "Max unavailable". Defaults: surge `1`, unavailable `0`.

### 18. Cluster autoscaler info *(HCP, non-blocking)*

Info messages when `MaxNodesTotal` is set. Does not block create.

### 19. Tags

`GetString` — "Tags" (optional). **Re-prompt**; tag parse/duplicate → **Exit**.

### 20. Create node pool

OCM `CreateNodePool` → **Exit** on API error. **No final yes/no confirm.**

### Survey primitive summary (production)

| Primitive | HCP interactive steps |
|-----------|----------------------|
| `GetString` | Name, labels, taints, tags, capacity reservation ID, disk size, node drain, max surge/unavailable |
| `GetInt` | Replicas, min/max replicas |
| `GetBool` | Subnet selection, autoscaling, autorepair |
| `GetOption` | Image type, OpenShift version, subnet ID, AZ, instance type, IMDSv2, capacity preference |
| `GetMultipleOptions` | Security groups, tuning configs, kubelet config |

**Highest Bubble Tea migration complexity:** replica branching (step 6), subnet/AZ tree (4–5), long instance-type list (8), and **MultiSelect** (7, 11, 13).

---

## Demo golden path (24 steps)

Fixed script for `machinepool-demo`, `machinepool-bubble`, and `machinepool-bubble-new`. Same validators and fixtures; **prompt order differs** from production for steps 10–14 (metadata block grouped before instance type).

| # | Primitive | Prompt |
|---|-----------|--------|
| 1 | `GetString` | Machine pool name |
| 2 | `GetOption` | Image type |
| 3 | `GetOption` | OpenShift version |
| 4 | `GetBool` | Select subnet for hosted machine pool? → **No** |
| 5 | `GetOption` | AWS availability zone |
| 6 | `GetOption` | Subnet ID |
| 7 | `GetBool` | Enable autoscaling? → **Yes** |
| 8 | `GetInt` | Min replicas |
| 9 | `GetInt` | Max replicas |
| 10 | `GetString` | Labels |
| 11 | `GetString` | Taints |
| 12 | **`GetMultipleOptions`** | **Additional Machine Pool Security Group IDs** |
| 13 | `GetString` | Tags |
| 14 | `GetOption` | Instance type |
| 15 | `GetBool` | Autorepair |
| 16 | **`GetMultipleOptions`** | **Tuning configs** |
| 17 | `GetString` | Capacity Reservation ID |
| 18 | `GetOption` | Capacity Reservation Preference |
| 19 | **`GetMultipleOptions`** | **Kubelet config** |
| 20 | `GetOption` | IMDSv2 optional / required |
| 21 | `GetString` | Root disk size (GiB or TiB) |
| 22 | `GetString` | Node drain grace period |
| 23 | `GetString` | Max surge |
| 24 | `GetString` | Max unavailable |

Then: optional autoscaler info → faked create success in demo commands.

| Primitive | Count (demo) |
|-----------|----------------|
| `GetString` | 9 |
| `GetOption` | 7 |
| `GetBool` | 3 |
| `GetInt` | 2 |
| **`GetMultipleOptions`** | **3** |

### Demo vs production prompt order

| Block | Production (`CreateNodePools`) | Demo (`pkg/machinepooldemo`) |
|-------|-------------------------------|------------------------------|
| After autoscaling | Security groups → instance type → labels → taints → autorepair → … → tags (near end) | Labels → taints → security groups → tags → instance type → autorepair → … |

---

## Cluster and environment prerequisites

For the **full** golden path (all 24 demo prompts):

| Requirement | Purpose |
|-------------|---------|
| HCP cluster in **ready** state | Phase 0 pre-checks |
| **Commercial** region (not GovCloud) | Image type, capacity, kubelet blocks |
| OpenShift version supporting **day-2 security groups** on node pools | Step 12 multi-select |
| **Non-empty** attachable security groups in VPC | Step 12 multi-select |
| **Tuning configs** on cluster (OCM) | Step 16 multi-select |
| **Kubelet configs** on cluster (OCM) | Step 19 multi-select |
| VPC private subnets: AZ with **multiple** subnets | Steps 5–6 |
| Valid instance types for region/AZ/image type | Step 14 |
| AWS + OCM credentials working | Instance type fetch, create API |

If a prerequisite is missing, production **skips** the step; the demo may still show it with fixtures.

### Fallback: minimal MultiSelect demo

When tuning/kubelet gates are unavailable:

- Commercial HCP cluster with **day-2 security groups** and **SGs in VPC**
- **Skip** tuning (step 16) and kubelet (step 19) when OCM lists are empty
- Still use **autoscaling yes** and **subnet no → AZ**

~21 prompts; enough to demo MultiSelect without full cluster prep.

---

## Bubble Tea POC

| Command | UX | Recording |
|---------|-----|-----------|
| `rosa create machinepool-bubble -i` | Survey-parity styling and transcript | [poc-survey-parity.mp4](./videos/poc-survey-parity.mp4) |
| `rosa create machinepool-bubble-new -i` | UX-first Charm styling, progress header, `bubbles/list` checkbox delegate for multi-select | [poc-ux-first.mp4](./videos/poc-ux-first.mp4) |

Shared behavior:

1. **Same demo question order** as `pkg/machinepooldemo` (24 steps above).
2. **Completed answers visible** — Bubble Tea uses scrollback (`tea.Println` or in-view summary).
3. **Fake create** — no real `CreateNodePool`; dry-run success summary only.
4. **Golden-path guardrails** (demo only) — subnet **Yes** and autoscaling **No** re-prompt with a message.

### Native vs workaround (`machinepool-bubble`)

**Canonical classification table** — update whenever demo behavior changes.

**Source files:** `pkg/interactive/bubbletea/machinepoolbubble/`

| Area | Classification | Implementation | teatest note |
|------|----------------|----------------|--------------|
| Single select | **Native** | `bubbles/list` + `surveySelectDelegate` | ↑/↓, Enter, `/`, type-to-filter |
| Text / int | **Native** | `bubbles/textinput` | type + Enter |
| Multi-select (×3) | **Workaround** | Custom `multiSelectModel` | **space** + Enter |
| Prior answers | **Light workaround** | `tea.Println` transcript | Native API; layout differs from Survey |
| Type-to-filter without `/` | **Light workaround** | `maybeStartTypeToFilter` | List filter is native underneath |
| Golden-path re-prompts | **Demo-only** | Demo runners | Not production |

### Native vs workaround (`machinepool-bubble-new`)

**Source files:** `pkg/interactive/bubbletea/machinepoolbubblenew/`

| Area | Classification | Implementation |
|------|----------------|----------------|
| Single select | **Native** | `bubbles/list` + `NewDefaultDelegate` |
| Text / int | **Native** | `bubbles/textinput` |
| Multi-select (×3) | **Native delegate** | `bubbles/list` + checkbox `ItemDelegate` (space toggle, `/` filter) |
| Progress / summary | **Native** | lipgloss + in-view completed summary |

---

## What we are not proving

- Identical Survey UX (layout and TUI differ by design on UX-first command)
- Classic machine pools (`CreateMachinePool`)
- FedRAMP/GovCloud interactive behavior
- Non-interactive flag-only creates
- Edit or delete machine pool flows

---

## Maintenance

- **Native vs workaround tables** — keep in sync with `machinepool-bubble` and `machinepool-bubble-new` changes.
- **24-step demo script** — update when `pkg/machinepooldemo` or demo wizards change.
- **Production flow** — update when `pkg/machinepool/machinepool.go` (`CreateNodePools`) changes.
- **Migration how-to** — [survey-to-bubbletea-migration.md](./survey-to-bubbletea-migration.md).
