package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

// copyText puts text on the clipboard; tests replace it.
var copyText = func(text string) error { return mygo.Clipboard.Write(transfer.TextData(text)) }

// songMenu is three dots that open the menu of tr.
func (a *app) songMenu(c *ui.Context, tr *music.Track, size float32, color ui.Color) ui.Element {
	b := iconButton(c, iconMore, size, color, T("More"))
	b.Menu(a.songItems(tr, ""))
	return b
}

// songItems builds the menu of what can be done with tr, behind its dots
// or a right click on its row: copying its title, its title with its
// artists, or the link to its page, downloading it, opening its artists',
// album's or show's, and adding it to a playlist or taking it out of pid, the
// playlist it is listed in, if set; as far as its source can.
func (a *app) songItems(tr *music.Track, pid string) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		lib := a.trackLib(tr)
		if m.Item(T("Play")).Disabled(tr.Unavailable).Chosen() {
			a.pl.PlayTrack([]music.Track{*tr}, 0)
		}
		if m.Item(T("Play Next")).Disabled(tr.Unavailable).Chosen() {
			a.queueNext([]music.Track{*tr}, true)
		}
		if m.Item(T("Add to Queue")).Disabled(tr.Unavailable).Chosen() {
			a.queueNext([]music.Track{*tr}, false)
		}
		m.Separator()
		if m.Item(T("Copy Title")).Chosen() {
			copyText(tr.Name)
		}
		if names := tr.ArtistNames(); names != "" && m.Item(T("Copy Title and Artists")).Chosen() {
			copyText(tr.Name + " - " + names)
		}
		if tr.URL != "" {
			m.Separator()
			if m.Item(T("Copy Link")).Chosen() {
				copyText(tr.URL)
			}
		}
		if a.canDownload(tr) {
			m.Separator()
			if m.Item(T("Download…")).Chosen() {
				a.download(*tr)
			}
		}
		// Its artists', album's and show's pages, those the source has.
		linked := tr.Album.ID != "" && lib.can("album") || tr.ShowID != "" && lib.can("show")
		for _, ar := range tr.Artists {
			linked = linked || ar.ID != "" && lib.can("artist")
		}
		if linked {
			m.Separator()
		}
		if tr.ShowID != "" && lib.can("show") && m.Item(T("Go to Podcast")).Chosen() {
			a.openPage(route{page: pageShow, src: lib.id, id: tr.ShowID})
		}
		for _, ar := range tr.Artists {
			if ar.ID != "" && lib.can("artist") && m.Item(Tf("Go to Artist: %s", ar.Name)).Chosen() {
				a.openArtist(lib.id, ar.ID)
			}
		}
		if tr.Album.ID != "" && lib.can("album") && m.Item(Tf("Go to Album: %s", tr.Album.Name)).Chosen() {
			a.openPage(route{page: pageAlbum, src: lib.id, id: tr.Album.ID})
		}
		a.playlistItems(m, tr, pid)
	}
}

// queueNext puts tracks after the song playing, or at the end of the queue,
// and says so: the queue is out of sight.
func (a *app) queueNext(tracks []music.Track, next bool) {
	playable := make([]music.Track, 0, len(tracks))
	for _, tr := range tracks {
		if !tr.Unavailable {
			playable = append(playable, tr)
		}
	}
	if len(playable) == 0 {
		return
	}
	idle := a.pl.Current() == nil
	if next {
		a.pl.PlayNext(playable)
	} else {
		a.pl.AddToQueue(playable)
	}
	switch {
	case idle:
	case next && len(playable) == 1:
		a.toast = Tf("%s will play next", playable[0].Name)
	case next:
		a.toast = Tf("%d songs will play next", len(playable))
	case len(playable) == 1:
		a.toast = Tf("Added %s to the queue", playable[0].Name)
	default:
		a.toast = Tf("Added %d songs to the queue", len(playable))
	}
}
