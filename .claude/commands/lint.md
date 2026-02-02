---
description: Run linters and code quality checks
---

Run code quality checks for greg using the justfile.

## Quick Check

```bash
# Run all quality checks
just lint
```

This runs:
- `go vet ./...`
- `gofmt` check
- `golangci-lint` (if installed)

## Individual Checks

### 1. Go Vet
```bash
go vet ./...
```

### 2. Format Check
```bash
# Check formatting
gofmt -l .

# Auto-fix formatting
just fmt
```

### 3. Golangci-lint
```bash
golangci-lint run
```

If not installed:
```bash
just tools  # Installs golangci-lint and other dev tools
```

### 4. Module Tidiness
```bash
go mod tidy
git diff go.mod go.sum  # Check for changes
```

## Pre-Commit Check

Run all checks before committing:
```bash
just pre-commit
```

This runs: `fmt` → `lint` → `test`

## Common Issues

### Unused Imports/Variables
```bash
# golangci-lint will catch these
golangci-lint run --enable=unused
```

### Error Handling
```bash
# Check for unchecked errors
golangci-lint run --enable=errcheck
```

### Shadow Variables
```bash
golangci-lint run --enable=govet
```

### Printf Format Issues
```bash
go vet ./...
```

## Auto-Fix

```bash
# Fix formatting
just fmt

# This runs:
# - gofmt -w .
# - goimports -w .
```

## TODO/FIXME Comments

Find all TODO comments:
```bash
grep -rn "TODO\|FIXME\|XXX\|HACK" --include="*.go" .
```

## Output

Report:
1. Total issues found
2. Issues by category (error, warning)
3. Files affected
4. Suggestions for fixes
5. Commands to auto-fix (where applicable)

If issues found, offer to run `just fmt` to fix formatting automatically.
