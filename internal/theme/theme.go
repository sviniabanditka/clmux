package theme

import "github.com/charmbracelet/lipgloss"

var (
	// Core palette
	Accent    = lipgloss.Color("69")  // bright blue
	AccentBg  = lipgloss.Color("24")  // dark blue bg for selection
	White     = lipgloss.Color("255")
	Light     = lipgloss.Color("250")
	Muted     = lipgloss.Color("245")
	Dim       = lipgloss.Color("240")
	Faint     = lipgloss.Color("236")
	Green     = lipgloss.Color("114")
	Yellow    = lipgloss.Color("220")
	Red       = lipgloss.Color("203")
	Border    = lipgloss.Color("236")

	// Sidebar chrome
	SidebarStyle = lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Border)

	SidebarFocusedStyle = lipgloss.NewStyle().
				BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(Accent)

	PanelStyle = lipgloss.NewStyle()

	// Section header
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Bold(true)

	// Project name
	ProjectStyle = lipgloss.NewStyle().
			Foreground(White).
			Bold(true)

	// Project path
	ProjectPathStyle = lipgloss.NewStyle().
				Foreground(Dim)

	// Selection — bright bg, unmissable
	SelectedStyle = lipgloss.NewStyle().
			Background(AccentBg).
			Foreground(White).
			Bold(true)

	// Active thread in panel
	ActiveThreadStyle = lipgloss.NewStyle().
				Foreground(Accent).
				Bold(true)

	// Thread names
	ThreadStyle    = lipgloss.NewStyle().Foreground(Light)
	ThreadDimStyle = lipgloss.NewStyle().Foreground(Dim)

	// Status dots
	StatusOpen      = lipgloss.NewStyle().Foreground(Green)
	StatusSuspended = lipgloss.NewStyle().Foreground(Yellow)
	StatusClosed    = lipgloss.NewStyle().Foreground(Dim)

	// Buttons
	ButtonStyle      = lipgloss.NewStyle().Foreground(Dim)
	CloseButtonStyle = lipgloss.NewStyle().Foreground(Dim)

	// Modal
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).
			Padding(1, 2)

	// General text
	DimStyle   = lipgloss.NewStyle().Foreground(Dim)
	MutedStyle = lipgloss.NewStyle().Foreground(Muted)
	ErrorStyle = lipgloss.NewStyle().Foreground(Red)

	PlaceholderStyle = lipgloss.NewStyle().
				Foreground(Dim).
				Italic(true)

	// Hint bar
	HintKeyStyle  = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	HintDescStyle = lipgloss.NewStyle().Foreground(Dim)
	SeparatorStyle = lipgloss.NewStyle().Foreground(Faint)

	// Context menu
	CtxMenuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent)

	CtxMenuItemStyle = lipgloss.NewStyle().
				Foreground(Light).
				Padding(0, 1)

	CtxMenuSelectedStyle = lipgloss.NewStyle().
				Background(Accent).
				Foreground(White).
				Bold(true).
				Padding(0, 1)
)
