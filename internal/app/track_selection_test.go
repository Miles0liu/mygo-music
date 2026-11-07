package app

import (
	"errors"
	"fmt"
	"github.com/egoist/mygo/ui"
	"mygo-music/internal/i18n"
	"mygo-music/internal/lyrics"
	"mygo-music/internal/music"
	"slices"
	"testing"
	"time"
)

// Batch removal sends one request with every selected ID, and keeps the
// visible playlist intact until its source confirms the edit.
func TestSelectedPlaylistBatchFailure(t *testing.T) {
	a := editTestApp()
	ch := posts(a)
	called := make(chan []string, 1)
	fake(a).removeFrom = func(pid string, ids []string) error {
		if pid != "10" {
			t.Errorf("playlist = %s", pid)
		}
		called <- slices.Clone(ids)
		return errors.New("offline")
	}
	tracks := a.playlists["10"].Tracks
	for i := range tracks {
		tracks[i].Source = "fake"
	}
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	if err := tt.Click(tracks[0].Name); err != nil {
		t.Fatal(err)
	}
	if err := tt.ClickWith(ui.Ctrl, tracks[1].Name); err != nil {
		t.Fatal(err)
	}
	if err := tt.RightClick(tracks[0].Name); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Remove from Playlist"); err != nil {
		t.Fatal(err)
	}
	if got := wait(t, called, "batch removal"); !slices.Equal(got, []string{"1", "2"}) {
		t.Fatal(got)
	}
	if len(a.playlists["10"].Tracks) != 2 {
		t.Fatal("unconfirmed edit removed songs")
	}
	runPost(t, ch)
}

// Selecting a different route clears the previous list's selection and cursor.
func TestTrackSelectionRouteChange(t *testing.T) {
	a, tt, tr := selectionView(t, 8)
	if err := tt.Click(tr[1].Name); err != nil {
		t.Fatal(err)
	}
	a.route.id = "another"
	tt.Frame()
	if a.trackSelection.Len() != 0 || a.trackCursor != -1 {
		t.Fatal("selection leaked to another playlist")
	}
}

// With MM_FRAMES, inspect the selected rows underneath a batch context menu.
func TestSelectedPlaylistMenuFrame(t *testing.T) {
	dir := framesDir(t)
	_, tt, tracks := selectionView(t, 8)
	if err := tt.Click(tracks[0].Name); err != nil {
		t.Fatal(err)
	}
	if err := tt.ClickWith(ui.Shift, tracks[3].Name); err != nil {
		t.Fatal(err)
	}
	if err := tt.RightClick(tracks[1].Name); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	tt.Frame()
	tt.Frame()
	saveFrame(t, dir, "selected-playlist-menu.png", tt.Image())
}

func selectionView(t *testing.T, count int) (*app, *ui.Tester, []music.Track) {
	t.Helper()
	a := newTestApp()
	tracks := make([]music.Track, count)
	for i := range tracks {
		tracks[i] = music.Track{ID: fmt.Sprint(i), Source: "fake", Name: fmt.Sprintf("Track %03d", i), Duration: 120000}
	}
	a.route = route{page: pagePlaylist, src: "fake", id: "test"}
	tt := ui.NewTester(func(c *ui.Context) {
		a.pal = paletteFor(c.Theme())
		c.Root().TextColor(a.pal.text)
		c.SetMenuPresentation(ui.PopoverMenus)
		a.trackRows(c, tracks, false, func(c *ui.Context) { ui.Text(c, "Playlist").Height(40) })
	}, 1200, 800)
	tt.Frame()
	return a, tt, tracks
}

// A single click only selects; double click and Enter explicitly start playback.
func TestTrackSelectionActivation(t *testing.T) {
	a, tt, tracks := selectionView(t, 8)
	if err := tt.Click(tracks[0].Name); err != nil {
		t.Fatal(err)
	}
	if a.pl.Current() != nil || a.trackSelection.Len() != 1 {
		t.Fatal("single click played or did not select")
	}
	tt.Advance(80 * time.Millisecond)
	if err := tt.Click(tracks[0].Name); err != nil {
		t.Fatal(err)
	}
	if !sameTrack(a.pl.Current(), &tracks[0]) {
		t.Fatal("double click did not play")
	}
	tt.Advance(time.Second)
	if err := tt.Click(tracks[2].Name); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !sameTrack(a.pl.Current(), &tracks[2]) {
		t.Fatal("Enter did not play selection")
	}
}

