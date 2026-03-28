package sidebar

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sviniabanditka/clmux/internal/store"
	"github.com/sviniabanditka/clmux/internal/theme"
)

type ItemType int

const (
	ItemProjectHeader ItemType = iota
	ItemProject
	ItemThread
)

type Item struct {
	Type      ItemType
	ProjectID string
	ThreadID  string
	Label     string
	Path      string
	Status    store.ThreadStatus
}

type (
	ToggleCollapseMsg struct{ ProjectID string }
	SelectThreadMsg   struct{ ProjectID, ThreadID string }
	AddProjectMsg     struct{}
	AddThreadMsg      struct{ ProjectID string }
	CloseThreadMsg    struct{ ProjectID, ThreadID string }
	RenameThreadMsg   struct{ ProjectID, ThreadID string }
	DeleteProjectMsg  struct{ ProjectID string }
	DeleteThreadMsg   struct{ ProjectID, ThreadID string }
)

type Model struct {
	items        []Item
	cursor       int
	focused      bool
	width        int
	height       int
	store        *store.Store
	activeThread string
	itemYRanges  []yRange
	// Context menu
	ctxMenu   bool
	ctxX      int
	ctxY      int
	ctxItems  []ctxMenuItem
	ctxCursor int
}

type yRange struct{ start, end int }
type ctxMenuItem struct {
	label  string
	action func() tea.Msg
}

func New(s *store.Store) Model {
	m := Model{store: s}
	m.rebuildItems()
	return m
}

func (m *Model) SetFocused(f bool)         { m.focused = f }
func (m Model) Focused() bool              { return m.focused }
func (m *Model) SetActiveThread(id string) { m.activeThread = id }
func (m *Model) SetSize(w, h int)          { m.width = w; m.height = h }

func (m *Model) rebuildItems() {
	m.items = nil
	m.items = append(m.items, Item{Type: ItemProjectHeader, Label: "Projects"})
	for _, p := range m.store.State.Projects {
		m.items = append(m.items, Item{
			Type: ItemProject, ProjectID: p.ID,
			Label: p.Name, Path: p.Path,
		})
		if !p.Collapsed {
			for _, t := range p.Threads {
				m.items = append(m.items, Item{
					Type: ItemThread, ProjectID: p.ID,
					ThreadID: t.ID, Label: t.Name, Status: t.Status,
				})
			}
		}
	}
	m.rebuildYRanges()
}

func (m *Model) rebuildYRanges() {
	m.itemYRanges = nil
	y := 0
	for _, item := range m.items {
		h := 1
		if item.Type == ItemProject {
			h = 2 // name + path
		}
		m.itemYRanges = append(m.itemYRanges, yRange{y, y + h})
		y += h
	}
}

func (m Model) SelectedItem() *Item {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		it := m.items[m.cursor]
		return &it
	}
	return nil
}

func (m *Model) Rebuild() {
	m.rebuildItems()
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.ctxMenu {
		return m.updateCtx(msg)
	}
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			return m, m.doSelect()
		case "a":
			return m, m.doAdd()
		case "x":
			return m, m.doClose()
		case "r":
			return m, m.doRename()
		case "d", "backspace":
			return m, m.doDelete()
		}
	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

// ── Mouse ──

func (m Model) hitItem(y int) int {
	for i, r := range m.itemYRanges {
		if y >= r.start && y < r.end {
			return i
		}
	}
	return -1
}

// Button layout (right-aligned, each 4 chars wide):
// ProjectHeader: [+]
// Project:       [+] [x]       — add thread, delete project
// Thread:        [r] [x] [−]   — rename, close/open, delete
// positions from right edge (0-indexed from right):
//   btn3: width-13..width-10  (only threads)
//   btn2: width-9..width-6
//   btn1: width-5..width-2

