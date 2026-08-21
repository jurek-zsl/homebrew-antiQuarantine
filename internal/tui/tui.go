package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"antiQuarantine/internal/history"
	"antiQuarantine/internal/quarantine"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#04B575")).
				Background(lipgloss.Color("#2E2E3E"))

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD"))

	quarantinedBadge = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF5F5F")).
				Render("🔴 QUARANTINED")

	cleanBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575")).
			Render("🟢 CLEAN")

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginTop(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F1F1F1")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)
)

type item struct {
	path     string
	filename string
	isDir    bool
	info     *quarantine.FileInfo
	selected bool
}

type Model struct {
	dir         string
	items       []item
	cursor      int
	width       int
	height      int
	statusMsg   string
	quarantined int
}

func NewModel(dir string) (Model, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	m := Model{
		dir: absDir,
	}
	m.refreshItems()
	return m, nil
}

func (m *Model) refreshItems() {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Error reading dir: %v", err)
		return
	}

	var items []item
	qCount := 0

	for _, e := range entries {
		p := filepath.Join(m.dir, e.Name())
		info, _ := quarantine.InspectFile(p)

		it := item{
			path:     p,
			filename: e.Name(),
			isDir:    e.IsDir(),
			info:     info,
		}
		if info != nil && info.HasQuarantine {
			qCount++
		}
		items = append(items, it)
	}

	m.items = items
	m.quarantined = qCount
	if m.cursor >= len(m.items) && len(m.items) > 0 {
		m.cursor = len(m.items) - 1
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case " ":
			if len(m.items) > 0 {
				m.items[m.cursor].selected = !m.items[m.cursor].selected
			}

		case "s": // Strip selected or highlighted
			m.stripCurrentOrSelected()

		case "a": // Strip all quarantined
			m.stripAllQuarantined()

		case "r": // Refresh
			m.refreshItems()
			m.statusMsg = "Refreshed file list"

		case "u": // Undo last
			rec, err := history.RestoreLast()
			if err != nil {
				m.statusMsg = fmt.Sprintf("Restore error: %v", err)
			} else {
				m.statusMsg = fmt.Sprintf("Restored: %s", filepath.Base(rec.FilePath))
				m.refreshItems()
			}
		}
	}

	return m, nil
}

func (m *Model) stripCurrentOrSelected() {
	var targets []int
	for i, it := range m.items {
		if it.selected && it.info != nil && it.info.HasQuarantine {
			targets = append(targets, i)
		}
	}

	if len(targets) == 0 && len(m.items) > 0 {
		if m.items[m.cursor].info != nil && m.items[m.cursor].info.HasQuarantine {
			targets = append(targets, m.cursor)
		}
	}

	if len(targets) == 0 {
		m.statusMsg = "No quarantined files selected"
		return
	}

	count := 0
	for _, idx := range targets {
		p := m.items[idx].path
		raw, _ := quarantine.GetRawQuarantine(p)
		_ = history.RecordStrip(p, raw)
		if err := quarantine.RemoveQuarantine(p); err == nil {
			count++
		}
	}

	m.statusMsg = fmt.Sprintf("Successfully stripped quarantine from %d file(s)", count)
	m.refreshItems()
}

func (m *Model) stripAllQuarantined() {
	count := 0
	for _, it := range m.items {
		if it.info != nil && it.info.HasQuarantine {
			raw, _ := quarantine.GetRawQuarantine(it.path)
			_ = history.RecordStrip(it.path, raw)
			if err := quarantine.RemoveQuarantine(it.path); err == nil {
				count++
			}
		}
	}
	m.statusMsg = fmt.Sprintf("Stripped all %d quarantined file(s)", count)
	m.refreshItems()
}

func (m Model) View() string {
	if len(m.items) == 0 {
		return titleStyle.Render(" antiQuarantine (aq) TUI ") + "\n\nNo files found in " + m.dir + "\n\nPress 'q' to quit."
	}

	// Left pane: File List
	var listBuilder strings.Builder
	for i, it := range m.items {
		cursor := " "
		if i == m.cursor {
			cursor = "❯"
		}
		checked := "[ ]"
		if it.selected {
			checked = "[●]"
		}

		status := " "
		if it.info != nil {
			if it.info.HasQuarantine {
				status = "🔴"
			} else {
				status = "🟢"
			}
		}

		name := it.filename
		if it.isDir {
			name += "/"
		}
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		line := fmt.Sprintf("%s %s %s %-30s", cursor, checked, status, name)
		if i == m.cursor {
			listBuilder.WriteString(selectedItemStyle.Render(line) + "\n")
		} else {
			listBuilder.WriteString(normalItemStyle.Render(line) + "\n")
		}
	}

	leftPane := borderStyle.Width(45).Render(listBuilder.String())

	// Right pane: Inspector Details
	var rightBuilder strings.Builder
	curr := m.items[m.cursor]
	rightBuilder.WriteString(fmt.Sprintf("File: %s\n", curr.filename))
	rightBuilder.WriteString(fmt.Sprintf("Type: %s\n\n", map[bool]string{true: "Directory / Bundle", false: "Regular File"}[curr.isDir]))

	if curr.info != nil {
		if curr.info.HasQuarantine {
			rightBuilder.WriteString("Status: " + quarantinedBadge + "\n\n")
			if curr.info.Metadata != nil {
				m := curr.info.Metadata
				rightBuilder.WriteString(fmt.Sprintf("Agent:     %s\n", m.Agent))
				rightBuilder.WriteString(fmt.Sprintf("Timestamp: %s\n", m.Timestamp.Format("2006-01-02 15:04:05 UTC")))
				rightBuilder.WriteString(fmt.Sprintf("Flags:     0x%s\n", m.FlagsHex))
				for _, lbl := range m.FlagLabels {
					rightBuilder.WriteString(fmt.Sprintf("  • %s\n", lbl))
				}
				if m.EventUUID != "" {
					rightBuilder.WriteString(fmt.Sprintf("UUID:      %s\n", m.EventUUID))
				}
			}
			if curr.info.Provenance != nil {
				p := curr.info.Provenance
				if p.DataURL != "" {
					rightBuilder.WriteString(fmt.Sprintf("\nOrigin URL:\n  %s\n", p.DataURL))
				}
				if p.OriginURL != "" {
					rightBuilder.WriteString(fmt.Sprintf("Referrer:\n  %s\n", p.OriginURL))
				}
			}
		} else {
			rightBuilder.WriteString("Status: " + cleanBadge + "\n\n")
			rightBuilder.WriteString("No Gatekeeper quarantine attribute detected.\n")
		}
	}

	rightPane := borderStyle.Width(50).Render(rightBuilder.String())

	header := titleStyle.Render(" antiQuarantine (aq) Interactive Inspector ") + " " +
		fmt.Sprintf("Path: %s | Quarantined: %d", m.dir, m.quarantined)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	help := helpStyle.Render("Navigation: [↑/↓, j/k] | Toggle: [Space] | Strip: [s] | Strip All: [a] | Restore: [u] | Refresh: [r] | Quit: [q]")

	status := ""
	if m.statusMsg != "" {
		status = "\n" + statusStyle.Render(m.statusMsg)
	}

	return header + "\n\n" + body + "\n" + help + status
}

// Run launches the Bubbletea interactive UI
func Run(dir string) error {
	m, err := NewModel(dir)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
