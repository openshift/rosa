# ROSA CLI command paths

Reference for contributors and agents working in this repository.
Lists every supported `rosa` command path and interactive-mode flows ranked by prompt complexity.

## Scope and maintenance

- **Audience:** humans and agents changing CLI commands, structure tests, or interactive flows.
- **Authoritative contracts:** when commands or flags change, update `cmd/rosa/structure_test/command_structure.yml` and matching `command_args` files (see `AGENTS.md`).
- **This document:** inventory of paths and interactive complexity; regenerate when the command tree or interactive package changes materially.
- **Verification:** build the CLI (`make rosa`) and walk `./rosa <path> help`. Hidden commands are registered in Cobra with `Hidden: true` and do not appear in default help output.
- **Last generated:** 2026-06-08 from branch inventory scripts against the local `rosa` binary.

## Table 1 — All CLI command paths

**Total paths:** 157 (142 visible via default help + 15 hidden)

| Path | Visibility |
|------|------------|
| `rosa` | visible |
| `rosa attach` | hidden |
| `rosa attach policy` | hidden |
| `rosa completion` | visible |
| `rosa config` | visible |
| `rosa config get` | visible |
| `rosa config set` | visible |
| `rosa create` | visible |
| `rosa create account-roles` | visible |
| `rosa create admin` | visible |
| `rosa create autoscaler` | visible |
| `rosa create break-glass-credential` | visible |
| `rosa create cluster` | visible |
| `rosa create decision` | visible |
| `rosa create dns-domain` | visible |
| `rosa create external-auth-provider` | visible |
| `rosa create iamserviceaccount` | visible |
| `rosa create idp` | visible |
| `rosa create image-mirror` | visible |
| `rosa create kubeletconfig` | visible |
| `rosa create log-forwarder` | visible |
| `rosa create machinepool` | visible |
| `rosa create managed-service` | hidden |
| `rosa create network` | visible |
| `rosa create ocm-role` | visible |
| `rosa create oidc-config` | visible |
| `rosa create oidc-provider` | visible |
| `rosa create operator-roles` | visible |
| `rosa create tuning-configs` | visible |
| `rosa create user-role` | visible |
| `rosa delete` | visible |
| `rosa delete account-roles` | visible |
| `rosa delete admin` | visible |
| `rosa delete autoscaler` | visible |
| `rosa delete cluster` | visible |
| `rosa delete dns-domain` | visible |
| `rosa delete external-auth-provider` | visible |
| `rosa delete iamserviceaccount` | visible |
| `rosa delete idp` | visible |
| `rosa delete image-mirror` | visible |
| `rosa delete ingress` | visible |
| `rosa delete kubeletconfig` | visible |
| `rosa delete log-forwarder` | visible |
| `rosa delete machinepool` | visible |
| `rosa delete managed-service` | hidden |
| `rosa delete ocm-role` | visible |
| `rosa delete oidc-config` | visible |
| `rosa delete oidc-provider` | visible |
| `rosa delete operator-roles` | visible |
| `rosa delete tuning-configs` | visible |
| `rosa delete upgrade` | visible |
| `rosa delete user-role` | visible |
| `rosa describe` | visible |
| `rosa describe access-request` | visible |
| `rosa describe addon` | visible |
| `rosa describe addon-installation` | visible |
| `rosa describe admin` | visible |
| `rosa describe autoscaler` | visible |
| `rosa describe break-glass-credential` | visible |
| `rosa describe cluster` | visible |
| `rosa describe external-auth-provider` | visible |
| `rosa describe iamserviceaccount` | visible |
| `rosa describe ingress` | visible |
| `rosa describe kubeletconfig` | visible |
| `rosa describe log-forwarder` | visible |
| `rosa describe machinepool` | visible |
| `rosa describe managed-service` | hidden |
| `rosa describe tuning-configs` | visible |
| `rosa describe upgrade` | visible |
| `rosa detach` | hidden |
| `rosa detach policy` | hidden |
| `rosa docs` | hidden |
| `rosa download` | visible |
| `rosa download openshift-client` | visible |
| `rosa download rosa-client` | visible |
| `rosa edit` | visible |
| `rosa edit addon` | visible |
| `rosa edit autoscaler` | visible |
| `rosa edit cluster` | visible |
| `rosa edit image-mirror` | visible |
| `rosa edit ingress` | visible |
| `rosa edit kubeletconfig` | visible |
| `rosa edit log-forwarder` | visible |
| `rosa edit machinepool` | visible |
| `rosa edit managed-service` | hidden |
| `rosa edit tuning-configs` | visible |
| `rosa grant` | visible |
| `rosa grant user` | visible |
| `rosa hibernate` | hidden |
| `rosa hibernate cluster` | hidden |
| `rosa init` | visible |
| `rosa install` | visible |
| `rosa install addon` | visible |
| `rosa link` | visible |
| `rosa link ocm-role` | visible |
| `rosa link user-role` | visible |
| `rosa list` | visible |
| `rosa list access-request` | visible |
| `rosa list account-roles` | visible |
| `rosa list addons` | visible |
| `rosa list break-glass-credentials` | visible |
| `rosa list clusters` | visible |
| `rosa list dns-domain` | visible |
| `rosa list external-auth-providers` | visible |
| `rosa list gates` | visible |
| `rosa list iamserviceaccounts` | visible |
| `rosa list idps` | visible |
| `rosa list image-mirrors` | visible |
| `rosa list ingresses` | visible |
| `rosa list instance-types` | visible |
| `rosa list kubeletconfigs` | visible |
| `rosa list log-forwarders` | visible |
| `rosa list machinepools` | visible |
| `rosa list managed-services` | hidden |
| `rosa list ocm-roles` | visible |
| `rosa list oidc-config` | visible |
| `rosa list oidc-providers` | visible |
| `rosa list operator-roles` | visible |
| `rosa list regions` | visible |
| `rosa list rh-regions` | hidden |
| `rosa list tuning-configs` | visible |
| `rosa list upgrades` | visible |
| `rosa list user-roles` | visible |
| `rosa list users` | visible |
| `rosa list versions` | visible |
| `rosa login` | visible |
| `rosa logout` | visible |
| `rosa logs` | visible |
| `rosa logs install` | visible |
| `rosa logs uninstall` | visible |
| `rosa register` | visible |
| `rosa register oidc-config` | visible |
| `rosa resume` | hidden |
| `rosa resume cluster` | hidden |
| `rosa revoke` | visible |
| `rosa revoke break-glass-credentials` | visible |
| `rosa revoke user` | visible |
| `rosa token` | visible |
| `rosa uninstall` | visible |
| `rosa uninstall addon` | visible |
| `rosa unlink` | visible |
| `rosa unlink ocm-role` | visible |
| `rosa unlink user-role` | visible |
| `rosa upgrade` | visible |
| `rosa upgrade account-roles` | visible |
| `rosa upgrade cluster` | visible |
| `rosa upgrade machinepool` | visible |
| `rosa upgrade operator-roles` | visible |
| `rosa upgrade roles` | visible |
| `rosa verify` | visible |
| `rosa verify network` | visible |
| `rosa verify openshift-client` | visible |
| `rosa verify permissions` | visible |
| `rosa verify quota` | visible |
| `rosa verify rosa-client` | visible |
| `rosa version` | visible |
| `rosa whoami` | visible |

