---
description: Debug and troubleshoot greg issues
---

Help debug issues in greg. Ask the user what they're experiencing.

## Known Bugs

**Active issues to be aware of:**

1. **Manga reading mode bugs** - Various UX issues, not production ready.

## Common Issues to Check

### 1. Build Failures
```bash
# Check dependencies
just deps
go mod tidy

# Verify build
just build

# Check for syntax errors
just lint
```

### 2. Runtime Crashes
- Check logs: `~/.local/state/greg/greg.log`
- Look for panic stack traces
- Check database permissions
- Verify config syntax

```bash
# Enable debug logging
greg --log-level debug

# Check config
greg config show
```

### 3. Provider Issues
```bash
# List providers and status
greg providers list

# Test specific provider
# Check if website structure changed
# Verify decryption logic still works
```

Common provider problems:
- Website structure changed (selector updates needed)
- Rate limiting
- Cloudflare protection
- API changes

### 4. Database Issues
```bash
# Check database exists
ls -la ~/.local/share/greg/greg.db

# Test database
sqlite3 ~/.local/share/greg/greg.db "SELECT name FROM sqlite_master WHERE type='table';"

# Reset database (WARNING: deletes all data)
just db-reset
```

Check for:
- Lock files (`*.db-wal`, `*.db-shm`)
- Permission issues
- Corrupted database (try reset)

### 5. Player Issues (mpv)

```bash
# Verify mpv installed
which mpv
mpv --version

# Test mpv directly
mpv "https://example.com/test.mp4"

# Test IPC socket
mpv --input-ipc-server=/tmp/test-mpv.sock --idle
```

**Windows users:** mpv IPC is broken. Use WSL or another platform.

Check:
- Socket permissions
- gopv connection timeouts
- Zombie mpv processes

### 6. Configuration Issues

```bash
# Show effective config
greg config show

# Validate YAML
cat ~/.config/greg/config.yaml | python3 -c "import sys,yaml; yaml.safe_load(sys.stdin)"

# Check XDG paths
echo $XDG_CONFIG_HOME  # defaults to ~/.config
echo $XDG_DATA_HOME    # defaults to ~/.local/share
```

### 7. AniList Issues

```bash
# Re-authenticate
greg auth anilist

# Check token stored
ls -la ~/.config/greg/
```

Common problems:
- Token expired (re-authenticate)
- Rate limiting (wait and retry)
- Network issues

## Debug Commands

```bash
# Full environment check
just doctor

# Run with debug logging
greg --log-level debug

# Check dependencies
go list -m all

# Run tests to verify functionality
just test
```

## Collect Debug Info

When reporting issues, gather:

```bash
# System info
uname -a
go version
mpv --version

# Greg info
./greg version
cat ~/.config/greg/config.yaml

# Recent logs
tail -100 ~/.local/state/greg/greg.log
```

## Output

Provide:
1. Diagnosis of the issue
2. Root cause analysis
3. Step-by-step fix
4. Prevention tips
5. Relevant log excerpts with explanation
