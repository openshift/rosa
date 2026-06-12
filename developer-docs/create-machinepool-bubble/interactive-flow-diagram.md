# `rosa create machinepool` — HCP interactive flowchart

Companion to [create-machinepool-interactive-flow.md](./create-machinepool-interactive-flow.md).  
**Branch:** `ROSAENG-4069-bubble-tea` · **Scope:** HCP (`CreateNodePools`) only.

## Diagram assumptions

| Assumption | Meaning |
|------------|---------|
| **100% interactive** | `-i` / `--interactive` on; no non-interactive shortcut path |
| **Flags unset** | No create flags pre-filled; every prompt runs |
| **Validation** | Any `GetString` / `GetInt` / `GetBool` / `GetOption` / `GetMultipleOptions` → invalid answer → **re-prompt same step** (not drawn per field) |
| **Classic** | Out of scope |

### Shape legend

| Shape | Meaning |
|-------|---------|
| `([...])` | Start / terminal (exit or success) |
| `[...]` | Survey prompt or system action |
| `{...}` | Decision (answer-driven or environment gate) |
| Subgraph | Phase grouping |

---

## Flowchart

```mermaid
flowchart TD
  START([rosa create machinepool -i --cluster HCP])

  subgraph P0["Phase 0 — Pre-checks"]
    START --> P0_CLUSTER{Valid cluster key<br/>and state ready?}
    P0_CLUSTER -->|No| X_P0([Exit])
    P0_CLUSTER -->|Yes| P0_AWS[Build AWS client]
    P0_AWS -->|Fail| X_P0
    P0_AWS --> P0_OCM[Load OCM autoscaler]
    P0_OCM -->|Fail| X_P0
  end

  P0_OCM --> P1

  subgraph P1["Identity"]
    P1["GetString: Machine pool name"]
    P1 --> P1_POST{Name regex OK?}
    P1_POST -->|No| X_NAME([Exit])
    P1_POST -->|Yes| FED{FedRAMP / GovCloud?}
  end

  subgraph P2["Image & version"]
    FED -->|No| P2_IMG["GetOption: Image Type"]
    FED -->|Yes| P2_VER_FETCH[Fetch & filter OpenShift versions]
    P2_IMG --> WIN{Windows LI selected?}
    WIN -->|No| P2_VER_FETCH
    WIN -->|Yes| WIN_TP{Tech preview active?}
    WIN_TP -->|No| X_WIN([Exit: Windows LI unavailable])
    WIN_TP -->|Yes| P2_WARN[Warn: Windows LI preview message]
    P2_WARN --> P2_VER_FETCH
    P2_VER_FETCH -->|Fail| X_VER_FETCH([Exit])
    P2_VER_FETCH --> P2_VER["GetOption: OpenShift version"]
    P2_VER --> P2_VAL{Version valid?}
    P2_VAL -->|No| X_VER([Exit])
    P2_VAL -->|Yes| NET
  end

  subgraph P3["Network — subnet / AZ"]
    NET["GetBool: Select subnet for hosted machine pool?"]
    NET --> NET_YES{User selects subnet?}
    NET_YES -->|Yes| NET_SUB["GetOption: Subnet ID"]
    NET_SUB --> NET_DONE[Subnet resolved]
    NET_YES -->|No| NET_AZ["GetOption: AWS availability zone"]
    NET_AZ -->|AWS error| X_NET([Exit])
    NET_AZ --> NET_COUNT{Subnets in AZ?}
    NET_COUNT -->|None| X_NET
    NET_COUNT -->|Exactly one| NET_AUTO[Use sole subnet]
    NET_COUNT -->|Multiple| NET_SUB
    NET_AUTO --> NET_DONE
  end

  subgraph P4["Scale — replicas"]
    NET_DONE --> P4_AS["GetBool: Enable autoscaling?"]
    P4_AS --> P4_BRANCH{Autoscaling enabled?}
    P4_BRANCH -->|Yes| P4_MIN["GetInt: Min replicas"]
    P4_MIN --> P4_MAX["GetInt: Max replicas"]
    P4_MAX --> P4_POST
    P4_BRANCH -->|No| P4_REP["GetInt: Replicas"]
    P4_REP --> P4_POST{Replica rules OK?}
    P4_POST -->|No| X_SCALE([Exit])
    P4_POST -->|Yes| P5_LBL
  end

  subgraph P5["Metadata — labels, taints, tags"]
    P5_LBL["GetString: Labels"]
    P5_LBL --> P5_TNT["GetString: Taints"]
    P5_TNT --> P5_PARSE{Parse labels/taints OK?}
    P5_PARSE -->|No| X_META([Exit])
    P5_PARSE -->|Yes| P5_TAGS["GetString: Tags"]
    P5_TAGS --> P6_FEAT
  end

  subgraph P6["Security groups"]
    P6_FEAT{Version supports<br/>day-2 security groups?}
    P6_FEAT -->|No| P7_FETCH
    P6_FEAT -->|Yes| P6_SG_AVAIL{SGs available in VPC?}
    P6_SG_AVAIL -->|No| P7_FETCH
    P6_SG_AVAIL -->|Yes| P6_SG["GetMultipleOptions: Additional Machine Pool Security Group IDs"]
    P6_SG -->|Survey error| X_SG([Exit])
    P6_SG --> P7_FETCH
  end

  subgraph P7["Compute"]
    P7_FETCH[Spinner: fetch instance types<br/>filtered by image type & AZ]
    P7_FETCH -->|Fail| X_IT([Exit])
    P7_FETCH --> P7_IT["GetOption: Instance type"]
    P7_IT --> P7_VAL{Instance type valid?}
    P7_VAL -->|No| X_IT
    P7_VAL -->|Yes| P7_AR["GetBool: Autorepair"]
    P7_AR --> P8_TUN_AVAIL
  end

  subgraph P8["Platform configs"]
    P8_TUN_AVAIL{Tuning configs<br/>on cluster?}
    P8_TUN_AVAIL -->|Yes| P8_TUN["GetMultipleOptions: Tuning configs"]
    P8_TUN_AVAIL -->|No| P8_FED
    P8_TUN --> P8_FED{FedRAMP?}
    P8_FED -->|Yes| P8_IMDS
    P8_FED -->|No| P8_CAP_ID["GetString: Capacity Reservation ID"]
    P8_CAP_ID --> P8_CAP_PREF{Reservation ID empty?}
    P8_CAP_PREF -->|Yes| P8_PREF_FULL["GetOption: Capacity preference<br/>none / only / open"]
    P8_CAP_PREF -->|No| P8_PREF_ID["GetOption: Capacity preference<br/>capacity-reservations-only"]
    P8_PREF_FULL --> P8_CAP_VAL
    P8_PREF_ID --> P8_CAP_VAL{Preference valid?}
    P8_CAP_VAL -->|No| X_CAP([Exit])
    P8_CAP_VAL -->|Yes| P8_KUBE_AVAIL{Kubelet configs<br/>on cluster?}
    P8_KUBE_AVAIL -->|No| P8_IMDS
    P8_KUBE_AVAIL -->|Yes| P8_KUBE["GetMultipleOptions: Kubelet config<br/>max 1 selection"]
    P8_KUBE --> P8_KUBE_VAL{Kubelet rules OK?}
    P8_KUBE_VAL -->|No| X_KUBE([Exit])
    P8_KUBE_VAL -->|Yes| P8_IMDS
    P8_IMDS["GetOption: IMDSv2 optional / required"]
    P8_IMDS --> P8_IMDS_VAL{IMDS value valid?}
    P8_IMDS_VAL -->|No| X_IMDS([Exit])
    P8_IMDS_VAL -->|Yes| P9_DISK
  end

  subgraph P9["Storage & upgrades"]
    P9_DISK["GetString: Root disk size GiB/TiB"]
    P9_DISK --> P9_DISK_VAL{Disk size valid?}
    P9_DISK_VAL -->|No| X_DISK([Exit])
    P9_DISK_VAL -->|Yes| P9_DRAIN["GetString: Node drain grace period"]
    P9_DRAIN --> P9_DRAIN_VAL{Drain period valid?}
    P9_DRAIN_VAL -->|No| X_DRAIN([Exit])
    P9_DRAIN_VAL -->|Yes| P9_SURGE["GetString: Max surge"]
    P9_SURGE --> P9_UNAVAIL["GetString: Max unavailable"]
    P9_UNAVAIL --> P10_INFO
  end

  subgraph P10["Create"]
    P10_INFO{Cluster autoscaler<br/>MaxNodesTotal set?}
    P10_INFO -->|Yes| P10_MSG[Info: replica vs autoscaler limits]
    P10_INFO -->|No| P10_API
    P10_MSG --> P10_API[OCM CreateNodePool]
    P10_API -->|Fail| X_API([Exit])
    P10_API -->|OK| OK([Success messages])
  end

  subgraph VALID["Validation note — applies to all Survey prompts"]
    direction LR
    V_NOTE["Invalid input → re-prompt same step<br/>Not shown on every edge above"]
  end

  style VALID fill:#f9f9f9,stroke:#999,stroke-dasharray: 5 5
  style X_P0 fill:#fee
  style X_NAME fill:#fee
  style X_WIN fill:#fee
  style X_VER_FETCH fill:#fee
  style X_VER fill:#fee
  style X_NET fill:#fee
  style X_SCALE fill:#fee
  style X_META fill:#fee
  style X_SG fill:#fee
  style X_IT fill:#fee
  style X_CAP fill:#fee
  style X_KUBE fill:#fee
  style X_IMDS fill:#fee
  style X_DISK fill:#fee
  style X_DRAIN fill:#fee
  style X_API fill:#fee
  style OK fill:#efe
```

