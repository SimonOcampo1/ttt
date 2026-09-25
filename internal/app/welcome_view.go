package app

import (
	"github.com/eugenioenko/ttt/internal/term"
	"github.com/eugenioenko/ttt/internal/textwidth"
	"github.com/eugenioenko/ttt/internal/widgets"
	"github.com/gdamore/tcell/v3"
)

// Largest first; the layout takes the biggest one the tab can hold.
var welcomeTitles = [][]string{
	{
		"████████╗████████╗████████╗",
		"╚══██╔══╝╚══██╔══╝╚══██╔══╝",
		"   ██║      ██║      ██║   ",
		"   ██║      ██║      ██║   ",
		"   ██║      ██║      ██║   ",
		"   ╚═╝      ╚═╝      ╚═╝   ",
	},
	{
		"┏┳┓ ┏┳┓ ┏┳┓",
		" ┃   ┃   ┃ ",
		" ╹   ╹   ╹ ",
	},
	{"TTT Editor"},
}

const (
	welcomeSubtitle  = "Terminal Text Tool"
	welcomeHint      = "↑↓ select   enter open"
	welcomeShortcutW = 6
)

type welcomeItem struct {
	label   string
	detail  string // shortcut or folder path, right-aligned and muted
	section string // heading drawn above this item, after a blank row
	run     func()
}

// Each section heading takes a blank row plus the heading row.
const welcomeSectionRows = 2

type welcomeLayout struct {
	title    []string
	subtitle bool
	gap      int
	hint     bool
	top      int
	height   int
}

// layoutWelcome fits the page into h rows for n actions under the given
// number of section headings. Space between the actions outranks the size of
// the title; with too little room even for the smallest title, only the
// actions are shown, without headings.
func layoutWelcome(w, h, n, sections int) welcomeLayout {
	for _, gap := range []int{1, 0} {
		for _, title := range welcomeTitles {
			if textwidth.String(title[0]) > w {
				continue
			}
			l := welcomeLayout{title: title, subtitle: len(title) > 1, gap: gap}
			l.height = len(title) + 1 + gap + n + (n-1)*gap + sections*welcomeSectionRows
			if l.subtitle {
				l.height += 2
			}
			if l.height > h {
				continue
			}
			if l.height+2 <= h {
				l.hint = true
				l.height += 2
			}
			l.top = (h - l.height) / 2
			return l
		}
	}
	return welcomeLayout{height: min(n, h), top: max((h-n)/2, 0)}
}

type welcomeView struct {
	widgets.BaseWidget
	items      []welcomeItem
	note       string // muted line under the items, laid out like a section
	selected   int
	wasPressed bool
	rowX, rowW int
	rowY       []int
}

func (v *welcomeView) Height() int { return 0 }
func (v *welcomeView) Width() int  { return 0 }

func (v *welcomeView) Render(surface widgets.Surface) {
	w, h := surface.Size()
	surface.Fill(term.Cell{Ch: ' ', Style: term.StyleDefault})
	if w <= 0 || h <= 0 {
		return
	}
	sections := 0
	if v.note != "" {
		sections++
	}
	for _, item := range v.items {
		if item.section != "" {
			sections++
		}
	}
	l := layoutWelcome(w, h, len(v.items), sections)
	y := l.top

	if len(l.title) > 0 {
		titleW := textwidth.String(l.title[0])
		for _, line := range l.title {
			surface.DrawText((w-titleW)/2, y, line, w, term.StyleBorderActive)
			y++
		}
		if l.subtitle {
			y++
			surface.DrawText((w-len(welcomeSubtitle))/2, y, welcomeSubtitle, w, term.StyleMuted)
			y++
		}
		y += 1 + l.gap
	}

	listW := 0
	for _, item := range v.items {
		listW = max(listW, textwidth.String(item.label)+welcomeShortcutW+textwidth.String(item.detail))
	}
	// Two cells of padding either side of the highlight.
	v.rowW = min(listW+4, w)
	v.rowX = max((w-v.rowW)/2, 0)
	origin := v.GetRect()
	// Too short for every action: scroll so the selection stays visible.
	first, last := 0, len(v.items)
	if len(l.title) == 0 && len(v.items) > h {
		first = min(max(v.selected-h+1, 0), len(v.items)-h)
		last = first + h
	}
	v.rowY = v.rowY[:0]
	for i := range v.items {
		if i < first || i >= last {
			v.rowY = append(v.rowY, -1)
			continue
		}
		if v.items[i].section != "" && len(l.title) > 0 {
			y++
			surface.DrawText(v.rowX+2, y, v.items[i].section, v.rowX+v.rowW-2, term.StyleMuted)
			y++
		}
		v.rowY = append(v.rowY, origin.Y+y)
		v.renderRow(surface, y, v.items[i].label, v.items[i].detail, i == v.selected)
		y += 1 + l.gap
	}

	if v.note != "" && len(l.title) > 0 {
		surface.DrawText((w-textwidth.String(v.note))/2, y, v.note, w, term.StyleMuted)
	}

	if l.hint {
		surface.DrawText((w-textwidth.String(welcomeHint))/2, l.top+l.height-1, welcomeHint, w, term.StyleMuted)
	}
}

func (v *welcomeView) renderRow(surface widgets.Surface, y int, label, shortcut string, selected bool) {
	labelStyle, shortcutStyle := term.StyleDefault, term.StyleMuted
	if selected {
		labelStyle, shortcutStyle = term.StylePaletteSelected, term.StylePaletteSelected
		for x := v.rowX; x < v.rowX+v.rowW; x++ {
			surface.SetCell(x, y, term.Cell{Ch: ' ', Style: term.StylePaletteSelected})
		}
	}
	// DrawText takes an absolute column limit, not a width.
	left, right := v.rowX+2, v.rowX+v.rowW-2
	surface.DrawText(left, y, label, right, labelStyle)
	if sw := textwidth.String(shortcut); sw > 0 && sw < right-left-textwidth.String(label) {
		surface.DrawText(right-sw, y, shortcut, right, shortcutStyle)
	}
}

func (v *welcomeView) rowAt(mx, my int) int {
	r := v.GetRect()
	if mx < r.X+v.rowX || mx >= r.X+v.rowX+v.rowW {
		return -1
	}
	for i, y := range v.rowY {
		if y == my {
			return i
		}
	}
	return -1
}

func (v *welcomeView) HandleEvent(ev tcell.Event) widgets.EventResult {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyUp:
			v.selected = (v.selected + len(v.items) - 1) % len(v.items)
		case tcell.KeyDown:
			v.selected = (v.selected + 1) % len(v.items)
		case tcell.KeyHome:
			v.selected = 0
		case tcell.KeyEnd:
			v.selected = len(v.items) - 1
		case tcell.KeyEnter:
			v.items[v.selected].run()
		default:
			return widgets.EventIgnored
		}
		return widgets.EventConsumed
	case *tcell.EventMouse:
		pressed := ev.Buttons()&tcell.Button1 != 0
		fresh := pressed && !v.wasPressed
		v.wasPressed = pressed
		i := v.rowAt(ev.Position())
		if i < 0 {
			return widgets.EventIgnored
		}
		v.selected = i
		if fresh {
			v.items[i].run()
		}
		return widgets.EventConsumed
	}
	return widgets.EventIgnored
}
