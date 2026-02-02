---
description: Scaffold a new streaming provider implementation
---

Scaffold a new streaming provider for greg.

## Important: Local Mode is Default

Providers run **locally by default** using internal scraping. The remote API server is **optional and experimental** - do not implement remote mode unless specifically requested.

## Provider Details Needed

Ask the user for:
1. **Provider name**
2. **Media type** (anime, movies, tv, or multiple)
3. **Base URL** of the website
4. **Authentication requirements** (if any)
5. **Special requirements** (decryption, GraphQL, etc.)

## Create Provider Structure

```bash
mkdir -p internal/providers/{provider_name}
```

Create these files:

### 1. provider.go - Main Implementation
```go
package providername

import (
    "context"
    "fmt"
    "sync"

    "github.com/justchokingaround/greg/internal/providers"
)

type Provider struct {
    name       string
    baseURL    string
    httpClient *http.Client
    cache      sync.Map // Cache media info (NOT stream URLs)
}

func New() *Provider {
    return &Provider{
        name:       "providername",
        baseURL:    "https://example.com",
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (p *Provider) Name() string {
    return p.name
}

func (p *Provider) Type() providers.MediaType {
    return providers.Anime // or Movie, TV, MovieTV
}

func (p *Provider) Search(ctx context.Context, query string) ([]providers.Media, error) {
    // Implementation
    return nil, fmt.Errorf("not implemented")
}

func (p *Provider) GetTrending(ctx context.Context) ([]providers.Media, error) {
    return nil, fmt.Errorf("not implemented")
}

func (p *Provider) GetRecent(ctx context.Context) ([]providers.Media, error) {
    return nil, fmt.Errorf("not implemented")
}

func (p *Provider) GetMediaDetails(ctx context.Context, id string) (*providers.MediaDetails, error) {
    return nil, fmt.Errorf("not implemented")
}

func (p *Provider) GetSeasons(ctx context.Context, mediaID string) ([]providers.Season, error) {
    return nil, fmt.Errorf("not implemented")
}

func (p *Provider) GetEpisodes(ctx context.Context, seasonID string) ([]providers.Episode, error) {
    return nil, fmt.Errorf("not implemented")
}

func (p *Provider) GetStreamURL(ctx context.Context, episodeID string, quality providers.Quality) (*providers.StreamURL, error) {
    return nil, fmt.Errorf("not implemented")
}

func (p *Provider) GetAvailableQualities(ctx context.Context, episodeID string) ([]providers.Quality, error) {
    return nil, fmt.Errorf("not implemented")
}

func (p *Provider) IsAvailable(ctx context.Context) bool {
    // Health check - test if provider is accessible
    return true
}
```

### 2. provider_test.go - Tests
```go
package providername_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/justchokingaround/greg/internal/providers/providername"
)

func TestSearch(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`mock response`))
    }))
    defer server.Close()

    p := providername.New()
    // Override base URL for testing
    // p.baseURL = server.URL

    results, err := p.Search(context.Background(), "test")
    assert.NoError(t, err)
    assert.NotEmpty(t, results)
}
```

## Key Patterns

### 1. Context Usage
Always use `context.Context` as first parameter and respect cancellation:
```go
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
}
```

### 2. Error Wrapping
```go
if err != nil {
    return nil, fmt.Errorf("provider %s search failed: %w", p.Name(), err)
}
```

### 3. Caching
Cache media info but **never cache stream URLs** (they expire):
```go
if cached, ok := p.cache.Load(mediaID); ok {
    return cached.(*providers.MediaDetails), nil
}
// ... fetch and cache
p.cache.Store(mediaID, details)
```

### 4. Movies vs TV
For movies, return empty seasons array:
```go
func (p *Provider) GetSeasons(ctx context.Context, mediaID string) ([]providers.Season, error) {
    if isMovie(mediaID) {
        return []providers.Season{}, nil // TUI will auto-skip
    }
    // Return actual seasons for TV
}
```

## Register Provider

Add to `internal/providers/registry.go`:
```go
func init() {
    Register("providername", func() Provider {
        return providername.New()
    })
}
```

## Run Tests

```bash
just test-pkg providers/providername
```

## Documentation

Update:
- `README.org` - Add to provider list
- `docs/PROVIDERS.org` - If unique implementation
- `docs/CONFIG.org` - If special config needed

## Output

Provide:
1. Created file paths
2. Provider interface methods implemented
3. Test results
4. Registration status
5. Next steps for user
