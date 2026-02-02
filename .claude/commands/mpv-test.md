---
description: Test mpv integration and player functionality
---

Test mpv integration with greg using gopv.

## Test Steps

### 1. Verify mpv Installation
```bash
which mpv
mpv --version
```

### 2. Test Basic mpv IPC
```bash
# Start mpv with IPC socket
mpv --input-ipc-server=/tmp/test-mpv.sock --idle &

# Test connection (Linux/macOS)
echo '{"command": ["get_property", "volume"]}' | socat - /tmp/test-mpv.sock

# Clean up
pkill -f "mpv.*test-mpv.sock"
```

### 3. Run Unit Tests
```bash
just test-pkg player
```

### 4. Run Integration Tests
```bash
# Requires mpv installed
just test-integration
```

### 5. Test with Stream URL
If user provides a test stream:
```bash
# Test direct playback
mpv "https://example.com/video.m3u8"

# Test with IPC
mpv --input-ipc-server=/tmp/greg-test.sock "https://example.com/video.m3u8"
```

## Common Issues

### Socket Permission Issues
```bash
# Check /tmp is writable
ls -la /tmp/

# Check for stale sockets
ls /tmp/greg-mpv-*.sock 2>/dev/null
rm /tmp/greg-mpv-*.sock 2>/dev/null  # Clean up
```

### Zombie Processes
```bash
# Find zombie mpv processes
ps aux | grep mpv

# Kill orphaned mpv
pkill -9 mpv
```

### IPC Timeout
- Increase timeout in player config
- Check if mpv started successfully
- Verify socket path exists

### JSON Parsing Errors
- gopv returns `interface{}` - always type assert
- Check mpv version compatibility

## Test Code

Create a test file to verify gopv:

```go
package main

import (
    "fmt"
    "time"
    "os/exec"

    "github.com/diniamo/gopv"
)

func main() {
    socketPath := "/tmp/test-mpv.sock"

    // Start mpv
    cmd := exec.Command("mpv",
        "--input-ipc-server="+socketPath,
        "--idle")
    cmd.Start()
    defer cmd.Process.Kill()

    // Wait for socket
    time.Sleep(500 * time.Millisecond)

    // Connect
    client, err := gopv.NewClient(socketPath)
    if err != nil {
        fmt.Println("Connection failed:", err)
        return
    }
    defer client.Close()

    // Test property
    vol, err := client.GetProperty("volume")
    if err != nil {
        fmt.Println("GetProperty failed:", err)
        return
    }
    fmt.Printf("Volume: %v\n", vol)

    fmt.Println("mpv integration working!")
}
```

Run with:
```bash
go run test_mpv.go
```

## Output

Report:
1. mpv version and path
2. Connection test result
3. Property read/write test
4. Command execution test
5. Performance metrics (latency)
6. Any errors encountered
7. Platform-specific notes
