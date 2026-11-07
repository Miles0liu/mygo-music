package app

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"mygo-music/internal/music"
)

// The hover play control starts the row with one click while row clicks select.
func TestHoverTrackPlay(t *testing.T) {
	a, tt, tracks := selectionView(t, 8)
	r, _ := tt.Find(tracks[2].Name)
	tt.Move(r.X+40, r.Y+r.H/2)
	tt.Frame()
	if err := tt.Click("Play"); err != nil {
		t.Fatal(err)
	}
	if !sameTrack(a.pl.Current(), &tracks[2]) || !a.trackSelection.Has(trackIdentity(tracks[2])) {
		t.Fatal("inline play did not activate and select its song")
	}
}

// Reduced motion locates immediately, and a wheel gesture can interrupt a glide.
func TestLocateMotionPreferences(t *testing.T) {
	a, tt, tracks := selectionView(t, 150)
	a.pl.Cue([]music.Track{tracks[120]})
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if err := tt.Click("Locate Playing Song"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Frame()
	if _, ok := tt.Find(tracks[120].Name); !ok {
		t.Fatal("reduced motion did not locate immediately")
	}
	tt.SetPreferences(ui.Preferences{})
	a.pl.Cue([]music.Track{tracks[0]})
	tt.Frame()
	if err := tt.Click("Locate Playing Song"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(80 * time.Millisecond)
	tt.Scroll(600, 300, 0, 100)
	tt.Frame()
	if a.trackScroll.Scrolling() {
		t.Fatal("wheel did not interrupt locate")
	}
}

// Locating a deep song accounts for the tall header in the virtual list.
func TestLocateLongPlaylist(t *testing.T) {
	a := editTestApp()
	tracks := make([]music.Track, 2500)
	for i := range tracks {
		tracks[i] = song(fmt.Sprint(i), fmt.Sprintf("Long track %d", i))
	}
	a.playlists["10"].Tracks = tracks
	a.pl.Cue([]music.Track{tracks[2300]})
	tt := ui.NewTester(a.view, 1200, 800)
	tt.Frame()
	if err := tt.Click("Locate Playing Song"); err != nil {
		t.Fatal(err)
	}
	for range 120 {
		tt.Advance(time.Second / 60)
		tt.Frame()
	}
	if box, ok := tt.Find(tracks[2300].Name); !ok || box.Y < 46 || box.Y > 716 || a.trackLocate != "" {
		first, last := a.trackList.Visible()
		t.Fatalf("deep song not located: %+v, visible=%v rows=%d..%d y=%f target=%f reference=%d active=%s", box, ok, first, last, a.trackScroll.Y, a.trackLocateTo, a.trackReferenceIndex, a.trackLocate)
	}
}

// Refreshing liked songs preserves all loaded pages, including on partial failure.
func TestLikedRefreshKeepsLoadedPages(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			a := newTestApp()
			ch := posts(a)
			a.likedList = []music.Track{song("1", "one"), song("2", "two"), song("3", "three"), song("4", "four")}
			a.likedPaged = paged{loaded: true, more: true}
			a.likedAt = time.Now().Add(-2 * listsFresh)
			before := a.likedList
			fake(a).liked = func(offset int) (*music.Page, error) {
				if offset == 2 && fail {
					return nil, errors.New("offline")
				}
				return &music.Page{Tracks: slices.Clone(before[offset : offset+2]), More: true}, nil
			}
			a.freshenLiked(a.library)
			if !sameList(a.likedList, before) {
				t.Fatal("refresh cleared visible cache")
			}
			runPost(t, ch)
			if len(a.likedList) != 4 || !a.likedPaged.more {
				t.Fatal("refresh discarded loaded pages")
			}
			if fail && !sameList(a.likedList, before) {
				t.Fatal("partial answer replaced cache")
			}
		})
	}
}

