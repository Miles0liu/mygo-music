package app

import (
	"cmp"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"mygo-music/internal/i18n"
	"mygo-music/internal/music"
)

// What a page's songs can be sorted by, from their column headers.
const (
	sortNone = iota
	sortTitle
	sortAlbum
	sortLength
)

// trackSort is how the listener sorted the songs of a page: by is one of the
// sort keys, and it lasts until another page opens.
type trackSort struct {
	route route
	by    int
	desc  bool
}

// sortedList keeps the songs of a page as sorted, for the next frame to reuse
// while neither the songs nor the order changed.
type sortedList struct {
	first  *music.Track
	n      int
	sort   trackSort
	tracks []music.Track
}

var (
	iconPlayNext = strokeIcon(`<path d="M3 6h11"/><path d="M3 12h7"/><path d="M3 18h7"/><path d="M15 13v7l6-3.5z"/>`)
	iconClock    = strokeIcon(`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`)
	iconKeyboard = strokeIcon(`<rect x="2" y="5" width="20" height="14" rx="2.5"/><path d="M6 9h.01M10 9h.01M14 9h.01M18 9h.01M6 13h.01M18 13h.01M10 13h4M7 16.5h10"/>`)
)

// trackWide says whether the songs of the page have room for the column of
// their albums. An album's page has none: its songs are all on it.
func (a *app) trackWide(c *ui.Context) bool {
	w, _ := c.Size()
	return w-sidebarWidth > 820 && a.route.page != pageAlbum
}

// sortable says whether the page's songs can be sorted: all of them must be
// there, not a page of them with more to come.
func (a *app) sortable() bool {
	switch a.route.page {
	case pageLiked:
		return !a.library.likedPaged.more
	case pageHome:
		return false
	}
	return true
}

// sortedTracks returns tracks in the order the listener sorted the page by,
// or as they are.
func (a *app) sortedTracks(tracks []music.Track) []music.Track {
	s := a.trackSort
	if s.route != a.route || s.by == sortNone || len(tracks) < 2 || !a.sortable() {
		return tracks
	}
	k := &a.trackSorted
	if k.first == &tracks[0] && k.n == len(tracks) && k.sort == s {
		return k.tracks
	}
	tag := language.English
	if i18n.Language() == i18n.Chinese {
		tag = language.SimplifiedChinese
	}
	coll := collate.New(tag, collate.Loose, collate.Numeric)
	out := slices.Clone(tracks)
	slices.SortStableFunc(out, func(x, y music.Track) int {
		var r int
		switch s.by {
		case sortTitle:
			r = coll.CompareString(x.Name, y.Name)
		case sortAlbum:
			r = coll.CompareString(x.Album.Name, y.Album.Name)
		case sortLength:
			r = cmp.Compare(x.Duration, y.Duration)
		}
		if s.desc {
			r = -r
		}
		return r
	})
	*k = sortedList{first: &tracks[0], n: len(tracks), sort: s, tracks: out}
	return out
}

// trackColumns heads the columns of the page's songs, lined up with their
// rows: a click on one sorts the songs by it, again the other way, and a
// third time puts them back as they were.
func (a *app) trackColumns(c *ui.Context, covers bool) {
	p := a.pal
	wide := a.trackWide(c)
	sortable := a.sortable()
	ui.Column(c.Key("columns")).Margin(0, 0, 6, 0).Children(func() {
		ui.Row(c).Height(30).PaddingX(12).Gap(14).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Width(26).Center().Children(func() {
				ui.Text(c, "#").FontSize(12).TextColor(p.faint)
			})
			if covers {
				ui.Box(c).Width(38)
			}
			a.columnHead(c, T("Title"), sortTitle, sortable, false).Grow(3).Basis(0).MinWidth(0)
			if wide {
				a.columnHead(c, T("Album"), sortAlbum, sortable, false).Grow(2).Basis(0).MinWidth(0)
			}
			ui.Box(c).Width(32)
			ui.Box(c).Width(32)
			a.columnHead(c, "", sortLength, sortable, true).Width(44)
		})
		ui.Box(c).Height(1).Margin(0, 12).Background(p.hairline)
	})
}

