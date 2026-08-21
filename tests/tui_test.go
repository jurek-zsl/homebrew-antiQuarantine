package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"antiQuarantine/internal/history"
	"antiQuarantine/internal/quarantine"
	"antiQuarantine/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIModelAndInteractions(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin only")
	}

	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "app1.zip")
	f2 := filepath.Join(tmpDir, "app2.dmg")
	f3 := filepath.Join(tmpDir, "clean.txt")

	_ = os.WriteFile(f1, []byte("zip"), 0644)
	_ = os.WriteFile(f2, []byte("dmg"), 0644)
	_ = os.WriteFile(f3, []byte("clean"), 0644)

	_ = quarantine.SetRawQuarantine(f1, sampleQuarantine)
	_ = quarantine.SetRawQuarantine(f2, sampleQuarantine)

	_ = history.ClearHistory()

	// 1. Initialize TUI model
	m, err := tui.NewModel(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize TUI model: %v", err)
	}

	if m.Init() != nil {
		t.Errorf("expected Init() to return nil")
	}

	// 2. Test initial View rendering
	initialView := m.View()
	if !strings.Contains(initialView, "antiQuarantine (aq) Interactive Inspector") {
		t.Errorf("expected header in initial view, got: %s", initialView)
	}
	if !strings.Contains(initialView, "🔴 QUARANTINED") {
		t.Errorf("expected quarantined badge in view")
	}
	if !strings.Contains(initialView, "Safari") {
		t.Errorf("expected agent 'Safari' in inspector details")
	}

	// 3. Test Navigation (down / j)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(tui.Model)

	// 4. Test Selection Toggle (space)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(tui.Model)

	// 5. Test Strip Selected ('s')
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = newModel.(tui.Model)

	viewAfterStrip := m.View()
	if !strings.Contains(viewAfterStrip, "Successfully stripped quarantine") {
		t.Errorf("expected success status message after 's', got: %s", viewAfterStrip)
	}

	// 6. Test Undo ('u')
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = newModel.(tui.Model)

	viewAfterUndo := m.View()
	if !strings.Contains(viewAfterUndo, "Restored:") {
		t.Errorf("expected restore status message after 'u', got: %s", viewAfterUndo)
	}

	// 7. Test Strip All ('a')
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newModel.(tui.Model)

	viewAfterStripAll := m.View()
	if !strings.Contains(viewAfterStripAll, "Stripped all") {
		t.Errorf("expected 'Stripped all' status message, got: %s", viewAfterStripAll)
	}

	// 8. Test Refresh ('r')
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = newModel.(tui.Model)

	if !strings.Contains(m.View(), "Refreshed file list") {
		t.Errorf("expected refreshed status message")
	}

	// 9. Test Window Resize
	newModel, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(tui.Model)

	// 10. Test Quit ('q')
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Errorf("expected tea.Quit command on 'q' keypress")
	}
}
