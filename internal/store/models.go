package store

import "time"

type ThreadStatus string

const (
	ThreadOpen      ThreadStatus = "open"
	ThreadSuspended ThreadStatus = "suspended"
	ThreadClosed    ThreadStatus = "closed"
)

type Thread struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	SessionID string       `json:"session_id"`
	Status    ThreadStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type Project struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Threads   []Thread `json:"threads"`
	Collapsed bool     `json:"collapsed"`
	CreatedAt time.Time `json:"created_at"`
}

type AppState struct {
	Projects      []Project `json:"projects"`
	ActiveProject string    `json:"active_project"`
	ActiveThread  string    `json:"active_thread"`
	SidebarWidth  int       `json:"sidebar_width"`
}

func DefaultState() *AppState {
	return &AppState{
		Projects:     []Project{},
		SidebarWidth: 44,
	}
}