### Notes

- `delete` is the user-facing name; implementation lives under `cmd/dlt/`.
- `managed-service` commands are hidden but registered and reachable.
- `attach`, `detach`, `hibernate`, and `resume` are hidden top-level groups.
- Structure tests may use slightly different naming in a few places; prefer live CLI behavior and structure-test YAML when they diverge.

## Table 2 — Interactive mode commands (by complexity)

Commands that use `-i` / `--interactive`, or auto-enable interactive in specific cases.
Sorted by static Survey prompt call-site count (higher = more complex).

**Methodology:** count `interactive.Get*` call sites in each command's Go sources, including shared `pkg/` helpers on that command's code path. Runtime prompt count is often lower because many prompts are conditional (cluster type, flags already set, FedRAMP, etc.).

| Rank | Command path | Prompt call sites | Notes |
|------|--------------|-------------------|-------|
| 1 | `rosa create cluster` | 86 | Up to ~90 prompts at runtime; many are conditional on STS/HCP/VPC/proxy/ingress choices. |
| 2 | `rosa create machinepool` | 45 | Classic vs HCP paths differ; autoscaling/tuning/kubelet prompts vary. |
| 3 | `rosa edit machinepool` | 45 | Shares machinepool interactive helpers with create. |
| 4 | `rosa edit cluster` | 25 | Many prompts gated on cluster state and flags already set. |
| 5 | `rosa create autoscaler` | 24 | GPU limit loop can add extra prompts. |
| 6 | `rosa edit autoscaler` | 24 | Same autoscaler prompt set as create. |
| 7 | `rosa create idp — ldap` | 12 | Includes shared idp prompts (type, name, mapping method). |
| 8 | `rosa create idp — openid` | 12 | Includes shared idp prompts. |
| 9 | `rosa create idp — github` | 10 | Org/team restriction prompts. |
| 10 | `rosa create account-roles` | 9 | — |
| 11 | `rosa create external-auth-provider` | 9 | — |
| 12 | `rosa create operator-roles` | 9 | — |
| 13 | `rosa create idp — gitlab` | 7 | — |
| 14 | `rosa create idp — htpasswd` | 7 | Loop for adding multiple users. |
| 15 | `rosa create log-forwarder` | 7 | CloudWatch / S3 / both branches. |
| 16 | `rosa edit ingress` | 7 | Can auto-enable interactive; classic-only fields. |
| 17 | `rosa edit log-forwarder` | 7 | — |
| 18 | `rosa create idp — google` | 6 | — |
| 19 | `rosa create ocm-role` | 6 | — |
| 20 | `rosa upgrade cluster` | 6 | Optional installer-role selection when multiple roles exist. |
| 21 | `rosa delete iamserviceaccount` | 5 | — |
| 22 | `rosa create user-role` | 4 | — |
| 23 | `rosa describe iamserviceaccount` | 3 | — |
| 24 | `rosa register oidc-config` | 3 | — |
| 25 | `rosa upgrade machinepool` | 3 | — |
| 26 | `rosa create break-glass-credential` | 2 | — |
| 27 | `rosa create kubeletconfig` | 2 | — |
| 28 | `rosa create oidc-config` | 2 | — |
| 29 | `rosa create tuning-configs` | 2 | — |
| 30 | `rosa delete account-roles` | 2 | — |
| 31 | `rosa delete ocm-role` | 2 | — |
| 32 | `rosa delete oidc-config` | 2 | — |
| 33 | `rosa delete user-role` | 2 | — |
| 34 | `rosa edit kubeletconfig` | 2 | — |
| 35 | `rosa install addon` | 2 | Plus variable addon-parameter prompts via GetAddonArgument. |
| 36 | `rosa list regions` | 2 | Auto-enables interactive when installer role ARN is missing. |
| 37 | `rosa attach policy` | 1 | Hidden command. |
| 38 | `rosa create oidc-provider` | 1 | — |
| 39 | `rosa delete image-mirror` | 1 | — |
| 40 | `rosa delete oidc-provider` | 1 | — |
| 41 | `rosa delete operator-roles` | 1 | — |
| 42 | `rosa detach policy` | 1 | Hidden command. |
| 43 | `rosa edit tuning-configs` | 1 | — |
| 44 | `rosa link ocm-role` | 1 | — |
| 45 | `rosa link user-role` | 1 | — |
| 46 | `rosa list instance-types` | 1 | Auto-enables interactive when installer role ARN is missing. |
| 47 | `rosa list operator-roles` | 1 | — |
| 48 | `rosa login` | 1 | Password prompt when token is not supplied. |
| 49 | `rosa unlink ocm-role` | 1 | — |
| 50 | `rosa unlink user-role` | 1 | — |
| 51 | `rosa upgrade account-roles` | 1 | Mode prompt only. |
| 52 | `rosa upgrade operator-roles` | 1 | Mode prompt only. |
| 53 | `rosa upgrade roles` | 1 | Mode prompt only. |
| 54 | `rosa create iamserviceaccount` | 0 | Has --interactive flag but no Survey prompts in the create path today. |
| 55 | `rosa edit addon` | variable | Variable: 0..N prompts depending on addon parameters. |

