package panel

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sviniabanditka/clmux/internal/theme"
)

type Model struct {
	width    int
	height   int
	focused  bool
	content  string
	threadID string
}

func New() Model {
	return Model{}
}

func (m *Model) SetFocused(f bool) { m.focused = f }
func (m *Model) SetSize(w, h int)  { m.width = w; m.height = h }
func (m *Model) SetContent(c string) { m.content = c }
func (m *Model) SetThread(id string) { m.threadID = id }
func (m Model) ThreadID() string     { return m.threadID }
func (m Model) Init() tea.Cmd        { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	if m.threadID == "" {
		return m.renderPlaceholder()
	}
	return m.renderContent()
}

func (m Model) renderPlaceholder() string {
	msg := theme.PlaceholderStyle.Render("Select or create a thread to start")
	pad := m.height / 2
	var b strings.Builder
	for i := 0; i < pad; i++ {
		b.WriteString("\n")
	}
	msgWidth := lipgloss.Width(msg)
	if msgWidth < m.width {
		b.WriteString(strings.Repeat(" ", (m.width-msgWidth)/2))
	}
	b.WriteString(msg)
	return b.String()
}

func (m Model) renderContent() string {
	if m.content == "" {
		return theme.DimStyle.Render("  Starting session...")
	}
	return m.content
}
