package panel

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/hinshun/vt10x"
)

// VTerm wraps a vt10x terminal emulator with thread-safe access.
type VTerm struct {
	vt   vt10x.Terminal
	mu   sync.Mutex
	rows int
	cols int
}

func NewVTerm(cols, rows int) *VTerm {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return &VTerm{
		vt:   vt10x.New(vt10x.WithSize(cols, rows)),
		rows: rows,
		cols: cols,
	}
}

func (t *VTerm) Write(data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.vt.Write(data)
}

func (t *VTerm) Resize(cols, rows int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if cols < 1 || rows < 1 {
		return
	}
	t.cols = cols
	t.rows = rows
	t.vt.Resize(cols, rows)
}

// Render produces a styled string from the virtual terminal screen buffer.
func (t *VTerm) Render() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	var b strings.Builder
	for row := 0; row < t.rows; row++ {
		if row > 0 {
			b.WriteByte('\n')
		}
		t.renderRow(&b, row)
	}
	return b.String()
}

func (t *VTerm) renderRow(b *strings.Builder, row int) {
	// Find last non-empty cell to avoid trailing spaces
	lastCol := t.cols - 1
	for lastCol >= 0 {
		g := t.vt.Cell(lastCol, row)
		if g.Char != 0 && g.Char != ' ' {
			break
		}
		lastCol--
	}

	var prevFG, prevBG vt10x.Color
	var prevBold bool
	prevFG = 0xFFFFFFFF // sentinel
	styleOpen := false

	for col := 0; col <= lastCol; col++ {
		g := t.vt.Cell(col, row)
		ch := g.Char
		if ch == 0 {
			ch = ' '
		}

		fg := g.FG
		bg := g.BG
		bold := g.Mode&1 != 0 // AttrBold

		// Check if style changed
		if fg != prevFG || bg != prevBG || bold != prevBold {
			if styleOpen {
				// Close previous styled segment - we handle this inline
			}
			prevFG = fg
			prevBG = bg
			prevBold = bold
			styleOpen = true
		}

		// Apply style per character for simplicity
		styled := t.styleChar(ch, fg, bg, bold)
		b.WriteString(styled)
	}
}

const (
	defaultFGColor vt10x.Color = 16777216  // vt10x default FG sentinel
	defaultBGColor vt10x.Color = 16777217  // vt10x default BG sentinel
)

func (t *VTerm) styleChar(ch rune, fg, bg vt10x.Color, bold bool) string {
	s := string(ch)

	// Check if any styling needed
	hasFG := fg != defaultFGColor && fg != 0
	hasBG := bg != defaultBGColor && bg != 0
	if !hasFG && !hasBG && !bold {
		return s
	}

	style := lipgloss.NewStyle()
	if bold {
		style = style.Bold(true)
	}
	if hasFG {
		style = style.Foreground(vt10xColorToLipgloss(fg))
	}
	if hasBG {
		style = style.Background(vt10xColorToLipgloss(bg))
	}
	return style.Render(s)
}

func vt10xColorToLipgloss(c vt10x.Color) lipgloss.Color {
	// vt10x Color: lower 8 bits for standard colors, or full 24-bit RGB
	v := uint32(c)

	// Standard 16 colors (0-15)
	if v < 16 {
		return lipgloss.Color(fmt.Sprintf("%d", v))
	}

	// 256 color palette (16-255)
	if v < 256 {
		return lipgloss.Color(fmt.Sprintf("%d", v))
	}

	// 24-bit true color
	r := (v >> 16) & 0xFF
	g := (v >> 8) & 0xFF
	b := v & 0xFF
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}
