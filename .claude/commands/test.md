---
description: Run tests with coverage and analysis
---

Run the test suite for greg using justfile commands.

## Quick Test

```bash
# Run all unit tests
just test
```

## Test Commands

| Command | Description |
|---------|-------------|
| `just test` | Run all unit tests |
| `just test-coverage` | Generate HTML coverage report |
| `just test-integration` | Run integration tests (requires network) |
| `just test-race` | Run with race detector |
| `just test-pkg <name>` | Test specific package |
| `just bench` | Run benchmarks |

## Examples

### Run All Tests
```bash
just test
```

### Test Specific Package
```bash
just test-pkg providers
just test-pkg player
just test-pkg tracker
just test-pkg downloader
```

### Coverage Report
```bash
just test-coverage
# Opens coverage.html in browser
```

### Integration Tests
```bash
# Requires network connection
just test-integration
```

These test real API endpoints. Use sparingly.

### Race Detection
```bash
just test-race
```

Finds concurrency bugs. Run before merging.

### Benchmarks
```bash
just bench
```

## Coverage Goals

| Package | Target |
|---------|--------|
| Providers | 80%+ |
| Player | 80%+ |
| Tracker | 80%+ |
| Downloader | 80%+ |
| TUI | 60%+ |

## View Uncovered Code

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -v "100.0%"
```

## Known Test Issues

**TODO: Some tests may be failing.** Run `just test` to identify:

```bash
just test 2>&1 | grep -E "FAIL|---"
```

## Writing Tests

### Table-Driven Tests
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid", "input", "output", false},
        {"empty", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Mock HTTP
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"data": "test"}`))
}))
defer server.Close()
```

### Integration Test Tag
```go
//go:build integration

package provider_test
```

## Output

Report:
1. Total tests run
2. Passed/Failed count
3. Coverage percentage
4. Failing tests with details
5. Suggestions for improvement

If tests fail, analyze failures and suggest fixes.
