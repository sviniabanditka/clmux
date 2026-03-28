package modal

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sviniabanditka/claudex/internal/theme"
)

type Kind int

const (
	ModalNone Kind = iota
	ModalAddProject
	ModalRenameThread
	ModalConfirmDelete
)

type SubmitMsg struct {
	Kind      Kind
	Value     string
	ProjectID string
	ThreadID  string
}

type CancelMsg struct{}

type Model struct {
	kind      Kind
	title     string
	message   string // for confirm dialogs
	input     textinput.Model
	projectID string
	threadID  string
	width     int
	height    int
	err       string
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 256
	return Model{kind: ModalNone, input: ti}
}

func (m Model) Active() bool {
	return m.kind != ModalNone
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) Show(kind Kind, title, placeholder string) {
	m.kind = kind
	m.title = title
	m.message = ""
	m.input.Placeholder = placeholder
	m.input.SetValue("")
	m.input.Focus()
	m.err = ""
	m.projectID = ""
	m.threadID = ""
}

func (m *Model) ShowWithContext(kind Kind, title, placeholder, projectID, threadID string) {
	m.Show(kind, title, placeholder)
	m.projectID = projectID
	m.threadID = threadID
}

func (m *Model) ShowConfirm(title, message, projectID, threadID string) {
	m.kind = ModalConfirmDelete
	m.title = title
	m.message = message
	m.input.Blur()
	m.err = ""
	m.projectID = projectID
	m.threadID = threadID
}

func (m *Model) SetError(err string) {
	m.err = err
}

func (m *Model) Hide() {
	m.kind = ModalNone
	m.input.Blur()
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.Active() {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.kind == ModalConfirmDelete {
			return m.updateConfirm(msg)
		}
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				m.err = "Value cannot be empty"
				return m, nil
			}
			kind := m.kind
			pid := m.projectID
			tid := m.threadID
			m.Hide()
			return m, func() tea.Msg {
				return SubmitMsg{Kind: kind, Value: val, ProjectID: pid, ThreadID: tid}
			}
		case "esc":
			m.Hide()
			return m, func() tea.Msg { return CancelMsg{} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		kind := m.kind
		pid := m.projectID
		tid := m.threadID
		m.Hide()
		return m, func() tea.Msg {
			return SubmitMsg{Kind: kind, Value: "confirm", ProjectID: pid, ThreadID: tid}
		}
	case "n", "esc", "q":
		m.Hide()
		return m, func() tea.Msg { return CancelMsg{} }
	}
	return m, nil
}

func (m Model) View() string {
	if !m.Active() {
		return ""
	}

	modalWidth := 50
	if m.width > 0 && m.width < modalWidth+10 {
		modalWidth = m.width - 10
	}

	var b strings.Builder

	if m.kind == ModalConfirmDelete {
		b.WriteString(theme.ErrorStyle.Render(m.title))
		b.WriteString("\n\n")
		b.WriteString(theme.MutedStyle.Render(m.message))
		b.WriteString("\n\n")
		b.WriteString(theme.HintKeyStyle.Render("y") + theme.HintDescStyle.Render("/enter: confirm   "))
		b.WriteString(theme.HintKeyStyle.Render("n") + theme.HintDescStyle.Render("/esc: cancel"))
	} else {
		b.WriteString(theme.HeaderStyle.Render(m.title))
		b.WriteString("\n\n")
		b.WriteString(m.input.View())
		if m.err != "" {
			b.WriteString("\n")
			b.WriteString(theme.ErrorStyle.Render(m.err))
		}
		b.WriteString("\n\n")
		b.WriteString(theme.MutedStyle.Render("enter: confirm  esc: cancel"))
	}

	content := theme.ModalStyle.Width(modalWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
