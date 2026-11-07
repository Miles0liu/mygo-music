package player

import "mygo-music/internal/music"

// PlayNext puts tracks right after the one playing, in their order, moving
// any of them already queued; with nothing playing, it plays them.
func (p *Player) PlayNext(tracks []music.Track) { p.enqueue(tracks, true) }

// AddToQueue puts tracks at the end of what is to play, moving any of them
// already queued; with nothing playing, it plays them.
func (p *Player) AddToQueue(tracks []music.Track) { p.enqueue(tracks, false) }

func (p *Player) enqueue(tracks []music.Track, next bool) {
	if len(tracks) == 0 {
		return
	}
	if p.track == nil || p.at < 0 || p.at >= len(p.order) {
		p.PlayQueue(tracks, 0)
		return
	}
	type key struct{ source, id string }
	cur := p.queue[p.order[p.at]]
	curKey := key{cur.Source, cur.ID}
	adding := make(map[key]bool, len(tracks))
	var added []music.Track
	for _, t := range tracks {
		k := key{t.Source, t.ID}
		if k == curKey || adding[k] {
			continue
		}
		adding[k] = true
		added = append(added, t)
	}
	if len(added) == 0 {
		return
	}
	// The queue in the order it plays, without those coming again: what
	// was shuffled stays as it would have played.
	seq := make([]music.Track, 0, len(p.order)+len(added))
	at := -1
	for k, i := range p.order {
		t := p.queue[i]
		if k == p.at {
			at = len(seq)
		} else if adding[key{t.Source, t.ID}] {
			continue
		}
		seq = append(seq, t)
	}
	pos := len(seq)
	if next {
		pos = at + 1
	}
	seq = append(seq[:pos], append(added, seq[pos:]...)...)
	p.queue = seq
	p.order = make([]int, len(seq))
	for i := range p.order {
		p.order[i] = i
	}
	p.at = at
	p.track = &p.queue[at]
	p.getAhead(p.aheadQuality())
	if p.Changed != nil {
		p.Changed()
	}
}
