# `rosa create machinepool` — interactive mode flow (HCP demo)

Reference for the Bubble Tea demo (`ROSAENG-4069`). Describes the **current Survey-based** interactive behavior on branch `ROSAENG-4069-bubble-tea`.

**Demo scope:** **Hosted Control Plane (HCP) clusters only** (`cluster.Hypershift().Enabled()` → `CreateNodePools`).  
A separate **classic** machine-pool flow exists (`CreateMachinePool`); it is **not covered here**.

**Entry command**

```text
rosa create machinepool --cluster=<hcp-cluster> [--interactive | -i] [flags...]
```

**Source:** `cmd/create/machinepool/cmd.go` → `pkg/machinepool/machinepool.go` (`CreateNodePools`)

**Legend**

| Outcome | Meaning |
|---------|---------|
| **Re-prompt** | Survey validation failed; user stays on the same question |
| **Exit** | Command stops (returned error or `os.Exit(1)`) |
| **Skip** | Step not shown (flag already set or prerequisite not met) |

---

## Cluster routing (context only)

```text
Cluster Hypershift (HCP) enabled?
├── Yes → CreateNodePools   ← this document
└── No  → CreateMachinePool (classic — out of scope for demo)
```

Classic adds multi-AZ vs single-AZ prompts, spot instances, and different subnet/BYOVPC rules. The steps below marked **(common)** also exist on classic with similar Survey primitives.

---

## Interactive mode activation (common)

- User passes `--interactive` / `-i`, **or**
- Machine pool `--name` is empty → CLI prints `Enabling interactive mode` and turns interactive on automatically
- With `-i`, many prompts run **even when flags were already provided**

---

## Phase 0 — Before any Survey prompt (common + HCP)

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

---

## HCP interactive flow (`CreateNodePools`)

### 1. Machine pool name **(common)**

| | |
|---|---|
| **Prompt** | `GetString` — "Machine pool name" (required) |
| **Default** | `--name` if set |
| **Validation** | Regex `^[a-z]([-a-z0-9]*[a-z0-9])?$` |
| **Fails** | **Re-prompt** (Survey validator) |
| **After prompt** | Re-check regex → **Exit** if still invalid |

**Skip prompt:** only if `--name` set **and** interactive mode off.

---

### 2. Image type *(HCP only)*

**If** interactive **and** not GovCloud:

| Prompt | `GetOption` — "Image Type" (optional; default / Windows) |
|--------|----------------------------------------------------------|

| After selection | Outcome |
|-----------------|---------|
| Invalid type | **Exit** |
| Windows + tech preview not active | **Exit** |
| Windows + preview active | **Warn** with preview message; continue |

**Skip:** FedRAMP/GovCloud.

---

### 3. OpenShift version *(HCP only)*

**If** `--version` set **or** interactive:

| Step | What happens |
|------|----------------|
| 3a | Fetch version list; filter to allowed node-pool range |
| 3b | Interactive: `GetOption` — "OpenShift version" (required) |
| 3c | `ValidateVersion` against filtered list |

| Validation fails | Outcome |
|------------------|---------|
| Survey | **Re-prompt** |
| Version not in allowed range | **Exit** |

---

### 4. Subnet selection *(HCP)*

`getSubnetFromUser` **always** runs on HCP:

| Step | Prompt |
|------|--------|
| 4a | `GetBool` — "Select subnet for a hosted machine pool" (skip bool if `--subnet` set) |
| 4b | If yes → `GetOption` — "Subnet ID" (from VPC private subnets) |

| Validation fails | Outcome |
|------------------|---------|
| Invalid answer | **Re-prompt** |
| AWS / fetch error | **Exit** |

---

### 5. Availability zone → subnet *(HCP only, if no subnet from step 4)*

| Step | Prompt / action |
|------|-----------------|
| 5a | `GetOption` — "AWS availability zone" (from VPC private subnets) |
| 5b | If multiple subnets in that AZ → info message + return to **step 4** subnet pick |
| 5c | No private subnet for AZ | **Exit** |

Classic uses a different subnet/AZ tree (multi-AZ bool, BYOVPC rules); not in demo scope.

---

### 6. Replicas / autoscaling **(common)**

| Step | Interactive condition | Prompt |
|------|----------------------|--------|
| 6a | No `--replicas`, autoscaling off, `--enable-autoscaling` not set | `GetBool` — "Enable autoscaling" |
| 6b | Autoscaling **enabled** | `GetInt` — "Min replicas" (if `-i` or `--min-replicas` unset) |
| 6c | Autoscaling **enabled** | `GetInt` — "Max replicas" (if `-i` or `--max-replicas` unset) |
| 6d | Autoscaling **disabled** | `GetInt` — "Replicas" (if `-i` or `--replicas` unset) |

| Validation | Outcome |
|------------|---------|
| Survey int validators (HCP max 500 nodes, min ≤ max, etc.) | **Re-prompt** |
| `--replicas` set while autoscaling enabled | **Exit** |
| `--min-replicas` / `--max-replicas` without autoscaling | **Exit** |
| Post-prompt validator failure | **Exit** |

Classic adds multi-AZ “multiple of 3” replica rules; HCP does not.

---

### 7. Additional security groups **(common primitive, HCP gating)**

**If** interactive **and** `--additional-security-group-ids` not set **and** node-pool version supports additional day-2 security groups:

| Prompt | `GetMultipleOptions` — "Additional 'Machine Pool' Security Group IDs" (optional; AWS VPC list) |
|--------|--------------------------------------------------------------------------------------------------|

| Validation fails | Outcome |
|------------------|---------|
| Survey error | **Exit** (`os.Exit(1)` in helper) |
| No SGs available | **Skip** (empty selection) |

