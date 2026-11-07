package app

import (
	"fmt"
	"runtime"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/player"
)

// keyName writes a shortcut as the platform does: ⌘ on macOS, Ctrl
// elsewhere.
func keyName(mac, other string) string {
	if runtime.GOOS == "darwin" {
		return mac
	}
	return other
}

// volumeAccelerators turn the volume up and down from the menu: ⌘↑ and ⌘↓
// as in Music; elsewhere Ctrl+↑ and Ctrl+↓ move through the songs, so Alt
// joins them.
func volumeAccelerators() (up, down string) {
	if runtime.GOOS == "darwin" {
		return "Cmd+Up", "Cmd+Down"
	}
	return "Ctrl+Alt+Up", "Ctrl+Alt+Down"
}

// nudgeVolume turns the volume by by, and says where it is.
func (a *app) nudgeVolume(by float64) {
	a.volume = min(1, max(0, a.volume+by))
	a.pl.SetVolume(a.volume)
	a.toast = Tf("Volume %d%%", int(a.volume*100+0.5))
}

// shortcutList is what the listener can do from the keyboard, as the
// window of shortcuts lists it.
func shortcutList() [][2]string {
	cmd := keyName("⌘", "Ctrl+")
	up, down := keyName("⌘↑", "Ctrl+Alt+↑"), keyName("⌘↓", "Ctrl+Alt+↓")
	return [][2]string{
		{T("Play/Pause"), T("Space")},
		{T("Next"), cmd + "→"},
		{T("Previous"), cmd + "←"},
		{T("Volume Up"), up},
		{T("Volume Down"), down},
		{T("Like/Unlike"), cmd + "D"},
		{T("Now Playing"), cmd + "L"},
		{T("Search"), cmd + "F"},
		{T("Back"), cmd + "["},
		{T("Forward"), cmd + "]"},
		{T("New Playlist…"), cmd + "N"},
		{T("Select All Songs"), cmd + "A"},
		{T("Play the Chosen Song"), "Enter"},
		{T("Select More Songs"), keyName("⌘ / ⇧ + Click", "Ctrl / Shift + Click")},
		{T("Switch Now Playing Tabs"), cmd + "1…5"},
		{T("Close Now Playing"), "Esc"},
		{T("Settings…"), cmd + ","},
		{T("Keyboard Shortcuts"), cmd + "/"},
	}
}

// shortcutsDialog lists the keyboard shortcuts, two columns of them.
func (a *app) shortcutsDialog(c *ui.Context) {
	p := a.pal
	ui.DialogBase(c.Key("shortcuts"), &a.shortcutsOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
		panel.Width(620).Padding(26).Radius(18).Gap(14).
			Background(p.bg).Border(1, p.hairline).Shadow(0, 24, 60, -12, ui.RGBA(0, 0, 0, 0.4)).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.94}, Exit: &ui.Motion{Scale: 0.96}})
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Icon(c, iconKeyboard).Size(20, 20).TextColor(p.accent)
			ui.Text(c, T("Keyboard Shortcuts")).FontSize(17).FontWeight(700).Grow(1)
			if iconButton(c.Key("close"), iconX, 14, p.muted, T("Close")).Clicked() {
				a.shortcutsOpen = false
			}
		})
		list := shortcutList()
		ui.Grid(c).Columns(2).GapX(28).GapY(2).Children(func() {
			for i, s := range list {
				ui.Row(c.Key(i)).Height(32).AlignItems(ui.Center).Gap(12).Children(func() {
					ui.Text(c, s[0]).FontSize(13).TextColor(p.text).Grow(1).Shrink(1).MinWidth(0).SingleLine()
					ui.Text(c, s[1]).FontSize(12).FontWeight(600).FontFeatures("tnum").TextColor(p.muted).
						Padding(3, 8).Radius(6).Background(p.hover).SingleLine()
				})
			}
		})
	})
}

// upNext is the name of the song that plays next, if there is one.
func (a *app) upNext() string {
	order, at := a.pl.PlayOrder()
	q := a.pl.Queue()
	if at < 0 || len(order) == 0 {
		return ""
	}
	next := at + 1
	if next >= len(order) {
		if a.pl.Repeat() == player.RepeatOff && a.pl.Radio() == "" {
			return ""
		}
		next = 0
	}
	if a.pl.Repeat() == player.RepeatOne {
		next = at
	}
	if i := order[next]; i < len(q) {
		return fmt.Sprintf("%s · %s", q[i].Name, q[i].ArtistNames())
	}
	return ""
}
