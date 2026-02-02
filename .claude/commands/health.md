# Provider Health Check Command

Quick health check for streaming providers.

## Usage

```
/health [provider]
```

**Arguments:**
- (none) - Check all providers
- `<name>` - Check specific provider (hianime, allanime, sflix, flixhq, hdrezka, comix)

## Workflow

### 1. List Available Providers

```go
// internal/providers/registry.go
var Registry = map[string]Provider{
    "hianime": &hianime.Provider{},
    "allanime": &allanime.Provider{},
    // ...
}
```

### 2. Test Each Provider

For each provider, test:

| Test | Method | Pass Criteria |
|------|--------|---------------|
| Search | `Search("one piece")` | Returns results |
| Info | `GetInfo(id)` | Returns valid MediaInfo |
| Sources | `GetSources(id)` | Returns playable URLs |

### 3. Check Response Times

```
< 2s  = Good
2-5s  = Acceptable  
> 5s  = Slow (may indicate blocking)
```

### 4. Detect Common Issues

| Issue | Detection | Likely Cause |
|-------|-----------|--------------|
| Timeout | No response > 10s | Site down or blocked |
| 403/Cloudflare | Status code check | Anti-bot protection |
| Empty results | Valid response, no data | Selector changes |
| Parse errors | Malformed data | HTML structure changed |

## Provider Status Table

| Provider | Type | Region Issues | Notes |
|----------|------|---------------|-------|
| hianime | Anime | Some ISP blocks | Primary anime |
| allanime | Anime | Rare | Backup anime |
| sflix | Movies/TV | Some regions | |
| flixhq | Movies/TV | Some regions | |
| hdrezka | Movies/TV | Works globally | Russian site |
| comix | Manga | None known | **Not production ready** |

## Output Format

```markdown
## Provider Health Report

| Provider | Status | Search | Info | Sources | Latency |
|----------|--------|--------|------|---------|---------|
| hianime  | ✅ OK  | ✅     | ✅   | ✅      | 1.2s    |
| allanime | ✅ OK  | ✅     | ✅   | ✅      | 0.8s    |
| sflix    | ⚠️ Slow | ✅    | ✅   | ✅      | 4.5s    |
| flixhq   | ❌ Down | ❌    | -    | -       | timeout |
| hdrezka  | ✅ OK  | ✅     | ✅   | ✅      | 1.5s    |
| comix    | ⚠️ Beta | ✅    | ✅   | ⚠️      | 1.0s    |

### Issues Detected
- **flixhq**: Connection timeout - site may be down or blocked
- **sflix**: Slow response - possible rate limiting

### Recommendations
- Consider adding retry logic for flixhq
- Monitor sflix for continued slowness
```

## Running Tests Manually

```bash
# Test specific provider
just test-pkg providers/hianime

# Integration tests (requires network)
just test-integration
```

## Known Provider Issues

| Provider | Issue | Status |
|----------|-------|--------|
| comix | Manga reading mode buggy | Not production ready |
| All | No offline caching | Feature not implemented |