---

## Reading guide

### Answer-driven forks (decision diamonds)

| Decision | Branches |
|----------|----------|
| **FedRAMP?** | Commercial → image type + capacity + kubelet prompts; GovCloud → skip those blocks |
| **Windows LI?** | Tech preview off → exit; on → warn and continue |
| **Select subnet?** | Yes → subnet list; No → AZ path |
| **Subnets in AZ?** | One → auto; Multiple → subnet pick; None → exit |
| **Autoscaling?** | Yes → min + max replicas; No → single replicas count |
| **Reservation ID empty?** | Changes capacity **preference** options |
| **Tuning / kubelet / SG on cluster?** | Data from OCM/AWS; skip multi-select if empty |
| **Version supports day-2 SGs?** | Skip security-group multi-select if unsupported |

### Linear stretches (no fork, same next step)

OpenShift version (after image branch), autorepair, IMDS, disk size, node drain, max surge → max unavailable, create.

### Order note vs numbered text doc

This diagram follows **`CreateNodePools` call order** in code: **labels, taints, and tags** run **before** instance type (security groups sit between taints and the instance-type fetch). The numbered list in the companion doc groups tags near the end for readability; the chart matches execution order for the Bubble Tea POC.

### Exit nodes (hard stop)

Pre-checks, name regex, Windows LI, version/API validation, network/AWS, replica post-check, label/taint parse, security-group helper, instance type, capacity preference, kubelet, IMDS, disk, drain, `CreateNodePool` API.

---

## Maintenance

Update when `pkg/machinepool/machinepool.go` (`CreateNodePools`) or helpers change. Keep in sync with [create-machinepool-interactive-flow.md](./create-machinepool-interactive-flow.md).
