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

func indexTimeToSample(sampleRate int, t *cue.Time) int {
	return (t.Min*60+t.Sec)*sampleRate + sampleRate/75*t.Frames
}

func (t *Track) SetIndexes(sampleRate int, indexes []*cue.Index) error {
	// INDEX 01 defines the beginning of this track
	for _, i := range indexes {
		if i.Number == 1 {
			t.FirstSample = indexTimeToSample(sampleRate, i.Time)
			return nil
		}
	}

	return fmt.Errorf("track number %d doesn't have INDEX 01", t.Number)
}

func (t *Track) SetNextIndexes(sampleRate int, indexes []*cue.Index) {
	// INDEX 00 of the next track will indicate the end of this track
	// no INDEX 00 found? the end is the beginning of the next track
	for _, i := range indexes {
		if i.Number == 0 {
			t.LastSample = indexTimeToSample(sampleRate, i.Time) - 1
			return
		} else if i.Number == 1 && t.LastSample == 0 {
			t.LastSample = indexTimeToSample(sampleRate, i.Time) - 1
		}
	}
}
