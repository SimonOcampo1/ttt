package terminal

import (
	"strings"
	"testing"

	xterm "github.com/gitpod-io/xterm-go"
)

const fishPrompt = "\x1b]133;A;click_events=1\x1b\\/home/user/a/rather/long/project/path\r\n❯ \x1b]133;B\x1b\\"

// resize runs the emulator half of Terminal.Resize; there is no pty here.
func (t *Terminal) resizeEmulator(cols, rows int) {
	t.clearPromptForRedraw()
	trimForReflow(t.term.NormalBuffer(), t.cols, t.rows, cols, rows)
	t.cols, t.rows = cols, rows
	t.term.Resize(cols, rows)
}

func newPromptTerminal(cols, rows int) *Terminal {
	t := &Terminal{cols: cols, rows: rows}
	t.term = xterm.New(xterm.WithCols(cols), xterm.WithRows(rows), xterm.WithScrollback(100))
	t.watchPromptMarks()
	return t
}

// fish repaints after SIGWINCH by moving up to the first prompt row and
// clearing from there; a reflowed prompt used to survive that as a copy.
func TestResizeLeavesOnePromptAfterShellRedraw(t *testing.T) {
	term := newPromptTerminal(60, 10)
	term.term.WriteString("earlier output\r\n" + fishPrompt)

	for _, cols := range []int{20, 50, 15, 60} {
		term.resizeEmulator(cols, 10)
		term.term.WriteString("\r\x1b[A\x1b[J" + fishPrompt)
	}

	screen := term.term.String()
	if n := strings.Count(screen, "/home/user"); n != 1 {
		t.Fatalf("%d prompts after resizing, want 1:\n%s", n, screen)
	}
	if !strings.Contains(screen, "earlier output") {
		t.Fatalf("output above the prompt was lost:\n%s", screen)
	}
}

func TestResizeKeepsRunningCommandOutput(t *testing.T) {
	term := newPromptTerminal(60, 10)
	term.term.WriteString(fishPrompt + "ls\r\n\x1b]133;C\x1b\\file-one file-two")

	term.resizeEmulator(30, 10)

	if !strings.Contains(term.term.String(), "file-one") {
		t.Fatalf("command output cleared on resize:\n%s", term.term.String())
	}
}
