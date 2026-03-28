package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/sviniabanditka/claudex/internal/modal"
	"github.com/sviniabanditka/claudex/internal/panel"
	"github.com/sviniabanditka/claudex/internal/session"
	"github.com/sviniabanditka/claudex/internal/sidebar"
	"github.com/sviniabanditka/claudex/internal/store"
)

type focus int

const (
	focusSidebar focus = iota
	focusPanel
)

type Model struct {
	sidebar  sidebar.Model
	panel    panel.Model
	modal    modal.Model
	store    *store.Store
	manager  *session.Manager
	focus    focus
	width    int
	height   int
	inputBuf map[string][]byte
	quitting bool
}

func New(s *store.Store) Model {
	sb := sidebar.New(s)
	sb.SetFocused(true)
	if s.State.ActiveThread != "" {
		sb.SetActiveThread(s.State.ActiveThread)
	}
	p := panel.New()
	m := modal.New()
	mgr := session.NewManager()

	return Model{
		sidebar:  sb,
		panel:    p,
		modal:    m,
		store:    s,
		manager:  mgr,
		focus:    focusSidebar,
		inputBuf: make(map[string][]byte),
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.manager.SetProgram(p)
}

func (m *Model) Init() tea.Cmd {
	return m.modal.Init()
}

func (m *Model) panelCols() uint16 {
	sw := m.store.State.SidebarWidth
	if sw <= 0 {
		sw = 44
	}
	cols := m.width - sw - 1 // sidebar border = 1
	if cols < 10 {
		cols = 80
	}
	return uint16(cols)
}

func (m *Model) panelRows() uint16 {
	if m.height < 1 {
		return 24
	}
	return uint16(m.height)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Loading modal — only allow spinner ticks and loading done
	if m.modal.IsLoading() {
		switch msg := msg.(type) {
		case modal.SpinnerTickMsg:
			newModal, cmd := m.modal.Update(msg)
			m.modal = newModal
			return m, cmd
		case modal.LoadingDoneMsg:
			m.modal.Hide()
			return m.handleLoadingDone(msg)
		case tea.WindowSizeMsg:
			m.handleResize(msg)
			return m, nil
		}
		return m, nil
	}

	// Regular modal
	if m.modal.Active() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			newModal, cmd := m.modal.Update(msg)
			m.modal = newModal
			return m, cmd
		case modal.SubmitMsg:
			return m.handleModalSubmit(msg)
		case modal.CancelMsg:
			return m, nil
		case tea.WindowSizeMsg:
			m.handleResize(msg)
			return m, nil
		}
		newModal, cmd := m.modal.Update(msg)
		m.modal = newModal
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleResize(msg)
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, Keys.Quit) {
			return m.startQuit()
		}
		if key.Matches(msg, Keys.ToggleFocus) {
			m.toggleFocus()
			return m, nil
		}
		if m.focus == focusPanel && m.panel.ThreadID() != "" {
			if m.manager.IsRunning(m.panel.ThreadID()) {
				raw := keyToBytes(msg)
				if raw != nil {
					m.manager.Write(m.panel.ThreadID(), raw)
					m.trackInput(m.panel.ThreadID(), msg)
				}
				return m, nil
			}
		}

	case tea.MouseMsg:
		sidebarWidth := m.store.State.SidebarWidth
		if sidebarWidth <= 0 {
			sidebarWidth = 44
		}
		if msg.X < sidebarWidth {
			if m.focus != focusSidebar {
				m.focus = focusSidebar
				m.sidebar.SetFocused(true)
				m.panel.SetFocused(false)
			}
			newSidebar, cmd := m.sidebar.Update(msg)
			m.sidebar = newSidebar
			return m, cmd
		} else {
			if m.focus != focusPanel {
				m.focus = focusPanel
				m.sidebar.SetFocused(false)
				m.panel.SetFocused(true)
			}
			if m.panel.ThreadID() != "" && m.manager.IsRunning(m.panel.ThreadID()) {
				if msg.Button == tea.MouseButtonWheelUp {
					m.manager.Write(m.panel.ThreadID(), []byte{27, '[', '5', '~'})
				} else if msg.Button == tea.MouseButtonWheelDown {
					m.manager.Write(m.panel.ThreadID(), []byte{27, '[', '6', '~'})
				}
			}
			return m, nil
		}

	case sidebar.AddProjectMsg:
		m.modal.Show(modal.ModalAddProject, "Add Project", "/path/to/project")
		m.modal.SetSize(m.width, m.height)
		return m, nil

	case sidebar.AddThreadMsg:
		return m.addThread(msg.ProjectID)

	case sidebar.ToggleCollapseMsg:
		p := m.store.FindProject(msg.ProjectID)
		if p != nil {
			p.Collapsed = !p.Collapsed
			m.sidebar.Rebuild()
			m.store.Save()
		}
		return m, nil

	case sidebar.SelectThreadMsg:
		return m.selectThread(msg.ProjectID, msg.ThreadID)

	case sidebar.CloseThreadMsg:
		return m.startCloseThread(msg.ProjectID, msg.ThreadID)

	case sidebar.RenameThreadMsg:
		t := m.store.FindThread(msg.ProjectID, msg.ThreadID)
		if t != nil {
			m.modal.ShowWithContext(modal.ModalRenameThread, "Rename Thread", t.Name, msg.ProjectID, msg.ThreadID)
			m.modal.SetSize(m.width, m.height)
		}
		return m, nil

	case sidebar.DeleteProjectMsg:
		p := m.store.FindProject(msg.ProjectID)
		name := "this project"
		if p != nil {
			name = "project \"" + p.Name + "\""
		}
		m.modal.ShowConfirm("Delete Project", "Are you sure you want to delete "+name+"?", msg.ProjectID, "")
		m.modal.SetSize(m.width, m.height)
		return m, nil

	case sidebar.DeleteThreadMsg:
		t := m.store.FindThread(msg.ProjectID, msg.ThreadID)
		name := "this thread"
		if t != nil {
			name = "thread \"" + t.Name + "\""
		}
		m.modal.ShowConfirm("Delete Thread", "Are you sure you want to delete "+name+"?", msg.ProjectID, msg.ThreadID)
		m.modal.SetSize(m.width, m.height)
		return m, nil

	case modal.SubmitMsg:
		return m.handleModalSubmit(msg)

	case session.PtyOutputMsg:
		if msg.ThreadID == m.panel.ThreadID() {
			rendered := m.manager.RenderTerm(msg.ThreadID)
			m.panel.SetContent(rendered)
		}
		t := m.findThreadInStore(msg.ThreadID)
		if t != nil && t.Status != store.ThreadOpen {
			t.Status = store.ThreadOpen
			t.UpdatedAt = time.Now()
			m.sidebar.Rebuild()
			m.store.Save()
		}
		return m, nil

	case session.ProcessExitedMsg:
		t := m.findThreadInStore(msg.ThreadID)
		if t != nil {
			t.Status = store.ThreadSuspended
			t.UpdatedAt = time.Now()
			m.sidebar.Rebuild()
			m.store.Save()
		}
		return m, nil

	case modal.LoadingDoneMsg:
		m.modal.Hide()
		return m.handleLoadingDone(msg)
	}

	if m.focus == focusSidebar {
		newSidebar, cmd := m.sidebar.Update(msg)
		m.sidebar = newSidebar
		cmds = append(cmds, cmd)
	} else {
		newPanel, cmd := m.panel.Update(msg)
		m.panel = newPanel
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	sidebarView := m.sidebar.View()
	panelView := m.panel.View()
	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, panelView)

	if m.modal.Active() {
		return m.modal.View()
	}
	return layout
}

