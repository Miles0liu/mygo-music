package app

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

// A column's head sorts the songs by it, again the other way, and a third
// time puts them back in the playlist's order.
func TestColumnHeadSorts(t *testing.T) {
	a, tt, tracks := selectionView(t, 4)
	above := func(x, y string) bool {
		rx, okx := tt.Find(x)
		ry, oky := tt.Find(y)
		return okx && oky && rx.Y < ry.Y
	}
	if err := tt.Click("Sort by Title"); err != nil {
		t.Fatal(err)
	}
	if !above(tracks[0].Name, tracks[3].Name) {
		t.Fatal("ascending title sort moved the first song down")
	}
	if err := tt.Click("Sort by Title"); err != nil {
		t.Fatal(err)
	}
	if !above(tracks[3].Name, tracks[0].Name) || !a.trackSort.desc {
		t.Fatal("second click did not reverse the songs")
	}
	if err := tt.Click("Sort by Title"); err != nil {
		t.Fatal(err)
	}
	if !above(tracks[0].Name, tracks[3].Name) || a.trackSort.by != sortNone {
		t.Fatal("third click did not restore the order")
	}
}

// With two songs selected, the bar over the page puts them after the song
// playing, in the order they are listed.
func TestSelectionBarPlaysNext(t *testing.T) {
	a, tt, tracks := selectionView(t, 6)
	a.pl.Cue([]music.Track{{ID: "x", Source: "fake", Name: "Playing"}, {ID: "y", Source: "fake", Name: "After"}})
	if err := tt.Click(tracks[4].Name); err != nil {
		t.Fatal(err)
	}
	if err := tt.ClickWith(ui.Ctrl, tracks[2].Name); err != nil {
		t.Fatal(err)
	}
	// The bar rises into view.
	tt.Advance(400 * time.Millisecond)
	if err := tt.Click("Play Next"); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, tr := range a.pl.Queue() {
		ids = append(ids, tr.ID)
	}
	if got := strings.Join(ids, ","); got != "x,2,4,y" {
		t.Fatalf("queue %s", got)
	}
}
