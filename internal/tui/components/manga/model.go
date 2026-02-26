package manga

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gorm.io/gorm"

	"github.com/justchokingaround/greg/internal/config"
	"github.com/justchokingaround/greg/internal/database"
	"github.com/justchokingaround/greg/internal/tui/common"
)

type Model struct {
	Config                *config.Config
	DB                    *gorm.DB
	Pages                 []string
	Title                 string
	Chapter               string
	CurrentPage           int
	Width                 int
	Height                int
	ImageContent          string
	Err                   error
	Loading               bool
	ShowHelp              bool
	InputMode             bool
	InputBuffer           string
	ShowNextChapterPrompt bool
	ShowQuitPrompt        bool

	// Metadata for history
	MediaID       string
	EpisodeNumber int
	ProviderName  string
	AniListID     *int
	StatusMessage string
}

type PageRenderedMsg struct {
	Content string
	Err     error
}

func New(cfg *config.Config, db *gorm.DB) Model {
	return Model{
		Config: cfg,
		DB:     db,
	}
}

func (m Model) Init() tea.Cmd {
	// Hide cursor when the manga reader is initialized
	// Also return a batch command to clear screen if needed
	// Use multiple methods to ensure cursor is hidden
	return tea.Batch(tea.HideCursor, tea.ClearScreen)
}

// Define a command to periodically hide cursor
func hideCursorPeriodically() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tea.HideCursor
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if len(m.Pages) > 0 {
			return m, tea.Batch(m.renderPage(), hideCursorPeriodically())
		}

	case PageRenderedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Err = msg.Err
			return m, hideCursorPeriodically()
		}
		m.ImageContent = msg.Content
		m.Err = nil
		// Clear screen and ensure cursor is hidden after image rendering
		return m, tea.Batch(tea.ClearScreen, tea.HideCursor, hideCursorPeriodically())

	case tea.KeyMsg:
		// Handle quit prompt
		if m.ShowQuitPrompt {
			switch msg.String() {
			case "y", "Y", "enter":
				m.updateHistory()
				m.ShowQuitPrompt = false

				var cmds []tea.Cmd
				cmds = append(cmds, func() tea.Msg { return common.MangaQuitMsg{} })

				// Check if completed
				if m.CurrentPage >= len(m.Pages)-1 {
					cmds = append(cmds, func() tea.Msg {
						return common.ChapterCompletedMsg{MediaID: m.MediaID, Chapter: m.EpisodeNumber}
					})
				}
				// Hide cursor when dismissing prompt
				cmds = append(cmds, tea.HideCursor, hideCursorPeriodically())
				return m, tea.Batch(cmds...)
			case "n", "N":
				m.ShowQuitPrompt = false
				// Hide cursor when dismissing prompt
				return m, tea.Batch(func() tea.Msg { return common.MangaQuitMsg{} }, tea.HideCursor, hideCursorPeriodically())
			case "esc", "q":
				m.ShowQuitPrompt = false
				// Hide cursor when dismissing prompt
				return m, tea.Batch(tea.HideCursor, hideCursorPeriodically())
			}
			return m, hideCursorPeriodically()
		}

		// Handle next chapter prompt
		if m.ShowNextChapterPrompt {
			switch msg.String() {
			case "y", "Y", "enter":
				m.ShowNextChapterPrompt = false
				// Hide cursor when dismissing prompt
				return m, tea.Batch(
					func() tea.Msg { return common.NextChapterMsg{} },
					func() tea.Msg { return common.ChapterCompletedMsg{MediaID: m.MediaID, Chapter: m.EpisodeNumber} },
					tea.HideCursor,
					hideCursorPeriodically(),
				)
			case "n", "N", "esc", "q":
				m.ShowNextChapterPrompt = false
				// Hide cursor when dismissing prompt
				return m, tea.Batch(tea.HideCursor, hideCursorPeriodically())
			default:
				return m, hideCursorPeriodically()
			}
		}

		// Handle input mode for page jumping
		if m.InputMode {
			switch msg.String() {
			case "esc":
				m.InputMode = false
				m.InputBuffer = ""
				// Hide cursor when exiting input mode
				return m, tea.Batch(tea.HideCursor, hideCursorPeriodically())
			case "enter":
				if page, err := strconv.Atoi(m.InputBuffer); err == nil {
					// Convert 1-based input to 0-based index
					pageIndex := page - 1
					if pageIndex >= 0 && pageIndex < len(m.Pages) {
						m.CurrentPage = pageIndex
						m.Loading = true
						m.InputMode = false
						m.InputBuffer = ""
						m.updateHistory()
						// Hide cursor when exiting input mode
						return m, tea.Batch(m.renderPage(), tea.HideCursor, hideCursorPeriodically())
					}
				}
				// Invalid input or page number, just exit input mode
				m.InputMode = false
				m.InputBuffer = ""
				// Hide cursor when exiting input mode
				return m, tea.Batch(tea.HideCursor, hideCursorPeriodically())
			case "backspace":
				if len(m.InputBuffer) > 0 {
					m.InputBuffer = m.InputBuffer[:len(m.InputBuffer)-1]
				}
				return m, nil
			default:
				// Only allow digits
				if _, err := strconv.Atoi(msg.String()); err == nil {
					m.InputBuffer += msg.String()
				}
				return m, nil
			}
		}

		switch msg.String() {
		case "?":
			m.ShowHelp = !m.ShowHelp
			if m.ShowHelp {
				// Show cursor isn't necessary for help, but hide when hiding help
				return m, hideCursorPeriodically()
			} else {
				// Hide cursor when hiding help overlay
				return m, tea.Batch(tea.HideCursor, hideCursorPeriodically())
			}
		case "q", "esc", "ctrl+c":
			m.ShowQuitPrompt = true
			return m, hideCursorPeriodically()
		case "g":
			m.InputMode = true
			m.InputBuffer = ""
			// Show cursor for input - don't use periodic hiding in this mode
			return m, tea.ShowCursor
		case "right", "l", "n", " ", "j":
			if m.CurrentPage < len(m.Pages)-1 {
				m.CurrentPage++
				m.Loading = true
				m.updateHistory() // Save progress on page turn
				return m, tea.Batch(m.renderPage(), hideCursorPeriodically())
			} else if len(m.Pages) > 0 {
				// End of chapter
				m.updateHistory() // Ensure completed status is saved
				m.ShowNextChapterPrompt = true
				return m, hideCursorPeriodically()
			}
		case "left", "h", "p", "k":
			if m.CurrentPage > 0 {
				m.CurrentPage--
				m.Loading = true
				m.updateHistory() // Save progress on page turn
				return m, tea.Batch(m.renderPage(), hideCursorPeriodically())
			}
		}
	}
	// For normal operation, keep cursor hidden periodically
	return m, hideCursorPeriodically()
}