// columnHead is the head of one column of songs, which sorts them by by.
// The column of lengths has a clock in place of a name.
func (a *app) columnHead(c *ui.Context, label string, by int, sortable, end bool) ui.Element {
	p := a.pal
	s := a.trackSort
	on := s.route == a.route && s.by == by && sortable
	name := label
	if name == "" {
		name = T("Duration")
	}
	b := ui.ButtonBase(c.Key(by)).Height(26).Gap(4).AlignItems(ui.Center).FocusRing(false).Label(Tf("Sort by %s", name))
	if end {
		b.Justify(ui.End)
	} else {
		b.Justify(ui.Start)
	}
	if !sortable {
		b.Disabled(true)
	}
	color := p.faint
	if on || b.Hovered() {
		color = p.text
	}
	b.Children(func() {
		if label == "" {
			ui.Icon(c, iconClock).Size(13, 13).TextColor(color)
		} else {
			ui.Text(c, label).FontSize(12).FontWeight(600).TextColor(color).SingleLine()
		}
		if on {
			arrow := ui.Icon(c, iconDown).Size(11, 11).TextColor(p.accent)
			if !s.desc {
				arrow.Rotate(180)
			}
		}
	})
	if b.Clicked() {
		switch {
		case !on:
			a.trackSort = trackSort{route: a.route, by: by}
		case !s.desc:
			a.trackSort.desc = true
		default:
			a.trackSort = trackSort{}
		}
	}
	return b
}

// selectionBar floats over the foot of the page while songs are selected:
// how many, and what can be done with them all, without a right click.
func (a *app) selectionBar(c *ui.Context, tracks []music.Track, bottom float32, pid string) {
	p := a.pal
	selected := a.selectedTracks(tracks)
	n := len(selected)
	shown := float32(0)
	if n > 1 {
		shown = 1
	}
	row := ui.Row(c.Key("selection-bar")).Absolute().Left(0).Right(0).Bottom(bottom + 18).Justify(ui.Center).PassThrough()
	v := row.AnimateTo("shown", shown, ui.Snappy)
	if v < 0.01 {
		return
	}
	playable := slices.DeleteFunc(slices.Clone(selected), func(tr music.Track) bool { return tr.Unavailable })
	pop := popoverColors(p.dark)
	row.Children(func() {
		bar := ui.Row(c).Height(46).Padding(0, 6, 0, 16).Gap(4).Radius(999).AlignItems(ui.Center).
			Background(pop.fill).Border(0.5, pop.edge).Shadow(0, 10, 30, -6, ui.RGBA(0, 0, 0, 0.28)).
			Opacity(v).Translate(0, (1-v)*12)
		bar.Children(func() {
			ui.Text(c, Tf("%d songs selected", max(n, 2))).FontSize(13).FontWeight(600).TextColor(p.text).Margin(0, 8, 0, 0).SingleLine()
			action := func(key string, icon *ui.SVG, label string, disabled bool) ui.Element {
				b := iconButton(c.Key(key), icon, 16, p.text, label).Size(34, 34).Tooltip(label)
				if disabled {
					b.Disabled(true).Opacity(0.35)
				}
				return b
			}
			if action("play", iconPlay, T("Play Selected"), len(playable) == 0).Clicked() {
				a.pl.PlayQueue(playable, 0)
			}
			if action("next", iconPlayNext, T("Play Next"), len(playable) == 0).Clicked() {
				a.queueNext(playable, true)
			}
			if action("queue", iconQueue, T("Add to Queue"), len(playable) == 0).Clicked() {
				a.queueNext(playable, false)
			}
			if len(selected) > 0 {
				more := action("more", iconMore, T("More"), false)
				i := slices.IndexFunc(tracks, func(tr music.Track) bool { return trackIdentity(tr) == trackIdentity(selected[0]) })
				if i >= 0 {
					more.Menu(a.selectedSongItems(tracks, i, pid))
				}
			}
			ui.Box(c).Width(1).Height(20).Background(p.hairline).Margin(0, 4)
			if action("clear", iconX, T("Deselect All"), false).Clicked() {
				a.trackSelection.Clear()
			}
		})
	})
}

// pageBar is what the bar over a page of songs shows once the page's header
// has scrolled under it: its title, and a button to play its songs.
type pageBar struct {
	title  string
	cover  string
	tracks []music.Track
	past   bool // the header has scrolled under the bar
}

