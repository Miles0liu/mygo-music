package app

import (
	"math"

	"github.com/egoist/mygo/ui"
	"mygo-music/internal/music"
)

type wallWarmState struct {
	queue    []music.Track
	w, h     float32
	col, row int
}

// Warm the initial wall viewport while listening elsewhere, and its next
// row and column as it is dragged. The decoded size matches wallTile's use.
func (a *app) warmWall(c *ui.Context) {
	if a.art == nil {
		return
	}
	queue := a.pl.Queue()
	order, at := a.pl.PlayOrder()
	if len(queue) == 0 || at < 0 || at >= len(order) {
		if len(a.wallWarm.queue) > 0 {
			a.wallWarm = wallWarmState{}
			a.art.prefetch(nil)
		}
		return
	}
	w, h := c.Size()
	h -= topBar(c) + npTabsHeight + min(max(h*0.36, 200), 320) - 36
	l := layWall(len(queue), w, h, 72, 112)
	off := l.centre(order[at], w, h, 72, 112)
	if a.nowPlaying && a.npPane == paneWall && a.wallState.track != "" {
		off = addPoint(a.wallState.shown, a.wallState.pan)
	}
	col := int(math.Floor(float64(-off.X / wallPitch)))
	row := int(math.Floor(float64(-off.Y / wallPitch)))
	was := a.wallWarm
	if sameList(was.queue, queue) && was.w == w && was.h == h && was.col == col && was.row == row {
		return
	}
	a.wallWarm = wallWarmState{queue: queue, w: w, h: h, col: col, row: row}
	var urls []string
	seen := map[string]bool{}
	cols, rows := int(math.Ceil(float64(w/wallPitch))), int(math.Ceil(float64(h/wallPitch)))
	for r := max(0, row-1); r <= min(l.rows-1, row+rows+1); r++ {
		for c := max(0, col-1); c <= min(l.cols-1, col+cols+1); c++ {
			i := r*l.cols + c
			if i >= len(queue) {
				break
			}
			url := sized(queue[i].Album.ImageURL, 240)
			if url != "" && !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
		}
	}
	a.art.prefetch(urls)
}
