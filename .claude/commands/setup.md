---
description: Setup development environment for greg
---

Set up the development environment for greg.

## Prerequisites

- **Go 1.21+** (project uses 1.25.3)
- **mpv** - Video player (required for playback)
- **ffmpeg** - Video processing (required for downloads)
- **just** - Command runner
- **git**

## Setup Steps

### 1. Check Prerequisites

```bash
# Go (need 1.21+)
go version

# mpv
which mpv
mpv --version

# ffmpeg
which ffmpeg
ffmpeg -version

# git
git --version
```

### 2. Install Missing Tools

#### Go
Download from https://go.dev/dl/

#### mpv
```bash
# macOS
brew install mpv

# Ubuntu/Debian
sudo apt install mpv

# Arch
sudo pacman -S mpv

# Windows
# Download from https://mpv.io/
# Note: mpv IPC is broken on Windows, use WSL
```

#### ffmpeg
```bash
# macOS
brew install ffmpeg

# Ubuntu/Debian
sudo apt install ffmpeg

# Arch
sudo pacman -S ffmpeg
```

#### just (command runner)
```bash
# macOS
brew install just

# Linux (cargo)
cargo install just

# Or download from https://github.com/casey/just/releases
```

### 3. Clone and Setup

```bash
git clone https://github.com/justchokingaround/greg.git
cd greg

# Install dev tools (golangci-lint, goimports, air)
just tools

# Download dependencies
just deps

# Verify environment
just doctor
```

### 4. Build and Test

```bash
# Build
just build

# Run tests
just test

# Check code quality
just lint
```

### 5. Initialize Config (Optional)

```bash
just config-init
# Creates ~/.config/greg/config.yaml
```

### 6. Run

```bash
# Run directly
just run

# Hot reload development
just dev
```

## Optional: Git Hooks

Set up pre-commit hook:

```bash
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/sh
just pre-commit
EOF
chmod +x .git/hooks/pre-commit
```

## Optional: Air (Hot Reload)

Already installed via `just tools`. Config is in `.air.toml`.

```bash
# Run with hot reload
just dev
```

## Verify Setup

```bash
# Full environment check
just doctor

# Build and run
just run

# Run with hot reload
just dev

# Run tests
just test

# Check code quality
just lint
```

## Common Issues

### Go Version Too Old
```
error: go version 1.21+ required
```
Update Go from https://go.dev/dl/

### mpv Not Found
```bash
# Verify installation
which mpv

# Install if missing (see above)
```

**Windows users:** mpv IPC doesn't work on Windows. Use WSL.

### just Not Found
```bash
# Install just
# macOS: brew install just
# Linux: cargo install just
# Or download binary from GitHub releases
```

### Permission Denied
```bash
# Make scripts executable
chmod +x scripts/*.sh
```

### gopv Issues
The project uses a local fork at `../gopv`. If you get import errors:
```bash
# Clone gopv next to greg
cd ..
git clone https://github.com/diniamo/gopv.git
cd greg
```

## Quick Reference

| Command | Description |
|---------|-------------|
| `just` | List all commands |
| `just build` | Build binary |
| `just run` | Build and run |
| `just dev` | Hot reload |
| `just test` | Run tests |
| `just lint` | Run linters |
| `just pre-commit` | Full check before commit |
| `just doctor` | Check environment |

## Next Steps

1. Read `CONTRIBUTING.org` for development workflow
2. Review `ARCHITECTURE.org` to understand codebase
3. Check open issues for tasks
4. Run `just` to see all available commands