Classic requires BYOVPC for this prompt; HCP uses version feature support instead.

---

### 8. Instance type **(common)**

| Step | What happens |
|------|----------------|
| 8a | Spinner + OCM/AWS fetch of instance types (AZ from cluster or selected subnet) |
| 8b | Interactive: `GetOption` — "Instance type" (required; long list) |

HCP filters the list by **image type** (e.g. Windows LI).

| Validation fails | Outcome |
|------------------|---------|
| Survey / empty selection | **Re-prompt** or **Exit** |
| Type not valid for cluster | **Exit** |
| Non-interactive and empty instance type | **Exit** |

---

### 9. Labels and taints **(common)**

| Prompt | Type | Validation fails |
|--------|------|------------------|
| "Labels" | `GetString` — comma-separated `key=value` | **Re-prompt**; parse error → **Exit** |
| "Taints" | `GetString` — comma-separated `key=value:Effect` | **Re-prompt**; parse error → **Exit** |

---

### 10. Autorepair *(HCP only)*

| Prompt | `GetBool` — "Autorepair" (default: true) |
|--------|------------------------------------------|
| Fails | **Re-prompt** |

---

### 11. Tuning configs *(HCP only)*

**If** interactive **and** cluster has tuning configs in OCM:

| Prompt | `GetMultipleOptions` — "Tuning configs" (optional) |
|--------|-----------------------------------------------------|

**Skip** if none available (warn only if `--tuning-configs` had values).

---

### 12. Capacity reservation *(HCP only, not GovCloud)*

| Step | Prompt |
|------|--------|
| 12a | `GetString` — "Capacity Reservation ID" (optional) |
| 12b | `GetOption` — "Capacity Reservation Preference" (`none` / `capacity-reservations-only` / `open`; options depend on whether ID was set) |

| Validation fails | Outcome |
|------------------|---------|
| Preference vs ID rules | **Exit** |
| FedRAMP + reservation flags | **Exit** |

---

### 13. Kubelet configs *(HCP only, not GovCloud)*

**If** interactive **or** `--kubelet-configs` set:

| Step | Condition | Prompt |
|------|-----------|--------|
| 13a | Kubelet configs exist on cluster | `GetMultipleOptions` — "Kubelet config" (max **1** selected) |
| 13b | None available | **Warn**; ignore input |

| Validation fails | Outcome |
|------------------|---------|
| More than one selected | **Re-prompt** (validator) |
| Post-validation | **Exit** |

---

### 14. EC2 metadata / IMDSv2 *(HCP only)*

| Prompt | `GetOption` — "Configure the use of IMDSv2 for ec2 instances" (`optional` / `required`) |
|--------|----------------------------------------------------------------------------------------|
| Fails | **Re-prompt**; invalid value → **Exit** |

---

### 15. Root disk size **(common)**

**If** interactive **or** `--disk-size` set:

| Prompt | `GetString` — "Root disk size (GiB or TiB)" |
|--------|---------------------------------------------|
| Validator | Node-pool disk rules (HCP validator) |
| Fails during prompt | **Re-prompt** |
| Parse / validation after prompt | **Exit** |

---

### 16. Node drain grace period *(HCP only)*

| Prompt | `GetString` — "Node drain grace period" (optional; minutes/hours) |
|--------|-------------------------------------------------------------------|
| Fails during prompt | **Re-prompt** |
| Invalid after parse | **Exit** |

---

### 17. Upgrade strategy — max surge / unavailable *(HCP only)*

**If** interactive (or flags changed):

| Prompt | `GetString` — "Max surge" then "Max unavailable" |
|--------|---------------------------------------------------|
| Fails | **Re-prompt** |

Defaults from flags: surge `1`, unavailable `0`.

---

### 18. Cluster autoscaler info *(HCP only, non-blocking)*

If cluster autoscaler `MaxNodesTotal` is set, CLI may print **info** messages about replica totals. Does not block create.

---

### 19. AWS tags **(common)**

| Prompt | `GetString` — "Tags" (optional) |
|--------|----------------------------------|
| Validation fails | **Re-prompt**; tag parse/duplicate → **Exit** |

---

### 20. Create node pool *(HCP)*

| Step | Outcome |
|------|---------|
| OCM `CreateNodePool` | **Exit** on API error |
| Success | Info messages (or JSON with `-o json`) |

**No final yes/no confirm** on create.

Classic path calls `CreateMachinePool` instead; otherwise similar success/output pattern.

---

## HCP demo — prompt-type summary

| Survey primitive | HCP interactive steps |
|------------------|----------------------|
| `GetString` | Name, labels, taints, tags, capacity reservation ID, disk size, node drain, max surge/unavailable |
| `GetInt` | Replicas, min/max replicas |
| `GetBool` | Subnet selection, autoscaling, autorepair |
| `GetOption` | Image type, OpenShift version, subnet ID, AZ, instance type, IMDSv2, capacity preference |
| `GetMultipleOptions` | Security groups, tuning configs, kubelet config |

**Highest complexity for Bubble Tea POC (HCP):**

- Replica branching (**step 6**)
- Subnet / AZ tree (**steps 4–5**)
- Long instance-type select (**step 8**)
- **MultiSelect** — security groups, tuning configs, kubelet config (**steps 7, 11, 13**)

**Not in HCP demo (classic only):** multi-AZ machine pool bool, spot instances, spot max price.

---

## Maintenance

Update when `pkg/machinepool/machinepool.go` (`CreateNodePools`), `pkg/machinepool/helper.go`, or `pkg/helper/machinepools/helpers.go` change interactive behavior.
