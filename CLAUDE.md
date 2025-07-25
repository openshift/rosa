# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is the ROSA (Red Hat OpenShift Service on AWS) CLI tool repository - a comprehensive Go-based command-line interface for managing OpenShift clusters on AWS. The project follows enterprise Go development patterns with extensive testing, CI/CD integration, and modular architecture.

## Common Development Commands

### Building and Installation
- `make rosa` - Build the rosa binary
- `make install` - Install rosa to $GOPATH/bin  
- `go build -ldflags="-X github.com/openshift/rosa/pkg/info.Build=$(git rev-parse --short HEAD)" ./cmd/rosa` - Build with version info

### Testing
- `make test` - Run unit tests (excludes /tests/ directory)
- `make coverage` - Generate test coverage report
- `make e2e_test` - Run E2E tests with Ginkgo (requires LabelFilter env var)
- `ginkgo run --label-filter day1 tests/e2e --timeout 2h` - Run day1 E2E tests
- `ginkgo run --label-filter '(Critical,High)&&(day1-post,day2)&&!Exclude' tests/e2e` - Run filtered E2E tests

### Code Quality
- `make fmt` - Format code and organize imports using gci
- `make lint` - Run golangci-lint with 5m timeout
- `make verify` - Format, tidy modules, vendor, and check for changes
- `make generate` - Generate assets and mocks using go-bindata and mockgen

### Development Workflow
- `make clean` - Remove build artifacts
- `make diff` - Check for uncommitted changes
- `commits/check` - Verify commit message format

## Architecture Overview

### Core Structure
- **`cmd/rosa/`** - Main CLI entry point with cobra command structure
- **`pkg/`** - Core business logic organized by domain:
  - `aws/` - AWS SDK integrations and cloud operations
  - `ocm/` - OpenShift Cluster Manager API interactions  
  - `interactive/` - User interaction and prompts
  - `machinepool/`, `network/`, `ingress/` - Resource management
  - `arguments/`, `config/`, `reporter/` - CLI utilities

### Command Architecture
The CLI uses Cobra framework with hierarchical command structure:
- Root command: `rosa`
- Major command groups: `create`, `delete`, `describe`, `edit`, `list`, `upgrade`
- Each command group has subcommands for specific resources (clusters, machinepools, etc.)

### Package Organization
- **API Interfaces** (`pkg/aws/api_interface/`) - AWS service abstractions
- **Mocks** (`pkg/*/mocks/`) - Generated mocks for testing
- **Helpers** (`pkg/helper/`) - Cross-cutting utilities
- **Domain Logic** - Each major feature has its own package

## Testing Framework

### Test Structure
- **Framework**: Ginkgo v2 + Gomega for BDD-style testing
- **Unit Tests**: `*_test.go` files with `*_suite_test.go` suite initializers
- **E2E Tests**: `tests/e2e/` directory with comprehensive integration testing
- **Test Labels**: Sophisticated labeling system for test categorization (Critical, High, day1, day2, etc.)

### Test Execution Patterns
- Unit tests exclude `/tests/` directory
- E2E tests use label-based filtering for selective execution
- Parallel execution support with timeout management
- Extensive mocking using gomock for external dependencies

### Test Organization
- Suite pattern with `RegisterFailHandler(Fail)` and `RunSpecs()`
- Context/Describe/It structure for readable test specifications
- BeforeEach/AfterEach hooks for setup/teardown
- DeferCleanup for resource management

## Development Guidelines

### Commit Message Format
Follow conventional commits with JIRA ticket references:
```
<JIRA-TICKET> | <type>: <message>

[optional body]

[optional footer]
```

Example: `OCM-6141 | feat: Allow longer cluster names up to 54 chars`

### Code Quality Standards
- All code must be covered by tests using Ginkgo
- Use `make fmt` to maintain consistent formatting
- Follow golangci-lint rules (5m timeout configured)
- No `os.Exit()` calls in commands - use proper error handling
- Use `Run: run` instead of `RunE: runE` to prevent usage info on errors

### Adding New Commands
1. Add command to `cmd/rosa/structure_test/command_structure.yml`
2. Create `command_args.yml` in appropriate `cmd/rosa/structure_test/command_args/` subdirectory
3. List all supported flags in the args file
4. Follow existing patterns in `cmd/rosa/main.go` for registration

### Dependencies and Modules
- Go 1.23.1 minimum version
- Major dependencies: AWS SDK v2, Cobra, Ginkgo v2, OCM SDK
- Use `go mod tidy` and `go mod vendor` as part of verification
- Mock generation using `go.uber.org/mock/gomock`

## Key Files and Entry Points

### Main Entry Points
- `cmd/rosa/main.go` - CLI application entry point and command registration
- `pkg/rosa/runner.go` - Core runtime logic

### Configuration
- `Makefile` - Build system and development commands
- `go.mod` - Go module dependencies
- `.golangciversion` - Linter version for CI
- `codecov.yml` - Code coverage configuration

### Testing Entry Points  
- `tests/e2e/e2e_suite_test.go` - E2E test suite
- `pkg/test/helpers.go` - Test utilities and helpers
- Individual `*_suite_test.go` files - Package-level test suites

## Special Considerations

### AWS Integration
- Extensive AWS SDK v2 usage across multiple services
- Mock interfaces for all AWS API clients
- CloudFormation template management in `templates/`

### Error Handling
- Use reporter pattern for consistent error messaging
- Avoid `os.Exit()` in commands
- Proper error wrapping and context

### Interactive Features
- Survey-based user interactions in `pkg/interactive/`
- Confirmation prompts and input validation
- Support for both interactive and non-interactive modes

## Mocking Guidelines
- Mocks are typically generated. Do not modify generated mock files directly. Instead, generate them with `make generate`

## Best Practices for AWS Integration

### Function and Implementation Guidelines
- Prefer to use existing functions rather than implement new functions if the existing functions can accomplish the task
  - For example, use the existing `EnsureRole` and `EnsurePolicy` functions instead of creating a specific function to attach policies to service account roles

## Acronyms and Definitions

- OIDC = OpenIDConnect
