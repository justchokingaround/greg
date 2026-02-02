# TODO Scanner Command

Scan and manage TODO/FIXME comments across the codebase.

## Usage

```
/todo [action]
```

**Actions:**
- (none) - List all TODOs
- `priority` - Categorize by urgency
- `file <path>` - TODOs in specific file/package
- `clean` - Find stale/outdated TODOs

## Workflow

### 1. Scan Codebase

```bash
# Find all TODO/FIXME comments
grep -rn "TODO\|FIXME\|HACK\|XXX" --include="*.go" .
```

### 2. Categorize by Type

| Tag | Meaning | Priority |
|-----|---------|----------|
| FIXME | Known bug, needs fixing | High |
| TODO | Enhancement needed | Medium |
| HACK | Temporary workaround | Medium |
| XXX | Dangerous/problematic | High |

### 3. Categorize by Area

```
internal/tui/       - UI/UX improvements
internal/player/    - Player functionality
internal/providers/ - Scraping/parsing
internal/anilist/   - API integration
internal/db/        - Database/persistence
```

### 4. Report Format

```markdown
## TODOs by Priority

### High Priority (FIXME/XXX)
- [ ] file.go:123 - Description

### Medium Priority (TODO/HACK)
- [ ] file.go:456 - Description

## TODOs by Package
...
```

## Known TODOs in Greg

These are documented issues:

| Location | Issue | Status |
|----------|-------|--------|
| `internal/player/mpv/` | Windows IPC broken (gopv bug) | Blocked upstream |
| `internal/tui/model.go` | j/k navigation in filter mode | Bug |
| `internal/tui/` | Manga reading mode | Not production ready |

## Output

Provide:
1. **Count summary** - Total TODOs by type
2. **Prioritized list** - Grouped by urgency
3. **Stale check** - TODOs older than 6 months (check git blame)
4. **Recommendations** - Which to tackle first

## Example Output

```
## TODO Summary

Found 6 TODOs across 4 files:
- FIXME: 1
- TODO: 4
- HACK: 1

### High Priority
1. internal/providers/hianime/hianime.go:234
   FIXME: Rate limiting not implemented

### Medium Priority
2. internal/tui/components/player/player.go:89
   TODO: Add subtitle track selection
...

### Recommendations
- Consider fixing #1 first (affects stability)
- #2-4 are good first issues for contributors
```