// Ctrl toggles a row, Shift extends a range, and menus keep the batch selected.
func TestTrackSelectionModifiersAndMenu(t *testing.T) {
	a, tt, tr := selectionView(t, 8)
	for _, step := range []struct {
		mods  ui.Modifiers
		row   int
		count int
	}{{0, 0, 1}, {ui.Ctrl, 2, 2}, {ui.Shift, 4, 3}, {ui.Ctrl, 0, 4}, {ui.Ctrl, 0, 3}} {
		tt.Advance(time.Second)
		if err := tt.ClickWith(step.mods, tr[step.row].Name); err != nil {
			t.Fatal(err)
		}
		if a.trackSelection.Len() != step.count {
			t.Fatalf("selection = %d, want %d", a.trackSelection.Len(), step.count)
		}
	}
	if err := tt.RightClick(tr[3].Name); err != nil {
		t.Fatal(err)
	}
	if a.trackSelection.Len() != 3 || !slices.Contains(tt.Menu(), "Play Selected") {
		t.Fatal("right click lost selected batch")
	}
	if err := tt.ChooseMenuItem("Play Selected"); err != nil {
		t.Fatal(err)
	}
	if len(a.pl.Queue()) != 3 || a.pl.Queue()[0].ID != tr[2].ID {
		t.Fatal("batch playback order changed")
	}
	if err := tt.RightClick(tr[0].Name); err != nil {
		t.Fatal(err)
	}
	if a.trackSelection.Len() != 1 || !a.trackSelection.Has(trackIdentity(tr[0])) {
		t.Fatal("right click did not select the new row")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if err := tt.Click(tr[1].Name); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Frame()
	if a.trackSelection.Len() != len(tr) {
		t.Fatal("Ctrl+A did not select all songs")
	}
}

// Locate scrolls an off-screen playing song into view, without restarting it.
func TestLocatePlayingTrack(t *testing.T) {
	a, tt, tr := selectionView(t, 150)
	a.pl.Cue([]music.Track{tr[120]})
	tt.Frame()
	if err := tt.Click("Locate Playing Song"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := tt.Find(tr[120].Name); !ok {
		t.Fatal("playing song is not visible")
	}
	if a.pl.Playing() || !a.trackSelection.Has(trackIdentity(tr[120])) {
		t.Fatal("locating changed playback or lost highlight")
	}
	if _, ok := tt.Find("Locate Playing Song"); ok {
		t.Fatal("locate button stayed after the song became visible")
	}
	// The list still builds this row, but the page header covers it.
	a.trackList.ScrollTo(121, ui.Start)
	tt.Frame()
	tt.Frame()
	tt.Scroll(600, 300, 0, 100)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("Locate Playing Song"); !ok {
		t.Fatal("locate button did not return after scrolling the song behind the header")
	}
	if err := tt.Click("Locate Playing Song"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find("Locate Playing Song"); ok {
		t.Fatal("locate button did not disappear again")
	}
	a.pl.Cue([]music.Track{tr[0]})
	tt.Frame()
	if _, ok := tt.Find("Locate Playing Song"); !ok {
		t.Fatal("off-screen next song did not restore locate button")
	}
}

// With MM_FRAMES, inspect Chinese navigation at its real size and weight.
func TestChineseTypographyFrame(t *testing.T) {
	dir := framesDir(t)
	previous := i18n.Language()
	i18n.SetLanguage(i18n.Chinese)
	defer i18n.SetLanguage(previous)
	a := newTestApp()
	a.route = route{page: pageShows, src: "fake"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme := *c.Theme()
		applyTypography(&theme)
		c.SetTheme(&theme)
		a.pal = paletteFor(&theme)
		ui.Column(c).Width(232).Padding(12).Gap(4).Background(a.pal.sidebar).Children(func() {
			a.navItem(c, iconHome, "为你推荐", route{page: pageHome})
			a.navItem(c, iconCalendar, "每日推荐", route{page: pageBrowse})
			a.navItem(c, iconRadio, "私人 FM", route{page: pageStations})
			a.navItem(c, iconCharts, "排行榜", route{page: pageBrowse, id: "charts"})
			a.navItem(c, iconPodcast, "播客", route{page: pageShows})
		})
	}, 232, 230)
	tt.SetDark(true)
	tt.Frame()
	saveFrame(t, dir, "chinese-navigation.png", tt.Image())
}

// The vinyl pauses in place and reduced motion stops rotation.
func TestVinylRotationPause(t *testing.T) {
	var v vinylRotation
	now := time.Unix(100, 0)
	v.step(now, true, false)
	angle := v.step(now.Add(time.Second), true, false)
	if angle != 12 {
		t.Fatal(angle)
	}
	if v.step(now.Add(3*time.Second), false, false) != angle {
		t.Fatal("paused vinyl rotated")
	}
	if v.step(now.Add(40*time.Second), true, false) != angle {
		t.Fatal("resume jumped")
	}
	if v.step(now.Add(5*time.Second), true, true) != 0 {
		t.Fatal("reduced motion rotated")
	}
}

// With MM_FRAMES, inspect the cover and lyrics across desktop and ultrawide sizes.
func TestDesktopLayoutFrames(t *testing.T) {
	dir := framesDir(t)
	for _, size := range [][2]int{{1180, 780}, {2560, 1080}, {5120, 2088}} {
		for _, style := range []string{"", "vinyl"} {
			a := newTestApp()
			tracks := framesTracks(a)
			a.pl.Cue(tracks)
			a.pl.SetLyrics(lyrics.Build("[00:04.00]The sky goes green at the edges\n[00:09.00]And everyone stops to look\n[00:14.00]Nobody says a word\n[00:19.00]Nobody needs to", ""))
			a.pl.Hold(10 * time.Second)
			a.nowPlaying, a.npPane, a.settings.CoverStyle = true, paneLyrics, style
			tt := ui.NewTester(a.view, size[0], size[1])
			tt.SetDark(true)
			tt.Advance(2 * time.Second)
			tt.Frame()
			saveFrame(t, dir, fmt.Sprintf("desktop-%d-%s.png", int(size[0]), style), tt.Image())
		}
	}
}
