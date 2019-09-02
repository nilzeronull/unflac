package main

import (
	"fmt"

	"github.com/vchimishuk/chub/cue"
)

type Track struct {
	Number      int
	Performer   string
	SongWriter  string
	Title       string
	Genre       string
	Date        string
	FirstSample int
	LastSample  int
}

func (t *Track) OutputPath(in *Input, ext string) (path string) {
	path = fmt.Sprintf(in.TrackNumberFmt, t.Number)
	if t.Title != "" {
		path += " - " + t.Title
	}
	path += ext
	return
}

func (t *Track) SetIndexes(sampleRate int, indexes []*cue.Index) {
	time := indexes[0].Time
	t.FirstSample = (time.Min*60+time.Sec)*sampleRate + sampleRate/75*time.Frames
}
