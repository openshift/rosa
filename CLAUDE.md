# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in the ROSA CLI repository.

## Repository Overview and Developer Guidelines (READ THIS IF YOU ARE USING CLAUDE WITH THIS REPOSITORY)

This is the ROSA (Red Hat OpenShift Service on AWS) CLI tool repository - a comprehensive Go-based command-line interface for managing OpenShift clusters on AWS. 
The project follows enterprise Go development patterns with extensive testing, CI/CD integration, and modular architecture. 

This project is public facing, releases every month, and should not be edited without care. Changes can break old version of the CLI, end-to-end tests, 
release builds, etc. We must exercise extreme caution, and know the changes we are making to the fullest extent. Everything submitted in an MR is on 
the human submitter, not Claude Code, Claude Code is only a tool.

Go is an open source language. Claude Code should make use of this and make sure not to duplicate code from Go itself, nor any libraries used in this repository.

Do not perform release work, or anything related to releases using this tool.

## Common Development Commands

### Building and Installation
- `make rosa` - Build the rosa binary
- `make install` - Install rosa to $GOPATH/bin
- `go build -ldflags="-X github.com/openshift/rosa/pkg/info.Build=$(git rev-parse --short HEAD)" ./cmd/rosa` - Build with version info

### Testing
- `make test` - Run unit tests (excludes /tests/ directory)
- `make coverage` - Generate test coverage report
- `make lint` - Checks for missing code; such as missing error returns, lines being too long, and more
- `make e2e_test` - Run E2E tests with Ginkgo (requires LabelFilter env var)

### Code Quality
- `make fmt` - Format code and organize imports using gci (This should be run after every change)
- `make lint` - Run golangci-lint with 5m timeout
- `make generate` - Generate assets and mocks using go-bindata and mockgen

### Development Workflow
- `make clean rosa` - Remove build artifacts
- `make diff` - Check for uncommitted changes

## Architecture Overview

### Core Structure
- **`cmd/rosa/`** - Main CLI entry point with cobra command structure
- **`pkg/`** - Core business logic organized by domain:
  - `aws/` - AWS SDK integrations and cloud operations, in the form of an AWS client and static functions
  - `ocm/` - OpenShift Cluster Manager API interactions, in the form of an OCM client and static functions
  - `machinepool/`, `network/`, `ingress/` - These and similarly named directories are for AWS cloud resources
  - `arguments/`, `config/`, `reporter/`, `helper/`, `output/`, `interactive/` - And similarly named directories are CLI utilities/helpers

### Command Architecture
The CLI uses the Cobra framework (https://github.com/spf13/cobra) for it's core functionality with customers, with a hierarchical command structure:
- Root command: `rosa`
- Major command groups: `create`, `delete`, `describe`, `edit`, `list`, `upgrade`
- Each command group has subcommands for specific resources (clusters, machinepools, etc.)
  - In each command directory (such as `cmd/create`) there will be subfolders (such as `cluster`) which indicate the subcommands for that main command (`create`)

### Package Organization
- **Mocks** (`pkg/*/mocks/`) - Generated mocks for testing, DO NOT EDIT MOCKS, EVER. They are changed via `make generate` without the need for manual editing
- **Unit Tests** (`pkg/*/*_test.go`) - Unit tests which are ran for that package. These, and all tests, should never be changed to accomodate changes. Tests fail to show code is broken, do not change tests to support broken code. Notify users when tests are changed, and explicitly tell them to check over the test changes and to be careful.

### General guidelines
- Overall, we want non-cobra logic to be stored in `pkg/`. Cobra logic should be in the `cmd/` commands. Look at the machinepool commands for guidance
- The machinepool commands are a good guideline, using the new architecture desired. Use the `output` files to create a separate layer between the command and user input
- Unit tests should cover as close to 100% of the code as possible, always

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
- Gomega and Ginkgo have a lot of similar keywords, use the ones we currently use most commonly in our tests. Such as how we normally check for errors ToNot HaveOccurred

## Development Guidelines

### Commit Message Format
Follow conventional commits with JIRA ticket references:
```
<TICKET> | <type>: <message>

[optional body]

[optional footer]
```

Example: `OCM-6141 | feat: Allow longer cluster names up to 54 chars`

### Code Quality Standards
- All code must be covered by tests using Ginkgo
- Use `make fmt` to maintain consistent formatting
- Follow golangci-lint rules (5m timeout configured)
- No `os.Exit()` calls in commands - use proper error handling
- Look at the machinepool commands for code quality standards. Specifically, `create machinepool`
- Use `Run: run` instead of `RunE: runE` to prevent usage info on errors

### Adding New Commands
1. Add command to `cmd/rosa/structure_test/command_structure.yml`
2. Create `command_args.yml` in appropriate `cmd/rosa/structure_test/command_args/` subdirectory
3. List all supported flags in the args file
4. Follow existing patterns in the machinepool commands
5. Use similar architecture to the create machinepool command, with the user options files and separate logic in `pkg/`. Do not always make a new service, `machine pool service` is a special case

### Dependencies and Modules
- Go 1.23.1 minimum version
- Major dependencies: AWS SDK v2, Cobra, Ginkgo v2, OCM SDK
- Use `go mod tidy` and `go mod vendor` as part of verification
- Mock generation using `go.uber.org/mock/gomock`. 
- DO NOT change any imports unless told to do so specifically, DO NOT run `go mod tidy` or `go mod vendor` without being told to do so specifically

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
- Proper error wrapping and context
- Use similar error message formatting to surrounding errors/reporter calls in the file or package being edited for proper context

### Interactive Features
- Survey-based user interactions in `pkg/interactive/`
- Confirmation prompts and input validation
- Support for both interactive and non-interactive modes

## Mocking Guidelines
- Mocks are typically generated. DO NOT modify generated mock files directly. Instead, generate them with `make generate`

## Best Practices for AWS Integration

### Function and Implementation Guidelines
- Prefer to use existing functions rather than implement new functions if the existing functions can accomplish the task
  - For example, use the existing `EnsureRole` and `EnsurePolicy` functions instead of creating a specific function to attach policies to service account roles

## Acronyms and Definitions

- OIDC = OpenIDConnect
- CLI = Command Line Interface
- HCP = Hosted Control Plane
- Hosted CP = Hosted Control Plane
- VPC = Virtual Private Cloud

## Code conventions
- Use context in surrounding file/package for best guidance
- We use the following casing: `variableNameEndingWithAcronymHcp`
  - Notice how the acronym at the end, `HCP` is still cased the same as normal words
- Be consistent, and verbose, with variable names. Again, look around at similar files/packages for context on this