func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing..."
	}

	// Styles
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("235")).
		Width(m.Width).
		Align(lipgloss.Center)

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Width(m.Width).
		Align(lipgloss.Center)

	// Header
	headerText := fmt.Sprintf("%s - %s - Page %d/%d", m.Title, m.Chapter, m.CurrentPage+1, len(m.Pages))
	if m.Loading && m.Config != nil && m.Config.UI.ShowLoading {
		headerText += "..."
	}
	header := headerStyle.Render(headerText)

	// Footer
	footerText := "j/k/Space: Next/Prev | g: Go to | ?: Help | q: Quit"
	if m.StatusMessage != "" {
		footerText = m.StatusMessage
	}
	footer := footerStyle.Render(footerText)

	// Calculate available height for image
	// Header and Footer are 1 line each
	// We reserve 2 lines total (Header + Footer) plus 1 line safety margin
	availableHeight := m.Height - 3
	if availableHeight < 1 {
		availableHeight = 1
	}

	var view string
	// Check for overlays first, as they should take precedence
	if m.ShowNextChapterPrompt || m.ShowQuitPrompt || m.ShowHelp {
		// Clear any image content and show a clean overlay
		// Start with a clean slate using ANSI codes to clear the screen
		clearScreen := "\x1b[2J" // Clear entire screen
		moveToTop := "\x1b[H"    // Move cursor to top-left

		if m.ShowHelp {
			helpText := []string{
				"Manga Reader Controls",
				"",
				"  Right/l/n/j/Space : Next Page",
				"  Left/h/p/k        : Previous Page",
				"  g                 : Go to Page",
				"  ?                 : Toggle Help",
				"  q/Esc             : Quit Reader",
			}

			helpBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 2).
				Render(strings.Join(helpText, "\n"))

			view = lipgloss.Place(
				m.Width,
				m.Height,
				lipgloss.Center,
				lipgloss.Center,
				helpBox,
				lipgloss.WithWhitespaceChars(" "),
				lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
			)
		} else if m.ShowNextChapterPrompt {
			promptBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 2).
				Render("End of chapter. Go to next chapter? (y/n)")

			// Create a full-screen centered prompt
			view = lipgloss.Place(
				m.Width,
				m.Height,
				lipgloss.Center,
				lipgloss.Center,
				promptBox,
				lipgloss.WithWhitespaceChars(" "),
				lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
			)
		} else if m.ShowQuitPrompt { // This is the else case for ShowQuitPrompt
			promptBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 2).
				Render("Save progress before quitting? (y/n)")

			// Create a full-screen centered prompt
			view = lipgloss.Place(
				m.Width,
				m.Height,
				lipgloss.Center,
				lipgloss.Center,
				promptBox,
				lipgloss.WithWhitespaceChars(" "),
				lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
			)
		}

		// Add comprehensive terminal reset codes
		terminalReset := "\x1b[?25l" // Hide cursor
		view = clearScreen + moveToTop + terminalReset + view
	} else if m.InputMode {
		// For input mode, we need to make sure we have a proper interface
		// Clear screen and show the input interface with full screen management
		clearScreen := "\x1b[2J" // Clear entire screen
		moveToTop := "\x1b[H"    // Move cursor to top-left

		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Render(fmt.Sprintf("Go to page: %s_", m.InputBuffer))

		// Center overlay on the whole screen
		view = lipgloss.Place(
			m.Width,
			m.Height,
			lipgloss.Center,
			lipgloss.Center,
			inputBox,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
		)

		// Add the clear screen codes to ensure clean display
		view = clearScreen + moveToTop + view
	} else if m.Err != nil {
		// Apply comprehensive cursor hiding to all text modes
		hideCursor := "\x1b[?25l"      // Hide cursor
		resetCursor := "\x1b[?12l"     // Disable cursor blinking
		resetCursorStyle := "\x1b[0 q" // Reset cursor style

		msg := fmt.Sprintf("Error rendering page: %v", m.Err)
		centeredMsg := lipgloss.Place(m.Width, availableHeight, lipgloss.Center, lipgloss.Center, msg)
		// Apply persistent cursor hiding for consistency
		forceHideCursor := "\x1b[?25l\x1b[?25l\x1b[?25l" // Multiple hide commands for persistence
		view = fmt.Sprintf("%s%s%s%s\n%s\n%s%s", header, hideCursor, resetCursor, resetCursorStyle, centeredMsg, footer, forceHideCursor)

		// Add cursor-moving command to avoid interfering with header rendering
		moveCursorAway := "\x1b[999;999H" // Move cursor to far bottom-right corner (likely invisible)
		view += moveCursorAway
	} else if m.ImageContent != "" {
		// Show image (even if loading next page) to prevent UI jumping
		imageContent := strings.Trim(m.ImageContent, "\n\r")

		// Thoroughly hide cursor during image display to prevent blinking cursor artifact
		hideCursor := "\x1b[?25l"      // Hide cursor
		resetCursor := "\x1b[?12l"     // Disable cursor blinking
		resetCursorStyle := "\x1b[0 q" // Reset cursor style

		// Use ANSI escape codes to ensure footer is at the bottom
		// \x1b[H moves cursor to specific position (row;col)
		// \x1b[2K clears the current line (for footer)
		moveToBottom := fmt.Sprintf("\x1b[%d;1H", m.Height)
		clearLine := "\x1b[2K"

		// Combine the header, image content, and footer, then add cursor controls carefully
		// First the header (with cursor hiding), then the image content, then position for footer
		// For image content, ensure multiple cursor-hiding commands are sent to override chafa's potential terminal state changes
		forceHideCursor := "\x1b[?25l\x1b[?25l\x1b[?25l" // Multiple hide commands for persistence
		view = fmt.Sprintf("%s%s%s%s\n%s%s%s%s%s", header, hideCursor, resetCursor, resetCursorStyle, imageContent, moveToBottom, clearLine, footer, forceHideCursor)

		// Add the cursor-moving command AFTER the main content to avoid interfering with header rendering
		// This moves cursor to a position that won't interfere with the visible content
		moveCursorAway := "\x1b[999;999H" // Move cursor to far bottom-right corner (likely invisible)
		view += moveCursorAway
	} else if m.Loading {
		// Hide cursor during loading state as well
		hideCursor := "\x1b[?25l"
		resetCursor := "\x1b[?12l"
		resetCursorStyle := "\x1b[0 q"

		msg := fmt.Sprintf("Loading page %d/%d...", m.CurrentPage+1, len(m.Pages))
		centeredMsg := lipgloss.Place(m.Width, availableHeight, lipgloss.Center, lipgloss.Center, msg)
		// Apply persistent cursor hiding for consistency
		forceHideCursor := "\x1b[?25l\x1b[?25l\x1b[?25l" // Multiple hide commands for persistence
		view = fmt.Sprintf("%s%s%s%s\n%s\n%s%s", header, hideCursor, resetCursor, resetCursorStyle, centeredMsg, footer, forceHideCursor)

		// Add cursor-moving command to avoid interfering with header rendering
		moveCursorAway := "\x1b[999;999H" // Move cursor to far bottom-right corner (likely invisible)
		view += moveCursorAway
	} else {
		// Also hide cursor in other states
		hideCursor := "\x1b[?25l"
		resetCursor := "\x1b[?12l"
		resetCursorStyle := "\x1b[0 q"

		var msg string
		if len(m.Pages) == 0 {
			msg = "No pages loaded."
		} else {
			msg = "Waiting for render..."
		}
		centeredMsg := lipgloss.Place(m.Width, availableHeight, lipgloss.Center, lipgloss.Center, msg)
		// Apply persistent cursor hiding for consistency
		forceHideCursor := "\x1b[?25l\x1b[?25l\x1b[?25l" // Multiple hide commands for persistence
		view = fmt.Sprintf("%s%s%s%s\n%s\n%s%s", header, hideCursor, resetCursor, resetCursorStyle, centeredMsg, footer, forceHideCursor)

		// Add cursor-moving command to avoid interfering with header rendering
		moveCursorAway := "\x1b[999;999H" // Move cursor to far bottom-right corner (likely invisible)
		view += moveCursorAway
	}

	// Ensure we fill the screen height exactly to overwrite old content
	// We target m.Height - 1 lines to avoid scrolling
	// NOTE: We cannot reliably count lines for Sixel/Graphics content, so we skip padding
	// when image content is present to avoid scrolling issues.
	//
	// Update: When ImageContent is present, we use ANSI codes to clear/position, so we skip this.
	// But when showing overlays, we handled that above.
	if m.ImageContent == "" && !m.ShowNextChapterPrompt && !m.ShowQuitPrompt && !m.InputMode && !m.ShowHelp {
		lines := strings.Count(view, "\n") + 1
		if lines < m.Height-1 {
			view += strings.Repeat("\n", (m.Height-1)-lines)
		}
	}

	return view
}