// ── Resize ──

func (m *Model) handleResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	sidebarWidth := m.store.State.SidebarWidth
	if sidebarWidth <= 0 {
		sidebarWidth = 44
	}
	panelWidth := m.width - sidebarWidth - 1 // border = 1 col

	m.sidebar.SetSize(sidebarWidth, m.height)
	m.panel.SetSize(panelWidth, m.height)
	m.modal.SetSize(m.width, m.height)

	m.manager.ResizeAll(m.panelRows(), m.panelCols())
}

// ── Focus ──

func (m *Model) toggleFocus() {
	if m.focus == focusSidebar {
		m.focus = focusPanel
		m.sidebar.SetFocused(false)
		m.panel.SetFocused(true)
	} else {
		m.focus = focusSidebar
		m.sidebar.SetFocused(true)
		m.panel.SetFocused(false)
	}
}

// ── Thread lifecycle ──

func (m *Model) selectThread(projectID, threadID string) (tea.Model, tea.Cmd) {
	m.store.State.ActiveProject = projectID
	m.store.State.ActiveThread = threadID
	m.panel.SetThread(threadID)
	m.sidebar.SetActiveThread(threadID)
	m.store.Save()

	rendered := m.manager.RenderTerm(threadID)
	if rendered != "" {
		m.panel.SetContent(rendered)
	} else {
		m.panel.SetContent("")
	}

	if !m.manager.IsRunning(threadID) {
		p := m.store.FindProject(projectID)
		t := m.store.FindThread(projectID, threadID)
		if p != nil && t != nil {
			resume := store.ClaudeSessionExists(t.SessionID)
			rows := m.panelRows()
			cols := m.panelCols()
			m.manager.Open(threadID, t.SessionID, p.Path, t.Name, resume, rows, cols)
			t.Status = store.ThreadOpen
			t.UpdatedAt = time.Now()
			m.sidebar.Rebuild()
			m.store.Save()
		}
	}

	m.focus = focusPanel
	m.sidebar.SetFocused(false)
	m.panel.SetFocused(true)

	return m, nil
}

