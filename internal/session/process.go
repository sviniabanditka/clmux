package session

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type Process struct {
	SessionID string
	ThreadID  string
	Cmd       *exec.Cmd
	ptmx      *os.File
	mu        sync.Mutex
	running   bool
	exited    chan struct{} // closed when process exits
	onOutput  func(threadID string, data []byte)
	onExit    func(threadID string, err error)
}

type ProcessOpts struct {
	SessionID string
	ThreadID  string
	Cwd       string
	Resume    bool
	Name      string
	OnOutput  func(threadID string, data []byte)
	OnExit    func(threadID string, err error)
}

func NewProcess(opts ProcessOpts) *Process {
	args := []string{}
	if opts.Resume && opts.SessionID != "" {
		args = append(args, "--resume", opts.SessionID)
	} else {
		args = append(args, "--session-id", opts.SessionID)
	}
	if opts.Name != "" && !opts.Resume {
		args = append(args, "--name", opts.Name)
	}

	cmd := exec.Command("claude", args...)
	cmd.Dir = opts.Cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	return &Process{
		SessionID: opts.SessionID,
		ThreadID:  opts.ThreadID,
		Cmd:       cmd,
		exited:    make(chan struct{}),
		onOutput:  opts.OnOutput,
		onExit:    opts.OnExit,
	}
}

func (p *Process) Start(rows, cols uint16) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	size := &pty.Winsize{Rows: rows, Cols: cols}
	ptmx, err := pty.StartWithSize(p.Cmd, size)
	if err != nil {
		return err
	}
	p.ptmx = ptmx
	p.running = true

	// Reader goroutine — the single place that calls Cmd.Wait()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := p.ptmx.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				if p.onOutput != nil {
					p.onOutput(p.ThreadID, data)
				}
			}
			if err != nil {
				break
			}
		}

		exitErr := p.Cmd.Wait()

		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		close(p.exited)

		if p.onExit != nil {
			p.onExit(p.ThreadID, exitErr)
		}
	}()

	return nil
}

func (p *Process) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ptmx == nil {
		return 0, io.ErrClosedPipe
	}
	return p.ptmx.Write(data)
}

func (p *Process) Resize(rows, cols uint16) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ptmx == nil {
		return nil
	}
	return pty.Setsize(p.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (p *Process) Stop() {
	p.mu.Lock()
	proc := p.Cmd.Process
	p.mu.Unlock()

	if proc == nil {
		return
	}

	// SIGTERM — Claude handles it gracefully: saves session, removes lock file
	proc.Signal(syscall.SIGTERM)

	// Wait up to 5 seconds for clean exit
	select {
	case <-p.exited:
		goto cleanup
	case <-time.After(5 * time.Second):
	}

	// SIGKILL as last resort + manual lock cleanup
	proc.Signal(os.Kill)
	select {
	case <-p.exited:
	case <-time.After(1 * time.Second):
	}
	cleanupSessionLock(proc.Pid)

cleanup:
	p.mu.Lock()
	if p.ptmx != nil {
		p.ptmx.Close()
		p.ptmx = nil
	}
	p.running = false
	p.mu.Unlock()
}

func (p *Process) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// cleanupSessionLock removes Claude's session lock file for a given PID.
// Claude stores these at ~/.claude/sessions/<pid>.json
func cleanupSessionLock(pid int) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	lockFile := filepath.Join(home, ".claude", "sessions", fmt.Sprintf("%d.json", pid))
	os.Remove(lockFile)
}
