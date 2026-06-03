# CLAUDE.md

<!-- Canonical source: AGENTS.md. This file is auto-generated for Claude Code compatibility. -->

This file provides guidance to AI coding assistants when working with this repository.

## Project Overview

OCM SDK for Go — a client library that simplifies interaction with the OpenShift Cluster Manager (OCM) API. Provides type-safe Go bindings generated from the OCM API model, with authentication, pagination, and error handling built in.

## Build & Test Commands

```bash
make generate        # Regenerate SDK code from model definitions
make fmt             # Format Go source code
make goimports       # Organize imports
make lint            # Run golangci-lint
make examples        # Build example programs
make clean           # Remove generated files
```

Tests use Ginkgo:
```bash
make ginkgo-install  # Install Ginkgo test runner
ginkgo ./...         # Run all tests
```

## Architecture

The SDK is organized by OCM API service area, each in its own top-level package:

- **clustersmgmt/**: Cluster management API bindings
- **accountsmgmt/**: Account management API bindings
- **addonsmgmt/**: Add-on management API bindings
- **servicelogs/**: Service logs API bindings
- **authorizations/**: Authorization API bindings
- **authentication/**: Authentication utilities and token handling
- **configuration/**: SDK configuration
- **errors/**: Error type definitions
- **helpers/**: Shared utility functions
- **logging/**: Logging interfaces
- **testing/**: Test utilities and mock transport
- **examples/**: Usage examples

## Key Conventions

- Module path: `github.com/openshift-online/ocm-sdk-go`
- Most code is auto-generated from the OCM API model — do not edit generated files directly
- Uses builder pattern for constructing API requests
- Ginkgo/Gomega for testing
- Generated files can be identified by their header comments