// startCloseThread shows loading modal and closes in background.
func (m *Model) startCloseThread(projectID, threadID string) (tea.Model, tea.Cmd) {
	if !m.manager.IsRunning(threadID) {
		// Not running — just update status
		t := m.store.FindThread(projectID, threadID)
		if t != nil {
			t.Status = store.ThreadSuspended
			t.UpdatedAt = time.Now()
		}
		m.sidebar.Rebuild()
		m.store.Save()
		return m, nil
	}

	m.modal.ShowLoading(
		"Closing session",
		"Saving conversation and stopping Claude...",
		"close_thread", projectID, threadID,
	)
	m.modal.SetSize(m.width, m.height)

	pid, tid := projectID, threadID
	mgr := m.manager
	return m, tea.Batch(modal.SpinnerTick(), func() tea.Msg {
		mgr.Close(tid)
		return modal.LoadingDoneMsg{Action: "close_thread", ProjectID: pid, ThreadID: tid}
	})
}

// startQuit shows loading modal and stops all sessions in background.
func (m *Model) startQuit() (tea.Model, tea.Cmd) {
	// Check if any sessions are running
	hasRunning := false
	for _, p := range m.store.State.Projects {
		for _, t := range p.Threads {
			if m.manager.IsRunning(t.ID) {
				hasRunning = true
				break
			}
		}
	}

	if !hasRunning {
		return m, tea.Quit
	}

	m.quitting = true
	m.modal.ShowLoading(
		"Shutting down",
		"Saving all sessions and stopping Claude processes...",
		"quit", "", "",
	)
	m.modal.SetSize(m.width, m.height)

	mgr := m.manager
	return m, tea.Batch(modal.SpinnerTick(), func() tea.Msg {
		mgr.StopAll()
		return modal.LoadingDoneMsg{Action: "quit"}
	})
}

