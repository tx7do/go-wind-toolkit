# Contributing to protoc-gen-redact

Thank you for your interest in contributing to protoc-gen-redact! This document provides guidelines and information for contributors.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Makefile Targets](#makefile-targets)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Protocol Buffers](#protocol-buffers)
- [Submitting Changes](#submitting-changes)

## Getting Started

### Prerequisites

- Go 1.20 or higher
- Protocol Buffers compiler (protoc)
- Buf CLI (optional, for buf.build publishing)
- golangci-lint (for linting)
- staticcheck (for static analysis)

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/menta2k/protoc-gen-redact.git
cd protoc-gen-redact

# Install required tools
make install-tools

# Download dependencies
make deps

# Build the plugin
make build

# Run tests
make test
```

## Development Workflow

### 1. Set Up Your Environment

```bash
# Install all development tools
make install-tools

# Verify your setup
make info
```

### 2. Make Your Changes

```bash
# Format code automatically
make fmt

# Run linters
make lint

# Fix lint issues automatically (when possible)
make lint-fix
```

### 3. Test Your Changes

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run only fast tests
make test-short

# Run integration tests
make test-integration

# Run benchmarks
make test-bench
```

### 4. Build

```bash
# Build the plugin
make build

# Build with debug symbols
make build-debug

# Install to $GOPATH/bin
make install
```

## Makefile Targets

The project includes a comprehensive Makefile with organized targets. To see all available targets:

```bash
make help
```

### Common Targets

#### General

- `make help` - Display help with all available targets
- `make info` - Show project information and tool versions
- `make all` - Build and test everything

#### Building

- `make build` - Build the protoc-gen-redact plugin
- `make build-debug` - Build with debug symbols (for debugging)
- `make install` - Install plugin to $GOPATH/bin
- `make clean` - Clean all build artifacts

#### Testing

- `make test` - Run all tests with race detection
- `make test-short` - Run quick tests only
- `make test-integration` - Run integration tests (requires build)
- `make test-coverage` - Generate coverage report (HTML + terminal)
- `make test-bench` - Run performance benchmarks

#### Code Quality

- `make fmt` - Format all Go code
- `make vet` - Run go vet
- `make lint` - Run all linters (golangci-lint + staticcheck)
- `make lint-fix` - Auto-fix linting issues where possible

#### Protocol Buffers

- `make buf-lint` - Lint proto files with buf
- `make buf-format` - Format proto files
- `make buf-breaking` - Check for breaking changes
- `make buf-push` - Push to buf.build/menta2k/redact
- `make buf-push-tag TAG=v1.0.0` - Push with a specific tag

#### Legacy Proto Generation

- `make generate` - Generate Go code from redact.proto
- `make generate-examples` - Regenerate example code
- `make generate-testdata` - Regenerate test data

#### CI/CD

- `make ci` - Run CI pipeline (deps + lint + test)
- `make ci-full` - Full CI with coverage and buf checks
- `make pre-commit` - Run pre-commit checks (fmt + lint + test-short)

## Testing

### Writing Tests

- Place tests in `*_test.go` files
- Use table-driven tests where appropriate
- Include both positive and negative test cases
- Add benchmarks for performance-critical code

### Test Categories

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test with actual protoc compilation
3. **Benchmarks**: Performance tests for critical paths

### Running Specific Tests

```bash
# Run specific test
go test -v -run TestName ./...

# Run tests in a specific package
go test -v ./module_test.go

# Run with coverage for specific package
go test -v -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Code Quality

### Linting Configuration

The project uses:
- **golangci-lint**: Comprehensive Go linter (`.golangci.yml`)
- **staticcheck**: Go static analysis
- **buf**: Proto file linting and formatting

### Code Style Guidelines

- Follow standard Go conventions
- Use `gofmt` for formatting (automatic with `make fmt`)
- Write clear, descriptive variable and function names
- Add comments for exported functions and types
- Keep functions focused and reasonably sized

### Pre-Commit Checklist

Before committing, run:

```bash
make pre-commit
```

This will:
1. Format your code
2. Run all linters
3. Run fast tests

## Protocol Buffers

### Working with Proto Files

The main proto file is `redact/v3/redact.proto`. When modifying proto definitions:

1. Make your changes to the `.proto` file
2. Lint your changes: `make buf-lint`
3. Check for breaking changes: `make buf-breaking`
4. Regenerate Go code: `make generate`
5. Test your changes: `make test`

### Publishing to buf.build

```bash
# Lint before pushing
make buf-lint

# Push to buf.build
make buf-push

# Push with a version tag
make buf-push-tag TAG=v1.2.3
```

## Submitting Changes

### Pull Request Process

1. **Fork the Repository**
   ```bash
   git clone https://github.com/YOUR_USERNAME/protoc-gen-redact.git
   cd protoc-gen-redact
   git remote add upstream https://github.com/menta2k/protoc-gen-redact.git
   ```

2. **Create a Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Your Changes**
   - Write code
   - Add/update tests
   - Update documentation if needed
   - Run `make pre-commit`

4. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` - New features
   - `fix:` - Bug fixes
   - `docs:` - Documentation changes
   - `test:` - Test additions/changes
   - `refactor:` - Code refactoring
   - `chore:` - Maintenance tasks

5. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

   Then create a Pull Request on GitHub.

### PR Requirements

- [ ] All tests pass (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linters pass (`make lint`)
- [ ] New code has tests
- [ ] Documentation is updated (if needed)
- [ ] Commit messages follow conventions
- [ ] PR description explains what and why

### Code Review Process

1. Maintainers will review your PR
2. Address any feedback
3. Once approved, your PR will be merged

## CI/CD and Automation

The project uses GitHub Actions for automated testing and publishing:

- **Automated tests** run on all pull requests
- **Code coverage** reports are generated automatically
- **Buf.build publishing** happens on main branch pushes and tagged releases
- **Release artifacts** are built for all platforms on version tags

For information on setting up buf.build publishing and understanding the CI/CD pipeline, see [docs/GITHUB_SETUP.md](docs/GITHUB_SETUP.md)

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/menta2k/protoc-gen-redact/issues)
- **Discussions**: Use GitHub Discussions for questions
- **Documentation**: Check the [README](README.md) and inline code comments

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0, consistent with the project's license.

Thank you for contributing to protoc-gen-redact! 🎉
