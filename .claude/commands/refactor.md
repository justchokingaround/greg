---
description: Safe refactoring with LSP verification
---

Perform safe refactoring operations with LSP verification.

## Safety First

**CRITICAL:** Always verify changes don't break the build:
1. Run `just build` after each refactoring step
2. Run `just test` to catch regressions
3. Use LSP tools for safe renames and references

## Refactoring Types

### 1. Rename Symbol (Safest)
Use LSP for project-wide renames:

```bash
# Check if rename is valid first
# Use lsp_prepare_rename tool

# Then perform the rename
# Use lsp_rename tool
```

### 2. Extract Function
Move code into a new function:

1. Identify the code block to extract
2. Determine parameters needed (variables used)
3. Determine return values
4. Create new function
5. Replace original code with call
6. Verify with `just build && just test`

### 3. Extract Interface
Create interface from concrete type:

1. Identify methods to abstract
2. Create interface with those methods
3. Update consumers to use interface
4. Verify with `just build`

### 4. Move to New File
Split large files:

1. Identify cohesive code group
2. Create new file in same package
3. Move types/functions
4. Update imports if needed
5. Verify with `just build`

### 5. Inline Function/Variable
Remove unnecessary indirection:

1. Find all usages with `lsp_find_references`
2. Replace each usage with inlined code
3. Remove original definition
4. Verify with `just build && just test`

## Large File Refactoring

For files like `model.go` (7000+ lines):

### Step 1: Analyze Structure
```bash
# Get file outline
# Use lsp_document_symbols tool
```

### Step 2: Identify Groupings
Look for:
- Related types and their methods
- Feature-specific code (e.g., all AniList handling)
- View-specific update/view functions

### Step 3: Extract Incrementally
1. Start with most isolated group
2. Create new file: `model_{feature}.go`
3. Move related code
4. Keep in same package (no import changes)
5. Build and test after EACH move

### Step 4: Common Splits for TUI

```
model.go (main)           → Core state, Init, main Update switch
model_anilist.go          → AniList view handling
model_downloads.go        → Download view handling
model_player.go           → Player/playback handling
model_search.go           → Search and results handling
model_manga.go            → Manga-specific handling
model_messages.go         → Message type definitions
model_commands.go         → Tea.Cmd functions
```

## LSP Tools Reference

| Tool | Use For |
|------|---------|
| `lsp_hover` | Get type info at position |
| `lsp_goto_definition` | Find where symbol is defined |
| `lsp_find_references` | Find ALL usages of symbol |
| `lsp_document_symbols` | Get file outline/structure |
| `lsp_workspace_symbols` | Search symbols across project |
| `lsp_prepare_rename` | Check if rename is valid |
| `lsp_rename` | Rename symbol project-wide |
| `lsp_diagnostics` | Get errors/warnings |

## Workflow

### Small Refactor
```
1. Identify change
2. Use LSP to find all references
3. Make change
4. just build
5. just test
6. Done
```

### Large Refactor
```
1. Create branch: git checkout -b refactor/description
2. Plan extraction groups
3. For each group:
   a. Move code to new file
   b. just build
   c. just test
   d. git commit -m "refactor: extract X to model_x.go"
4. Final verification: just pre-commit
5. Create PR for review
```

## Common Patterns in Greg

### Extracting View Handler
```go
// Before: in model.go
func (m Model) updateAniListView(msg tea.Msg) (Model, tea.Cmd) {
    // 200 lines...
}

// After: in model_anilist.go (same package)
func (m Model) updateAniListView(msg tea.Msg) (Model, tea.Cmd) {
    // Same code, just moved
}
```

### Extracting Message Types
```go
// Before: scattered in model.go
type searchResultsMsg []providers.Media
type errorMsg error
// ...

// After: in model_messages.go
type (
    searchResultsMsg   []providers.Media
    errorMsg           error
    progressMsg        float64
    // All message types together
)
```

## Verification Checklist

After refactoring:
- [ ] `just build` passes
- [ ] `just test` passes
- [ ] `just lint` passes
- [ ] No new LSP diagnostics
- [ ] Functionality unchanged (manual test)

## Output

Report:
1. What was refactored
2. Files created/modified
3. Build status
4. Test status
5. Any issues found
6. Suggested follow-up refactors