func (m *Model) handleLoadingDone(msg modal.LoadingDoneMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case "close_thread":
		t := m.store.FindThread(msg.ProjectID, msg.ThreadID)
		if t != nil {
			t.Status = store.ThreadSuspended
			t.UpdatedAt = time.Now()
		}
		if m.panel.ThreadID() == msg.ThreadID {
			m.panel.SetThread("")
			m.panel.SetContent("")
			m.store.State.ActiveThread = ""
			m.sidebar.SetActiveThread("")
		}
		m.sidebar.Rebuild()
		m.store.Save()
		return m, nil

	case "delete_thread":
		if m.panel.ThreadID() == msg.ThreadID {
			m.panel.SetThread("")
			m.panel.SetContent("")
			m.store.State.ActiveThread = ""
			m.sidebar.SetActiveThread("")
		}
		p := m.store.FindProject(msg.ProjectID)
		if p != nil {
			for i, t := range p.Threads {
				if t.ID == msg.ThreadID {
					p.Threads = append(p.Threads[:i], p.Threads[i+1:]...)
					break
				}
			}
		}
		m.sidebar.Rebuild()
		m.store.Save()
		return m, nil

	case "delete_project":
		if m.store.State.ActiveProject == msg.ProjectID {
			m.panel.SetThread("")
			m.panel.SetContent("")
			m.store.State.ActiveProject = ""
			m.store.State.ActiveThread = ""
			m.sidebar.SetActiveThread("")
		}
		projects := m.store.State.Projects
		for i, proj := range projects {
			if proj.ID == msg.ProjectID {
				m.store.State.Projects = append(projects[:i], projects[i+1:]...)
				break
			}
		}
		m.sidebar.Rebuild()
		m.store.Save()
		return m, nil

	case "quit":
		return m, tea.Quit
	}
	return m, nil
}

// ── Modal ──

func (m *Model) handleModalSubmit(msg modal.SubmitMsg) (tea.Model, tea.Cmd) {
	switch msg.Kind {
	case modal.ModalAddProject:
		return m.addProject(msg.Value)
	case modal.ModalRenameThread:
		t := m.store.FindThread(msg.ProjectID, msg.ThreadID)
		if t != nil {
			t.Name = msg.Value
			t.UpdatedAt = time.Now()
			m.sidebar.Rebuild()
			m.store.Save()
		}
		return m, nil
	case modal.ModalConfirmDelete:
		return m.startDelete(msg.ProjectID, msg.ThreadID)
	}
	return m, nil
}

// startDelete shows loading and runs delete in background.
func (m *Model) startDelete(projectID, threadID string) (tea.Model, tea.Cmd) {
	if threadID != "" {
		// Delete thread
		t := m.store.FindThread(projectID, threadID)
		name := "thread"
		if t != nil {
			name = t.Name
		}
		running := m.manager.IsRunning(threadID)

		m.modal.ShowLoading(
			"Deleting thread",
			"Stopping \""+name+"\" and cleaning up...",
			"delete_thread", projectID, threadID,
		)
		m.modal.SetSize(m.width, m.height)

		mgr := m.manager
		pid, tid := projectID, threadID
		return m, tea.Batch(modal.SpinnerTick(), func() tea.Msg {
			if running {
				mgr.Close(tid)
			}
			return modal.LoadingDoneMsg{Action: "delete_thread", ProjectID: pid, ThreadID: tid}
		})
	}

	// Delete project
	p := m.store.FindProject(projectID)
	name := "project"
	if p != nil {
		name = p.Name
	}

	m.modal.ShowLoading(
		"Deleting project",
		"Stopping sessions in \""+name+"\" and cleaning up...",
		"delete_project", projectID, "",
	)
	m.modal.SetSize(m.width, m.height)

	// Collect running thread IDs
	var runningThreads []string
	if p != nil {
		for _, t := range p.Threads {
			if m.manager.IsRunning(t.ID) {
				runningThreads = append(runningThreads, t.ID)
			}
		}
	}

	mgr := m.manager
	pid := projectID
	return m, tea.Batch(modal.SpinnerTick(), func() tea.Msg {
		for _, tid := range runningThreads {
			mgr.Close(tid)
		}
		return modal.LoadingDoneMsg{Action: "delete_project", ProjectID: pid}
	})
}

func (m *Model) addProject(path string) (tea.Model, tea.Cmd) {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		m.modal.Show(modal.ModalAddProject, "Add Project", "/path/to/project")
		m.modal.SetError("Directory does not exist: " + path)
		m.modal.SetSize(m.width, m.height)
		return m, nil
	}

	absPath, _ := filepath.Abs(path)
	name := filepath.Base(absPath)

	project := store.Project{
		ID:        uuid.New().String(),
		Name:      name,
		Path:      absPath,
		Threads:   []store.Thread{},
		CreatedAt: time.Now(),
	}

	m.store.State.Projects = append(m.store.State.Projects, project)
	m.sidebar.Rebuild()
	m.store.Save()
	return m, nil
}

