package app

import (
	"math"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"mygo-music/internal/music"
)

func trackIdentity(tr music.Track) string { return tr.Source + "\x00" + tr.ID }

// Selection survives focus moving into a menu and follows songs when rows move.
func (a *app) prepareTrackSelection(c *ui.Context, tracks []music.Track) {
	if !a.trackSelectionReady || a.trackSelectionRoute != a.route {
		a.trackSelection.Clear()
		a.trackAnchor, a.trackFocus, a.trackCursor = "", "", -1
		a.trackSelectionRoute, a.trackSelectionReady = a.route, true
	}
	a.trackCursor = slices.IndexFunc(tracks, func(tr music.Track) bool { return trackIdentity(tr) == a.trackFocus })
	present := make(map[string]bool, len(tracks))
	for _, tr := range tracks {
		present[trackIdentity(tr)] = true
	}
	for key := range a.trackSelection.All() {
		if !present[key] {
			a.trackSelection.Remove(key)
		}
	}
	a.trackList.OnShortcut(c, ui.Cmd, ui.KeyA, func() {
		for _, tr := range tracks {
			a.trackSelection.Add(trackIdentity(tr))
		}
	})
	a.trackList.OnShortcut(c, 0, ui.KeyEscape, func() { a.trackSelection.Clear() })
	a.trackList.OnShortcut(c, 0, ui.KeyEnter, func() {
		if i := a.trackCursor; i >= 0 && i < len(tracks) && !tracks[i].Unavailable {
			a.pl.PlayTrack(tracks, i)
		}
	})
	for _, mods := range []ui.Modifiers{0, ui.Shift, ui.Ctrl, ui.Ctrl | ui.Shift} {
		for _, direction := range []struct {
			key ui.Key
			by  int
		}{{ui.KeyUp, -1}, {ui.KeyDown, 1}} {
			a.trackList.OnShortcut(c, mods, direction.key, func() {
				a.trackLocate = ""
				if len(tracks) == 0 {
					return
				}
				i := max(0, min(len(tracks)-1, a.trackCursor+direction.by))
				if mods == ui.Ctrl {
					a.trackCursor = i
					a.trackFocus = trackIdentity(tracks[i])
				} else {
					a.selectTrack(tracks, i, mods)
				}
				a.trackList.ScrollIntoView(i + 1)
			})
		}
	}
}

func (a *app) selectTrack(tracks []music.Track, i int, mods ui.Modifiers) {
	a.trackLocate = ""
	key := trackIdentity(tracks[i])
	additive := mods&(ui.Ctrl|ui.Cmd) != 0
	if mods&ui.Shift != 0 {
		anchor := slices.IndexFunc(tracks, func(tr music.Track) bool { return trackIdentity(tr) == a.trackAnchor })
		if anchor < 0 {
			anchor = i
			a.trackAnchor = key
		}
		if !additive {
			a.trackSelection.Clear()
		}
		for n := min(anchor, i); n <= max(anchor, i); n++ {
			a.trackSelection.Add(trackIdentity(tracks[n]))
		}
	} else {
		if additive && a.trackSelection.Has(key) {
			a.trackSelection.Remove(key)
		} else {
			if !additive {
				a.trackSelection.Clear()
			}
			a.trackSelection.Add(key)
		}
		a.trackAnchor = key
	}
	a.trackCursor, a.trackFocus = i, key
}

func (a *app) selectedTracks(tracks []music.Track) []music.Track {
	var selected []music.Track
	for _, tr := range tracks {
		if a.trackSelection.Has(trackIdentity(tr)) {
			selected = append(selected, tr)
		}
	}
	return selected
}

func (a *app) selectedSongItems(tracks []music.Track, i int, pid string) func(*ui.Menu) {
	return func(m *ui.Menu) {
		if !a.trackSelection.Has(trackIdentity(tracks[i])) {
			a.selectTrack(tracks, i, 0)
		}
		selected := a.selectedTracks(tracks)
		if len(selected) == 1 {
			a.songItems(&tracks[i], pid)(m)
			return
		}
		m.Item(Tf("%d songs selected", len(selected))).Disabled(true)
		m.Separator()
		playable := slices.DeleteFunc(slices.Clone(selected), func(tr music.Track) bool { return tr.Unavailable })
		if m.Item(T("Play Selected")).Disabled(len(playable) == 0).Chosen() {
			a.pl.PlayQueue(playable, 0)
		}
		m.Separator()
		if m.Item(T("Copy Titles and Artists")).Chosen() {
			lines := make([]string, len(selected))
			for n, tr := range selected {
				lines[n] = tr.Name + " - " + tr.ArtistNames()
			}
			copyText(strings.Join(lines, "\n"))
		}
		lib := a.trackLib(&tracks[i])
		if lib.signedOut() {
			return
		}
		ids := make([]string, 0, len(selected))
		for _, tr := range selected {
			if tr.Source == lib.id && !slices.Contains(ids, tr.ID) {
				ids = append(ids, tr.ID)
			}
		}
		if len(ids) == 0 {
			return
		}
		if lib.can("addToPlaylist") {
			m.Separator()
			m.Submenu(T("Add to Playlist"), func(m *ui.Menu) {
				for _, pl := range lib.mine {
					if pl.ID != pid && !pl.Fixed && m.Item(pl.Name).Chosen() {
						a.changeSelectedPlaylist(lib, pl.ID, ids, false)
					}
				}
			})
		}
		if lib == a.library && a.editable(pid) != nil && lib.can("removeFromPlaylist") {
			if m.Item(T("Remove from Playlist")).Chosen() {
				a.changeSelectedPlaylist(lib, pid, ids, true)
			}
		}
	}
}

