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

// The arm occupies the space above the platter, independently of the
// rotating artwork. Reflections stay fixed in the room as the disc turns.
func (a *app) vinylCover(c *ui.Context, tr *music.Track, art *artwork, size float32) {
	wrap := ui.Box(c.Key("vinyl-cover")).Size(size, vinylHeight(size)).Label(T("Album Cover"))
	rotation := ui.Local(wrap, "rotation", func() vinylRotation { return vinylRotation{} })
	angle := rotation.step(c.Now(), a.pl.Playing(), c.Preferences().ReduceMotion)
	if a.pl.Playing() && !c.Preferences().ReduceMotion {
		c.After(time.Second / 30)
	}
	wrap.Children(func() {
		diameter := size * 0.94
		left, top := (size-diameter)/2, size*0.28
		ui.Box(c.Key("platter")).Absolute().Left(0).Top(top-left).Size(size, size).
			Radius(size/2).Background(white.Alpha(0.035)).Border(1, white.Alpha(0.07)).PassThrough()
		ui.Box(c.Key("record")).Absolute().Left(left).Top(top).Size(diameter, diameter).
			Radius(diameter/2).Border(1.5, ui.RGB(5, 6, 7)).Shadow(0, 12, 30, -5, ui.RGBA(0, 0, 0, 0.4)).
			Draw(drawVinylSurface).Children(func() {
			label := diameter * 0.68
			ui.Box(c.Key("label")).Absolute().Left((diameter-label)/2).Top((diameter-label)/2).
				Size(label, label).Radius(label / 2).Clip().Rotate(angle).
				Background(ui.RGB(30, 31, 34)).Children(func() {
				if bitmap(art) != nil {
					artImage(c, bitmap(art), label/2, 1)
				} else {
					ui.Box(c).Fill().Center().Children(func() {
						ui.Icon(c, iconNote).Size(label*0.3, label*0.3).TextColor(white.Alpha(0.6))
					})
				}
			})
			ui.Box(c).Absolute().Left((diameter-label)/2-2).Top((diameter-label)/2-2).
				Size(label+4, label+4).Radius(label/2+2).Border(2, ui.RGBA(0, 0, 0, 0.65)).PassThrough()
		})
		arm := ui.Box(c.Key("tonearm")).Absolute().Left(size*0.5).Top(size*0.025).
			Size(size*0.5, size*0.52).TransformOrigin(0, 0).PassThrough()
		target := float32(-28)
		if a.pl.Playing() {
			target = 0
		}
		if !c.Preferences().ReduceMotion {
			target = arm.AnimateTo("angle", target, ui.Animation{Duration: 350 * time.Millisecond})
		}
		arm.Rotate(target).Draw(func(p *ui.Painter, r ui.Rect) { drawTonearm(p, r, size) })
		pivot := size * 0.052
		ui.Box(c.Key("arm-pivot")).Absolute().Left(size*0.5-pivot/2).Top(size*0.025-pivot/2).
			Size(pivot, pivot).Radius(pivot/2).Background(ui.RGB(238, 239, 240)).
			Border(size*0.016, ui.RGBA(0, 0, 0, 0.18)).Shadow(0, 2, 5, 0, ui.RGBA(0, 0, 0, 0.2)).Center().Children(func() {
			ui.Box(c).Size(pivot*0.34, pivot*0.34).Radius(pivot / 2).Background(ui.RGB(194, 197, 200))
		})
	})
}

func vinylHeight(size float32) float32 { return size * 1.24 }

func drawVinylSurface(p *ui.Painter, r ui.Rect) {
	p.FillConicGradient(r, ui.ConicGradient{Angle: 25, Stops: []ui.GradientStop{
		{Color: ui.RGB(14, 15, 17), At: 0}, {Color: ui.RGB(43, 44, 46), At: 0.18},
		{Color: ui.RGB(12, 13, 15), At: 0.34}, {Color: ui.RGB(7, 8, 10), At: 0.50},
		{Color: ui.RGB(40, 41, 43), At: 0.69}, {Color: ui.RGB(15, 16, 18), At: 0.84},
		{Color: ui.RGB(14, 15, 17), At: 1},
	}}, r.W/2)
	for ring := 0; ring < 52; ring++ {
		inset := r.W * (0.014 + float32(ring)*0.0028)
		circle := ui.Rect{X: r.X + inset, Y: r.Y + inset, W: r.W - 2*inset, H: r.H - 2*inset}
		p.Stroke(circle, white.Alpha(0.027), circle.W/2, 0.65)
	}
	p.Stroke(r, ui.RGBA(0, 0, 0, 0.8), r.W/2, 2)
}

func drawTonearm(p *ui.Painter, r ui.Rect, size float32) {
	var arm ui.Path
	arm.MoveTo(r.X, r.Y).
		CubeTo(r.X+size*0.02, r.Y+size*0.10, r.X+size*0.045, r.Y+size*0.24, r.X+size*0.09, r.Y+size*0.29).
		LineTo(r.X+size*0.265, r.Y+size*0.455)
	p.StrokePath(&arm, size*0.018, ui.RGBA(0, 0, 0, 0.18))
	p.StrokePath(&arm, size*0.012, ui.RGB(241, 242, 243))
	var cartridge ui.Path
	cartridge.MoveTo(r.X+size*0.237, r.Y+size*0.454).
		LineTo(r.X+size*0.266, r.Y+size*0.425).
		LineTo(r.X+size*0.326, r.Y+size*0.484).
		LineTo(r.X+size*0.297, r.Y+size*0.514).Close()
	p.FillPath(&cartridge, ui.RGB(235, 237, 239))
	p.StrokePath(&cartridge, size*0.003, white.Alpha(0.72))
	p.Line(r.X+size*0.280, r.Y+size*0.474, r.X+size*0.305, r.Y+size*0.449, size*0.003, ui.RGB(180, 183, 188))
}