func (m *Model) SetContent(pages []string, title, chapter string, mediaID string, episodeNumber int, providerName string, anilistID *int) {
	m.Pages = pages
	m.Title = title
	m.Chapter = chapter
	m.MediaID = mediaID
	m.EpisodeNumber = episodeNumber
	m.ProviderName = providerName
	m.AniListID = anilistID
	m.CurrentPage = 0
	m.ImageContent = ""
	m.Err = nil
	m.Loading = true
	m.ShowNextChapterPrompt = false
	m.ShowQuitPrompt = false

	// Check history to resume
	if m.DB != nil && m.MediaID != "" {
		var history database.History
		err := m.DB.Where("media_id = ? AND episode = ? AND completed = false", m.MediaID, m.EpisodeNumber).
			Order("watched_at DESC").
			First(&history).Error
		if err == nil && history.Page > 0 && history.Page <= len(m.Pages) {
			m.CurrentPage = history.Page - 1
		}
	}

	// Initial history update - REMOVED to avoid auto-save on load
	// m.updateHistory()
}

func (m *Model) updateHistory() {
	if m.DB == nil || m.MediaID == "" {
		return
	}

	// Calculate progress percentage
	progressPercent := 0.0
	if len(m.Pages) > 0 {
		progressPercent = (float64(m.CurrentPage+1) / float64(len(m.Pages))) * 100.0
	}

	completed := m.CurrentPage >= len(m.Pages)-1

	// Check for existing record
	var existing database.History
	err := m.DB.Where("media_id = ? AND episode = ? AND completed = false", m.MediaID, m.EpisodeNumber).
		Order("watched_at DESC").
		First(&existing).Error

	if err == nil {
		// Update existing
		existing.Page = m.CurrentPage + 1
		existing.TotalPages = len(m.Pages)
		existing.ProgressPercent = progressPercent
		existing.WatchedAt = time.Now()
		existing.Completed = completed
		m.DB.Save(&existing)
	} else {
		// Create new
		history := database.History{
			MediaID:         m.MediaID,
			MediaTitle:      m.Title,
			MediaType:       "manga",
			Episode:         m.EpisodeNumber,
			Page:            m.CurrentPage + 1,
			TotalPages:      len(m.Pages),
			ProgressPercent: progressPercent,
			WatchedAt:       time.Now(),
			Completed:       completed,
			ProviderName:    m.ProviderName,
			AniListID:       m.AniListID,
		}
		m.DB.Create(&history)
	}
}