func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	switch {
	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
		idx := m.hitItem(msg.Y)
		if idx < 0 {
			return m, nil
		}
		item := m.items[idx]
		m.cursor = idx
		x := msg.X
		btn1 := x >= m.width-5  // rightmost button
		btn2 := x >= m.width-9 && x < m.width-5
		btn3 := x >= m.width-13 && x < m.width-9

		switch item.Type {
		case ItemProjectHeader:
			if btn1 {
				return m, func() tea.Msg { return AddProjectMsg{} }
			}
			return m, func() tea.Msg { return AddProjectMsg{} }
		case ItemProject:
			pid := item.ProjectID
			if btn1 {
				// [x] delete project
				return m, func() tea.Msg { return DeleteProjectMsg{pid} }
			}
			if btn2 {
				// [+] add thread
				return m, func() tea.Msg { return AddThreadMsg{pid} }
			}
			return m, func() tea.Msg { return ToggleCollapseMsg{item.ProjectID} }
		case ItemThread:
			pid, tid := item.ProjectID, item.ThreadID
			if btn1 {
				// [−] delete
				return m, func() tea.Msg { return DeleteThreadMsg{pid, tid} }
			}
			if btn2 {
				// [x] close (if open)
				if item.Status == store.ThreadOpen {
					return m, func() tea.Msg { return CloseThreadMsg{pid, tid} }
				}
			}
			if btn3 {
				// [r] rename
				return m, func() tea.Msg { return RenameThreadMsg{pid, tid} }
			}
			return m, func() tea.Msg { return SelectThreadMsg{pid, tid} }
		}

	case msg.Button == tea.MouseButtonRight && msg.Action == tea.MouseActionPress:
		idx := m.hitItem(msg.Y)
		if idx < 0 {
			return m, nil
		}
		m.cursor = idx
		m.openCtx(msg.X, msg.Y, m.items[idx])
		return m, nil

	case msg.Button == tea.MouseButtonWheelUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case msg.Button == tea.MouseButtonWheelDown:
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	}
	return m, nil
}

// ── Context menu ──

func (m *Model) openCtx(x, y int, item Item) {
	m.ctxMenu = true
	m.ctxX = x
	m.ctxY = y
	m.ctxCursor = 0
	m.ctxItems = nil

	switch item.Type {
	case ItemProjectHeader:
		m.ctxItems = []ctxMenuItem{
			{"Add project", func() tea.Msg { return AddProjectMsg{} }},
		}
	case ItemProject:
		pid := item.ProjectID
		m.ctxItems = []ctxMenuItem{
			{"New thread", func() tea.Msg { return AddThreadMsg{ProjectID: pid} }},
			{"Delete project", func() tea.Msg { return DeleteProjectMsg{ProjectID: pid} }},
		}
	case ItemThread:
		pid, tid := item.ProjectID, item.ThreadID
		m.ctxItems = []ctxMenuItem{
			{"Open", func() tea.Msg { return SelectThreadMsg{pid, tid} }},
			{"Rename", func() tea.Msg { return RenameThreadMsg{pid, tid} }},
		}
		if item.Status == store.ThreadOpen {
			m.ctxItems = append(m.ctxItems,
				ctxMenuItem{"Close", func() tea.Msg { return CloseThreadMsg{pid, tid} }})
		}
		m.ctxItems = append(m.ctxItems,
			ctxMenuItem{"Delete", func() tea.Msg { return DeleteThreadMsg{pid, tid} }})
	}
}

func (m Model) updateCtx(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.ctxMenu = false
		case "up", "k":
			if m.ctxCursor > 0 {
				m.ctxCursor--
			}
		case "down", "j":
			if m.ctxCursor < len(m.ctxItems)-1 {
				m.ctxCursor++
			}
		case "enter", " ":
			if m.ctxCursor < len(m.ctxItems) {
				act := m.ctxItems[m.ctxCursor].action
				m.ctxMenu = false
				return m, func() tea.Msg { return act() }
			}
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			if msg.Button == tea.MouseButtonLeft {
				ci := msg.Y - m.ctxY
				if ci >= 0 && ci < len(m.ctxItems) {
					act := m.ctxItems[ci].action
					m.ctxMenu = false
					return m, func() tea.Msg { return act() }
				}
			}
			m.ctxMenu = false
		}
	}
	return m, nil
}

// ── Actions ──

func (m Model) doSelect() tea.Cmd {
	it := m.SelectedItem()
	if it == nil {
		return nil
	}
	switch it.Type {
	case ItemProjectHeader:
		return func() tea.Msg { return AddProjectMsg{} }
	case ItemProject:
		return func() tea.Msg { return ToggleCollapseMsg{it.ProjectID} }
	case ItemThread:
		return func() tea.Msg { return SelectThreadMsg{it.ProjectID, it.ThreadID} }
	}
	return nil
}

func (m Model) doAdd() tea.Cmd {
	it := m.SelectedItem()
	if it == nil {
		return nil
	}
	if it.Type == ItemProjectHeader {
		return func() tea.Msg { return AddProjectMsg{} }
	}
	pid := it.ProjectID
	return func() tea.Msg { return AddThreadMsg{pid} }
}