func (m *Model) addThread(projectID string) (tea.Model, tea.Cmd) {
	p := m.store.FindProject(projectID)
	if p == nil {
		return m, nil
	}

	threadNum := len(p.Threads) + 1
	thread := store.Thread{
		ID:        uuid.New().String(),
		Name:      fmt.Sprintf("Thread %d", threadNum),
		SessionID: uuid.New().String(),
		Status:    store.ThreadClosed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	p.Threads = append(p.Threads, thread)
	p.Collapsed = false
	m.sidebar.Rebuild()
	m.store.Save()
	return m, nil
}

// ── Auto-naming ──

func (m *Model) trackInput(threadID string, msg tea.KeyMsg) {
	t := m.findThreadInStore(threadID)
	if t == nil {
		return
	}
	if !strings.HasPrefix(t.Name, "Thread ") {
		return
	}

	buf := m.inputBuf[threadID]

	if msg.Type == tea.KeyEnter && len(buf) > 0 {
		name := strings.TrimSpace(string(buf))
		if name != "" {
			if utf8.RuneCountInString(name) > 40 {
				runes := []rune(name)
				name = string(runes[:40]) + "..."
			}
			t.Name = name
			t.UpdatedAt = time.Now()
			m.sidebar.Rebuild()
			m.store.Save()
		}
		delete(m.inputBuf, threadID)
		return
	}

	if msg.Type == tea.KeyRunes && len(buf) < 200 {
		buf = append(buf, []byte(string(msg.Runes))...)
		m.inputBuf[threadID] = buf
	} else if msg.Type == tea.KeyBackspace && len(buf) > 0 {
		s := string(buf)
		runes := []rune(s)
		if len(runes) > 0 {
			m.inputBuf[threadID] = []byte(string(runes[:len(runes)-1]))
		}
	}
}

// ── Helpers ──

func (m *Model) findThreadInStore(threadID string) *store.Thread {
	for i := range m.store.State.Projects {
		for j := range m.store.State.Projects[i].Threads {
			if m.store.State.Projects[i].Threads[j].ID == threadID {
				return &m.store.State.Projects[i].Threads[j]
			}
		}
	}
	return nil
}

func keyToBytes(msg tea.KeyMsg) []byte {
	switch msg.Type {
	case tea.KeyEnter:
		return []byte{'\r'}
	case tea.KeyTab:
		return []byte{'\t'}
	case tea.KeyBackspace:
		return []byte{127}
	case tea.KeyEscape:
		return []byte{27}
	case tea.KeyUp:
		return []byte{27, '[', 'A'}
	case tea.KeyDown:
		return []byte{27, '[', 'B'}
	case tea.KeyRight:
		return []byte{27, '[', 'C'}
	case tea.KeyLeft:
		return []byte{27, '[', 'D'}
	case tea.KeyHome:
		return []byte{27, '[', 'H'}
	case tea.KeyEnd:
		return []byte{27, '[', 'F'}
	case tea.KeyDelete:
		return []byte{27, '[', '3', '~'}
	case tea.KeyPgUp:
		return []byte{27, '[', '5', '~'}
	case tea.KeyPgDown:
		return []byte{27, '[', '6', '~'}
	case tea.KeyCtrlA:
		return []byte{1}
	case tea.KeyCtrlD:
		return []byte{4}
	case tea.KeyCtrlE:
		return []byte{5}
	case tea.KeyCtrlK:
		return []byte{11}
	case tea.KeyCtrlL:
		return []byte{12}
	case tea.KeyCtrlU:
		return []byte{21}
	case tea.KeyCtrlW:
		return []byte{23}
	case tea.KeySpace:
		return []byte{' '}
	case tea.KeyRunes:
		if len(msg.Runes) > 0 {
			return []byte(string(msg.Runes))
		}
	}
	return nil
}
