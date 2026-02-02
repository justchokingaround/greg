---
description: Create quick prototype or experiment with code
---

Create a quick prototype or experiment in greg.

## Prototype Location

Create prototypes in temporary directory:
```bash
mkdir -p /tmp/greg-proto
cd /tmp/greg-proto
go mod init proto
```

Or use the scripts directory:
```bash
mkdir -p scripts/prototypes
```

## Common Prototypes

### 1. HTTP Scraper Test
Test scraping a website:

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/PuerkitoBio/goquery"
)

func main() {
    resp, err := http.Get("https://example.com")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        panic(err)
    }

    doc.Find("selector").Each(func(i int, s *goquery.Selection) {
        fmt.Println(s.Text())
    })
}
```

### 2. GraphQL Query Test
```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

func main() {
    query := `{"query": "{ Page { media(search: \"test\") { id title { romaji } } } }"}`

    resp, err := http.Post(
        "https://graphql.anilist.co",
        "application/json",
        bytes.NewBufferString(query),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Printf("%+v\n", result)
}
```

### 3. TUI Prototype
```go
package main

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type model struct {
    cursor int
    items  []string
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q":
            return m, tea.Quit
        case "j", "down":
            if m.cursor < len(m.items)-1 {
                m.cursor++
            }
        case "k", "up":
            if m.cursor > 0 {
                m.cursor--
            }
        }
    }
    return m, nil
}

func (m model) View() string {
    s := "Prototype:\n\n"
    for i, item := range m.items {
        if i == m.cursor {
            s += lipgloss.NewStyle().Bold(true).Render("> "+item) + "\n"
        } else {
            s += "  " + item + "\n"
        }
    }
    s += "\nq to quit"
    return s
}

func main() {
    m := model{items: []string{"Item 1", "Item 2", "Item 3"}}
    if _, err := tea.NewProgram(m).Run(); err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)
    }
}
```

### 4. Database Query Test
```go
package main

import (
    "fmt"

    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type TestModel struct {
    ID   uint
    Name string
}

func main() {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        panic(err)
    }

    db.AutoMigrate(&TestModel{})

    db.Create(&TestModel{Name: "test"})

    var result TestModel
    db.First(&result)
    fmt.Printf("Result: %+v\n", result)
}
```

### 5. Decryption Algorithm Test
```go
package main

import (
    "crypto/aes"
    "crypto/cipher"
    "encoding/base64"
    "fmt"
)

func main() {
    // Test decryption logic
    key := []byte("your-key-here...")
    iv := []byte("your-iv-here....")
    encrypted := "base64-encrypted-data"

    data, _ := base64.StdEncoding.DecodeString(encrypted)

    block, _ := aes.NewCipher(key)
    mode := cipher.NewCBCDecrypter(block, iv)
    mode.CryptBlocks(data, data)

    fmt.Printf("Decrypted: %s\n", data)
}
```

## Running Prototypes

```bash
# Run directly
go run main.go

# With dependencies from greg
go run -mod=mod main.go
```

## After Prototyping

1. Run and verify output
2. Explain results
3. Suggest how to integrate into main codebase
4. Ask if user wants to convert to proper implementation

## Cleanup

```bash
rm -rf /tmp/greg-proto
# or
rm scripts/prototypes/main.go
```
