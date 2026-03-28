package session

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sviniabanditka/clmux/internal/panel"
)

// PtyOutputMsg delivers raw PTY output for a specific thread.
type PtyOutputMsg struct {
	ThreadID string
}

// ProcessExitedMsg signals that a Claude process has exited.
type ProcessExitedMsg struct {
	ThreadID string
	Err      error
}

// Manager handles multiple Claude PTY sessions.
type Manager struct {
	processes map[string]*Process
	terms     map[string]*panel.VTerm
	mu        sync.RWMutex
	program   *tea.Program
	rows      uint16
	cols      uint16
}

func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*Process),
		terms:     make(map[string]*panel.VTerm),
		rows:      24,
		cols:      80,
	}
}

func (m *Manager) SetProgram(p *tea.Program) {
	m.program = p
}

// Open starts or resumes a Claude session for the given thread.
func (m *Manager) Open(threadID, sessionID, cwd, name string, resume bool, rows, cols uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Already running
	if proc, ok := m.processes[threadID]; ok && proc.IsRunning() {
		return nil
	}

	// Always create a fresh virtual terminal for a new process
	m.terms[threadID] = panel.NewVTerm(int(cols), int(rows))

	m.rows = rows
	m.cols = cols

	proc := NewProcess(ProcessOpts{
		SessionID: sessionID,
		ThreadID:  threadID,
		Cwd:       cwd,
		Resume:    resume,
		Name:      name,
		OnOutput: func(tid string, data []byte) {
			m.mu.RLock()
			term := m.terms[tid]
			m.mu.RUnlock()
			if term != nil {
				term.Write(data)
			}
			if m.program != nil {
				m.program.Send(PtyOutputMsg{ThreadID: tid})
			}
		},
		OnExit: func(tid string, err error) {
			if m.program != nil {
				m.program.Send(ProcessExitedMsg{ThreadID: tid, Err: err})
			}
		},
	})

	if err := proc.Start(rows, cols); err != nil {
		return err
	}

	m.processes[threadID] = proc
	return nil
}

// Close stops a running session.
func (m *Manager) Close(threadID string) {
	// Extract process under lock, then stop outside lock to avoid deadlock
	// (reader goroutines need RLock in onOutput callbacks)
	m.mu.Lock()
	proc, ok := m.processes[threadID]
	if ok {
		delete(m.processes, threadID)
	}
	m.mu.Unlock()

	if ok {
		proc.Stop()
	}
}

// Write sends input to the active process.
func (m *Manager) Write(threadID string, data []byte) error {
	m.mu.RLock()
	proc, ok := m.processes[threadID]
	m.mu.RUnlock()
	if !ok || !proc.IsRunning() {
		return nil
	}
	_, err := proc.Write(data)
	return err
}

// Resize updates the PTY size and virtual terminal for a session.
func (m *Manager) Resize(threadID string, rows, cols uint16) {
	m.mu.RLock()
	proc, ok := m.processes[threadID]
	term := m.terms[threadID]
	m.mu.RUnlock()
	if ok {
		proc.Resize(rows, cols)
	}
	if term != nil {
		term.Resize(int(cols), int(rows))
	}
}

// ResizeAll updates all PTY sizes and virtual terminals.
func (m *Manager) ResizeAll(rows, cols uint16) {
	m.mu.Lock()
	m.rows = rows
	m.cols = cols
	procs := make(map[string]*Process, len(m.processes))
	terms := make(map[string]*panel.VTerm, len(m.terms))
	for k, v := range m.processes {
		procs[k] = v
	}
	for k, v := range m.terms {
		terms[k] = v
	}
	m.mu.Unlock()

	for _, proc := range procs {
		proc.Resize(rows, cols)
	}
	for _, term := range terms {
		term.Resize(int(cols), int(rows))
	}
}

// RenderTerm returns the rendered content of a thread's virtual terminal.
func (m *Manager) RenderTerm(threadID string) string {
	m.mu.RLock()
	term, ok := m.terms[threadID]
	m.mu.RUnlock()
	if !ok {
		return ""
	}
	return term.Render()
}

// IsRunning checks if a thread's process is active.
func (m *Manager) IsRunning(threadID string) bool {
	m.mu.RLock()
	proc, ok := m.processes[threadID]
	m.mu.RUnlock()
	return ok && proc.IsRunning()
}

// StopAll gracefully stops all running processes.
func (m *Manager) StopAll() {
	m.mu.Lock()
	procs := make(map[string]*Process, len(m.processes))
	for k, v := range m.processes {
		procs[k] = v
	}
	m.processes = make(map[string]*Process)
	m.mu.Unlock()

	for _, proc := range procs {
		proc.Stop()
	}
}
