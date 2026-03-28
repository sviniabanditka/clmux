package app

import "github.com/sviniabanditka/clmux/internal/store"

// PtyOutputMsg delivers raw PTY output for a specific thread.
type PtyOutputMsg struct {
	ThreadID string
	Data     []byte
}

// ProcessExitedMsg signals that a Claude process has exited.
type ProcessExitedMsg struct {
	ThreadID string
	Err      error
}

// ProjectAddedMsg signals a new project was added.
type ProjectAddedMsg struct {
	Project store.Project
}

// ThreadCreatedMsg signals a new thread was created.
type ThreadCreatedMsg struct {
	ProjectID string
	Thread    store.Thread
}

// ThreadClosedMsg signals a thread was closed.
type ThreadClosedMsg struct {
	ThreadID string
}

// FocusSidebarMsg switches focus to the sidebar.
type FocusSidebarMsg struct{}

// FocusPanelMsg switches focus to the panel.
type FocusPanelMsg struct{}
