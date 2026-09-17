package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/eugenioenko/ttt/internal/workspace"
)

// folderPickerRows is how many directory rows the list shows. Tall enough to
// browse without scrolling constantly, short enough that the dialog still reads
// as a dialog on a laptop screen.
const (
	folderPickerRows    = 12
	folderPickerPlacesW = 18
)

// folderPicker is the state behind the Open/Add Folder dialog: an editable path
// on top and a navigable listing under it, kept in sync in both directions.
//
// Typing a path stays the fastest route for anyone who knows where they are
// going, so the input remains the source of truth; the list is a way to discover
// and refine it, not a mode you have to enter.
type folderPicker struct {
	app    *App
	input  *widgets.InputWidget
	places *widgets.TreeWidget
	list   *widgets.TreeWidget
	dir    string
}

// folderPickerEntry is one row. dir is where selecting it takes you; places and
// the parent entry are just rows whose dir happens to be elsewhere.
type folderPickerEntry struct {
	label string
	dir   string
}

// ShowFolderPicker asks for a directory, with the shortcuts the user already
// keeps in their file manager and a listing they can walk.
func (a *App) ShowFolderPicker(title, confirmLabel, initial string, onPick func(string)) {
	fp := &folderPicker{app: a}

	submit := func(text string) {
		path := strings.TrimSpace(text)
		if path == "" {
			return
		}
		a.DismissDialog()
		onPick(path)
	}

	fp.input = widgets.NewInputWidget(widgets.InputConfig{
		Placeholder: "Folder path",
		OnSubmit:    submit,
		OnChange:    func(text string) { fp.syncFromInput(text) },
	})

	fp.places = widgets.NewListWidgetFromConfig(widgets.ListConfig{
		EmptyText: "No places",
		OnSelect:  func(node *widgets.TreeNode) { fp.enter(node.ID) },
	})
	fp.places.SetItems(placeNodes(workspace.Places()))

	fp.list = widgets.NewListWidgetFromConfig(widgets.ListConfig{
		EmptyText: "No subdirectories",
		OnSelect:  func(node *widgets.TreeNode) { fp.enter(node.ID) },
	})

	if initial == "" {
		initial = a.Workspace.Primary()
	}
	fp.setDir(initial)
	fp.input.SetText(fp.dir)

	dialog := widgets.NewDialogWidget(64)
	dialog.Title = title
	dialog.Borders = *a.Borders
	// Places on the left, the listing on the right, the way a file manager puts
	// them: shortcuts are always reachable without scrolling past them to get to
	// the directory you are actually in.
	placesBox := widgets.NewBoxWidget(widgets.BoxModel{PaddingRight: 1})
	placesBox.Child = fp.places
	placesBox.FixedWidth = folderPickerPlacesW

	// Both trees measure as grow children (height 0), so the dialog would collapse
	// them away; the stack is what reserves the rows.
	columns := widgets.NewHStackWidget(placesBox, fp.list)
	columns.FixedHeight = folderPickerRows
	columns.Gap = 1

	dialog.SetContent(widgets.NewVStackWidget(fp.input, columns))
	dialog.Buttons = []widgets.DialogButton{
		{Label: "&Cancel", Handler: func() { a.DismissDialog() }},
		{Label: "&" + confirmLabel, Handler: func() { submit(fp.input.Text()) }},
	}
	dialog.OnDismiss = func() { a.DismissDialog() }
	dialog.Build()
	a.ShowDialog(ui.NewWidgetAdapter(dialog))
}

// enter descends into dir and puts it in the input, so confirming right after
// picking a row does what it looks like it will do.
func (fp *folderPicker) enter(dir string) {
	if dir == "" {
		return
	}
	fp.setDir(dir)
	fp.input.SetText(fp.dir)
}

// syncFromInput follows along as the path is typed. It only re-lists when the
// text names a real directory, so a half-typed path leaves the listing where it
// was instead of emptying out on every keystroke.
func (fp *folderPicker) syncFromInput(text string) {
	path := expandFolderPath(strings.TrimSpace(text))
	if path == "" {
		return
	}
	// A trailing separator means "inside this one"; otherwise the last segment is
	// still being typed and the parent is the interesting listing.
	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			path = filepath.Dir(path)
		}
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return
	}
	if abs, err := filepath.Abs(path); err == nil && abs != fp.dir {
		fp.setDir(abs)
	}
}

func (fp *folderPicker) setDir(dir string) {
	abs, err := filepath.Abs(expandFolderPath(dir))
	if err != nil {
		return
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return
	}
	fp.dir = abs
	fp.list.SetItems(folderPickerNodes(folderPickerEntries(abs)))
}

// folderPickerEntries builds the rows for dir: the way up, then what is inside.
func folderPickerEntries(dir string) []folderPickerEntry {
	var out []folderPickerEntry
	if parent := filepath.Dir(dir); parent != dir {
		out = append(out, folderPickerEntry{label: "..", dir: parent})
	}
	out = append(out, subdirectories(dir)...)
	return out
}

// subdirectories lists dir's child directories, sorted, with dotfiles last:
// they are worth reaching but rarely what someone is looking for.
func subdirectories(dir string) []folderPickerEntry {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []folderPickerEntry
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		out = append(out, folderPickerEntry{
			label: item.Name(),
			dir:   filepath.Join(dir, item.Name()),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		hiddenI := strings.HasPrefix(out[i].label, ".")
		hiddenJ := strings.HasPrefix(out[j].label, ".")
		if hiddenI != hiddenJ {
			return hiddenJ
		}
		return strings.ToLower(out[i].label) < strings.ToLower(out[j].label)
	})
	return out
}

func folderPickerNodes(entries []folderPickerEntry) []*widgets.TreeNode {
	nodes := make([]*widgets.TreeNode, len(entries))
	for i, e := range entries {
		nodes[i] = &widgets.TreeNode{ID: e.dir, Label: e.label}
	}
	return nodes
}

func placeNodes(places []workspace.Place) []*widgets.TreeNode {
	nodes := make([]*widgets.TreeNode, len(places))
	for i, p := range places {
		nodes[i] = &widgets.TreeNode{ID: p.Path, Label: p.Name}
	}
	return nodes
}

func expandFolderPath(path string) string {
	return workspace.ExpandPath(path)
}