func (m Model) doClose() tea.Cmd {
	it := m.SelectedItem()
	if it == nil || it.Type != ItemThread {
		return nil
	}
	return func() tea.Msg { return CloseThreadMsg{it.ProjectID, it.ThreadID} }
}

func (m Model) doRename() tea.Cmd {
	it := m.SelectedItem()
	if it == nil || it.Type != ItemThread {
		return nil
	}
	return func() tea.Msg { return RenameThreadMsg{it.ProjectID, it.ThreadID} }
}

func (m Model) doDelete() tea.Cmd {
	it := m.SelectedItem()
	if it == nil {
		return nil
	}
	switch it.Type {
	case ItemProject:
		return func() tea.Msg { return DeleteProjectMsg{it.ProjectID} }
	case ItemThread:
		return func() tea.Msg { return DeleteThreadMsg{it.ProjectID, it.ThreadID} }
	}
	return nil
}

// ── View ──

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	cw := m.width // lipgloss Width = content width, border is added on top
	if cw < 1 {
		cw = 1
	}

	var b strings.Builder
	y := 0
	hintsH := 2

	// Both selected and normal lines: 1 char gutter + (cw-1) chars content = cw total
	contentW := cw - 1

	const (
		ansiSelBg = "\033[48;5;24m"
		ansiSelFg = "\033[38;5;255m"
		ansiBarFg = "\033[38;5;69m"
		ansiReset = "\033[0m"
	)

	for i, item := range m.items {
		isSel := m.focused && i == m.cursor
		rows := m.buildItemRows(item, contentW, isSel)

		for _, row := range rows {
			if y >= m.height-hintsH {
				break
			}
			// Pad row to exactly contentW visible chars
			runeLen := len([]rune(row))
			if isSel {
				// For selected: use rune count (plain text, no ANSI)
				if runeLen < contentW {
					row += strings.Repeat(" ", contentW-runeLen)
				}
				// ▎ (1 col) + row (contentW cols) = cw cols
				b.WriteString(ansiBarFg + ansiSelBg + "▎" + ansiSelFg + row + ansiReset + "\n")
			} else {
				// For styled: use lipgloss width (accounts for ANSI)
				w := lipgloss.Width(row)
				if w < contentW {
					row += strings.Repeat(" ", contentW-w)
				}
				// space (1 col) + row (contentW cols) = cw cols
				b.WriteString(" " + row + "\n")
			}
			y++
		}
	}

	for y < m.height-hintsH {
		b.WriteString(strings.Repeat(" ", cw) + "\n")
		y++
	}

	// Hints
	b.WriteString(theme.SeparatorStyle.Render(strings.Repeat("─", cw)) + "\n")
	b.WriteString(m.renderHints())

	style := theme.SidebarStyle
	if m.focused {
		style = theme.SidebarFocusedStyle
	}
	result := style.Width(m.width).Height(m.height).Render(b.String())

	if m.ctxMenu {
		result = m.renderCtxOverlay(result)
	}
	return result
}

// buildItemRows returns lines for an item.
// sel=true: returns 100% plain text (zero ANSI codes) so caller can wrap uniformly.
// sel=false: returns styled text via lipgloss.
func (m Model) buildItemRows(item Item, cw int, sel bool) []string {
	if sel {
		return m.buildPlain(item, cw)
	}
	return m.buildStyled(item, cw)
}

func (m Model) buildPlain(item Item, cw int) []string {
	switch item.Type {
	case ItemProjectHeader:
		return []string{rightPad(" PROJECTS", "[+] ", cw)}
	case ItemProject:
		p := m.store.FindProject(item.ProjectID)
		chev := "▼"
		if p == nil || p.Collapsed {
			chev = "▶"
		}
		btns := "[+] [x] "
		line1 := rightPad(" "+chev+" "+item.Label, btns, cw)
		line2 := "     " + shortenHome(item.Path)
		if len(line2) > cw {
			line2 = line2[:cw-1] + "…"
		}
		return []string{line1, line2}
	case ItemThread:
		active := item.ThreadID == m.activeThread
		dot := "○"
		if item.Status == store.ThreadOpen {
			dot = "●"
		} else if item.Status == store.ThreadSuspended {
			dot = "●"
		}
		prefix := "     "
		if active {
			prefix = "   › "
		}
		btns := "        [−] "
		if item.Status == store.ThreadOpen {
			btns = "[r] [x] [−] "
		} else {
			btns = "[r]     [−] "
		}
		return []string{rightPad(prefix+dot+" "+item.Label, btns, cw)}
	}
	return []string{""}
}

