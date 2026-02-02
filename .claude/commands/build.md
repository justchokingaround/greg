---
description: Build greg binary with version information
---

Build the greg binary using the justfile commands.

## Build Types

### 1. Development Build (Default)
Fast build without optimizations:
```bash
just build
```

Output: `./greg` binary in project root.

### 2. Release Build
Optimized build with version info embedded:
```bash
just build-release
```

Includes:
- Stripped symbols (`-s -w`)
- Version from git tags
- Commit hash
- Build date

### 3. Cross-Platform Builds
Build for all supported platforms:
```bash
just build-all
```

Targets:
- `greg-linux-amd64`
- `greg-darwin-amd64` (macOS Intel)
- `greg-darwin-arm64` (macOS Apple Silicon)
- `greg-windows-amd64.exe`

### 4. Windows Only
```bash
just build-windows
```

## Verify Build

```bash
./greg version
# or
./greg --version
```

Shows:
- Version (from git tag)
- Commit hash
- Build date

## Build Info

Version info is injected via ldflags:
```bash
-X main.version=${VERSION}
-X main.commit=${COMMIT}
-X main.date=${DATE}
```

## Release Archives

Create release archives for distribution:
```bash
just release
```

Creates tar.gz/zip archives for each platform in `dist/`.

## Quick Reference

| Command | Description |
|---------|-------------|
| `just build` | Dev build |
| `just build-release` | Release build |
| `just build-all` | All platforms |
| `just build-windows` | Windows only |
| `just release` | Create archives |
| `just clean` | Remove build artifacts |

## Output

Report:
1. Build type used
2. Binary location
3. Binary size
4. Version info (`./greg version`)
5. Build time