// collectionTint washes the top of a page of songs in the color of its cover,
// fading into the page, and scrolls away with the header.
func (a *app) collectionTint(c *ui.Context) {
	tint, ok := a.coverTint(a.bar.cover)
	if !ok {
		return
	}
	shown := ui.Box(c.Key("tint")).Absolute().Top(0).Left(0).Right(0).Height(420).PassThrough()
	in := shown.AnimateTo("in", 1, ui.Animation{Duration: 400 * time.Millisecond})
	strength := float32(0.32)
	if !a.pal.dark {
		strength = 0.2
	}
	shown.Translate(0, -min(a.trackScroll.Y, 420)).
		LinearGradient(ui.LinearGradient{Angle: 180, Oklab: true, From: tint.Alpha(strength * in), To: tint.Alpha(0)})
}

// coverTint returns the most vivid color of the cover at url, once loaded.
func (a *app) coverTint(url string) (ui.Color, bool) {
	if url == "" {
		return ui.Color{}, false
	}
	art := a.art.get(sized(url, 600), false)
	if art == nil || art.tint.A == 0 {
		return ui.Color{}, false
	}
	return art.tint, true
}

// compactBar shows the page's title over the page once its header has
// scrolled away, with a button to play it.
func (a *app) compactBar(c *ui.Context) {
	p := a.pal
	b := a.bar
	shown := float32(0)
	if b.title != "" && b.past {
		shown = 1
	}
	row := ui.Row(c.Key("compact-bar")).Absolute().Top(0).Left(0).Right(0).Height(topBar(c)).AlignItems(ui.Center).Justify(ui.Center).PassThrough()
	v := row.AnimateTo("shown", shown, ui.Animation{Duration: 160 * time.Millisecond})
	if v < 0.01 || b.title == "" {
		return
	}
	row.Opacity(v).Children(func() {
		ui.Text(c, b.title).FontSize(14).FontWeight(700).TextColor(p.text).SingleLine().MaxWidth(420)
	})
	if len(b.tracks) == 0 {
		return
	}
	ui.Row(c.Key("compact-play")).Absolute().Top(0).Right(20).Height(topBar(c)).AlignItems(ui.Center).Opacity(v).Children(func() {
		btn := ui.ButtonBase(c.Key("play")).Size(30, 30).Radius(999).Center().Background(p.accent).TextColor(p.onAccent).
			Label(T("Play")).FocusRing(false).Disabled(v < 0.5)
		btn.Children(func() { ui.Icon(c, iconPlay).Size(12, 12).Margin(0, 0, 0, 2) })
		if btn.Clicked() {
			if a.pl.Shuffle() {
				a.pl.ToggleShuffle()
			}
			a.pl.PlayQueue(b.tracks, firstPlayable(b.tracks))
		}
	})
}

// totalLength says how long tracks play together, as a page's header does.
func totalLength(tracks []music.Track) string {
	var total time.Duration
	for i := range tracks {
		total += tracks[i].Length()
	}
	if total <= 0 {
		return ""
	}
	minutes := int(total.Round(time.Minute) / time.Minute)
	if minutes >= 60 {
		return Tf("%d hr %d min", minutes/60, minutes%60)
	}
	return Tf("%d min", max(minutes, 1))
}

// playingFrom remembers which loaded playlist the queue is, for the queue
// and the song it was found for.
type playingFrom struct {
	track string
	n     int
	id    string
}

// playingPlaylist is the ID of the listener's playlist the queue playing came
// from, as far as can be told: a loaded one with the same songs, or "".
func (a *app) playingPlaylist() string {
	tr := a.pl.Current()
	if tr == nil {
		return ""
	}
	q := a.pl.Queue()
	key := trackIdentity(*tr)
	if a.playingList.track == key && a.playingList.n == len(q) {
		return a.playingList.id
	}
	id := ""
	for pid, pl := range a.playlists {
		if pl == nil || len(pl.Tracks) != len(q) || len(q) == 0 {
			continue
		}
		// Given in the playlist's order, whether shuffled or not: the same
		// ends, and the song playing among them.
		n := len(q) - 1
		if sameTrack(&pl.Tracks[0], &q[0]) && sameTrack(&pl.Tracks[n], &q[n]) &&
			slices.ContainsFunc(q, func(t music.Track) bool { return sameTrack(&t, tr) }) {
			id = pid
			break
		}
	}
	a.playingList = playingFrom{track: key, n: len(q), id: id}
	return id
}
