// Package tags provides shared helpers for AWS resource tag maps used by ROSA HCP
// cluster and machine-pool (node pool) consumers (ROSA CLI, Terraform RHCS, CAPA).
//
// Validate and Diff operate on map[string]string only. CLI string parsing stays in
// the ROSA CLI. System/protected tags are omitted by the OCM API when GET/list is
// called with fetchUserTagsOnly=true; this package does not filter system tags.
//
// Phase 2 consumers should depend on an ocm-common release that includes this package,
// call Validate before create/update, read current tags with fetchUserTagsOnly=true,
// then Equal/ComputeDiff against the desired map. Tag count limits remain API-enforced.
package tags
