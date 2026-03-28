# claudex

A terminal multiplexer for [Claude Code](https://claude.ai/claude-code). Manage multiple Claude sessions across projects from a single TUI.

## Features

- **Project-based organization** — group Claude sessions by project directory
- **Multiple concurrent sessions** — run several Claude instances in parallel, switch between them instantly
- **Session persistence** — close a session and resume it later with full conversation history
- **Native Claude rendering** — embedded VT100 terminal emulator displays Claude's full TUI output
- **Mouse support** — click to navigate, right-click context menus, clickable buttons
- **Auto-naming** — threads are automatically named from your first message
- **Keyboard-driven** — full keyboard navigation for power users

## Install

```bash
go install github.com/sviniabanditka/claudex@latest
```

Or build from source:

```bash
git clone https://github.com/sviniabanditka/claudex.git
cd claudex
go build -o claudex .
```

> Requires [Claude Code](https://docs.anthropic.com/en/docs/claude-code) to be installed and available in your PATH.

## Usage

```bash
claudex
```

### Sidebar

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Open thread / toggle project |
| `a` | Add project or thread |
| `r` | Rename thread |
| `x` | Close running thread |
| `d` | Delete (with confirmation) |
| `Ctrl+B` | Switch focus to panel |

Mouse: click items to select, click `[+]` `[x]` `[r]` `[−]` buttons, right-click for context menu, scroll to navigate.

### Panel

When focused on the panel, all input goes directly to the running Claude session. Press `Ctrl+B` to switch focus back to the sidebar.

## How it works

```
┌─────────────────────┬──────────────────────────────────────┐
│  PROJECTS       [+] │                                      │
│  ▼ myapp    [+] [x] │  Claude session output               │
│    ~/projects/myapp  │                                      │
│  › ● fix auth bug    │  > fix the auth middleware           │
│    ● refactor db     │                                      │
│    ○ old thread      │  I'll fix the auth middleware...     │
│                      │                                      │
│                      │                                      │
│──────────────────────│                                      │
│ a:add enter:open     │                                      │
└─────────────────────┴──────────────────────────────────────┘
```

Each project maps to a directory where Claude runs. Threads are individual Claude sessions within that project. Claudex spawns Claude in a PTY, pipes output through a VT100 emulator, and renders it in the panel.

Sessions are stopped with `SIGTERM` on close — Claude saves the conversation. Reopening a thread resumes the conversation via `claude --resume`.

## Config

State is stored in `~/.config/claudex/state.json`. Claude's own session data lives in `~/.claude/projects/`.

## Tech

Built with:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — styling
- [creack/pty](https://github.com/creack/pty) — PTY management
- [vt10x](https://github.com/hinshun/vt10x) — terminal emulation

## License

MIT
