# Changelog Generator Command

Generate changelog entries following conventional commits.

## Usage

```
/changelog [range]
```

**Arguments:**
- (none) - Changes since last tag
- `<range>` - Git range (e.g., `v0.1.0..HEAD`, `HEAD~10..HEAD`)

## Workflow

### 1. Get Commits

```bash
# Since last tag
git log $(git describe --tags --abbrev=0)..HEAD --oneline

# Or specific range
git log v0.1.0..HEAD --oneline
```

### 2. Parse Conventional Commits

| Prefix | Category | Example |
|--------|----------|---------|
| `feat:` | Features | New functionality |
| `fix:` | Bug Fixes | Bug corrections |
| `perf:` | Performance | Speed improvements |
| `refactor:` | Refactoring | Code restructuring |
| `docs:` | Documentation | README, comments |
| `test:` | Tests | Test additions/fixes |
| `chore:` | Chores | Dependencies, CI |
| `breaking:` | Breaking Changes | API changes |

### 3. Group by Scope

Common scopes in greg:
- `tui` - Terminal UI
- `player` - mpv integration
- `providers` - Streaming sources
- `anilist` - AniList sync
- `db` - Database/persistence

### 4. Generate Entry

```markdown
## [Unreleased]

### Features
- **tui**: Add search history (#123)
- **providers**: Add hdrezka provider

### Bug Fixes
- **player**: Fix volume persistence
- **tui**: Fix j/k navigation in filter mode

### Performance
- **providers**: Parallel source fetching

### Breaking Changes
- **config**: Renamed `api_key` to `anilist_token`
```

## Output Format

### For CHANGELOG.md

```markdown
## [0.2.0] - 2025-01-20

### Added
- New hdrezka provider for movies/TV
- Search history with fuzzy matching
- Keyboard shortcut help overlay (?)

### Fixed
- Player volume not persisting between sessions
- Crash when provider returns empty results

### Changed
- Improved error messages for network failures

### Removed
- Deprecated `--legacy-player` flag
```

### For GitHub Release

```markdown
## What's New

### Features
- New hdrezka provider for movies/TV
- Search history with fuzzy matching

### Bug Fixes
- Player volume not persisting
- Crash on empty provider results

### Full Changelog
https://github.com/user/greg/compare/v0.1.0...v0.2.0
```

## Conventional Commit Examples

```bash
# Feature
git commit -m "feat(tui): add search history"

# Fix
git commit -m "fix(player): persist volume setting"

# Breaking change
git commit -m "feat(config)!: rename api_key to anilist_token

BREAKING CHANGE: Config file requires update"

# With issue reference
git commit -m "fix(providers): handle empty results (#42)"
```

## Notes

- Greg doesn't currently have a CHANGELOG.md
- Consider creating one with this command
- Follow [Keep a Changelog](https://keepachangelog.com/) format
- Use [Semantic Versioning](https://semver.org/)