### Parent interactive flags

| Parent command | Flag | Applies to |
|----------------|------|------------|
| `rosa edit` | `-i` / `--interactive` | All edit subcommands listed above |
| `rosa upgrade` | `-i` / `--interactive` | All upgrade subcommands listed above |
| `rosa install` | `-i` / `--interactive` | `rosa install addon` |

### Edit subcommands with `-i` but no Survey prompts today

- `rosa edit image-mirror`
- `rosa edit managed-service` (hidden)
- `rosa edit addon` — prompts only when the addon exposes unset parameters

### Survey prompt types in use

| Type | Helper | Typical use |
|------|--------|-------------|
| Text input | `GetString` | Names, ARNs, labels, proxy URLs |
| Integer | `GetInt` | Replicas, autoscaler limits |
| Float | `GetFloat` | Autoscaler utilization threshold |
| Boolean | `GetBool` | Yes/no toggles |
| Single select | `GetOption` | Version, region, billing account |
| Multi select | `GetMultipleOptions` | Subnets, security groups, tuning configs |
| CIDR | `GetIPNet` | Machine/service/pod CIDR |
| Password | `GetPassword` | Secrets, login token |
| Certificate path | `GetCert` | CA / trust bundle files |
| Mode | `GetOptionMode` | `auto` vs `manual` AWS changes |

## Related files

- `cmd/rosa/structure_test/command_structure.yml` — command tree contract
- `cmd/rosa/structure_test/command_args/**/command_args.yml` — flag contracts
- `pkg/interactive/` — Survey wrapper and validators
- `guidelines/command-guidelines.md` — command authoring expectations
