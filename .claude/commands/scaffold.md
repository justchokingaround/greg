---
description: Generate boilerplate code for various components
---

Generate boilerplate code for greg components.

## Component Types

### 1. TUI Component

Create Bubble Tea component in `internal/tui/components/{name}/`:

```go
package componentname

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/justchokingaround/greg/internal/tui/styles"
)

type Model struct {
    width  int
    height int
    // Add component state
}

func New() Model {
    return Model{}
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "esc":
            // Handle quit/back
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    return m, nil
}

func (m Model) View() string {
    // Use styles from internal/tui/styles/
    return lipgloss.JoinVertical(
        lipgloss.Left,
        "Header",
        "Content",
        "Footer",
    )
}

// SetSize allows parent to set dimensions
func (m *Model) SetSize(w, h int) {
    m.width = w
    m.height = h
}
```

### 2. Database Model

Add to `internal/database/models.go`:

```go
type NewModel struct {
    gorm.Model
    Field1 string `gorm:"not null"`
    Field2 int    `gorm:"default:0"`
}

func (NewModel) TableName() string {
    return "new_models"
}
```

Add migration in `database.go`:
```go
db.AutoMigrate(&NewModel{})
```

### 3. CLI Command

Create in `cmd/greg/` or add to existing:

```go
var newCmd = &cobra.Command{
    Use:   "newcmd",
    Short: "Short description",
    Long:  `Long description with examples.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(newCmd)
    newCmd.Flags().StringP("flag", "f", "", "Flag description")
}
```

### 4. HTTP Client Wrapper

```go
package client

import (
    "context"
    "net/http"
    "time"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
}

func New(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
    if err != nil {
        return nil, err
    }
    return c.httpClient.Do(req)
}
```

### 5. Background Worker

```go
package worker

import (
    "context"
    "time"
)

type Worker struct {
    interval time.Duration
    done     chan struct{}
}

func New(interval time.Duration) *Worker {
    return &Worker{
        interval: interval,
        done:     make(chan struct{}),
    }
}

func (w *Worker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-w.done:
            return
        case <-ticker.C:
            w.doWork()
        }
    }
}

func (w *Worker) Stop() {
    close(w.done)
}

func (w *Worker) doWork() {
    // Implementation
}
```

### 6. Test Suite

```go
package pkg_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "expected",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## After Scaffolding

1. Verify syntax: `go build ./...`
2. Run tests: `just test`
3. Check lint: `just lint`
4. Fill in TODO comments
5. Add proper tests
