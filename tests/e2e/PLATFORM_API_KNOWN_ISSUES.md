# Platform API v2 Known Issues

This document tracks known Platform API v2 bugs that cause test failures and are labeled for skipping.

## List/Describe Inconsistency for Node Pools

**Status:** Known Issue - Labeled with `seaplane-known-issue`  
**Bug Filed:** [Pending]  
**Affects:** Platform API v2 (Seaplane)

### Description

The Platform API v2 has an inconsistency between the `list machinepool` and `describe machinepool` endpoints:

- `rosa list machinepool` returns node pools with their IDs and shows them as `Ready`
- `rosa describe machinepool <id>` immediately returns "Node pool not found" for the same ID

This persists even after retries with backoff (tested up to 3 retries over 6 seconds), indicating it's not an eventual consistency issue but a backend bug.

### Evidence

```bash
# List shows node pool exists
$ rosa list machinepool -c fd8182d6-7b7a-4831-a3af-b8af0b8d5aaa
ID                                    NAME       STATE
b10c0033-95ec-4695-92a8-01051579c671  workers-1  Ready

# Describe says not found (even after retries)
$ rosa describe machinepool b10c0033-95ec-4695-92a8-01051579c671 -c fd8182d6-7b7a-4831-a3af-b8af0b8d5aaa
ERR: Node pool 'b10c0033-95ec-4695-92a8-01051579c671' not found
```

Log evidence: `tests/output/rosa-seaplane-basic/failed-by-id-20261110-172215.log`

### Affected Tests

Tests labeled with `labels.Seaplane.KnownIssue`:

| Test ID | Test Name | File | Line |
|---------|-----------|------|------|
| id:66359 | volume size test | test_rosacli_cluster_post.go | 82 |
| id:75150 | compute_machine_type test | test_rosacli_cluster_post.go | 493 |

These tests:
1. Call `rosa list machinepool` to get node pool list
2. Extract the first node pool ID
3. Call `rosa describe machinepool <id>` to get details
4. **FAIL** with "not found" error despite list showing it exists

### Retry Logic Implemented

The test framework includes retry logic in [`machinepool_service.go`](../utils/exec/rosacli/machinepool_service.go):

```go
// Platform API v2 list/describe operations may be eventually consistent.
// If describe returns "not found", retry a few times with backoff.
if err != nil && strings.Contains(output.String(), "not found") {
    maxRetries := 3
    for attempt := 1; attempt <= maxRetries; attempt++ {
        sleepDuration := time.Duration(attempt) * time.Second
        Logger.Infof("Node pool describe returned 'not found', retrying after %v (attempt %d/%d)",
            sleepDuration, attempt, maxRetries)
        time.Sleep(sleepDuration)
        // Retry describe...
    }
}
```

**However,** the retries do not resolve the issue, confirming it's a persistent backend bug.

### Workarounds

#### Option A: Skip Tests (Current Approach)

Tests are labeled with `labels.Seaplane.KnownIssue` and automatically skipped in seaplane mode:

```go
// In e2e_suite_test.go
var _ = BeforeEach(func() {
    if isSeaplaneMode() && slices.Contains(CurrentSpecReport().Labels(), "seaplane-known-issue") {
        Skip("This test has a known Platform API v2 issue")
    }
})
```

#### Option B: Use List Data Instead of Describe

Modify tests to extract all needed information from the `list machinepool` output instead of calling describe.

**Pros:** Tests can run  
**Cons:** List output may not have all the detail fields that describe provides

#### Option C: Wait for Platform API Fix

File a P1 bug with the Platform API team and wait for a backend fix.

### How to Skip Tests

When a test fails due to a known Platform API issue:

1. Add the label to the test:
   ```go
   It("test name - [id:xxxxx]",
       labels.Seaplane.Validated, labels.Seaplane.KnownIssue,  // Add this
       func() {
           // test code
       })
   ```

2. The test will be automatically skipped when running in seaplane mode

3. Document the issue in this file

### Running Tests

```bash
# Tests with seaplane-known-issue are automatically skipped
make e2e_test

# To run ONLY known-issue tests (for verification after fix):
ginkgo run --label-filter='seaplane-known-issue' ./tests/e2e/

# To run seaplane tests EXCLUDING known issues (default):
ginkgo run --label-filter='seaplane-validated && !seaplane-known-issue' ./tests/e2e/
```

### Reporting Bugs

When filing Platform API bugs, include:

1. **Cluster ID** and **Node Pool ID** that reproduce the issue
2. **Endpoint URL** (e.g., `https://ehuc28agui.execute-api.us-east-1.amazonaws.com/prod`)
3. **Log file** showing the list/describe inconsistency
4. **Retry evidence** showing it's not timing-related
5. Link to this document for test impact

### Removing Labels

When the Platform API team fixes an issue:

1. Verify the fix works by running the specific test
2. Remove `labels.Seaplane.KnownIssue` from the test
3. Update this document to mark the issue as resolved
4. Consider keeping retry logic as defensive programming

## List Endpoint Missing Labels and Taints

**Status:** Known Issue - Labeled with `seaplane-known-issue`  
**Bug Filed:** [Pending]  
**Affects:** Platform API v2 (Seaplane)

### Description

The Platform API v2 `rosa list machinepool` command does not return labels and taints in the output, even though they exist on the node pools. The `rosa describe machinepool` command correctly shows the labels and taints.

### Evidence

```bash
# List output shows EMPTY labels/taints columns:
$ rosa list machinepool -c 6b4d5607-7721-4e87-8a0d-6441f6b77f69
ID        NAME            LABELS  TAINTS
49c1e...  np-metadata-eh          replacement=new:PreferNoSchedule

# Describe shows them correctly:
$ rosa describe machinepool np-metadata-eh -c 6b4d5607-7721-4e87-8a0d-6441f6b77f69
Labels: empty=, original=value
Taints: original=value:NoSchedule, empty=:NoExecute
```

Log evidence: `tests/output/rosa-seaplane-basic/e2e-20261111-123729.log` lines 11985-11989, 11956

### Affected Tests

Tests labeled with `labels.Seaplane.KnownIssue`:

| Test | File | Line |
|------|------|------|
| labels/taints test | test_rosacli_node_pool.go | 79 |
| CRUD test [id:56782] | test_rosacli_node_pool.go | 139 |

These tests:
1. Create node pools with labels and taints
2. Call `rosa list machinepool` to get node pool list
3. Parse labels/taints from the list output
4. **FAIL** because list output has empty LABELS and TAINTS columns

### Workarounds

#### Option A: Skip Tests (Current Approach)

Tests are labeled with `labels.Seaplane.KnownIssue` and automatically skipped in seaplane mode.

#### Option B: Use Describe Instead of List

Modify tests to use `rosa describe machinepool` for each pool instead of parsing the list output.

**Pros:** Tests can run  
**Cons:** Slower (N API calls versus 1), changes test behavior

---

## DNS Domain Route Not Found (Resolved)

**Status:** Resolved - `seaplane-known-issue` label removed from id:65793

### Description

The Platform API v2 `rosa create dns-domain` command used to fail with `ROUTE-001: route not found`.

### Root Cause

The Platform API does not expose a `dns_domains` route. It exposes named DNS reservations on
`dns_reservations`, where the reservation name is the identifier and the allocated domain is
reported in `status.baseDomain` once the reservation is Ready.

### Fix

`pkg/seaplane/dnsdomains.go` now creates, lists, and deletes `dns_reservations` and maps each
reservation to the OCM-shaped `DNSDomain` view the CLI prints.

---

## Other Known Issues

(Add additional known issues here as they are discovered)