// Apply the batch once, then reload from the source so a failed operation
// cannot roll back a more recent edit with a stale optimistic snapshot.
func (a *app) changeSelectedPlaylist(lib *library, pid string, ids []string, remove bool) {
	ids = slices.Clone(ids)
	go func() {
		var err error
		if remove {
			err = lib.src.RemoveFromPlaylist(pid, ids)
		} else {
			err = lib.src.AddToPlaylist(pid, ids)
		}
		a.post(func() {
			if err != nil {
				a.actionFailed(lib, err, T("Couldn't update the playlist. Try again later."))
			}
			a.playlistChanged(lib, pid)
		})
	}()
}

func (a *app) locatePlayingButton(c *ui.Context, tracks []music.Track, bottom float32) {
	at := slices.IndexFunc(tracks, func(tr music.Track) bool { return sameTrack(a.pl.Current(), &tr) })
	if at < 0 || a.playingTrackVisible(c, at+1) {
		return
	}
	p := a.pal
	ui.Box(c.Key("locate-playing")).Absolute().Right(24).Bottom(bottom + 20).Children(func() {
		if iconButton(c, iconLocate, 20, p.text.Alpha(0.85), T("Locate Playing Song")).
			Size(42, 42).Radius(21).Background(p.bg.Alpha(0.72)).
			Border(1, p.text.Alpha(0.10)).Shadow(0, 3, 12, 0, p.text.Alpha(0.08)).Tooltip(T("Locate Playing Song")).Clicked() {
			a.selectTrack(tracks, at, 0)
			a.trackLocate, a.trackLocateTo = trackIdentity(tracks[at]), -1
			a.advanceTrackLocate(c, tracks, bottom)
			a.trackVisibilityFrame = true
		}
	})
}

// Virtual lists revise their estimated offsets as rows are measured. Anchor
// the destination to an actual visible song so a tall playlist header cannot
// make a long glide end hundreds of songs short of its target.
func (a *app) advanceTrackLocate(c *ui.Context, tracks []music.Track, bottom float32) {
	if a.trackLocate == "" {
		return
	}
	at := slices.IndexFunc(tracks, func(tr music.Track) bool { return trackIdentity(tr) == a.trackLocate })
	if at < 0 || !sameTrack(a.pl.Current(), &tracks[at]) {
		a.trackLocate = ""
		return
	}
	if !a.trackScroll.Scrolling() && a.playingTrackVisible(c, at+1) {
		a.trackLocate = ""
		return
	}
	row, viewport := a.trackReference.Bounds(c), a.trackList.Bounds(c)
	if row.H == 0 || viewport.H == 0 {
		return
	}
	top, _ := a.insets(c)
	y := a.trackScroll.Y + float32(at-a.trackReferenceIndex)*54 + row.Y + row.H/2 - viewport.Y - (top+viewport.H-bottom)/2
	y = max(0, min(y, a.trackScroll.MaxY))
	if math.Abs(float64(y-a.trackLocateTo)) > 0.5 {
		a.trackLocateTo = y
		a.trackScroll.ScrollTo(0, y, ui.Animation{Spring: ui.SpringOptions{Duration: 420 * time.Millisecond}})
	}
}

// Rows behind the header/player or built as virtual-list overscan are not
// visible to the listener. Any exposed part of the song row counts as visible.
func (a *app) playingTrackVisible(c *ui.Context, row int) bool {
	first, last := a.trackList.Visible()
	if row < first || row > last {
		return false
	}
	bounds := a.trackPlayingRow.Bounds(c)
	viewport := a.trackList.Bounds(c)
	top, bottom := a.insets(c)
	return bounds.H > 0 && bounds.Y+bounds.H > viewport.Y+top && bounds.Y < viewport.Y+viewport.H-bottom
}
