package app

import (
	"github.com/egoist/mygo/ui"
)

// accentChoices are the colors the app can take for its accent: the red it
// comes in, the color of the cover playing, or one of a few others.
var accentChoices = []struct {
	id    string
	label string
	hue   float32 // in Oklch, for the fixed colors
}{
	{"", N("NetEase Red"), 0},
	{"cover", N("Follow Cover"), 0},
	{"blue", N("Blue"), 255},
	{"green", N("Green"), 152},
	{"purple", N("Purple"), 300},
	{"orange", N("Orange"), 55},
	{"pink", N("Pink"), 350},
}

// withAccent makes p's accent the one the listener chose. Following the
// cover, a gray one, or none yet, keeps the red.
func (a *app) withAccent(p palette) palette {
	switch id := a.settings.Accent; id {
	case "":
	case "cover":
		if tint, ok := a.playingTint(); ok {
			h, s, _ := hsv(float64(tint.R)/255, float64(tint.G)/255, float64(tint.B)/255)
			if s >= 0.18 {
				// Deep enough for white on it, light enough to read on the
				// dark background, as the red is.
				s = min(max(s, 0.55), 0.8)
				v := 0.78
				if p.dark {
					v = 0.86
				}
				p.accent = hsvColor(h, s, v)
			}
		}
	default:
		for _, ch := range accentChoices {
			if ch.id == id {
				if p.dark {
					p.accent = ui.Oklch(0.66, 0.17, ch.hue)
				} else {
					p.accent = ui.Oklch(0.6, 0.18, ch.hue)
				}
			}
		}
	}
	return p
}

// playingTint is the most vivid color of the cover playing, once loaded.
func (a *app) playingTint() (ui.Color, bool) {
	tr := a.pl.Current()
	if tr == nil || tr.Album.ImageURL == "" {
		return ui.Color{}, false
	}
	// The size the player's bar shows it at, which it has loaded.
	if art := a.art.get(sized(tr.Album.ImageURL, 120), false); art != nil && art.tint.A > 0 {
		return art.tint, true
	}
	return ui.Color{}, false
}
