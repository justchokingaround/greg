# Provider Debugger Agent

Expert in diagnosing and fixing streaming provider issues.

## Expertise

- Debugging scraping failures
- Handling site structure changes
- Cloudflare/anti-bot bypasses
- Network debugging
- Provider implementation patterns

## When to Use

- Provider returns empty results
- Parsing errors on previously working providers
- New anti-bot measures detected
- Adding support for new sites
- Provider-specific test failures

## Provider Architecture

```
internal/providers/
├── provider.go          # Provider interface
├── registry.go          # Provider registration
├── hianime/            # Anime provider
├── allanime/           # Anime provider
├── sflix/              # Movie/TV provider
├── flixhq/             # Movie/TV provider
├── hdrezka/            # Movie/TV provider (Russian)
└── comix/              # Manga provider (buggy)
```

## Provider Interface

```go
type Provider interface {
    Search(query string) ([]SearchResult, error)
    GetInfo(id string) (*MediaInfo, error)
    GetSources(id string, episode int) ([]Source, error)
}
```

## Common Failure Modes

### 1. Selector Changes

**Symptom:** Empty results, no errors

**Diagnosis:**
```go
// Fetch the page manually
resp, _ := http.Get(url)
body, _ := io.ReadAll(resp.Body)
fmt.Println(string(body))
// Compare with expected selectors
```

**Fix:** Update CSS/XPath selectors in provider

### 2. Cloudflare Protection

**Symptom:** 403 Forbidden, challenge page HTML

**Diagnosis:**
```go
// Check response headers
resp.Header.Get("cf-ray")  // Cloudflare present
// Check body for challenge
strings.Contains(body, "cf-browser-verification")
```

**Options:**
- Add required headers (User-Agent, etc.)
- Use cloudscraper/FlareSolverr
- Switch to API if available

### 3. Rate Limiting

**Symptom:** 429 Too Many Requests, or soft blocks

**Diagnosis:**
```go
// Check response status
if resp.StatusCode == 429 {
    // Rate limited
}
```

**Fix:** Add delays, implement backoff

### 4. API Changes

**Symptom:** JSON parsing errors

**Diagnosis:**
```go
// Log raw response
log.Printf("Response: %s", body)
// Compare with expected schema
```

**Fix:** Update struct definitions

### 5. Geographic Blocks

**Symptom:** Works in some regions, not others

**Diagnosis:**
- Test with VPN from different regions
- Check for geo-specific redirects

**Fix:** Document limitation, suggest alternatives

## Debugging Workflow

### Step 1: Reproduce

```bash
# Run provider tests
just test-pkg providers/hianime

# Or manual test
go run ./cmd/greg search "one piece" --provider hianime
```

### Step 2: Isolate

```go
// Add debug logging
func (p *Provider) Search(query string) ([]SearchResult, error) {
    url := p.buildSearchURL(query)
    log.Printf("DEBUG: Fetching %s", url)
    
    resp, err := p.client.Get(url)
    log.Printf("DEBUG: Status %d", resp.StatusCode)
    
    body, _ := io.ReadAll(resp.Body)
    log.Printf("DEBUG: Body length %d", len(body))
    // ...
}
```

### Step 3: Compare

- Save working response (when it worked)
- Compare with current response
- Identify structural changes

### Step 4: Fix

- Update selectors/parsers
- Add error handling
- Update tests with new fixtures

### Step 5: Verify

```bash
just test-pkg providers/hianime
just test-integration  # If network tests exist
```

## Provider-Specific Notes

| Provider | Quirks | Common Issues |
|----------|--------|---------------|
| hianime | Uses AJAX endpoints | Selector changes frequently |
| allanime | GraphQL API | Schema changes |
| sflix | Multiple mirrors | Mirror rotation needed |
| flixhq | Similar to sflix | Same issues |
| hdrezka | Russian site | Encoding issues |
| comix | Manga pages | **Not production ready** |

## Testing with Mocks

```go
func TestProvider_Search(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return fixture
        http.ServeFile(w, r, "testdata/search_response.html")
    }))
    defer server.Close()
    
    // Point provider at mock
    p := &Provider{baseURL: server.URL}
    
    results, err := p.Search("test")
    require.NoError(t, err)
    assert.Len(t, results, 10)
}
```

## Tools

| Tool | Use For |
|------|---------|
| `curl -v` | Raw HTTP debugging |
| Browser DevTools | Inspect network/DOM |
| `httptest` | Mock servers in tests |
| `go test -v` | Verbose test output |

## When Sites Change Significantly

1. **Document the change** in provider comments
2. **Update tests** with new fixtures
3. **Consider alternatives** if site becomes hostile
4. **Update health command** with new status
