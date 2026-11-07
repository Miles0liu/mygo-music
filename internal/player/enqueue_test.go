package player

import (
	"strings"
	"testing"

	"mygo-music/internal/music"
)

func ids(q []music.Track) string {
	s := make([]string, len(q))
	for i, t := range q {
		s[i] = t.ID
	}
	return strings.Join(s, "")
}

// Play Next puts songs right after the one playing, moving those already
// queued, and leaves the one playing where it is.
func TestPlayNext(t *testing.T) {
	p := &Player{at: -1}
	p.Cue([]music.Track{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}})
	p.at, p.track = 1, &p.queue[1]
	p.PlayNext([]music.Track{{ID: "d"}, {ID: "x"}, {ID: "b"}})
	if got := ids(p.Queue()); got != "abdxc" {
		t.Fatalf("queue %s", got)
	}
	if p.Current().ID != "b" || p.at != 1 {
		t.Fatalf("current %s at %d", p.Current().ID, p.at)
	}
}

// Add to Queue puts songs at the end of what plays, in the shuffled order
// the listener was hearing.
func TestAddToQueueKeepsShuffledOrder(t *testing.T) {
	p := &Player{at: -1}
	p.Cue([]music.Track{{ID: "a"}, {ID: "b"}, {ID: "c"}})
	p.order = []int{2, 0, 1}
	p.at, p.track = 0, &p.queue[2]
	p.AddToQueue([]music.Track{{ID: "a"}, {ID: "y"}})
	if got := ids(p.Queue()); got != "cbay" {
		t.Fatalf("queue %s", got)
	}
	if p.Current().ID != "c" || p.at != 0 {
		t.Fatalf("current %s at %d", p.Current().ID, p.at)
	}
}
