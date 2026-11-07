package app

import (
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

func npContentScale(w float32) float32 { return max(1, min(w/1600, 1.7)) }

func npColumnGap(w float32) float32 { return max(32, min(w*0.045, 144)) }

func npLyricSize(c *ui.Context) float32 {
	w, h := c.Size()
	return max(26, min(52, min(w/48, h/25)))
}

// The angle is accumulated only while playing, so pause and resume hold
// the record in place. Reduced motion never asks for animation frames.
type vinylRotation struct {
	angle   float32
	last    time.Time
	playing bool
}

func (v *vinylRotation) step(now time.Time, playing, reduced bool) float32 {
	if playing && v.playing && !reduced && !v.last.IsZero() {
		v.angle += float32(now.Sub(v.last).Seconds()) * 12
	}
	v.last, v.playing = now, playing && !reduced
	if reduced {
		return 0
	}
	return v.angle
}

func (a *app) vinylCover(c *ui.Context, tr *music.Track, art *artwork, size float32) {
	wrap := ui.Box(c.Key("vinyl-cover")).Size(size, size).Label(T("Album Cover"))
	rotation := ui.Local(wrap, "rotation", func() vinylRotation { return vinylRotation{} })
	angle := rotation.step(c.Now(), a.pl.Playing(), c.Preferences().ReduceMotion)
	if a.pl.Playing() && !c.Preferences().ReduceMotion {
		c.After(time.Second / 30)
	}
	wrap.Children(func() {
		ui.Box(c).Fill().Radius(size/2).Background(ui.RGB(22, 22, 25)).Border(1, white.Alpha(0.18)).
			Shadow(0, 24, 52, -10, ui.RGBA(0, 0, 0, 0.5)).Rotate(angle).Children(func() {
			for ring := 0; ring < 14; ring++ {
				inset := size * (0.045 + float32(ring)*0.021)
				ui.Box(c).Absolute().Top(inset).Left(inset).Right(inset).Bottom(inset).
					Radius(size/2).Border(1, white.Alpha(0.055)).PassThrough()
			}
			label := size * 0.43
			ui.Box(c).Absolute().Top((size-label)/2).Left((size-label)/2).Size(label, label).
				Radius(label / 2).Clip().Background(ui.RGB(58, 58, 64)).Children(func() {
				if bitmap(art) != nil {
					artImage(c, bitmap(art), label/2, 1)
				} else {
					ui.Box(c).Fill().Center().Children(func() { ui.Icon(c, iconNote).Size(label*0.3, label*0.3).TextColor(white.Alpha(0.7)) })
				}
			})
			hole := max(float32(8), size*0.025)
			ui.Box(c).Absolute().Top((size-hole)/2).Left((size-hole)/2).Size(hole, hole).
				Radius(hole/2).Background(ui.RGB(12, 12, 14)).Border(1, white.Alpha(0.25)).PassThrough()
		})
	})
}
