---
description: Update and synchronize project documentation
---

Keep greg documentation synchronized with code changes.

## Documentation Files

All documentation uses `.org` format (Emacs org-mode):

| File | Purpose |
|------|---------|
| `README.org` | User documentation, features, installation |
| `docs/dev/ARCHITECTURE.org` | System design, interfaces, data flow |
| `docs/dev/CONTRIBUTING.org` | Development guidelines |
| `docs/PROVIDERS.org` | Provider implementation guide |
| `docs/CONFIG.org` | Configuration reference |
| `docs/COMMANDS.org` | CLI command reference |
| `CLAUDE.md` | AI assistant guidance |
| `AGENTS.md` | AI agent descriptions |

## Tasks

### 1. Check for Documentation Drift

Compare code with docs:

```bash
# Check Provider interface
grep -A 20 "type Provider interface" internal/providers/provider.go

# Check config struct
grep -A 50 "type Config struct" internal/config/config.go

# Check view states
grep -A 30 "type ViewState" internal/tui/model.go
```

### 2. Update Documentation

After code changes, update:

- `ARCHITECTURE.org` - If interfaces or architecture changed
- `CONFIG.org` - If config options added/removed
- `PROVIDERS.org` - If provider interface changed
- `README.org` - If features added/removed
- `CLAUDE.md` - If development patterns changed

### 3. Verify Code Examples

Ensure all code snippets in docs compile:

```bash
# Test that documented interfaces match reality
go build ./...

# Check for broken patterns
just lint
```

### 4. Update Roadmap

In `README.org`:
- Mark completed features with `[X]`
- Add new planned features
- Update version targets

### 5. Check Known Issues

Verify known issues list in `CLAUDE.md` matches reality:
- Manga reading mode issues
- Provider health check verification needed

## Specific Checks

### Interface Changes
```bash
# Compare Provider interface in code vs docs
diff <(grep -A 15 "type Provider interface" internal/providers/provider.go) \
     <(grep -A 15 "type Provider interface" docs/PROVIDERS.org)
```

### Config Changes
Check `internal/config/config.go` for new fields not in `docs/CONFIG.org`.

### New Features
Search for recent additions:
```bash
git log --oneline -20
git diff HEAD~10 --name-only | grep -E '\.(go|org|md)$'
```

## Output

Provide:
1. List of documentation updates needed
2. Suggested changes with diffs
3. Missing documentation sections
4. Outdated examples to fix
5. Broken links found
