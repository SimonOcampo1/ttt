package app

import (
	"os"
	"path/filepath"

	"github.com/eugenioenko/ttt/internal/core/buffer"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/workspace"
)

const welcomeTabID = "welcome"

var welcomeCommands = []struct{ label, commandID string }{
	{"Open Folder…", "workspace.openFolder"},
	{"New File", "file.new"},
	{"Open Workspace…", "workspace.open"},
	{"Settings", "settings.openUI"},
	{"Command Palette", "command.palette"},
}

// welcomeItems lists the start actions with their shortcuts, then the
// favorite folders that exist, each with its path as written in settings.
func (a *App) welcomeItems() []welcomeItem {
	var items []welcomeItem
	for _, c := range welcomeCommands {
		shortcut := ""
		if cmd, ok := a.Reg.Get(c.commandID); ok {
			shortcut = cmd.Shortcut
		}
		items = append(items, welcomeItem{label: c.label, detail: shortcut, run: func() { a.Reg.Execute(c.commandID) }})
	}
	for _, fav := range a.Settings.Welcome.Favorites {
		abs, err := filepath.Abs(workspace.ExpandPath(fav))
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err != nil || !info.IsDir() {
			continue
		}
		item := welcomeItem{label: filepath.Base(abs), detail: fav, run: func() { a.openFolderPath(abs) }}
		if len(items) == len(welcomeCommands) {
			item.section = "Favorites"
		}
		items = append(items, item)
	}
	return items
}

func (a *App) ShowWelcome() {
	if a.EditorGroup.SwitchToTabByPath(welcomeTabID) {
		a.FocusEditor()
		return
	}
	view := &welcomeView{items: a.welcomeItems()}
	adapter := ui.NewWidgetAdapter(view)
	onlyBlankUntitled := a.EditorGroup.TabCount() == 1 && a.EditorGroup.IsActiveVirtual() && isBlank(a.EditorGroup.ActiveBuffer())
	a.EditorGroup.OpenPluginTab(welcomeTabID, "Welcome", adapter)
	if onlyBlankUntitled {
		a.EditorGroup.CloseOtherTabs()
	}
	a.FocusEditor()
	adapter.SetFocused(true)
}

func isBlank(b *buffer.Buffer) bool {
	return b != nil && !b.Dirty && len(b.Lines) <= 1 && (len(b.Lines) == 0 || b.Lines[0] == "")
}

// ShowEmptyState makes the welcome page the editor's empty state: it stays
// while nothing is open, and the sidebar has nothing to add to it.
func (a *App) ShowEmptyState() {
	a.ShowWelcome()
	a.EditorGroup.EmptyStateID = welcomeTabID
	a.welcomeIsEmptyState = true
	a.HideSidebar()
}

// SyncEmptyState steps the welcome page aside once something else opens.
func (a *App) SyncEmptyState() {
	if a.welcomeIsEmptyState && a.EditorGroup.TabCount() > 1 {
		a.closeWelcome()
	}
}

func (a *App) closeWelcome() {
	a.welcomeIsEmptyState = false
	a.EditorGroup.EmptyStateID = ""
	a.EditorGroup.ClosePluginTab(welcomeTabID)
}