func (m Model) buildStyled(item Item, cw int) []string {
	switch item.Type {
	case ItemProjectHeader:
		left := " " + theme.HeaderStyle.Render("PROJECTS")
		right := theme.ButtonStyle.Render("[+]") + " "
		return []string{rightPad(left, right, cw)}

	case ItemProject:
		p := m.store.FindProject(item.ProjectID)
		collapsed := p == nil || p.Collapsed
		chev := "▼"
		if collapsed {
			chev = "▶"
		}
		row1 := " " + theme.DimStyle.Render(chev) + " " + theme.ProjectStyle.Render(item.Label)
		right := theme.ButtonStyle.Render("[+]") + " " + theme.CloseButtonStyle.Render("[x]") + " "
		line1 := rightPad(row1, right, cw)

		short := shortenHome(item.Path)
		row2 := "     " + theme.ProjectPathStyle.Render(short)
		if lipgloss.Width(row2) > cw {
			row2 = trunc(row2, cw)
		}
		return []string{line1, row2}

	case ItemThread:
		active := item.ThreadID == m.activeThread

		var dot string
		switch item.Status {
		case store.ThreadOpen:
			dot = theme.StatusOpen.Render("●")
		case store.ThreadSuspended:
			dot = theme.StatusSuspended.Render("●")
		default:
			dot = theme.StatusClosed.Render("○")
		}

		var name string
		if active {
			name = theme.ActiveThreadStyle.Render(item.Label)
		} else if item.Status == store.ThreadClosed {
			name = theme.ThreadDimStyle.Render(item.Label)
		} else {
			name = theme.ThreadStyle.Render(item.Label)
		}

		prefix := "     "
		if active {
			prefix = "   " + theme.ActiveThreadStyle.Render("›") + " "
		}

		var right string
		if item.Status == store.ThreadOpen {
			right = theme.ButtonStyle.Render("[r]") + " " + theme.CloseButtonStyle.Render("[x]") + " " + theme.CloseButtonStyle.Render("[−]") + " "
		} else {
			right = theme.ButtonStyle.Render("[r]") + "     " + theme.CloseButtonStyle.Render("[−]") + " "
		}

		return []string{rightPad(prefix+dot+" "+name, right, cw)}
	}
	return []string{""}
}

func rightPad(left, right string, total int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	gap := total - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderCtxOverlay(base string) string {
	if len(m.ctxItems) == 0 {
		return base
	}
	var mb strings.Builder
	for i, ci := range m.ctxItems {
		if i == m.ctxCursor {
			mb.WriteString(theme.CtxMenuSelectedStyle.Render(ci.label))
		} else {
			mb.WriteString(theme.CtxMenuItemStyle.Render(ci.label))
		}
		if i < len(m.ctxItems)-1 {
			mb.WriteString("\n")
		}
	}
	menu := theme.CtxMenuStyle.Render(mb.String())

	baseLines := strings.Split(base, "\n")
	menuLines := strings.Split(menu, "\n")
	sy := m.ctxY
	if sy+len(menuLines) > len(baseLines) {
		sy = len(baseLines) - len(menuLines)
	}
	if sy < 0 {
		sy = 0
	}
	for i, ml := range menuLines {
		if sy+i < len(baseLines) {
			baseLines[sy+i] = ml
		}
	}
	return strings.Join(baseLines, "\n")
}

func (m Model) renderHints() string {
	it := m.SelectedItem()
	h := []string{hint("a", "add"), hint("enter", "open")}
	if it != nil {
		switch it.Type {
		case ItemThread:
			h = append(h, hint("r", "ren"))
			if it.Status == store.ThreadOpen {
				h = append(h, hint("x", "close"))
			}
			h = append(h, hint("d", "del"))
		case ItemProject:
			h = append(h, hint("d", "del"))
		}
	}
	return " " + strings.Join(h, " ")
}

func hint(k, d string) string {
	return theme.HintKeyStyle.Render(k) + theme.HintDescStyle.Render(":"+d)
}

func shortenHome(p string) string {
	abs, err := filepath.Abs(p)
	if err == nil {
		p = abs
	}
	if strings.HasPrefix(p, "/Users/") {
		parts := strings.SplitN(p, "/", 4)
		if len(parts) >= 4 {
			return "~/" + parts[3]
		}
	}
	return p
}

func trunc(s string, w int) string {
	n := 0
	for i, r := range s {
		n++
		if r > 127 {
			n++
		}
		if n > w-1 {
			return s[:i] + "…"
		}
	}
	return s
}
