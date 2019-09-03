package main

import (
	"fmt"

	"github.com/vchimishuk/chub/cue"
)

type Track struct {
	Number        int    `json:"number,omitempty"`
	TotalTracks   int    `json:"totalTracks"`
	Performer     string `json:"performer,omitempty"`
	SongWriter    string `json:"songWriter,omitempty"`
	Album         string `json:"album,omitempty"`
	Title         string `json:"title,omitempty"`
	Genre         string `json:"genre,omitempty"`
	Date          string `json:"date,omitempty"`
	StartAtSample int    `json:"firstSample"`
	EndAtSample   int    `json:"lastSample,omitempty"`
}

func indexTimeToSample(sampleRate int, t *cue.Time) int {
	return (t.Min*60+t.Sec)*sampleRate + sampleRate/75*t.Frames
}

func (t *Track) SetIndexes(sampleRate int, indexes []*cue.Index) error {
	// INDEX 01 defines the beginning of this track
	for _, i := range indexes {
		if i.Number == 1 {
			t.StartAtSample = indexTimeToSample(sampleRate, i.Time)
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
			t.EndAtSample = indexTimeToSample(sampleRate, i.Time)
			return
		} else if i.Number == 1 && t.EndAtSample == 0 {
			t.EndAtSample = indexTimeToSample(sampleRate, i.Time)
		}
	}
}