// A play request joins an in-flight playlist read and account changes discard it.
func TestPlaylistPlayWhileLoading(t *testing.T) {
	for _, forget := range []bool{false, true} {
		t.Run(fmt.Sprint(forget), func(t *testing.T) {
			a := newTestApp()
			ch := posts(a)
			fake(a).playlist = func(id string) (*music.Playlist, error) {
				return &music.Playlist{ID: id, Tracks: []music.Track{song("song", "Song")}}, nil
			}
			a.loadPlaylist(a.library, "p", false)
			a.loadPlaylist(a.library, "p", true)
			if forget {
				a.library.forget()
			}
			runPost(t, ch)
			if forget {
				if a.pl.Current() != nil || len(a.playlists) != 0 {
					t.Fatal("stale account answer was used")
				}
			} else if a.pl.Current() == nil || a.pl.Current().ID != "song" {
				t.Fatal("pending play was lost")
			}
		})
	}
}

// Sidebar and card context menus play fixed, owned and saved playlists in place.
func TestPlaylistContextPlay(t *testing.T) {
	for _, kind := range []string{"fixed", "owned", "saved", "card"} {
		t.Run(kind, func(t *testing.T) {
			a := editTestApp()
			a.plAt["10"] = time.Now()
			pl := a.mine[0]
			if kind == "fixed" {
				pl.Fixed = true
				a.mine[0].Fixed = true
			}
			if kind == "saved" {
				pl.Mine = false
				pl.Saved = true
				a.mine = nil
				a.saved = []music.Playlist{pl}
			}
			tt := ui.NewTester(func(c *ui.Context) {
				a.pal = paletteFor(c.Theme())
				if kind == "card" {
					ui.Column(c).Width(220).Children(func() { a.playlistCard(c, &pl) })
				} else {
					ui.Column(c).Width(232).Children(func() { a.myPlaylists(c, "Playlists", []music.Playlist{pl}, pl.Mine, &a.mineCollapsed) })
				}
			}, 800, 600)
			tt.Frame()
			before := a.route
			if err := tt.RightClick(pl.Name); err != nil {
				t.Fatal(err)
			}
			if err := tt.ChooseMenuItem("Play"); err != nil {
				t.Fatal(err)
			}
			if a.pl.Current() == nil || a.pl.Current().ID != "1" || a.route != before {
				t.Fatal("context play failed or changed page")
			}
		})
	}
}

// Comments keep a usable composer at the bottom as the discussion scrolls.
func TestCommentComposerStaysAtBottom(t *testing.T) {
	a, tt, _ := commentsApp()
	for i := 0; i < 25; i++ {
		a.comments.list = append(a.comments.list, music.Comment{ID: fmt.Sprintf("extra-%d", i), Text: "A longer conversation with more comments."})
	}
	tt.Frame()
	input, ok := tt.Find("Write a comment…")
	if !ok || input.Y < 600 || input.Y+input.H > 800 {
		t.Fatalf("composer outside bottom area: %+v", input)
	}
	tt.Scroll(950, 400, 0, 550)
	tt.Frame()
	after, _ := tt.Find("Write a comment…")
	if after.Y != input.Y {
		t.Fatal("composer scrolled away with comments")
	}
}

// With MM_FRAMES, inspect the comment composer and playback modes in context.
func TestPlaybackRefinementFrames(t *testing.T) {
	dir := framesDir(t)
	a, tt, _ := commentsApp()
	tracks := framesTracks(a)
	a.pl.Cue(tracks)
	a.comments.track, a.comments.id = trackIdentity(tracks[0]), tracks[0].ID
	a.pl.ToggleShuffle()
	a.pl.CycleRepeat()
	for i := 0; i < 12; i++ {
		a.comments.list = append(a.comments.list, music.Comment{ID: fmt.Sprintf("extra-%d", i), Author: "听众", Text: "音乐让故事在这里继续。", Time: time.Now().UnixMilli()})
	}
	tt.Advance(time.Second)
	tt.Frame()
	saveFrame(t, dir, "comments-bottom.png", tt.Image())
	a.pl.CycleRepeat()
	tt.Frame()
	saveFrame(t, dir, "repeat-one.png", tt.Image())
	tt.SetSize(2560, 1044)
	tt.Move(800, 300)
	tt.Advance(2 * time.Second)
	tt.Frame()
	saveFrame(t, dir, "comments-wide.png", tt.Image())
	a, tt, tracks = selectionView(t, 150)
	a.pl.Cue([]music.Track{tracks[120]})
	tt.Frame()
	saveFrame(t, dir, "locate-icon.png", tt.Image())
}
