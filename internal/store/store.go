package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Store struct {
	path  string
	State *AppState
}

func New() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".config", "claudex")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Store{
		path: filepath.Join(dir, "state.json"),
	}
	if err := s.Load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.State = DefaultState()
			return nil
		}
		return err
	}
	state := &AppState{}
	if err := json.Unmarshal(data, state); err != nil {
		s.State = DefaultState()
		return nil
	}
	s.State = state
	return nil
}

func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.State, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) FindProject(id string) *Project {
	for i := range s.State.Projects {
		if s.State.Projects[i].ID == id {
			return &s.State.Projects[i]
		}
	}
	return nil
}

// ClaudeSessionExists checks if a Claude session file exists on disk.
// Searches by UUID across all project directories to avoid path encoding issues.
func ClaudeSessionExists(sessionID string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	pattern := filepath.Join(home, ".claude", "projects", "*", sessionID+".jsonl")
	matches, _ := filepath.Glob(pattern)
	return len(matches) > 0
}

func (s *Store) FindThread(projectID, threadID string) *Thread {
	p := s.FindProject(projectID)
	if p == nil {
		return nil
	}
	for i := range p.Threads {
		if p.Threads[i].ID == threadID {
			return &p.Threads[i]
		}
	}
	return nil
}