func (m *Model) RenderCurrentPage() tea.Cmd {
	return m.renderPage()
}

func (m *Model) renderPage() tea.Cmd {
	if len(m.Pages) == 0 {
		return nil
	}
	url := m.Pages[m.CurrentPage]
	width := m.Width
	height := m.Height

	// If dimensions are not set, default to a reasonable size or wait
	if width == 0 || height == 0 {
		width = 80
		height = 24
	}

	// Calculate available height for image (subtract header/footer)
	// Header and Footer are 1 line each
	// We reserve 2 lines total (Header + Footer) plus 1 line safety margin
	availableHeight := height - 3
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Get display method from config
	method := "sixel" // Default
	if m.Config != nil && m.Config.UI.MangaMethod != "" {
		method = m.Config.UI.MangaMethod
	}

	// Detect if we're running in Ghostty terminal emulator and override to Kitty if using Sixel
	// This is because Ghostty doesn't support Sixel graphics properly
	if method == "sixel" {
		termEnv := os.Getenv("TERM")
		if strings.Contains(termEnv, "ghostty") {
			method = "kitty"
		}
	}

	return func() tea.Msg {
		// Create a temp file
		tmpFile, err := os.CreateTemp("", "greg-manga-*.png")
		if err != nil {
			return PageRenderedMsg{Err: fmt.Errorf("failed to create temp file: %w", err)}
		}
		tmpFileName := tmpFile.Name()
		defer func() { _ = os.Remove(tmpFileName) }()

		// Download the image
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return PageRenderedMsg{Err: fmt.Errorf("failed to create request: %w", err)}
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Referer", "https://comix.to/")

		resp, err := client.Do(req)
		if err != nil {
			return PageRenderedMsg{Err: fmt.Errorf("failed to download image: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return PageRenderedMsg{Err: fmt.Errorf("image download returned status %d", resp.StatusCode)}
		}

		downloadFile, err := os.CreateTemp("", "greg-manga-download-*.img")
		if err != nil {
			return PageRenderedMsg{Err: fmt.Errorf("failed to create download temp file: %w", err)}
		}
		downloadFileName := downloadFile.Name()
		defer func() { _ = os.Remove(downloadFileName) }()

		_, err = io.Copy(downloadFile, resp.Body)
		if err != nil {
			return PageRenderedMsg{Err: fmt.Errorf("failed to save image: %w", err)}
		}
		_ = downloadFile.Close()

		// Convert image to PNG if needed (WebP, AVIF, etc.)
		convertCmd := exec.Command("magick", downloadFileName, tmpFileName)
		if err := convertCmd.Run(); err != nil {
			// If convert fails, try using the original file
			tmpFileName = downloadFileName
		}

		_ = tmpFile.Close()

		// Run chafa
		// We set --size to the available area
		// We set --view-size to the available area
		// We set --align mid,mid to center it
		args := []string{
			"-s", fmt.Sprintf("%dx%d", width, availableHeight),
			"--view-size", fmt.Sprintf("%dx%d", width, availableHeight),
			"--align", "mid,mid",
		}

		// Add format flag based on config
		switch method {
		case "sixel":
			args = append(args, "-f", "sixel")
		case "kitty":
			args = append(args, "-f", "kitty")
		case "iterm":
			args = append(args, "-f", "iterm")
		}
		// "symbols" or others will use default (no -f flag or specific one if needed, but chafa defaults to symbols)

		// Add file path
		args = append(args, tmpFileName)

		cmd := exec.Command("chafa", args...)
		output, err := cmd.Output()
		if err != nil {
			return PageRenderedMsg{Err: fmt.Errorf("chafa failed: %w", err)}
		}

		return PageRenderedMsg{Content: strings.Trim(string(output), "\n\r\t ")}
	}
}
