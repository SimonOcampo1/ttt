package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eugenioenko/ttt/internal/widgets"
)

func TestWelcomeTabListsStartActions(t *testing.T) {
	h := newTestHarness(t, 100, 30)
	defer h.stop()

	h.exec("help.welcome")
	h.redraw()
	for _, label := range []string{"Welcome", "Open Folder…", "New File"} {
		h.assertContains(label)
	}
}

func TestEmptyExplorerRunsItsActions(t *testing.T) {
	h := newTestHarness(t, 100, 30)
	defer h.stop()

	var ran []string
	h.app.Explorer.OnAction = func(id string) { ran = append(ran, id) }
	h.app.Explorer.SetRoots(nil)
	h.exec("sidebar.explorer")
	h.redraw()
	h.assertContains("No folder open")

	h.app.Explorer.Tree.SelectByID("command:workspace.openFolder")
	h.app.Explorer.Tree.ActivateSelected()
	if len(ran) != 1 || ran[0] != "workspace.openFolder" {
		t.Fatalf("actions run = %v, want [workspace.openFolder]", ran)
	}
}

func TestWelcomeRowRunsOnClick(t *testing.T) {
	h := newTestHarness(t, 100, 40)
	defer h.stop()

	h.exec("help.welcome")
	h.redraw()
	for y := 0; y < 40; y++ {
		row := h.screenRow(y)
		if x := displayColumnOf(row, "Settings"); x >= 0 && !strings.Contains(row, "Welcome") {
			h.click(x, y)
			h.redraw()
			h.assertContains("Tab size")
			return
		}
	}
	t.Fatalf("Settings row not found:\n%s", h.screenText())
}

func TestClosingTheWorkspaceShowsWelcome(t *testing.T) {
	h := newTestHarness(t, 100, 30)
	defer h.stop()

	h.exec("workspace.close")
	h.redraw()
	if n := len(h.app.Workspace.Paths()); n != 0 {
		t.Fatalf("%d folders left after Close Workspace", n)
	}
	h.assertContains("Welcome")
	h.assertNotContains("untitled")
	if h.app.Sidebar.Visible {
		t.Error("sidebar still shown next to the welcome page")
	}
}

func TestRemovingTheLastFolderShowsWelcome(t *testing.T) {
	h := newTestHarness(t, 100, 30)
	defer h.stop()

	for _, p := range h.app.Workspace.Paths() {
		h.app.FileOpRemoveRoot(p)
	}
	h.redraw()
	if n := len(h.app.Workspace.Paths()); n != 0 {
		t.Fatalf("%d folders left after removing every root", n)
	}
	h.assertContains("Welcome")
}

// With no folder open the welcome page is the editor's empty state: it gives
// way to anything opened and comes back once that closes.
func TestWelcomeIsTheEmptyState(t *testing.T) {
	h := newTestHarness(t, 100, 30)
	defer h.stop()

	h.exec("workspace.close")
	h.exec("file.new")
	h.assertNotContains("Welcome")

	h.exec("tab.close")
	h.assertContains("Welcome")

	h.exec("tab.close")
	h.assertContains("Welcome")
}

// Favorites that exist are listed with their path and open on click; missing
// ones are left out.
func TestWelcomeFavoriteOpensFolder(t *testing.T) {
	h := newTestHarness(t, 100, 40)
	defer h.stop()

	fav := filepath.Join(t.TempDir(), "fav-project")
	if err := os.Mkdir(fav, 0o755); err != nil {
		t.Fatal(err)
	}
	h.app.Settings.Welcome.Favorites = []string{fav, filepath.Join(fav, "missing-project")}
	h.exec("workspace.close")
	h.redraw()
	h.assertNotContains("missing-project")
	for y := 0; y < 40; y++ {
		row := h.screenRow(y)
		if x := displayColumnOf(row, "fav-project"); x >= 0 {
			h.click(x, y)
			h.redraw()
			if paths := h.app.Workspace.Paths(); len(paths) != 1 || paths[0] != fav {
				t.Fatalf("workspace = %v, want [%s]", paths, fav)
			}
			return
		}
	}
	t.Fatalf("favorite row not found:\n%s", h.screenText())
}

// Adding the open folder saves it and lists it on the welcome page; removing
// it brings back the hint.
func TestWelcomeFavoriteCommands(t *testing.T) {
	h := newTestHarness(t, 100, 40)
	defer h.stop()

	root := h.app.Workspace.Paths()[0]
	h.exec("welcome.addFavorite")
	if favs := h.app.Settings.Welcome.Favorites; len(favs) != 1 {
		t.Fatalf("favorites = %v, want the open folder", favs)
	}
	h.exec("workspace.close")
	h.redraw()
	h.assertContains("Favorites")
	h.assertContains(filepath.Base(root))
	h.assertNotContains("Right-click a folder")

	h.app.ExplorerContextNode = &widgets.TreeNode{ID: root}
	h.exec("welcome.removeFavorite")
	if favs := h.app.Settings.Welcome.Favorites; len(favs) != 0 {
		t.Fatalf("favorites = %v after removing, want none", favs)
	}
	h.redraw()
	h.assertContains("Right-click a folder")
}
