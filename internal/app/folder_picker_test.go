package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func labels(entries []folderPickerEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.label
	}
	return out
}

func TestFolderPickerEntriesOrdersTheListing(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"zeta", ".hidden", "Alpha", "beta"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a-file.txt"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	got := labels(folderPickerEntries(dir))
	want := []string{"..", "Alpha", "beta", "zeta", ".hidden"}
	if !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

// The root has no parent, and offering ".." there would just point at itself.
func TestFolderPickerEntriesOmitsParentAtRoot(t *testing.T) {
	got := labels(folderPickerEntries(string(os.PathSeparator)))
	if slices.Contains(got, "..") {
		t.Fatalf("entries at root = %v, want no \"..\"", got)
	}
}

func TestFolderPickerEntriesOnUnreadableDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nope")
	// A directory that cannot be listed still offers the way back out.
	if got := labels(folderPickerEntries(dir)); !slices.Equal(got, []string{".."}) {
		t.Fatalf("entries = %v, want just [..]", got)
	}
}
