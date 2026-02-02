---
description: Review recent code changes for issues and improvements
---

Review recent code changes in the greg project.

## Quick Review

```bash
# See what changed
git status
git diff --stat

# See actual changes
git diff
```

## Review Process

### 1. Identify Changes
```bash
git status
git diff --name-only
```

### 2. For Each Modified File

Check for:
- Code quality issues
- Potential bugs
- Performance concerns
- Security vulnerabilities
- Missing error handling
- Missing tests
- Documentation gaps

### 3. Greg-Specific Checks

#### Provider Code
- [ ] Uses `context.Context` as first parameter
- [ ] Wraps errors with `fmt.Errorf("context: %w", err)`
- [ ] Caches media info (not stream URLs)
- [ ] Handles movies vs TV correctly

#### TUI Code
- [ ] Handles `tea.WindowSizeMsg`
- [ ] Uses oxocarbon theme colors
- [ ] Keyboard navigation works
- [ ] No blocking in `Update()`

#### Player Code
- [ ] Cleans up mpv processes
- [ ] Handles context cancellation
- [ ] Type-asserts gopv returns

### 4. Security Checks
- [ ] No hardcoded credentials
- [ ] Input validated before use
- [ ] GORM parameterized queries (no SQL injection)
- [ ] No command injection

### 5. Run Quality Checks
```bash
just lint
just test
```

### 6. Test Coverage
```bash
just test-coverage
```

Check that new code has tests.

## Common Issues to Flag

### High Priority
```go
// ❌ Error ignored
data, _ := provider.Search(ctx, query)

// ✅ Error handled
data, err := provider.Search(ctx, query)
if err != nil {
    return fmt.Errorf("search failed: %w", err)
}
```

### Resource Leaks
```go
// ❌ Response body not closed
resp, _ := http.Get(url)

// ✅ Properly closed
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
```

### Race Conditions
```go
// ❌ Unprotected shared state
func (c *Cache) Set(k string, v interface{}) {
    c.data[k] = v
}

// ✅ Mutex protected
func (c *Cache) Set(k string, v interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[k] = v
}
```

## Report Format

```markdown
## Code Review Summary

### Files Changed
- file1.go (added)
- file2.go (modified)

### Issues Found

#### High Priority ⚠️
1. [Error Handling] file.go:42 - Error ignored
   - Suggestion: Add error check

#### Medium Priority 📝
1. [Code Quality] file.go:100 - Magic number
   - Suggestion: Extract to constant

#### Low Priority ℹ️
1. [Style] file.go:55 - Long line
   - Suggestion: Break into multiple lines

### Positive Notes ✨
- Good use of context cancellation
- Well-structured error messages

### Test Coverage
- New code has tests: Yes/No
- Coverage: X%

### Recommendation
- [ ] Approved
- [ ] Approved with minor changes
- [ ] Changes required
```

## If Committing

Suggest commit message following conventional commits:
```
type(scope): description

feat: add new feature
fix: fix bug
refactor: code improvement
test: add tests
docs: documentation update
chore: maintenance
```
